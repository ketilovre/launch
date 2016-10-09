SHELL=/bin/bash

BUILD_FLAGS=-i -ldflags "-s -w" -o
LINUX=GOARCH=amd64 GOOS=linux
OSX=GOARCH=amd64 GOOS=darwin

build:
	go build -i

build_linux:
	$(LINUX) go build $(BUILD_FLAGS) launch_x64_linux

build_osx:
	$(OSX) go build $(BUILD_FLAGS) launch_x64_osx

dist: build_linux build_osx

qa:
	go fmt ./...
	go vet ./...
