// Package gotie provides high-level bindings and a simple command line client
// for the DCSO Threat Intelligence Engine (TIE) API.
package gotie

// DCSO gotie API bindings
// Copyright (c) 2016, DCSO GmbH

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tent/http-link-go"
)

const (
	MAX_RETRIES                = 3
	WAIT_FAIL_DURATION_SECONDS = 5
	WAIT_DURATION_MILLISECONDS = 100
)

var (
	// Debug turns on verbose output
	Debug bool
	// IOCLimit defines the maximum number of IOCs to query per request
	IOCLimit = 1000
	// AuthToken can be generated in the TIE webinterface and is used for authentication
	AuthToken string

	apiURL      = "https://tie.dcso.de/api/v1/"
	pingbackURL = "https://tie-fb.xyz/api/v1/"
	client      = http.Client{}
)

func IOCQuery(baseuri string, outchan chan IOCResult, data IOCQueryStruct) {
	var outData IOCQueryStruct
	var msg apiMessage
	var offset int
	// The TIE API uses a paging mechanism to return all matched IOCs.
	// So we have to loop until the API tells us to stop.
	for data.HasMore == true {

		uri := baseuri + "&offset=" + strconv.Itoa(offset)

		if Debug {
			log.Println("Asking API for more IOCs at offset", strconv.Itoa(offset), uri)
		}

		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			outchan <- IOCResult{IOC: nil, Error: err}
			break
		}
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Authorization", "Bearer "+AuthToken)

		resp, err := client.Do(req)
		if err != nil {
			outchan <- IOCResult{IOC: nil, Error: err}
			close(outchan)
			return
		}
		defer resp.Body.Close()

		if Debug {
			dump, _ := httputil.DumpResponse(resp, true)
			log.Println(string(dump))
		}

		if resp.StatusCode != 200 {
			err = json.NewDecoder(resp.Body).Decode(&msg)
			if err != nil {
				outchan <- IOCResult{IOC: nil, Error: err}
				close(outchan)
				return
			}
			errStr := fmt.Sprintf("TIE returned an error: %v %v", msg.Message, msg.Errors)
			this_err := errors.New(errStr)
			outchan <- IOCResult{IOC: nil, Error: this_err}
			close(outchan)
			return
		}

		err = json.NewDecoder(resp.Body).Decode(&data)
		if err != nil {
			outchan <- IOCResult{IOC: nil, Error: err}
			close(outchan)
			return
		}
		_, err = json.Marshal(data.Iocs)
		if err != nil {
			outchan <- IOCResult{IOC: nil, Error: err}
			close(outchan)
			return
		}
		for i := range data.Iocs {
			outchan <- IOCResult{IOC: &data.Iocs[i], Error: nil}
		}
		outData.Params = data.Params
		outData.HasMore = data.HasMore
		offset += IOCLimit
	}
	close(outchan)
}

func GetIOCChan(query string, dataType string, extraArgs string) <-chan IOCResult {
	data := IOCQueryStruct{HasMore: true}
	outchan := make(chan IOCResult)

	uri := apiURL +
		"iocs?data_type=" + strings.ToLower(dataType) +
		"&ivalue=" + query +
		"&limit=" + strconv.Itoa(IOCLimit) +
		"&date_format=rfc3339" +
		extraArgs

	go IOCQuery(uri, outchan, data)

	return outchan
}

func GetIOCPeriodFeedChan(feedPeriod string, dataType string, extraArgs string) <-chan IOCResult {
	data := IOCQueryStruct{HasMore: true}
	outchan := make(chan IOCResult)

	uri := apiURL +
		"iocs/feed/" + feedPeriod + "?data_type=" + strings.ToLower(dataType) +
		"&limit=" + strconv.Itoa(IOCLimit) +
		"&date_format=rfc3339" +
		extraArgs

	go IOCQuery(uri, outchan, data)

	return outchan
}

