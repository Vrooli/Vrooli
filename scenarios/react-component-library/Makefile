# React Component Library Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/react-component-library-api). ALWAYS use these commands.
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

.PHONY: help start run stop restart test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check setup open orient temporal-models endpoints matrix matrix-diff determinism determinism-check corpus-report ledger-audit restore-authored-mirror purge-unreadable plan-harness-capture plan-harness-compare

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

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
	@printf "  make matrix                     Emit the deterministic catalog gate verdict matrix\n"
	@printf "  make matrix-diff                Compare the live verdict matrix with the baseline\n"
	@printf "  make run                        Alias for make start\n"
	@printf "  make dev                        Alias for make start\n"
	@printf "\n"
	@printf "For scenario-specific commands, prefer the scenario CLI or 'vrooli scenario ...'.\n"

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

test: ## Run scenario tests
	@vrooli scenario test "$(SCENARIO_NAME)"

clean: ## Clean build artifacts
	@rm -rf ui/dist 2>/dev/null || true
	@rm -f api/$(SCENARIO_NAME)-api api/$(SCENARIO_NAME) 2>/dev/null || true

build: ## Build the scenario
	@if [ -f api/go.mod ]; then \
		cd api && GOWORK=off go build -o "$(SCENARIO_NAME)-api" .; \
	fi
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm run build; \
	fi

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

fmt-ui: ## Format UI and published library source
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm run format && pnpm run lint:fix && pnpm run strings:check; \
	fi

lint: lint-go lint-ui ## Lint all code

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

lint-ui: ## Lint UI code
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm run lint && pnpm run type-check; \
	fi

check: fmt lint test ## Run full quality gates

setup: ## Run scenario setup lifecycle
	@vrooli scenario setup "$(SCENARIO_NAME)"

temporal-models: ## Verify formal flow artifacts via the flow-verifier scenario CLI
	@flow-verifier verify check --root "$(CURDIR)"

open: ## Open the scenario in a browser
	@vrooli scenario open "$(SCENARIO_NAME)"

orient: ## Show generated-scenario orientation progress
	@template-manager orient "$(SCENARIO_NAME)"

endpoints: ## Regenerate .vrooli/endpoints.json from API modules (validated against cli/manifest.json)
	@cd api && go run ./cmd/gen-endpoints --output ../.vrooli/endpoints.json

matrix: ## Emit the deterministic catalog gate verdict matrix
	@cd api && GOWORK=off go run ./cmd/gate-matrix --root ../../..

matrix-diff: ## Compare the live verdict matrix with the committed baseline
	@cd api && GOWORK=off go run ./cmd/gate-matrix --root ../../.. --diff ../catalog/calibration/verdict-matrix-baseline.json

corpus-report: ## Emit version and retention corpus invariants
	@cd api && GOWORK=off go run ./cmd/corpus-report --root ../../..

ledger-audit: ## Audit authored release-hash entries without changing the tree
	@cd api && GOWORK=off go run ./cmd/ledger-audit --root ../../..

restore-authored-mirror: ## Preview or restore authored release files from the durable mirror
	@cd api && GOWORK=off go run ./cmd/restore-authored-mirror --root ../../.. $(ARGS)

purge-unreadable: ## Purge one explicitly named evicted version with no mirror
	@cd api && GOWORK=off go run ./cmd/purge-unreadable --root ../../.. $(ARGS)

plan-harness-capture: ## Capture the plan's gate, corpus, preflight, and build measures
	@cd api && GOWORK=off go run ./cmd/plan-harness capture --root ../../.. $(ARGS)

plan-harness-compare: ## Compare two plan-harness captures
	@cd api && GOWORK=off go run ./cmd/plan-harness compare $(ARGS)

determinism: ## Generate test-genie determinism capabilities from the gate registry
	@cd api && GOWORK=off go run ./cmd/gen-determinism --output ../.vrooli/test-genie.json

determinism-check: ## Verify generated determinism capabilities are current
	@cd api && GOWORK=off go run ./cmd/gen-determinism --check --output ../.vrooli/test-genie.json

run: start

dev: start
