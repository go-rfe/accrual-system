.ONESHELL:
	/bin/bash

.PHONY: deps clean build test run start

build: clean vet server

deps:
	go mod download

vet:
	go vet ./...

server:
	go build ./cmd/server

clean:
	rm -f server

test:
	go test ./...

run:
	go run ./cmd/server/main.go

start: build
	./server
