package gotie

import (
  "bytes"
  "encoding/json"
  "errors"
  "fmt"
  "io"
  "log"
  "net/http"
  "strconv"
  "strings"
  "time"

  "github.com/tent/http-link-go"
)

type MimeType string

func NewMimeType(outputFormat string) (t MimeType, err error) {
  switch outputFormat {
	case "bloom":
		return BLOOM, nil
	case "csv":
		return CSV, nil
	case "json":
		return JSON, nil
	case "stix":
		return STIX, nil
	default:
		return t, errors.New("Unsupported output format requested: " + outputFormat)
	}
}

func (t MimeType) Aggregator() PageContentAggregator {
  switch t {
  case BLOOM:
    return &BloomPageAggregator{}
  case CSV:
    return &PaginatedRawPageAggregator{}
  case JSON:
    return &JSONPageAggregator{}
  case STIX:
    return &PaginatedRawPageAggregator{}
  default:
    panic(fmt.Sprintf("unknown type %v", t))
  }
}

func (t MimeType) String() string {
	return string(t)
}

const (
	JSON  MimeType = "application/json"
	CSV   MimeType = "text/csv"
	BLOOM MimeType = "application/bloom"
	STIX  MimeType = "text/xml"
)

type Request interface {
	Url() string
}

type FeedRequest struct {
  Request

  FeedPeriod string
  DataType string
  ExtraArgs string
  MimeType
}

func (r *FeedRequest) Url() string {
  return apiURL+"iocs/feed/"+r.FeedPeriod+"/"+strings.ToLower(r.DataType)+
    "?limit="+strconv.Itoa(IOCLimit)+
    "&date_format=rfc3339"+
    r.ExtraArgs
}

type IOCRequest struct {
  Request

  Query string
  DataType string
  ExtraArgs string
  MimeType
}

func (r *IOCRequest) Url() string {
  return apiURL +
  			"iocs?data_type=" + strings.ToLower(r.DataType) +
  			"&ivalue=" + r.Query +
  			"&limit=" + strconv.Itoa(IOCLimit) +
  			"&date_format=rfc3339" +
  			r.ExtraArgs

}

// Do request and write result into w.
func Do(r Request, t MimeType, w io.Writer) (err error) {
  agg := t.Aggregator()
  defer agg.Finish(w)

  err = doRequest(r, t, func(buf io.Reader) error {
    agg.AddPage(buf)
    return nil
  })

  return
}

func DoCh(r Request, t MimeType, ch chan<- IOCResult) {
  var iocResult IOCResult

  err := doRequest(r, t, func(buf io.Reader) error {
    dec := json.NewDecoder(buf)
    if err := dec.Decode(&iocResult.IOC); err != nil {
      ch <- IOCResult{IOC: nil, Error: err}
      close(ch)
      return nil
    }

    ch <- iocResult

    return nil
  })

  if err != nil {
    ch <- IOCResult{IOC: nil, Error: err}
  }

  close(ch)

	return
}

func doRequest(r Request, t MimeType, f func(io.Reader) error) (err error) {
  url := r.Url()

  buf := bytes.NewBuffer([]byte{})

  for {
    if Debug {
      log.Printf("doRequest: GET %v", url)
    }

    next, err := doIteration(url, t, buf)
    if err != nil {
      return err
    }

    if err := f(buf); err != nil {
      return fmt.Errorf("f: %v", err)
    }

    if next != nil {
      url = next.URI
    } else {
      break
    }

    buf.Reset()
  }

	return
}

func doIteration(url string, t MimeType, w io.Writer) (next *link.Link, err error) {
  var code int
  var waitFail time.Duration = WAIT_FAIL_DURATION_SECONDS * time.Second

	<-time.After(WAIT_DURATION_MILLISECONDS * time.Millisecond)

  for i := 0; i < MAX_RETRIES; i++ {
    code, next, err = mustDoIteration(url, t, w)
    if code >= 500 {
      log.Printf("Status code %v (%v): retrying in %v...", code, err, waitFail)
			<-time.After(waitFail)
			waitFail *= 2
    } else if err != nil {
      return nil, err
    } else {
      break
    }
  }

  return
}

func mustDoIteration(url string, t MimeType, w io.Writer) (code int, next *link.Link, err error) {
  req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}

	req.Header.Add("Accept", t.String())
  req.Header.Add("Authorization", "Bearer "+AuthToken)

  if Debug {
    log.Printf("GET %v", url)
  }

  resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

  // Error handling for various Content types
	if code = resp.StatusCode; code > 299 {
		log.Printf("resp header: %v", resp.Header)

		if t := resp.Header.Get("Content-Type"); strings.Contains(t, string(JSON)) {
      var msg apiMessage

			err = json.NewDecoder(resp.Body).Decode(&msg)
			if err != nil {
				return
			}
			return code, nil, fmt.Errorf("TIE returned an error: %v %v", msg.Message, msg.Errors)
		} else {
			buf := &bytes.Buffer{}
			io.Copy(buf, resp.Body)
			return code, nil, fmt.Errorf("TIE returned an error: %v", buf.String())
		}
	}

  // Body processing
  if _, err = io.Copy(w, resp.Body); err != nil {
    return
  }

  // Next link parsing
  //
  // Due to the various output types we can not marshal and check the HasMore
  // header here. Fortunately TIE also returns a Link header for pagination.
  // Ref: https://tie.dcso.de/api-docs/api/v1/pagination.html
  if fetchLink := resp.Header.Get("link"); fetchLink != "" {
    links, err := link.Parse(fetchLink)
    if err != nil {
      return code, nil, fmt.Errorf("parse link: %v", err)
    }
    for _, l := range links {
      if l.Rel == "next" {
        next = &l
        break
      }
    }
  }

  return
}
