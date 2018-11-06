package gotie

// DCSO gotie API bindings
// Copyright (c) 2016-2018, DCSO GmbH

import "time"

// authToken is used to strictly marshal the returned
// TIE API access_token
type authToken struct {
	AccessToken string `json:"access_token"`
}

// apiMessage is used to strictly marshal the returned
// TIE API message
type apiMessage struct {
	Message string      `json:"message"`
	Errors  interface{} `json:"errors,omitempty"`
}

// IOC defines the basic data structure of IOCs in TIE
type IOC struct {
	ID                    string     `json:"id"`
	Value                 string     `json:"value"`
	DataType              string     `json:"data_type"`
	EntityIDs             []string   `json:"entity_ids"`
	EventIDs              []string   `json:"event_ids"`
	EventAttributes       []string   `json:"event_attributes"`
	Categories            []string   `json:"categories"`
	SourcePseudonyms      []string   `json:"source_pseudonyms"`
	SourceNames           []string   `json:"source_names"`
	NOccurrences          int        `json:"n_occurrences"`
	MinSeverity           int        `json:"min_severity"`
	MaxSeverity           int        `json:"max_severity"`
	FirstSeen             *time.Time `json:"first_seen"`
	LastSeen              *time.Time `json:"last_seen"`
	MinConfidence         int        `json:"min_confidence"`
	MaxConfidence         int        `json:"max_confidence"`
	Enrich                bool       `json:"enrich"`
	EnrichmentRequestedAt *time.Time `json:"enrichment_requested_at,omitempty"`
	EnrichedAt            *time.Time `json:"enriched_at,omitempty"`
	UpdatedAt             *time.Time `json:"updated_at"`
	CreatedAt             *time.Time `json:"created_at"`
	ObservationAttributes []string   `json:"observation_attributes"`
}

type IOCResult struct {
	IOC   *IOC
	Error error
}

// IOCParams contains all necessary query parameters
type IOCParams struct {
	NoDefaults       bool       `json:"no_defaults"`
	Direction        string     `json:"direction"`
	OrderBy          string     `json:"order_by"`
	Severity         string     `json:"severity"`
	Confidence       string     `json:"confidence"`
	Ivalue           string     `json:"ivalue"`
	GroupBy          []string   `json:"group_by"`
	Limit            int        `json:"limit"`
	Offset           int        `json:"offset"`
	WithCompositions bool       `json:"with_compositions"`
	FirstSeenSince   *time.Time `json:"first_seen_since,omitempty"`
	LastSeenSince    *time.Time `json:"last_seen_since,omitempty"`
	DateField        string     `json:"date_field"`
	Enriched         bool       `json:"enriched"`
	DateFormat       string     `json:"date_format"`
}

// IOCQueryStruct defines the returned data of a TIE API IOC query
type IOCQueryStruct struct {
	HasMore bool      `json:"has_more"`
	Iocs    []IOC     `json:"iocs"`
	Params  IOCParams `json:"params"`
}
