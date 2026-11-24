# Audio Tools Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/audio-tools-api). ALWAYS use these commands.

.PHONY: help start run stop restart test soak-accelerated lab-test logs status clean build build-product build-lab dev fmt fmt-go fmt-ui lint lint-go lint-ui type check setup open orient temporal-models endpoints

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

# Shared toolchain floor (mk/toolchain.mk) and build targets (mk/scenario.mk); a custom body for one of them is declared in SCENARIO_CUSTOM_TARGETS.
-include ../../mk/scenario.mk
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
	@printf "  make type                       Compile-check API, CLI, and UI\n"
	@printf "  make lab-test                   Run lab/qualification package tests\n"
	@printf "  make soak-accelerated           Run the gated 60-minute accelerated browser soak\n"
	@printf "  make temporal-models            Check formal temporal-model artifacts\n"
	@printf "  make open                       Open the scenario in a browser\n"
	@printf "  make orient                     Show generated-scenario orientation progress\n"
	@printf "  make endpoints                  Regenerate .vrooli/endpoints.json from API modules\n"
	@printf "  make catalog-gen                Regenerate provider catalogue Go/TS artifacts\n"
	@printf "  make operations-gen             Regenerate the audio operation UI specs\n"
	@printf "  make models-doc                 Regenerate docs/reference/models.md\n"
	@printf "  make catalog-check              Run catalog drift + cross-catalog conformance gates\n"
	@printf "  make portability                Cross-compile the API and CLI for Windows and macOS\n"
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

# Lab packages are compiled only into the audio_lab API artifact. Their import
# and product dependency-graph contracts are tested in api/internal/lab.
lab-test: ## Run lab/qualification package tests
	@cd api && go test -tags audio_lab -count=1 . ./internal/lab/... ./internal/corpus/... ./internal/experiment/... ./internal/eval/... ./internal/soak/... ./internal/modelpromotion ./internal/conformance ./handlers/corpus/... ./handlers/experiment/... ./handlers/eval/... ./handlers/soak/... ./handlers/validation/... ./internal/modules ./internal/server ./internal/bootstrap

# This gate runs through the same Test Genie lease and BAS Broker contract as
# the registered soak provider. The API selects the Dictation Studio UI and
# fixed WAV; callers cannot select a driver endpoint or fixture path.
soak-accelerated: ## Run the gated 60-minute accelerated browser soak
	@test -n "$(SOAK_API_BASE)" || { echo "SOAK_API_BASE is required (for example http://127.0.0.1:19630)" >&2; exit 2; }
	@test -n "$(VROOLI_AGENT_IDENTITY_TOKEN)" || { echo "VROOLI_AGENT_IDENTITY_TOKEN is required from a trusted Agent Manager run" >&2; exit 2; }
	@test -n "$(VROOLI_BROWSER_GRANT_RECEIPT)" || { echo "VROOLI_BROWSER_GRANT_RECEIPT is required from the active Agent Manager run" >&2; exit 2; }
	@test -n "$(TEST_GENIE_DEMAND_LEASE_RECEIPT_ID)" || { echo "TEST_GENIE_DEMAND_LEASE_RECEIPT_ID is required from an active Test Genie lease" >&2; exit 2; }
	@audio-tools --api-base "$(SOAK_API_BASE)" soak run \
		--lane accelerated --profile continuous --turns 3 --feed-ms 2000 \
		--reference-text "$(SOAK_REFERENCE_TEXT)" \
		--engine-id virtual-replay --model-id virtual-corpus \
		--strategy passthrough --policy default --shape burst \
		--simulated-minutes 60 --evidence-path "$(SOAK_EVIDENCE_PATH)" --json

clean: ## Clean build artifacts
	@rm -rf ui/dist 2>/dev/null || true
	@rm -f api/$(SCENARIO_NAME)-api api/$(SCENARIO_NAME) 2>/dev/null || true

fmt: fmt-go fmt-ui ## Format all code

lint: lint-go lint-ui ## Lint all code

type: ## Compile-check API, CLI, and UI
	@cd api && GOWORK=off go test -run='^$$' ./...
	@cd cli && GOWORK=off go test -run='^$$' ./...
	@if [ -f ui/package.json ]; then cd ui && pnpm run type-check; fi

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
	@cd api && go run -tags audio_lab ./cmd/gen-endpoints --output ../.vrooli/endpoints.json

build-product: ## Compile the product API without lab execution dependencies
	@cd api && GOWORK=off go build -o audio-tools-api .

build-lab: ## Compile the lab API with corpus, experiment, soak, and validation routes
	@cd api && GOWORK=off go build -tags audio_lab -o audio-tools-lab-api .

catalog-gen: ## Regenerate provider catalogue artifacts (Go + TypeScript)
	@cd api && go run ./cmd/gen-catalog

operations-gen: ## Regenerate the audio operation UI specs from the operation catalog
	@cd api && go run ./cmd/gen-operations

models-doc: ## Regenerate docs/reference/models.md from the model catalog
	@cd api && go run ./internal/modelcatalog/cmd/render-models

portability: ## Cross-compile the API and CLI for Windows and macOS
	@cd api && GOOS=windows GOARCH=amd64 go build ./... && GOOS=darwin GOARCH=arm64 go build ./...
	@cd cli && GOOS=windows GOARCH=amd64 go build ./... && GOOS=darwin GOARCH=arm64 go build ./...
	@echo "portability: windows/amd64 and darwin/arm64 builds passed"

catalog-check: ## Run catalog drift + cross-catalog conformance gates
	@cd api && go test ./internal/providercatalog ./internal/operationcatalog ./internal/modelcatalog ./internal/modelpromotion ./internal/catalogconformance ./internal/capabilities ./internal/sttengine ./internal/stt/whisperinfo ./handlers/models

run: start

dev: start
