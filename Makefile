# This makes sure that regex match for vVALID, vPATCH and vRC is working
SHELL:=/bin/bash

# MAKEFLAGS += -s

include .bingo/Variables.mk

# --------------------------------------------------------------------------------------------------------------------------------

.PHONY: toolsupdate

toolsupdate:
	@echo Updating tools
	@$(BINGO) list | tail -n +3 | awk '{print $$1}' | xargs -tI % sh -c '$(BINGO) get -l %@latest'

# --------------------------------------------------------------------------------------------------------------------------------

.PHONY: version

# similar to https://github.com/ahmetb/govvv
GITCOMMITFULL  ?= $(shell git rev-parse HEAD)
GITCOMMIT      ?= $(shell git rev-parse --short HEAD)
GITBRANCH	   ?= $(shell git rev-parse --abbrev-ref HEAD)
GITSTATE	   ?= $(shell test -z "$$(git status --porcelain)" && echo 'clean' || echo 'dirty')
GITSUMMARY     ?= $(shell git describe --tags --always --dirty --match=v*)
BUILDDATE      ?= $(shell git log -1 --format=%ci | $(TS) -f RFC3339)
VERSION        ?= $(shell git describe --always --tags --match=v*)

version:         ## Shows the current version based on git
	@$(call echo,GITCOMMITFULL : $(GITCOMMITFULL))
	@$(call echo,GITCOMMIT     : $(GITCOMMIT))
	@$(call echo,GITBRANCH     : $(GITBRANCH))
	@$(call echo,GITSTATE      : $(GITSTATE))
	@$(call echo,GITSUMMARY    : $(GITSUMMARY))
	@$(call echo,BUILDDATE     : $(BUILDDATE))
	@$(call echo,VERSION       : $(VERSION))

# Checks that there there are no uncommitted changes
.GIT.IS.DIRTY:
	@if [ "$(GITSTATE)" != "clean" ]; then \
		echo "🚧 Git working directory is dirty. Please commit or stash your changes."; \
		exit 1; \
	fi

# --------------------------------------------------------------------------------------------------------------------------------

.PHONY: tagdebug

vTAG := $(tag)
ifdef tag
	vTAG := ${shell [[ $(tag) = v* ]] && echo $(tag) || $(SVU) $(tag)}
else
	vTAG = v0
endif

# If the first argument is "bump"...
ifeq (bump,$(firstword $(MAKECMDGOALS)))
	# use the rest as arguments
  	BUMP_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  	# ...and turn them into do-nothing targets
  	$(eval $(BUMP_ARGS):;@:)
	BUMP_ARGS := $(if $(BUMP_ARGS),$(BUMP_ARGS),vINVALID)
	vTAG := ${shell [[ $(BUMP_ARGS) = v* ]] && echo $(BUMP_ARGS) || $(SVU) $(BUMP_ARGS)}
endif

# If the first argument is "changelog"...
ifeq (changelog,$(firstword $(MAKECMDGOALS)))
	# use the rest as arguments
  	CHANGELOG_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  	# ...and turn them into do-nothing targets
  	$(eval $(CHANGELOG_ARGS):;@:)
	CHANGELOG_ARGS := $(if $(CHANGELOG_ARGS),$(CHANGELOG_ARGS),vINVALID)
	vTAG := ${shell [[ $(CHANGELOG_ARGS) = v* ]] && echo $(CHANGELOG_ARGS) || $(SVU) $(CHANGELOG_ARGS)}
endif

# If the first argument is "tagdebug"...
ifeq (tagdebug,$(firstword $(MAKECMDGOALS)))
	# use the rest as arguments
  	TAGDEBUG_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  	# ...and turn them into do-nothing targets
  	$(eval $(TAGDEBUG_ARGS):;@:)
	TAGDEBUG_ARGS := $(if $(TAGDEBUG_ARGS),$(TAGDEBUG_ARGS),vINVALID)
	vTAG := ${shell [[ $(TAGDEBUG_ARGS) = v* ]] && echo $(TAGDEBUG_ARGS) || $(SVU) $(TAGDEBUG_ARGS)}
endif

vFULL  := $(vTAG)
vFULL := $(if $(vFULL),$(vFULL),INVALID)
# Is this a valid version?
vVALID := $(shell [[ $(vFULL) =~ ^v[0-9]+\.[0-9]+\.[0-9]+((-[0-9a-zA-Z]+)?$$|(-[0-9a-zA-Z]+\.[0-9a-zA-Z]+)?$$|(-[0-9a-zA-Z]+\.[0-9a-zA-Z]+\+[0-9a-zA-Z]+)?$$)$$ ]] && echo "yes")
# Is this a round version?
vROUND := $(shell [[ $(vFULL) =~ ^v[0-9]+(\.[0-9]+)?$$ ]] && echo "yes")
# Is this a patch?
vPATCH := $(shell [[ $(vFULL) =~ ^v[0-9]+\.[0-9]+\.([0-9]+)(-[0-9a-zA-Z]+)?(\.[0-9a-zA-Z]+)?(\+[0-9a-zA-Z]+)?$$ ]] && echo "yes")
# Is this a release candidate?
vRC    := $(shell [[ $(vFULL) =~ \-(rc|RC|Rc)(.[0-9a-zA-Z]+)?$$ ]] && echo "yes")

