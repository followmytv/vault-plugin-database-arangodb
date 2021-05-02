
GOARCH = amd64

UNAME = $(shell uname -s)

ifndef OS
	ifeq ($(UNAME), Linux)
		OS = linux
	else ifeq ($(UNAME), Darwin)
		OS = darwin
	endif
endif

.DEFAULT_GOAL := all

all: fmt build start

build:
	GOOS=$(OS) GOARCH="$(GOARCH)" go build -o ./build/plugins/arangodb-database-plugin ./arangodb-database-plugin/main.go

start:
	vault server -dev -dev-root-token-id=root -dev-plugin-dir=./build/plugins/

enable:
	vault secrets enable -path=arangodb arangodb-database-plugin

clean:
	rm -f ./build/plugins/arangodb-database-plugin

fmt:
	go fmt $$(go list ./...)

.PHONY: build clean fmt start enable
