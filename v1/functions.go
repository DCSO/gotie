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
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/peterhellberg/link"
)

var (
	// Debug turns on verbose output
	Debug bool
	// IOCLimit defines the maximum number of IOCs to query per request
	IOCLimit = 10000
	// AuthToken can be generated in the TIE webinterface and is used for authentication
	AuthToken string

	apiURL      = "https://tie.dcso.de/api/v1/"
	pingbackURL = "https://tie-fb.xyz/api/v1/"
	client      = http.Client{}
)

// GetIOCs allows queries for TIE IOC objects with "query" beeing a case
// insensitive string to search for.
func GetIOCs(query string, dataType string, extraArgs string) (*IOCQueryStruct, error) {
	var uri string
	var offset int
	var msg apiMessage
	var outData IOCQueryStruct
	data := IOCQueryStruct{HasMore: true}

	// The TIE API uses a paging mechanism to return all matched IOCs. So we have
	//  to loop until the API tells us to stop.
	for data.HasMore == true {
		uri = apiURL +
			"iocs?data_type=" + strings.ToLower(dataType) +
			"&ivalue=" + query +
			"&limit=" + strconv.Itoa(IOCLimit) +
			"&offset=" + strconv.Itoa(offset) +
			"&date_format=rfc3339" +
			extraArgs

		if Debug {
			fmt.Println("Asking API for more IOCs at offset", strconv.Itoa(offset), uri)
		}

		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			return &outData, err
		}
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Authorization", "Bearer "+AuthToken)

		resp, err := client.Do(req)
		if err != nil {
			return &outData, err
		}
		defer resp.Body.Close()

		if Debug {
			dump, _ := httputil.DumpResponse(resp, true)
			fmt.Println(string(dump))
		}

		if resp.StatusCode != 200 {
			err = json.NewDecoder(resp.Body).Decode(&msg)
			if err != nil {
				return &outData, err
			}
			errStr := fmt.Sprintf("TIE returned an error: %v %v", msg.Message, msg.Errors)
			return &outData, errors.New(errStr)
		}

		err = json.NewDecoder(resp.Body).Decode(&data)
		if err != nil {
			return &outData, err
		}
		_, err = json.Marshal(data.Iocs)
		if err != nil {
			return &outData, err
		}
		outData.Iocs = append(outData.Iocs, data.Iocs...)
		outData.Params = data.Params
		outData.HasMore = data.HasMore
		offset += IOCLimit
	}
	return &outData, nil
}

// PrintIOCs allows queries for TIE IOC objects with "query" beeing a case
// insensitive string to search for. The results are printed to stdout.
func PrintIOCs(query string, dataType string, extraArgs string, outputFormat string) error {
	var uri string
	var offset int
	var msg apiMessage

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
			fmt.Println("Asking API for more IOCs at offset", strconv.Itoa(offset), uri)
		}

		req, err := http.NewRequest("GET", uri, nil)
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
			return errors.New("Not supported output format requested: " + outputFormat)
		}

		req.Header.Add("Authorization", "Bearer "+AuthToken)

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			err = json.NewDecoder(resp.Body).Decode(&msg)
			if err != nil {
				return err
			}
			errStr := fmt.Sprintf("TIE returned an error: %v %v", msg.Message, msg.Errors)
			return errors.New(errStr)
		}

		_, err = io.Copy(os.Stdout, resp.Body)
		if err != nil {
			return err
		}

		// Due to the various output types we can not marshal and check the HasMore
		// header here. Fortunately TIE also returns a Link header for pagination.
		// Ref: https://tie.dcso.de/api-docs/api/v1/pagination.html
		group := link.ParseResponse(resp)
		if group["next"] == nil {
			break
		}

		offset += IOCLimit
	}
	return nil
}

// GetPeriodFeeds gets file based feeds for the given period and IOC data type.
// Valid outputFormats are: "csv" (default), "json" and "stix" and print to stdout
func GetPeriodFeeds(feedPeriod string, dataType string, outputFormat string) error {
	var msg apiMessage

	req, err := http.NewRequest("GET",
		apiURL+"iocs/feed/"+feedPeriod+"/"+strings.ToLower(dataType)+"?limit="+strconv.Itoa(IOCLimit),
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
		return errors.New("Not supported output format requested: " + outputFormat)
	}

	req.Header.Add("Authorization", "Bearer "+AuthToken)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if Debug {
		fmt.Println("Tried URL:" + apiURL + "iocs/feed/" + feedPeriod + "/" + dataType)
		dump, _ := httputil.DumpResponse(resp, true)
		fmt.Println(string(dump))
		fmt.Println("Requested outputFormat:", outputFormat)
	}

	if resp.StatusCode != 200 {
		err = json.NewDecoder(resp.Body).Decode(&msg)
		if err != nil {
			return err
		}
		errStr := fmt.Sprintf("TIE returned an error: %v %v", msg.Message, msg.Errors)
		return errors.New(errStr)
	}

	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		return err
	}

	return nil
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
		fmt.Println("Tried URL:" + apiURL + "submit")
		fmt.Println("Requested body data:", form.Encode())
	}

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return err
	}
	fmt.Println(string(dump))

	return nil
}
