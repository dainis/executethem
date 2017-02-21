GIT_HASH=$(shell git rev-parse HEAD)
BUILD_DATE=$(shell date)
VERSION="0.0.1"

updatedeps:
	go get -u github.com/kardianos/govendor
	govendor fetch +vendor

build:
	go build -ldflags "-X github.com/dainis/executethem/cmd.Version=$(VERSION) -X github.com/dainis/executethem/cmd.GitHash=$(GIT_HASH) -X 'github.com/dainis/executethem/cmd.BuildDate=$(BUILD_DATE)'" -o bin/executethem

.PHONY: updatedeps build
