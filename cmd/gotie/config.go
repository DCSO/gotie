package main

// DCSO gotie API bindings
// Copyright (c) 2016-2018, DCSO GmbH

import (
	"os/user"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// CONF contains the TIE authentication information read from the config file
var CONF = config{}

type config struct {
	TieToken      string `toml:"tie_token"`
	PingBackToken string `toml:"pingback_token"`
}

func getDefaultConfPath() string {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	dir := usr.HomeDir
	return filepath.Join(dir, ".gotie")
}

func loadConfig(path string) error {
	_, err := toml.DecodeFile(path, &CONF)
	if err != nil {
		return err
	}
	return nil
}
