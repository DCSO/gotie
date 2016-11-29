package main

// DCSO gotie API bindings
// Copyright (c) 2016, DCSO GmbH

import (
	"log"
	"os"
	"strings"

	"github.com/voxelbrain/goptions"

	"github.com/DCSO/gotie/v1"
)

func main() {
	var err error
	options := struct {
		ConfPath string        `goptions:"-c,--conf,description='Set non default config path'"`
		Debug    bool          `goptions:"-d,--debug,description='Print debug messages'"`
		Help     goptions.Help `goptions:"-h, --help, description='Show this help'"`

		goptions.Verbs
		IOCS struct {
			Query    string `goptions:"-q,--query, description='Query string (case insensitive)', obligatory"`
			Format   string `goptions:"-f,--format, description='Specify output format (csv|json|stix)'"`
			DataType string `goptions:"-t,--type, description='TIE IOC data type to search exclusively'"`
		} `goptions:"iocs"`
		Feed struct {
			Period   string `goptions:"-p,--period, description='Get TIE feed for given period (hourly|daily|weekly|monthly)', obligatory"`
			Format   string `goptions:"-f,--format, description='Specify output format (csv)'"`
			DataType string `goptions:"-t,--type, description='Specify a valid TIE IOC data type', obligatory"`
		} `goptions:"feed"`
		PingBack struct {
			DataType string `goptions:"-t,--type, description='Specify a valid TIE IOC data type', obligatory"`
			Value    string `goptions:"-v,--value, description='Specify a valid TIE IOC data value', obligatory"`
		} `goptions:"pingback"`
	}{ // Default values goes here
		ConfPath: getDefaultConfPath(),
	}
	goptions.ParseAndFail(&options)

	// If the user does not select any Verb we print the help and exit
	if options.IOCS.Query == "" &&
		options.Feed.Period == "" &&
		options.PingBack.DataType == "" {
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

	if options.IOCS.Query != "" {
		if options.IOCS.Format == "" {
			options.IOCS.Format = "csv"
		}
		err = gotie.PrintIOCs(options.IOCS.Query, options.IOCS.DataType,
			"&first_seen_since=2015-1-1", options.IOCS.Format)
		if err != nil {
			log.Fatal(err)
		}
	}

	if options.Feed.Period != "" {
		if options.Feed.Format == "" {
			options.Feed.Format = "csv"
		}
		err = gotie.GetPeriodFeeds(options.Feed.Period,
			strings.ToLower(options.Feed.DataType), options.Feed.Format)
		if err != nil {
			log.Fatal(err)
		}
	}

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
