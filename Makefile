.PHONY: build test format

build:
	go build -o gh-issue-sync ./cmd/gh-issue-sync

test:
	go test ./...

format:
	go fmt ./...
