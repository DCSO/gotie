package gotie

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/DCSO/bloom"
)

func TestBloomPageAggregator(t *testing.T) {
	iocsBuf, err := ioutil.TempFile("", "gotie-iocs")
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer os.Remove(iocsBuf.Name())
	bloomBuf, err := ioutil.TempFile("", "gotie-bloom")
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer os.Remove(bloomBuf.Name())

	err = WriteIOCs("google", "domainname", "&first_seen_since=2015-1-1", "json", iocsBuf)
	if err != nil {
		t.Fatalf(err.Error())
	}

	tmp := bytes.NewBufferString("")
	err = WriteIOCs("google", "domainname", "&first_seen_since=2015-1-1", "bloom", tmp)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if _, err = iocsBuf.Seek(0, 0); err != nil {
		t.Fatalf(err.Error())
	}

	agg := BloomPageAggregator{}
	agg.AddPage(tmp)
	agg.Finish(bloomBuf)

	if _, err = iocsBuf.Seek(0, 0); err != nil {
		t.Fatalf(err.Error())
	}

	jsonchan, err := GetIOCJSONInChan(iocsBuf)
	if err != nil {
		t.Fatalf(err.Error())
	}
	res, err := IOCChanCollect(jsonchan)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if len(res.Iocs) == 0 {
		t.Logf("ERROR: JSON input channel yielded no results")
		t.FailNow()
	}
	if len(res.Iocs) > 1000 {
		t.Fatalf("Expected not more than 1000 iocs (false-positive checking below)")
	}

	if _, err = bloomBuf.Seek(0, 0); err != nil {
		t.Fatalf(err.Error())
	}

	filter, err := bloom.LoadFromReader(bloomBuf, false)
	if err != nil {
		t.Fatalf(err.Error())
	}

	n := 0
	for _, ioc := range res.Iocs {
		if !filter.Check([]byte(ioc.Value)) {
			n++
		}
	}

	if filter.Check([]byte("www.bing.de")) && filter.Check([]byte("www.bing.com")) {
		t.Fatalf("More than 0.1% false-positives")
	}

	if n != 0 {
		t.Errorf("expected no negatives but got %v", n)
	}
}
