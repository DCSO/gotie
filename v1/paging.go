package gotie

// DCSO gotie API bindings
// Copyright (c) 2017, DCSO GmbH

import (
	"encoding/json"
//	"encoding/xml"
//	"errors"
//	"fmt"
	"io"
	"bytes"
//	"regexp"
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
	Params   IOCParams       `json:"params"`
	IOCs     []IOC           `json:"iocs"`
	has_more bool            `json:"has_more"`
}

type JSONPageAggregator struct {
	IOCs   []IOC             `json:"iocs"`
	Params IOCParams         `json:"params"`
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