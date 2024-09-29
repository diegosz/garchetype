# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.9. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Below generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for bingo variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(BINGO)
#	@echo "Running bingo"
#	@$(BINGO) <flags/args..>
#
BINGO := $(GOBIN)/bingo-v0.9.0
$(BINGO): $(BINGO_DIR)/bingo.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/bingo-v0.9.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=bingo.mod -o=$(GOBIN)/bingo-v0.9.0 "github.com/bwplotka/bingo"

GIT_CHGLOG := $(GOBIN)/git-chglog-v0.15.4
$(GIT_CHGLOG): $(BINGO_DIR)/git-chglog.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/git-chglog-v0.15.4"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=git-chglog.mod -o=$(GOBIN)/git-chglog-v0.15.4 "github.com/git-chglog/git-chglog/cmd/git-chglog"

GOLANGCI_LINT := $(GOBIN)/golangci-lint-v1.61.0
$(GOLANGCI_LINT): $(BINGO_DIR)/golangci-lint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/golangci-lint-v1.61.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=golangci-lint.mod -o=$(GOBIN)/golangci-lint-v1.61.0 "github.com/golangci/golangci-lint/cmd/golangci-lint"

SVU := $(GOBIN)/svu-v1.12.0
$(SVU): $(BINGO_DIR)/svu.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/svu-v1.12.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=svu.mod -o=$(GOBIN)/svu-v1.12.0 "github.com/caarlos0/svu"

TS := $(GOBIN)/ts-v0.0.7
$(TS): $(BINGO_DIR)/ts.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/ts-v0.0.7"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=ts.mod -o=$(GOBIN)/ts-v0.0.7 "github.com/liujianping/ts"

