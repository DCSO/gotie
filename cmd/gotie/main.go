package main

// DCSO gotie API bindings
// Copyright (c) 2016, DCSO GmbH

import (
	"log"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/voxelbrain/goptions"

	"github.com/DCSO/gotie/v1"
)

type IOCSParams struct {
	Query            string `goptions:"-q,--query, description='Query string (case insensitive)'"`
	Format           string `goptions:"-f,--format, description='Specify output format (bloom|csv|json|stix)'"`
	BloomP           string `goptions:"--bloom-p, description='Bloom output: false positive rate'"`
	Category         string `goptions:"-c,--category, description='specify comma-separated IOC categories'"`
	DataType         string `goptions:"-t,--type, description='TIE IOC data type to search exclusively'"`
	Severity         string `goptions:"--severity, description='Specify severity (can be a range)'"`
	Confidence       string `goptions:"--confidence, description='Specify confidence (can be a range)'"`
	Limit            string `goptions:"--limit, description='Specify limit of IOCs to query at once'"`
	Updated_since    string `goptions:"--updated-since, description='Limit to IOCs updated since the given date'"`
	Updated_until    string `goptions:"--updated-until, description='Limit to IOCs updated until the given date'"`
	Created_since    string `goptions:"--created-since, description='Limit to IOCs created since the given date'"`
	Created_until    string `goptions:"--created-until, description='Limit to IOCs created until the given date'"`
	First_seen_since string `goptions:"--first-seen-since, description='Limit to IOCs first seen since the given date'"`
	First_seen_until string `goptions:"--first-seen-until, description='Limit to IOCs first seen until the given date'"`
	Last_seen_since  string `goptions:"--last-seen-since, description='Limit to IOCs last seen since the given date'"`
	Last_seen_until  string `goptions:"--last-seen-until, description='Limit to IOCs last seen until the given date'"`
}

type FeedParams struct {
	Period           string `goptions:"-p,--period, description='Get TIE feed for given period (hourly|daily|weekly|monthly)', obligatory"`
	Format           string `goptions:"-f,--format, description='Specify output format (csv|json|stix)'"`
	Category         string `goptions:"-c,--category, description='specify comma-separated IOC categories'"`
	DataType         string `goptions:"-t,--type, description='Specify a valid TIE IOC data type', obligatory"`
	Severity         string `goptions:"--severity, description='Specify severity (can be a range)'"`
	Confidence       string `goptions:"--confidence, description='Specify confidence (can be a range)'"`
	Limit            string `goptions:"--limit, description='Specify limit of IOCs to query at once'"`
	Updated_since    string `goptions:"--updated-since, description='Limit to IOCs updated since the given date'"`
	Updated_until    string `goptions:"--updated-until, description='Limit to IOCs updated until the given date'"`
	Created_since    string `goptions:"--created-since, description='Limit to IOCs created since the given date'"`
	Created_until    string `goptions:"--created-until, description='Limit to IOCs created until the given date'"`
	First_seen_since string `goptions:"--first-seen-since, description='Limit to IOCs first seen since the given date'"`
	First_seen_until string `goptions:"--first-seen-until, description='Limit to IOCs first seen until the given date'"`
	Last_seen_since  string `goptions:"--last-seen-since, description='Limit to IOCs last seen since the given date'"`
	Last_seen_until  string `goptions:"--last-seen-until, description='Limit to IOCs last seen until the given date'"`
}

type PingBackParams struct {
	DataType string `goptions:"-t,--type, description='Specify a valid TIE IOC data type', obligatory"`
	Value    string `goptions:"-v,--value, description='Specify a valid TIE IOC data value', obligatory"`
}

type Params interface{}

func parseTime(timeString string) (time.Time, error) {
	var mtime time.Time
	var err error

	formats := []string{"2006-01-02",
		"2006-01-02 15:04",
		"2006/01/02",
		"2006/01/02 15:04",
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		time.RFC3339Nano}

	for _, format := range formats {
		mtime, err = time.Parse(format, timeString)
		if err == nil {
			return mtime, err
		}
	}

	return mtime, err
}

