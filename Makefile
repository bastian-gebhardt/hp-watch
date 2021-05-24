# TODO: check if PROGRAM_NAME and INSTALL_DIR != ""
# TODO: check if git installed
# TODO: pretty print INSTALL steps

PROGRAM_NAME=watch-hp
INSTALL_DIR=$(HOME)/bin

IS_GIT=$(shell git describe --tags --always || echo "false")
GIT_STATUS=$(shell git status -s --porcelain || echo "false")

# Build variables
ifeq ("$(IS_GIT)", "false")
VERSION 	?= "unknown"
COMMIT_HASH ?= "unknown"
else ifneq ("$(GIT_STATUS)", "")
VERSION 	?= local_changes@$(shell git describe --tags --always)
COMMIT_HASH ?= local_changes@$(shell git rev-parse --short HEAD 2>/dev/null)
else
VERSION 	?= $(shell git describe --tags --always)
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null)
endif

BUILD_DATE 	?= $(shell date +%FT%T%z)


# Go variables
GO      ?= go
GOOS    ?= $(shell $(GO) env GOOS)
GOARCH  ?= $(shell $(GO) env GOARCH)
GOHOST  ?= GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO)

LDFLAGS ?= "-X main.version=$(VERSION) -X main.commitHash=${COMMIT_HASH} -X main.buildDate=${BUILD_DATE}"

.PHONY: all
all: help

.PHONY: clean
clean: # clean all caches

// TODO: git stash to build HEAD (and not local changes)
.PHONY: build
build: ## Build the binary
	CGO_ENABLED=0 $(GOHOST) build -ldflags=$(LDFLAGS) -o $(PROGRAM_NAME) --mod=vendor ./cmd/$(PROGRAM_NAME)

.PHONY: install
install: ## Install tool
	cp $(PROGRAM_NAME) $(INSTALL_DIR)/$(PROGRAM_NAME)

.PHONY: uninstall
uninstall: ## Uninstall tool
	rm $(INSTALL_DIR)/$(PROGRAM_NAME)

.PHONY: test
test: ## Run all tests
	go test -v -race -failfast ./cmd/$(PROGRAM_NAME)...

.PHONY: help
help: ## Display this help
	@awk \
		-v "col=\033[36m" -v "nocol=\033[0m" \
		' \
			BEGIN { \
				FS = ":.*##" ; \
				printf "Usage:\n  make %s<target>%s\n", col, nocol \
			} \
			/^[a-zA-Z_-]+:.*?##/ { \
				printf "  %s%-12s%s %s\n", col, $$1, nocol, $$2 \
			} \
			/^##@/ { \
				printf "\n%s%s%s\n", nocol, substr($$0, 5), nocol \
			} \
		' $(MAKEFILE_LIST)