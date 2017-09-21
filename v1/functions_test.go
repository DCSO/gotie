package gotie

// DCSO gotie API bindings
// Copyright (c) 2016, DCSO GmbH

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func init() {
	// Get the test token from an environment variable.
	// This way we don't have to store it in the repository.
	AuthToken = os.Getenv("TIE_TOKEN")
}

// TestGetIocs tests all currently supported params for the iocs endpoint
// of the TIE API.
func TestGetIocs(t *testing.T) {
	var err error

	_, err = GetIOCs("google", "DomainName", "")
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}

	_, err = GetIOCs("google", "IPv4", "")
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}

	iocchan := GetIOCChan("google", "IPv4", "")
	if iocchan == nil {
		t.Logf("ERROR: no channel created")
		t.FailNow()
	}
	_, err = IOCChanCollect(iocchan)
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}

	feedchan := GetIOCPeriodFeedChan("daily", "DomainName", "")
	if feedchan == nil {
		t.Logf("ERROR: no channel created")
		t.FailNow()
	}
	_, err = IOCChanCollect(feedchan)
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}

	feedchan = GetIOCPeriodFeedChan("foobar", "DomainName", "")
	_, err = IOCChanCollect(feedchan)
	if err == nil {
		t.Log("ERROR: expected failure for invalid keyword", err)
		t.FailNow()
	} else {
		if !strings.Contains(fmt.Sprintf("%s", err), "invalid period") {
			t.Log("ERROR: expected invalid period error message", err)
			t.FailNow()
		}
	}

	err = WriteIOCs("google", "domainname", "&first_seen_since=2015-1-1", "csv", ioutil.Discard)
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}

	err = WriteIOCs("google", "domainname", "&first_seen_since=2015-1-1", "json", ioutil.Discard)
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}

	err = WriteIOCs("google", "domainname", "&first_seen_since=2015-1-1", "stix", ioutil.Discard)
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}

	err = WritePeriodFeeds("daily", "DomainName", "", "csv", ioutil.Discard)
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}

}

func TestWriteIocs(t *testing.T) {
	var info os.FileInfo
	var err error

	tmpfile, err := ioutil.TempFile("", "gotie-iocs")
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}
	defer os.Remove(tmpfile.Name())

	err = WriteIOCs("google", "domainname", "&first_seen_since=2015-1-1", "json", tmpfile)
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}

	info, err = tmpfile.Stat()
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}
	if info.Size() == 0 {
		t.Logf("ERROR: JSON file %s is empty but should contain IOC data", tmpfile.Name())
		t.FailNow()
	}
}

func TestReadWriteIocsJSON(t *testing.T) {
	var err error
	var jsonchan <-chan IOCResult
	var res *IOCQueryStruct

	tmpfile, err := ioutil.TempFile("", "gotie-iocs")
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}
	defer os.Remove(tmpfile.Name())

	err = WriteIOCs("google", "domainname", "&first_seen_since=2015-1-1", "json", tmpfile)
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}
	tmpfile.Sync()
	_, err = tmpfile.Seek(0, 0)
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}
	jsonchan, err = GetIOCJSONInChan(tmpfile)
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}
	res, err = IOCChanCollect(jsonchan)
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}
	if len(res.Iocs) == 0 {
		t.Logf("ERROR: JSON input channel yielded no results")
		t.FailNow()
	}
}
