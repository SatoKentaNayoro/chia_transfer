SHELL=/usr/bin/env bash

all: build
.PHONY: all

unexport GOFLAGS

BINS:=

chia_transfer:
	rm -f chia_transfer
	go build $(GOFLAGS) -o chia_transfer .
.PHONY: chia_transfer
BINS+=chia_transfer

build: chia_transfer

.PHONY: build

install: install-chia-transfer

install-chia-transfer:
	install -C ./chia_transfer /usr/local/bin/chia_transfer


buildall: $(BINS)

clean:
	rm -rf $(CLEAN) $(BINS)
.PHONY: clean