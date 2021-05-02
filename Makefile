
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
	vault secrets enable database

configure:
	vault write database/config/arango \
    plugin_name=arangodb-database-plugin \
    allowed_roles="my-role" \
    username="root" \
    password="root"

clean:
	rm -f ./build/plugins/arangodb-database-plugin

fmt:
	go fmt $$(go list ./...)

.PHONY: build clean fmt start enable
