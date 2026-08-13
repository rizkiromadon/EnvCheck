BINARY     := envcheck
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS    := -s -w -X main.version=$(VERSION)
PKG        := ./cmd/envcheck
DIST       := dist

.PHONY: all build test lint clean cross package-deb package-rpm package-arch

all: build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

test:
	go test ./... -v -race -cover

vet:
	go vet ./...

clean:
	rm -rf $(BINARY) $(DIST)

# Cross-compile static binaries for common targets. CGO is disabled so the
# resulting binaries have no dynamic library dependencies, which matters
# both for portability and for clean .deb/.rpm packaging.
cross: clean
	mkdir -p $(DIST)
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-linux-amd64   $(PKG)
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-linux-arm64   $(PKG)
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-darwin-amd64  $(PKG)
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-darwin-arm64  $(PKG)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-windows-amd64.exe $(PKG)

# Packaging targets assume `nfpm` (https://nfpm.goreleaser.com) is installed,
# which builds .deb/.rpm/.apk from one declarative config without needing
# a Debian or RPM build chroot. See packaging/nfpm.yaml.
package-deb: cross
	nfpm package --config packaging/nfpm.yaml --packager deb --target $(DIST)/

package-rpm: cross
	nfpm package --config packaging/nfpm.yaml --packager rpm --target $(DIST)/

package-arch:
	@echo "Build with: cd packaging/arch && makepkg -si"
