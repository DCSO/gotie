GOTIE [![Build Status](https://travis-ci.org/DCSO/gotie.svg?branch=master)](https://travis-ci.org/DCSO/gotie)
=====

Go bindings and simple command line client for the
[DCSO Threat Intelligence Engine API (TIE)](https://tie.dcso.de/).

## Install

To use the Go binding you will have to install a golang environment and
configure a $GOPATH for your user/system. Most modern distributions include a
recent version of Go. To install the bindings and the command line client into
your configured $GOPATH you can use the following command:

```bash
$ go get -u github.com/DCSO/gotie/...
```

The command line client expects a configuration file in your home directory
(~/.gotie) containing the following two variables:

```toml
tie_token = "<token>"
pingback_token = "<token>"
```

The `tie_token` is mandatory.
The `pingback_token` is optional.

**NOTE:**
You can always set an alternative path for the configuration file using the
*-c / --config* command line flag.

## Command-line Client

The example command-line client can be used to query the TIE API for IOCs and
feeds.


```bash
$ gotie iocs -q <query_string>
```

Run `gotie -h` to see all options.

### Output formats

Depending on your use case, you can choose between the output formats
CSV (default), JSON and Bloom filter. The latter integrates well with the
[DCSO Bloom filter CLI and lib](https://github.com/DCSO/bloom).


Retrieve IOCs of type DomainName created today in JSON format:
```bash
gotie iocs -t domainname -f json --created-since $(date +%F)
```

Print only value field using jq:
```bash
gotie iocs -f json --created-since $(date +%F) | jq '.iocs[] | .value'
```

Build a Bloom filter:
```bash
gotie iocs -f bloom --created-since $(date +%F) > test.bloom
```

Perform a check with the bloom CLI tool:
```bash
echo www.example.com | bloom check test.bloom
```

The value will be echoed for a match, otherwise the tool stays silent. Read
the [Bloom CLI Readme](https://github.com/DCSO/bloom) for further details.



## Tests

To run the included tests you have to set an environment variable containing
your API token:

```bash
$ TIE_TOKEN=<token> make test
```

## License

This software is released under a BSD 3-Clause license.
Please have a look at the LICENSE file included in the repository.

Copyright (c) 2016, DCSO Deutsche Cyber-Sicherheitsorganisation GmbH