func GetIOCJSONInChan(reader io.Reader) (<-chan IOCResult, error) {
	var iocs struct {
		IOCs []IOC
	}
	outchan := make(chan IOCResult)

	err := json.NewDecoder(reader).Decode(&iocs)
	if err != nil {
		return nil, err
	}

	go func() {
		for i, _ := range iocs.IOCs {
			outchan <- IOCResult{IOC: &iocs.IOCs[i], Error: nil}
		}
		close(outchan)
	}()

	return outchan, nil
}

func IOCChanCollect(inchan <-chan IOCResult) (*IOCQueryStruct, error) {
	var outData IOCQueryStruct

	for ioc := range inchan {
		if ioc.Error != nil {
			return &outData, ioc.Error
		}
		outData.Iocs = append(outData.Iocs, *(ioc.IOC))
	}

	return &outData, nil
}

// GetIOCs allows queries for TIE IOC objects with "query" being a case
// insensitive string to search for.
func GetIOCs(query string, dataType string, extraArgs string) (*IOCQueryStruct, error) {
	return IOCChanCollect(GetIOCChan(query, dataType, extraArgs))
}

// GetIOCPeriodFeeds gets file based feeds for the given period and IOC data type.
// Feed types are, for example, 'hourly', 'daily', 'weekly' or 'monthly'.
func GetIOCPeriodFeeds(feedPeriod string, dataType string, extraArgs string) (*IOCQueryStruct, error) {
	return IOCChanCollect(GetIOCPeriodFeedChan(feedPeriod, dataType, extraArgs))
}

func WriteIOCs(query, dataType, extraArgs, outputFormat string, dest io.Writer) error {
	var uri string
	var acceptHdr string
	var offset int
	var agg PageContentAggregator

	switch outputFormat {
	case "bloom":
		acceptHdr = "application/bloom"
		agg = &BloomPageAggregator{}
	case "csv":
		acceptHdr = "text/csv"
		agg = &PaginatedRawPageAggregator{}
	case "json":
		acceptHdr = "application/json"
		agg = &JSONPageAggregator{}
	case "stix":
		acceptHdr = "text/xml"
		agg = &PaginatedRawPageAggregator{}
	default:
		return errors.New("Unsupported output format requested: " + outputFormat)
	}

	// The TIE API uses a paging mechanism to return all matched IOCs. So we have
	//  to loop until the API tells us to stop.
	for {
		uri = apiURL +
			"iocs?data_type=" + strings.ToLower(dataType) +
			"&ivalue=" + query +
			"&limit=" + strconv.Itoa(IOCLimit) +
			"&offset=" + strconv.Itoa(offset) +
			"&date_format=rfc3339" +
			extraArgs

		if Debug {
			log.Println("Asking API for more IOCs at offset", strconv.Itoa(offset), uri)
		}

		body, fetchLink, err := fetchIOCs(uri, acceptHdr)
		if err != nil {
			return fmt.Errorf("fetch iocs: %v", err)
		}

		err = agg.AddPage(body)
		if err != nil {
			return err
		}
		body.Close()

		// Due to the various output types we can not marshal and check the HasMore
		// header here. Fortunately TIE also returns a Link header for pagination.
		// Ref: https://tie.dcso.de/api-docs/api/v1/pagination.html
		var links []link.Link
		found_next := false
		if fetchLink != "" {
			links, err = link.Parse(fetchLink)
			if err != nil {
				return err
			}
			for _, l := range links {
				if l.Rel == "next" {
					found_next = true
				}
			}
		}
		if !found_next {
			break
		}

		offset += IOCLimit
	}

	agg.Finish(dest)
	return nil
}

