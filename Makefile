GOPATH        = $(shell go env GOPATH)
GOLINT        = $(GOPATH)/bin/golint
GOIMPORTS     = $(GOPATH)/bin/goimports
MISSPELL      = $(GOPATH)/bin/misspell
GOSEC         = $(GOPATH)/bin/gosec
ERRCHECK      = $(GOPATH)/bin/errcheck
STATICCHECK   = $(GOPATH)/bin/staticcheck
GOCYCLO       = $(GOPATH)/bin/gocyclo
ARCH          = $(shell uname -p)


# go option
PKG        := ./...
TAGS       :=
TESTS      := .
TESTFLAGS  :=
GOFLAGS    :=

SHELL      = /usr/bin/env bash


GIT_COMMIT = $(shell git rev-parse HEAD)
GIT_SHA    = $(shell git rev-parse --short HEAD)
GIT_TAG    = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)


.PHONY: check
check: lint
check: misspell
check: gosec
check: err-check
check: static-check
check: vet
check: gocyclo

.PHONY: lint
lint:
	@echo
	@echo "==> Check & Review code <=="
	GO111MODULE=on go list ./... | grep -v /vendor/ | xargs -L1 $(GOLINT) -set_exit_status

.PHONY: misspell
misspell:
	@echo
	@echo "==> Correct commonly misspelled English words <=="
	GO111MODULE=on $(MISSPELL) transform pkg

.PHONY: gosec
gosec:
	@echo
	@echo "==> Inspects source code for security problems <=="
	GO111MODULE=on $(GOSEC) ./...

.PHONY: err-check
err-check: $(ERRCHECK)
	@echo
	@echo "==> Error check <=="
	GO111MODULE=on $(ERRCHECK) ./...

.PHONY: static-check
static-check: $(STATICCHECK)
	@echo
	@echo "==> Static check <=="
	GO111MODULE=on $(STATICCHECK) -checks all,-ST1000 ./...

.PHONY: vet
vet:
	@echo
	@echo "==> Vet <=="
	GO111MODULE=on go vet ./...

# https://github.com/fzipp/gocyclo
.PHONY: gocyclo
gocyclo: $(GOCYCLO)
	@echo
	@echo "==> GOCYCLO <=="
	GO111MODULE=on $(GOCYCLO) .


$(ERRCHECK):
	(cd /; GO111MODULE=on go get -u github.com/kisielk/errcheck)

$(STATICCHECK):
	(cd /; GO111MODULE=on go get -u honnef.co/go/tools/cmd/staticcheck)

$(GOCYCLO):
	(cd /; GO111MODULE=on go get github.com/fzipp/gocyclo)


.PHONY: info
info:
	@echo "Git Tag:           ${GIT_TAG}"
	@echo "Git Commit:        ${GIT_COMMIT}"


.PHONY: test
test:
	@echo
	@echo "==> Running unit tests <=="

	GO111MODULE=on go test $(GOFLAGS) -run $(TESTS) $(PKG) $(TESTFLAGS)

.PHONY: generated-code
generated-code:
	CGO_ENABLED=0 go generate ./...
