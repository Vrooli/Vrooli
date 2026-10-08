# mk/scenario.mk — the shared toolchain targets of a scenario Makefile.
#
# A scenario Makefile defines SCENARIO_NAME, includes this file, and gets the
# build, fmt-go, fmt-ui, lint-go and lint-ui recipes the react-vite template
# ships; the toolchain floor from mk/toolchain.mk comes with it. A scenario
# that needs its own body for one of these targets lists it in
# SCENARIO_CUSTOM_TARGETS before the include and defines it itself.
ifndef VROOLI_SCENARIO_MK
VROOLI_SCENARIO_MK := 1
include $(dir $(lastword $(MAKEFILE_LIST)))toolchain.mk

SCENARIO_CUSTOM_TARGETS ?=

ifeq ($(filter build,$(SCENARIO_CUSTOM_TARGETS)),)
build: ## Build the scenario
	@if [ -f api/go.mod ]; then \
		cd api && GOWORK=off go build -o "$(SCENARIO_NAME)-api" .; \
	fi
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm run build; \
	fi
endif

ifeq ($(filter fmt-go,$(SCENARIO_CUSTOM_TARGETS)),)
fmt-go: ## Format Go code
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd api && gofmt -w .; \
		fi; \
	fi
	@if [ -d cli ] && find cli -name "*.go" | head -1 | grep -q .; then \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd cli && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd cli && gofmt -w .; \
		fi; \
	fi
endif

ifeq ($(filter fmt-ui,$(SCENARIO_CUSTOM_TARGETS)),)
fmt-ui: ## Format UI code
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm run lint:fix && pnpm run strings:check; \
	fi
endif

ifeq ($(filter lint-go,$(SCENARIO_CUSTOM_TARGETS)),)
lint-go: ## Lint Go code
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && GOWORK=off golangci-lint run ./...; \
		elif command -v go >/dev/null 2>&1; then \
			cd api && GOWORK=off go vet ./... && GOWORK=off go fmt ./...; \
		fi; \
	fi
	@if [ -d cli ] && find cli -name "*.go" | head -1 | grep -q .; then \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd cli && GOWORK=off golangci-lint run ./...; \
		elif command -v go >/dev/null 2>&1; then \
			cd cli && GOWORK=off go vet ./... && GOWORK=off go fmt ./...; \
		fi; \
	fi
endif

ifeq ($(filter lint-ui,$(SCENARIO_CUSTOM_TARGETS)),)
lint-ui: ## Lint UI code
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm run lint && pnpm run type-check; \
	fi
endif
endif