// fetchIOCs using mustFetchIOCs, retry on Server Errors (>= 500).
func fetchIOCs(uri, acceptHdr string) (body io.ReadCloser, link string, err error) {
	var code int
	var waitFail time.Duration = WAIT_FAIL_DURATION_SECONDS * time.Second

	<-time.After(WAIT_DURATION_MILLISECONDS * time.Millisecond)

	for i := 0; i < MAX_RETRIES; i++ {
		body, code, link, err = mustFetchIOCs(uri, acceptHdr)
		if code >= 500 {
			log.Printf("Status code %v: retrying in %v...", code, waitFail)
			<-time.After(waitFail)
			waitFail *= 2
			if body != nil {
				body.Close()
			}
		} else {
			break
		}
	}

	return
}

func mustFetchIOCs(uri, acceptHdr string) (body io.ReadCloser, code int, link string, err error) {
	var msg apiMessage

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return
	}

	req.Header.Add("Accept", acceptHdr)
	req.Header.Add("Authorization", "Bearer "+AuthToken)

	resp, err := client.Do(req)
	if err != nil {
		return
	}

	if code = resp.StatusCode; code != 200 {
		err = json.NewDecoder(resp.Body).Decode(&msg)
		if err != nil {
			return
		}
		return nil, code, "", fmt.Errorf("TIE returned an error: %v %v", msg.Message, msg.Errors)
	}

	return resp.Body, 200, resp.Header.Get("Link"), nil
}

func WritePeriodFeeds(feedPeriod string, dataType string, extraArgs string, outputFormat string, dest io.Writer) error {
	var msg apiMessage

	req, err := http.NewRequest("GET",
		apiURL+"iocs/feed/"+feedPeriod+"/"+strings.ToLower(dataType)+
			"?limit="+strconv.Itoa(IOCLimit)+
			"&date_format=rfc3339"+
			extraArgs,
		nil)
	if err != nil {
		return err
	}

	switch outputFormat {
	case "csv":
		req.Header.Add("Accept", "text/csv")
	case "json":
		req.Header.Add("Accept", "application/json")
	case "stix":
		req.Header.Add("Accept", "text/xml")
	default:
		return errors.New("Unsupported output format requested: " + outputFormat)
	}

	req.Header.Add("Authorization", "Bearer "+AuthToken)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if Debug {
		log.Println("Tried URL:" + apiURL + "iocs/feed/" + feedPeriod + "/" + dataType)
		dump, _ := httputil.DumpResponse(resp, true)
		log.Println(string(dump))
		log.Println("Requested outputFormat:", outputFormat)
	}

	if resp.StatusCode != 200 {
		err = json.NewDecoder(resp.Body).Decode(&msg)
		if err != nil {
			return err
		}
		errStr := fmt.Sprintf("TIE returned an error: %v %v", msg.Message, msg.Errors)
		return errors.New(errStr)
	}

	_, err = io.Copy(dest, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// PrintIOCs allows queries for TIE IOC objects with "query" being a case
// insensitive string to search for. The results are printed to stdout.
func PrintIOCs(query, dataType, extraArgs, outputFormat string) error {
	return WriteIOCs(query, dataType, extraArgs, outputFormat, os.Stdout)
}

// PrintPeriodFeeds gets file based feeds for the given period and IOC data type.
// Valid outputFormats are: "csv" (default), "json" and "stix". Results are printed to stdout.
func PrintPeriodFeeds(feedPeriod string, dataType string, extraArgs string, outputFormat string) error {
	return WritePeriodFeeds(feedPeriod, dataType, extraArgs, outputFormat, os.Stdout)
}

// PingBackCall allows to tell the TIE about observed hits for IOCs
func PingBackCall(dataType string, value string, token string) error {
	currentDate := time.Now().UTC().Format(time.RFC3339)

	form := url.Values{}
	form.Add("data_type", dataType)
	form.Add("value", value)
	form.Add("seen", currentDate)

	req, err := http.NewRequest("POST", pingbackURL+"submit", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if Debug {
		log.Println("Tried URL:" + apiURL + "submit")
		log.Println("Requested body data:", form.Encode())
	}

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return err
	}
	fmt.Println(string(dump))

	return nil
}
