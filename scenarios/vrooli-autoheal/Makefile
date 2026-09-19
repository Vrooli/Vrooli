# Vrooli Autoheal Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/vrooli-autoheal-api). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs
#   make clean - Clean build artifacts
#
# This Makefile is a thin wrapper. All user-facing output (status, banners,
# colors) is owned by the `vrooli scenario` CLI itself.

.PHONY: help start run stop restart test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check setup open orient temporal-models endpoints loop-build loop-test loop-lint

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

SCENARIO_CUSTOM_TARGETS += build fmt-go lint-go
# Shared toolchain floor (mk/toolchain.mk) and build targets (mk/scenario.mk); a custom body for one of them is declared in SCENARIO_CUSTOM_TARGETS.
-include ../../mk/scenario.mk

help: ## Show the supported scenario entrypoints
	@printf "Vrooli scenario entrypoints for %s\n\n" "$(SCENARIO_NAME)"
	@printf "  make setup                      Run scenario setup lifecycle\n"
	@printf "  make start                      Start the scenario\n"
	@printf "  make stop                       Stop the scenario\n"
	@printf "  make restart                    Restart the scenario\n"
	@printf "  make status                     Show scenario runtime status\n"
	@printf "  make logs                       Show scenario logs\n"
	@printf "  make test                       Run scenario tests\n"
	@printf "  make temporal-models            Check formal temporal-model artifacts\n"
	@printf "  make open                       Open the scenario in a browser\n"
	@printf "  make orient                     Show generated-scenario orientation progress\n"
	@printf "  make endpoints                  Regenerate .vrooli/endpoints.json from API modules\n"
	@printf "  make loop-build                 Build the boot-recovery loop through vrooli scenario setup\n"
	@printf "  make loop-test                  Run the loop and langrecover module tests\n"
	@printf "  make loop-lint                  Lint the loop and langrecover modules\n"
	@printf "  make run                        Alias for make start\n"
	@printf "  make dev                        Alias for make start\n"
	@printf "\n"
	@printf "For scenario-specific commands, prefer the scenario CLI or 'vrooli scenario ...'.\n"

setup: ## Run scenario setup lifecycle
	@vrooli scenario setup "$(SCENARIO_NAME)"

start: ## Start the scenario
	@vrooli scenario start "$(SCENARIO_NAME)"

stop: ## Stop the scenario
	@vrooli scenario stop "$(SCENARIO_NAME)"

restart: ## Restart the scenario
	@vrooli scenario restart "$(SCENARIO_NAME)"

status: ## Show scenario runtime status
	@vrooli scenario status "$(SCENARIO_NAME)"

logs: ## Show scenario logs
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

test: loop-test ## Run scenario tests (the loop and langrecover modules first: nothing else runs them)
	@vrooli scenario test "$(SCENARIO_NAME)"

# The boot-recovery loop is the `loop` component in .vrooli/service.json: a
# nested module the lifecycle engine builds and stamps (cli/vrooli-autoheal-loop
# plus its .freshness.json manifest) on `vrooli scenario setup`. Nothing here
# runs `go build` on it: a binary built outside the engine carries no manifest,
# and the autoheal_watchdog safeguard reports such a binary as unknown, not
# fresh. These targets are the module test/lint coverage that `test`,
# `lint-go` and `fmt-go` fold in.
LOOP_MODULES := cli/loop langrecover

# `make build` is the shared react-vite body plus the loop, which is built
# through the engine and never through a direct go build.
build: loop-build ## Build the scenario (api, ui, and the loop through the lifecycle engine)
	@if [ -f api/go.mod ]; then \
		cd api && GOWORK=off go build -o "$(SCENARIO_NAME)-api" .; \
	fi
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm run build; \
	fi

loop-build: ## Build the boot-recovery loop through the lifecycle engine (stamps cli/vrooli-autoheal-loop.freshness.json)
	@vrooli scenario setup "$(SCENARIO_NAME)"

loop-test: ## Run the loop and langrecover module tests
	@for module in $(LOOP_MODULES); do \
		echo "==> go test $$module"; \
		GOWORK=off go -C "$$module" test ./... || exit 1; \
	done

loop-lint: ## Lint the loop and langrecover modules (golangci-lint when configured, go vet otherwise)
	@for module in $(LOOP_MODULES); do \
		echo "==> lint $$module"; \
		if command -v golangci-lint >/dev/null 2>&1 && [ -f "$$module/.golangci.yml" ]; then \
			(cd "$$module" && GOWORK=off golangci-lint run ./...) || exit 1; \
		else \
			GOWORK=off go -C "$$module" vet ./... || exit 1; \
		fi; \
	done

open: ## Open the scenario in a browser
	@vrooli scenario open "$(SCENARIO_NAME)"

orient: ## Show generated-scenario orientation progress
	@template-manager orient "$(SCENARIO_NAME)"

endpoints: ## Regenerate .vrooli/endpoints.json from API modules (validated against cli/manifest.json)
	@cd api && go run ./cmd/gen-endpoints --output ../.vrooli/endpoints.json

temporal-models: ## Verify formal flow artifacts via the flow-verifier scenario CLI
	@flow-verifier verify check --root "$(CURDIR)"

run: start

dev: start

clean: ## Clean build artifacts
	@rm -rf ui/dist 2>/dev/null || true
	@rm -f api/$(SCENARIO_NAME)-api api/$(SCENARIO_NAME) 2>/dev/null || true

fmt: fmt-go fmt-ui ## Format all code

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
	@if command -v gofumpt >/dev/null 2>&1; then \
		cd langrecover && gofumpt -w .; \
	elif command -v gofmt >/dev/null 2>&1; then \
		cd langrecover && gofmt -w .; \
	fi

lint: lint-go lint-ui ## Lint all code

lint-go: loop-lint ## Lint Go code
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

check: fmt lint test ## Run full quality gates
