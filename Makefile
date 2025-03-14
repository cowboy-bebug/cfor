OUTPUT := cfor
VERSION := $(shell git describe --tags --abbrev=0 | sed 's/^v//')
COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

build:
	go build -ldflags $(LDFLAGS) -o $(OUTPUT)

clean:
	rm -rf dist

.PHONY: build clean
