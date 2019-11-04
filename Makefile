WORK_DIR=work
VERSION := $(shell cat VERSION)
LDFLAGS=-ldflags "-X github.com/Hsn723/nginx-i2c/cmd.CurrentVersion=${VERSION}"

clean:
	rm -rf ${WORK_DIR}

setup:
	mkdir -p ${WORK_DIR}

lint: setup
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b ${WORK_DIR} v1.21.0
	golangci-lint run

build: clean setup
	go build ${LDFLAGS} -o ${WORK_DIR}/nginx-i2c .

.PHONY: clean setup lint build
