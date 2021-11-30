.PHONY: test
test:
	gotest -v ./... -coverprofile=coverage.out -covermode=atomic

build:
	go build
