SHELL=/usr/bin/env bash

all: build
.PHONY: all

unexport GOFLAGS

BINS:=

ldflags=-X=chia_transfer/build.CurrentCommit=+git.$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))
ifneq ($(strip $(LDFLAGS)),)
	ldflags+=-extldflags=$(LDFLAGS)
endif

GOFLAGS+=-ldflags="-s -w $(ldflags)"

move_sectors:
	rm -f chia_transfer
	go build $(GOFLAGS) -o chia_transfer ./cmd
.PHONY: chia_transfer
BINS+=move_sectors

build: chia_transfer

.PHONY: build

install: install-chia-transfer

install-move-sectors:
	install -C ./chia_transfer /usr/local/bin/chia_transfer


buildall: $(BINS)

clean:
	rm -rf $(CLEAN) $(BINS)
.PHONY: clean