func buildArgs(params Params, typestr string, debug bool) string {
	sharedParams := map[string]bool{
		"Severity":         true,
		"Confidence":       true,
		"Category":         true,
		"Updated_since":    true,
		"Updated_until":    true,
		"Created_since":    true,
		"Created_until":    true,
		"First_seen_since": true,
		"First_seen_until": true,
		"Last_seen_since":  true,
		"Last_seen_until":  true,
	}
	var p reflect.Value

	if typestr == "iocs" {
		iocparams := params.(IOCSParams)
		p = reflect.ValueOf(&iocparams).Elem()
	} else if typestr == "feed" {
		feedparams := params.(FeedParams)
		p = reflect.ValueOf(&feedparams).Elem()
	}
	values := []string{""}
	for i := 0; i < p.NumField(); i++ {
		field_name := p.Type().Field(i).Name
		field_value := p.Field(i).Interface().(string)
		if sharedParams[field_name] && field_value != "" {
			var err error
			outval := field_value
			matched, _ := regexp.MatchString("_(since|until)$", field_name)
			if matched {
				var timeval time.Time
				timeval, err = parseTime(field_value)
				if err != nil {
					log.Fatal(err)
				}
				outval = timeval.Format("2006-01-02T15:04:05Z")
			}
			argPair := url.QueryEscape(strings.ToLower(field_name)) + "=" + url.QueryEscape(outval)
			values = append(values, argPair)
		} else {
			if debug {
				log.Printf("unknown or empty parameter %s skipped\n", field_name)
			}
		}
	}
	return strings.Join(values, "&")
}

type Options struct {
	ConfPath string        `goptions:"-c,--conf,description='Set non default config path'"`
	Debug    bool          `goptions:"-d,--debug,description='Print debug messages'"`
	Help     goptions.Help `goptions:"-h, --help, description='Show this help'"`

	goptions.Verbs
	IOCS     IOCSParams     `goptions:"iocs"`
	Feed     FeedParams     `goptions:"feed"`
	PingBack PingBackParams `goptions:"pingback"`
}

func main() {
	var err error
	options := Options{
		ConfPath: getDefaultConfPath(),
		IOCS: IOCSParams{
			Format:           "csv",
			BloomP:           "0.001",
			First_seen_since: "2015-01-01",
			Limit:            "1000",
		},
		Feed: FeedParams{
			Format: "csv",
			Limit:  "1000",
		},
	}
	goptions.ParseAndFail(&options)

	// If the user does not select any Verb we print the help and exit
	if options.Verbs == "" {
		goptions.PrintHelp()
		os.Exit(1)
	}

	if options.Debug {
		log.Println("DEBUG mode activated")
		gotie.Debug = true
	}

	// Load the config file and fill the CONF stuct
	err = loadConfig(options.ConfPath)
	if err != nil {
		panic(err)
	}
	gotie.AuthToken = CONF.TieToken

	if options.Verbs == "iocs" {
		var s int64
		s, err = strconv.ParseInt(options.IOCS.Limit, 10, 32)
		if err == nil {
			gotie.IOCLimit = int(s)
		} else {
			log.Fatal(err)
		}
		if gotie.Debug {
			log.Println(buildArgs(options.IOCS, "iocs", options.Debug))
		}

		bloomP, err := strconv.ParseFloat(options.IOCS.BloomP, 64)
		if err != nil {
			log.Fatal(err)
		}

		err = gotie.PrintIOCs(options.IOCS.Query, options.IOCS.DataType,
			buildArgs(options.IOCS, "iocs", options.Debug), options.IOCS.Format,
			bloomP)
		if err != nil {
			log.Fatal(err)
		}
	}

	if options.Verbs == "feed" {
		var s int64
		s, err = strconv.ParseInt(options.Feed.Limit, 10, 32)
		if err == nil {
			gotie.IOCLimit = int(s)
		} else {
			log.Fatal(err)
		}
		err = gotie.PrintPeriodFeeds(options.Feed.Period,
			strings.ToLower(options.Feed.DataType),
			buildArgs(options.IOCS, "iocs", options.Debug), options.Feed.Format)
		if err != nil {
			log.Fatal(err)
		}
	}

	if options.Verbs == "pingback" {
		if options.PingBack.DataType != "" && options.PingBack.Value != "" {
			if CONF.PingBackToken == "" {
				log.Fatal("Please set a valid pingback_token in your config file!")
			}
			err = gotie.PingBackCall(options.PingBack.DataType, options.PingBack.Value,
				CONF.PingBackToken)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

}
