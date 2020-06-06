GO ?= go
CONFIG_DIR ?= undef

# Platforms to build for. You can add or remove platforms here
# Note that at this time, Linux build does not work outside of linux...
LOCAL_PLATFORM := $(shell uname -s | tr A-Z a-z)
PLATFORMS = linux darwin

SOURCE_FILES := $(shell find . -name '*.go')

PWD ?= $(shell pwd)
REAL_CONFIG_DIR := $(shell if [ -d $(CONFIG_DIR) -a ! -d /$(CONFIG_DIR) ]; then echo $(PWD)/$(CONFIG_DIR); else echo $(CONFIG_DIR); fi)
BINARY := $(shell basename $(PWD))
BUILD_DIR := $(PWD)
OUTPUT_BASE_DIR = $(BUILD_DIR)/output/
BUILD_DIR_LINK = $(shell readlink $(BUILD_DIR))

VET_REPORT := vet.report
TEST_REPORT := tests.xml

GOPRIVATE=github.com/AntonioMA/go-utils

GOARCH := amd64

VERSION?=?
COMMIT := $(shell git rev-parse HEAD)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS = -ldflags "-X main.VERSION=$(VERSION) -X main.COMMIT=$(COMMIT) -X main.BRANCH=$(BRANCH)"

OUTPUT_PLATFORM_DIRS = $(addprefix $(OUTPUT_BASE_DIR), $(PLATFORMS))

# To-do: Check if we can use this
#SUFFIXES+= .go .so
#.SUFFIXES: $(SUFFIXES)

# Those are calculated based on your configuration above
all: showconfig $(OUTPUT_PLATFORM_DIRS) $(PLATFORMS)

local: showconfig $(OUTPUT_PLATFORM_DIRS) $(LOCAL_PLATFORM)

showconfig:
	@echo Variables:
	@echo PLATFORMS: $(PLATFORMS)
	@echo OUTPUT_PLATFORM_DIRS: $(OUTPUT_PLATFORM_DIRS)
	@echo DEPLOY_TARGETS: $(DEPLOY_TARGETS)

$(OUTPUT_PLATFORM_DIRS):
	@echo "Creating output directories"
	@mkdir -p $@

$(PLATFORMS): OUTPUT_DIR = $(OUTPUT_BASE_DIR)$@/

$(PLATFORMS):
	@echo Building $@ from $(BUILD_DIR)
	@rm -f $(OUTPUT_DIR)$(BINARY)
	@cd $(BUILD_DIR); \
	GO111MODULE=on GOOS=$@ GOARCH=${GOARCH} GOPRIVATE=$(GOPRIVATE) $(GO) build -v ${LDFLAGS} -o $(OUTPUT_DIR)$(BINARY) .

run: clean all
	@source ./samples/test_env.sh ;\
	$(OUTPUT_BASE_DIR)$(LOCAL_PLATFORM)/$(BINARY) -i ./samples/template.txt

clean:
	-rm -f ${TEST_REPORT}
	-rm -f ${VET_REPORT}
	-rm -rf $(OUTPUT_BASE_DIR)

.PHONY: all link $(PLATFORMS) test vet fmt clean showconfig $(DEPLOY_TARGETS)

.SECONDEXPANSION:

