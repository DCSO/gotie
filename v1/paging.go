package gotie

// DCSO gotie API bindings
// Copyright (c) 2017, DCSO GmbH

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/DCSO/bloom"
)

type PageContentAggregator interface {
	AddPage(io.Reader) error
	Finish(io.Writer) error
	Reset()
}

type PaginatedRawPageAggregator struct {
	buf bytes.Buffer
}

func (pa *PaginatedRawPageAggregator) AddPage(reader io.Reader) error {
	_, err := pa.buf.ReadFrom(reader)
	return err
}

func (pa *PaginatedRawPageAggregator) Finish(writer io.Writer) error {
	_, err := pa.buf.WriteTo(writer)
	return err
}

func (pa *PaginatedRawPageAggregator) Reset() {
	pa.buf.Reset()
}

type JSONTopLevelResponse struct {
	Params   IOCParams `json:"params"`
	IOCs     []IOC     `json:"iocs"`
	has_more bool
}

type JSONPageAggregator struct {
	IOCs   []IOC     `json:"iocs"`
	Params IOCParams `json:"params"`
}

func (pa *JSONPageAggregator) AddPage(reader io.Reader) error {
	var tlr JSONTopLevelResponse

	err := json.NewDecoder(reader).Decode(&tlr)
	if err != nil {
		return err
	}

	pa.IOCs = append(pa.IOCs, tlr.IOCs...)
	pa.Params = tlr.Params

	return err
}

func (pa *JSONPageAggregator) Finish(writer io.Writer) error {
	var tlr JSONTopLevelResponse

	tlr.Params = pa.Params
	tlr.IOCs = pa.IOCs
	tlr.has_more = false
	tlr.Params.Offset = 0
	tlr.Params.Limit = len(tlr.IOCs)

	err := json.NewEncoder(writer).Encode(&tlr)
	return err
}

func (pa *JSONPageAggregator) Reset() {
	*pa = JSONPageAggregator{}
}

type BloomPageAggregator struct {
	f *bloom.BloomFilter
}

func (ba *BloomPageAggregator) AddPage(reader io.Reader) (err error) {
	f, err := bloom.LoadFromReader(reader, false)
	if err != nil {
		return fmt.Errorf("load bloom from reader: %v", err)
	}

	if ba.f == nil {
		ba.f = f
		return
	}

	if err = ba.f.Join(f); err != nil {
		return fmt.Errorf("join filters: %v", err)
	}

	return
}

func (ba *BloomPageAggregator) Finish(writer io.Writer) error {
	if ba.f == nil {
		if Debug {
			log.Printf("Writing empty bloom filter")
		}
		empty := bloom.Initialize(0, 0.01)
		ba.f = &empty
	}

	return ba.f.Write(writer)
}

func (ba *BloomPageAggregator) Reset() {
	ba.f = nil
}

// to be implemented
/*
type STIXPageAggregator struct {
	buf bytes.Buffer
}

func (pa *STIXPageAggregator) AddPage(reader io.Reader) error {
	return nil
}

func (pa *STIXPageAggregator) Finish(writer io.Writer) error {
	return nil
}

func (pa *STIXPageAggregator) Reset() {
	pa.buf.Reset()
}
*/
