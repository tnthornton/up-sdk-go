# ====================================================================================
# Setup Project

PROJECT_NAME := up-sdk-go
PROJECT_REPO := github.com/upbound/$(PROJECT_NAME)

PLATFORMS ?= linux_amd64 linux_arm64
# -include will silently skip missing files, which allows us
# to load those files with a target in the Makefile. If only
# "include" was used, the make command would fail and refuse
# to run a target until the include commands succeeded.
-include build/makelib/common.mk

# ====================================================================================
# Setup Images

# even though this repo doesn't build images (note the no-op img.build target below),
# some of the init is needed for the cross build container, e.g. setting BUILD_REGISTRY
-include build/makelib/image.mk
img.build:

# ====================================================================================
# Setup Go

# Set a sane default so that the nprocs calculation below is less noisy on the initial
# loading of this file
NPROCS ?= 1

# By default we reduce the parallelism to half the number of CPU cores.
GO_TEST_PARALLEL := $(shell echo $$(( $(NPROCS) / 2 )))

GO_SUBDIRS += errors service fake generate
GO111MODULE = on
GOLANGCILINT_VERSION := 1.64.8
-include build/makelib/golang.mk

# generate/generate.sh needs to know what subdirs to generate files for.
export GO_SUBDIRS

# ====================================================================================
# Targets

# run `make help` to see the targets and options

# We want submodules to be set up the first time `make` is run.
# We manage the build/ folder and its Makefiles as a submodule.
# The first time `make` is run, the includes of build/*.mk files will
# all fail, and this target will be run. The next time, the default as defined
# by the includes will be run instead.
fallthrough: submodules
	@echo Initial setup complete. Running make again . . .
	@make

# Generate a coverage report for cobertura applying exclusions on
# - generated file
cobertura:
	@cat $(GO_TEST_OUTPUT)/coverage.txt | \
		grep -v zz_generated.deepcopy | \
		$(GOCOVER_COBERTURA) > $(GO_TEST_OUTPUT)/cobertura-coverage.xml

# Update the submodules, such as the common build scripts.
submodules:
	@git submodule sync
	@git submodule update --init --recursive

go.generate: go.generate.apis
go.generate.apis:
	@$(INFO) "cd apis; go generate $(PLATFORM)"
	cd apis; CGO_ENABLED=0 $(GOHOST) generate $(GO_GENERATE_FLAGS) ./... || $(FAIL)
	@$(OK) "cd apis; go generate $(PLATFORM)"
	@$(INFO) "cd apis; go mod tidy"
	@cd apis; $(GOHOST) mod tidy || $(FAIL)
	@$(OK) "cd apis; go mod tidy"

go.lint: go.lint.apis
go.lint.apis: $(GOLANGCILINT)
	@$(INFO) "cd apis; golangci-lint"
	@cd apis; mkdir -p $(GO_LINT_OUTPUT)
	@cd apis; $(GOLANGCILINT) run $(GO_LINT_ARGS) || $(FAIL)
	@$(OK) "cd apis; golangci-lint"

.PHONY: cobertura submodules fallthrough

# ====================================================================================
# Special Targets

define UP_SDK_GO_HELP
Upbound Go SDK Targets:
    cobertura          Generate a coverage report for cobertura applying exclusions on generated files.
    submodules         Update the submodules, such as the common build scripts.

endef
export UP_SDK_GO_HELP

up-sdk-go.help:
	@echo "$$UP_SDK_GO_HELP"

help-special: up-sdk-go.help

.PHONY: up-sdk-go.help help-special