# If not valid but round then add .0 suffix and revalidate
ifeq ($(vVALID),)
ifeq ($(vROUND),yes)
	vFULL := $(addsuffix .0,$(vFULL))
	vVALID := $(shell [[ $(vFULL) =~ ^v[0-9]+\.[0-9]+\.[0-9]+((-[0-9a-zA-Z]+)?$$|(-[0-9a-zA-Z]+\.[0-9a-zA-Z]+)?$$)$$ ]] && echo "yes")
endif
endif

# The actual shorten version
vSHORT := $(word 1,$(subst ., ,$(vFULL))).$(word 2,$(subst ., ,$(vFULL)))

ifeq ($(vVALID),yes)
	TAG := $(vFULL)
endif

tagdebug:        ## Show the tag validation, run as: make tagdebug v0.0.1
	@$(call echo,vFULL  : $(vFULL))
	@$(call echo,vVALID : $(vVALID))
	@$(call echo,vROUND : $(vROUND))
	@$(call echo,vPATCH : $(vPATCH))
	@$(call echo,vRC    : $(vRC))
	@$(call echo,vSHORT : $(vSHORT))
	@$(call echo,TAG    : $(TAG))
	@$(call echo,STATUS : $(VERSION))

define verify_tag
	@if [ "$(vVALID)" != "yes" ]; then \
		echo Tag is invalid. Make sure that it complies with the Semantic Versioning specification; exit 1;\
	fi
endef

define verify_tag_version
	@$(call verify_tag)
	@if [ "$(TAG)" != "$(VERSION)" ]; then \
		echo TAG does not match git tag VERSION: $(VERSION); exit 1;\
	fi
endef

# --------------------------------------------------------------------------------------------------------------------------------

.PHONY: bump changelog

CHANGELOG_START_FOUND ?= $(shell head -n 1 .changelog_start.txt)
ifneq ($(wildcard .changelog_start.txt),) 
    CHANGELOG_START = $(CHANGELOG_START_FOUND)..
else
    CHANGELOG_START = 
endif

# bump:            ## Update the CHANGELOG file, commit and tag
bump: .GIT.IS.DIRTY $(GIT_CHGLOG) $(SVU)
	@$(call verify_tag)
	@$(GIT_CHGLOG) -c .chglog/config.yaml --next-tag $(TAG) -o CHANGELOG-REVIEW.md $(CHANGELOG_START)
	@meld CHANGELOG-REVIEW.md CHANGELOG.md
	@git add CHANGELOG.md
	@git commit -m "release: $(TAG)"
	@git tag -a $(TAG) -m "release: $(TAG)"

# changelog:       ## Update the CHANGELOG file with the especified tag
changelog: .GIT.IS.DIRTY $(GIT_CHGLOG) $(SVU)
	@$(call verify_tag)
	@$(GIT_CHGLOG) -c .chglog/config.yaml --next-tag $(TAG) -o CHANGELOG-REVIEW.md $(CHANGELOG_START)
	@meld CHANGELOG-REVIEW.md CHANGELOG.md
	@git add CHANGELOG.md
	@git commit -m "changelog: $(TAG)"

# --------------------------------------------------------------------------------------------------------------------------------

ifeq ($(shell echo "check_quotes"),"check_quotes")
   WINDOWS := yes
else
   WINDOWS := no
endif

ifeq ($(WINDOWS),yes) # ver si $(shell) se requiere en windows, con o sin MinGW
   mkdir = $(shell mkdir $(subst /,\,$(1)) > nul 2>&1 || (exit 0))
   rm = $(shell $(wordlist 2,65535,$(foreach FILE,$(subst /,\,$(1)),& del $(FILE) > nul 2>&1)) || (exit 0))
   rmdir = $(shell rmdir $(subst /,\,$(1)) > nul 2>&1 || (exit 0))
   echo = $(shell echo $(1))
else
   mkdir = mkdir -p $(1)
   rm = rm $(1) > /dev/null 2>&1 || true
   rmdir = rmdir $(1) > /dev/null 2>&1 || true
   echo = echo "$(1)"
endif

# --------------------------------------------------------------------------------------------------------------------------------

.PHONY: golint gotest gomodupdateall

golint: $(GOLANGCI_LINT)
	@$(GOLANGCI_LINT) run

gotest:
	go test -v ./...

gomodupdateall:
	go get -u ./...
	go mod tidy

# --------------------------------------------------------------------------------------------------------------------------------

.PHONY: build

build:
	go build -ldflags "-s -w" -o build/garchetype main.go
