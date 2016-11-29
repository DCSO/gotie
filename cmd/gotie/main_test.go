package main

// DCSO gotie API bindings
// Copyright (c) 2016, DCSO GmbH

import (
	"flag"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}
