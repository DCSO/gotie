// Package gotie provides high-level bindings and a simple command line client
// for the DCSO Threat Intelligence Engine (TIE) API.
package gotie

// DCSO gotie API bindings
// Copyright (c) 2016, DCSO GmbH

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
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

func GetIOCChan(query string, dataType string, extraArgs string) <-chan IOCResult {
	outchan := make(chan IOCResult)

	request := &IOCRequest{
		Query:     query,
		DataType:  dataType,
		ExtraArgs: extraArgs,
		MimeType:  JSON,
	}

	go DoCh(request, JSON, outchan)

	return outchan
}

func GetIOCPeriodFeedChan(feedPeriod string, dataType string, extraArgs string) <-chan IOCResult {
	outchan := make(chan IOCResult)

	request := &FeedRequest{
		FeedPeriod: feedPeriod,
		DataType:   dataType,
		ExtraArgs:  extraArgs,
		MimeType:   JSON,
	}

	go DoCh(request, JSON, outchan)

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
	tmp := GetIOCPeriodFeedChan(feedPeriod, dataType, extraArgs)
	return IOCChanCollect(tmp)
}

func WriteIOCs(query, dataType, extraArgs, outputFormat string, dest io.Writer) (err error) {
	t, err := NewMimeType(outputFormat)
	if err != nil {
		return
	}

	request := &IOCRequest{
		Query:     query,
		DataType:  dataType,
		ExtraArgs: extraArgs,
		MimeType:  t,
	}

	return Do(request, request.MimeType, dest)
}

func WritePeriodFeeds(feedPeriod string, dataType string, extraArgs string, outputFormat string, dest io.Writer) error {
	t, err := NewMimeType(outputFormat)
	if err != nil {
		return err
	}

	request := &FeedRequest{
		FeedPeriod: feedPeriod,
		DataType:   dataType,
		ExtraArgs:  extraArgs,
		MimeType:   t,
	}

	return Do(request, request.MimeType, dest)
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
