# Audio Tools Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/audio-tools-api). ALWAYS use these commands.

.PHONY: help start run stop restart test soak-accelerated logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check setup open orient temporal-models endpoints

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))
SOAK_REFERENCE_TEXT ?= the quick brown fox jumps.

help: ## Show the supported scenario entrypoints
	@printf "Vrooli scenario entrypoints for %s\n\n" "$(SCENARIO_NAME)"
	@printf "  make setup                      Run scenario setup lifecycle\n"
	@printf "  make start                      Start the scenario\n"
	@printf "  make stop                       Stop the scenario\n"
	@printf "  make restart                    Restart the scenario\n"
	@printf "  make status                     Show scenario runtime status\n"
	@printf "  make logs                       Show scenario logs\n"
	@printf "  make test                       Run scenario tests\n"
	@printf "  make soak-accelerated           Run the gated 60-minute accelerated browser soak\n"
	@printf "  make temporal-models            Check formal temporal-model artifacts\n"
	@printf "  make open                       Open the scenario in a browser\n"
	@printf "  make orient                     Show generated-scenario orientation progress\n"
	@printf "  make endpoints                  Regenerate .vrooli/endpoints.json from API modules\n"
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

# This is an explicit local/integration gate because it needs the running
# audio-tools API, BAS Playwright driver, and Dictation Studio UI. Test Genie
# does not execute arbitrary Make targets; its registered phase descriptors
# remain the owner of scenario-suite phases.
soak-accelerated: ## Run the gated 60-minute accelerated browser soak
	@test -n "$(SOAK_DRIVER_URL)" || { echo "SOAK_DRIVER_URL is required (for example http://127.0.0.1:24485)" >&2; exit 2; }
	@test -n "$(SOAK_UI_URL)" || { echo "SOAK_UI_URL is required (for example http://127.0.0.1:20004)" >&2; exit 2; }
	@test -n "$(SOAK_API_BASE)" || { echo "SOAK_API_BASE is required (for example http://127.0.0.1:19630)" >&2; exit 2; }
	@audio-tools --api-base "$(SOAK_API_BASE)" soak run \
		--driver-url "$(SOAK_DRIVER_URL)" --ui-url "$(SOAK_UI_URL)" \
		--fixture "$(CURDIR)/bas/fixtures/dictation-reference.wav" \
		--lane accelerated --profile continuous --turns 3 --feed-ms 2000 \
		--reference-text "$(SOAK_REFERENCE_TEXT)" \
		--engine-id virtual-replay --model-id virtual-corpus \
		--strategy passthrough --policy default --shape burst \
		--simulated-minutes 60 --evidence-path "$(SOAK_EVIDENCE_PATH)" --json

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

fmt-ui: ## Format UI code
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm run lint:fix && pnpm run strings:check; \
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

run: start

dev: start
