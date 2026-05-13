# Flow Verifier Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/flow-verifier-api). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs
#   make clean - Clean build artifacts

.PHONY: help setup start stop restart status logs test clean build fmt fmt-go fmt-ui lint lint-go lint-ui check dev run open orient endpoints temporal-models navigation

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
	@printf "  make navigation                 Validate + verify + reconcile flow-verifier's own navigation.json (also writes routes.generated.ts)\n"
	@printf "  make open                       Open the scenario in a browser\n"
	@printf "  make orient                     Show generated-scenario orientation progress\n"
	@printf "  make endpoints                  Regenerate .vrooli/endpoints.json from API modules\n"
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

test: ## Run scenario tests
	@vrooli scenario test "$(SCENARIO_NAME)"

temporal-models: ## Verify formal flow artifacts via the flow-verifier scenario CLI
	@flow-verifier verify check --root .

navigation: ## Validate + verify + reconcile the scenario's own navigation.json and emit routes.generated.ts
	@cd api && go run ./cmd/navigation-self-check --contract ../ui/flow/navigation.json --scenario .. --emit

open: ## Open the scenario in a browser
	@vrooli scenario open "$(SCENARIO_NAME)"

orient: ## Show generated-scenario orientation progress
	@vrooli scenario orient "$(SCENARIO_NAME)"

endpoints: ## Regenerate .vrooli/endpoints.json from API modules
	@cd api && go run ./cmd/gen-endpoints --output ../.vrooli/endpoints.json --seed cmd/gen-endpoints/cli_commands_seed.json

run: start

dev: start

clean: ## Remove build artifacts
	@rm -rf ui/dist api/bin cli/bin coverage/

build: ## Build API + UI bundles
	@cd api && go build ./...
	@cd cli && go build ./...
	@cd ui && npm run build

fmt: fmt-go fmt-ui ## Format Go + UI code

fmt-go: ## Format Go code via gofumpt (gracefully skips when gofumpt is not installed)
	@command -v gofumpt >/dev/null 2>&1 || { echo "gofumpt not installed; skipping fmt-go"; exit 0; }
	@test -d api && cd api && gofumpt -w . || true
	@test -d cli && cd cli && gofumpt -w . || true

fmt-ui: ## Format UI code via prettier (gracefully skips when prettier is not installed)
	@test -d ui || { echo "no ui directory; skipping fmt-ui"; exit 0; }
	@cd ui && npx --no-install prettier --write 'src/**/*.{ts,tsx,css}' 2>/dev/null || echo "prettier not installed; skipping fmt-ui"

lint: lint-go lint-ui ## Lint Go + UI code

lint-go: ## Lint Go code via golangci-lint (gracefully skips when golangci-lint is not installed)
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed; skipping lint-go"; exit 0; }
	@test -d api && cd api && golangci-lint run ./... || true
	@test -d cli && cd cli && golangci-lint run ./... || true

lint-ui: ## Lint UI code via eslint
	@test -d ui || { echo "no ui directory; skipping lint-ui"; exit 0; }
	@cd ui && npx --no-install eslint 'src/**/*.{ts,tsx}' 2>/dev/null || echo "eslint not installed; skipping lint-ui"

check: fmt lint navigation test ## Run fmt + lint + navigation self-check + test
