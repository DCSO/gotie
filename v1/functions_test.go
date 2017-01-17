package gotie

// DCSO gotie API bindings
// Copyright (c) 2016, DCSO GmbH

import (
	"fmt"
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
		t.Logf("ERROR: expected failure for invalid keyword", err)
		t.FailNow()
	} else {
		if !strings.Contains(fmt.Sprintf("%s", err), "invalid period") {
			t.Logf("ERROR: expected invalid period error message", err)
			t.FailNow()
		}
	}

	err = PrintIOCs("google", "domainname", "&first_seen_since=2015-1-1", "csv")
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}

	err = PrintIOCs("google", "domainname", "&first_seen_since=2015-1-1", "json")
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}

	err = PrintIOCs("google", "domainname", "&first_seen_since=2015-1-1", "stix")
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}

	err = PrintPeriodFeeds("daily", "DomainName", "", "csv")
	if err != nil {
		t.Logf("ERROR: %v", err)
		t.FailNow()
	}

}
