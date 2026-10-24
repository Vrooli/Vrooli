# mk/toolchain.mk — the toolchain floor for every Makefile in the repository.
#
# Included by the root Makefile, the scenario and resource template Makefiles
# and every scenario Makefile. It mirrors the control plane's BuildWidth lever
# (internal/tuning: min(4, max(1, NumCPU/4)), overridable through
# VROOLI_TUNING_BUILD_WIDTH) and the envkit.Toolchain composition rule: an
# inherited GOFLAGS keeps every token and gains -p=<width> only when no -p is
# present; GOMAXPROCS and the pnpm concurrency settings are set only when
# absent. Nothing here replaces an inherited value.
#
# require-no-root-wildcard refuses `go build ./...` at the repository root:
# on 2026-09-02 three agent sessions running that command drove a 32-core
# host to a 15-minute load of 1,499. Build a module list instead.
ifndef VROOLI_TOOLCHAIN_MK
VROOLI_TOOLCHAIN_MK := 1

TOOLCHAIN_NCPU := $(strip $(shell nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo "$${NUMBER_OF_PROCESSORS:-4}"))
ifneq ($(strip $(VROOLI_TUNING_BUILD_WIDTH)),)
BUILD_WIDTH := $(strip $(VROOLI_TUNING_BUILD_WIDTH))
else
BUILD_WIDTH := $(strip $(shell w=$$(( $(TOOLCHAIN_NCPU) / 4 )); [ "$$w" -lt 1 ] && w=1; [ "$$w" -gt 4 ] && w=4; echo "$$w"))
endif
BUILD_WIDTH_X2 := $(strip $(shell echo $$(( $(BUILD_WIDTH) * 2 ))))

ifeq ($(findstring -p=,$(GOFLAGS)),)
export GOFLAGS := $(strip $(GOFLAGS) -p=$(BUILD_WIDTH))
endif
ifeq ($(strip $(GOMAXPROCS)),)
export GOMAXPROCS := $(BUILD_WIDTH_X2)
endif
ifeq ($(strip $(npm_config_child_concurrency)),)
export npm_config_child_concurrency := $(BUILD_WIDTH)
endif
ifeq ($(strip $(npm_config_workspace_concurrency)),)
export npm_config_workspace_concurrency := $(BUILD_WIDTH)
endif

GO ?= go

# TOOLCHAIN_REPO_ROOT is the repository root when this include is reached
# from inside the monorepo; a scenario extracted to its own repository
# resolves nothing and the guard never fires there.
TOOLCHAIN_REPO_ROOT := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
TOOLCHAIN_REPO_ROOT := $(patsubst %/mk,%,$(TOOLCHAIN_REPO_ROOT))

.PHONY: require-no-root-wildcard
require-no-root-wildcard:
	@if [ "$(abspath $(CURDIR))" = "$(TOOLCHAIN_REPO_ROOT)" ] && printf '%s\n' $(MAKECMDGOALS) $(GOALS) $(PKGS) | grep -qx './\.\.\.'; then \
		echo "refusing 'go build ./...' at the repository root: it starts one compile per package across every module at once." >&2; \
		echo "Build a module list instead: make type, make verify-supervisory, or go build ./cmd/... ./internal/... (GOFLAGS=$(GOFLAGS))." >&2; \
		exit 2; \
	fi
endif
