.PHONY: test
test:
	gotest -v ./... -race -coverprofile=coverage.out -covermode=atomic

build:
	go build
