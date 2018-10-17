## simple makefile to log workflow
.PHONY: all test clean build install

GOFLAGS ?= $(GOFLAGS:)

all: install test

build:
	@go build $(GOFLAGS) ./...

install:
	@go get -t $(GOFLAGS) ./...

test: install
	@go vet $(GOFLAGS) ./...
	@go test -cover $(GOFLAGS) ./...

bench: install
	@go test -run=NONE -bench=. $(GOFLAGS) ./...

clean:
	@go clean $(GOFLAGS) -i ./...

release:
	@go get -t $(GOFLAGS) ./...
	@go build -v -o gotie_linux_amd64.bin cmd/gotie/*
	GOOS=windows GOARCH=amd64 go build -v -o gotie_windows_amd64.exe cmd/gotie/*
## EOF
