# Image Tools Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/image-tools-api). ALWAYS use these commands.
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

.PHONY: help start run stop restart test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui coverage-model-lifecycle model-install-e2e headless-ai-e2e gpu-e2e check setup open orient temporal-models endpoints

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

coverage-model-lifecycle: ## Check model lifecycle package coverage floors
	@cd api && for spec in \
		"./internal/models 85.0" \
		"./internal/ai 85.0" \
		"./internal/backends 85.0"; do \
		pkg=$$(printf "%s" "$$spec" | awk '{print $$1}'); \
		threshold=$$(printf "%s" "$$spec" | awk '{print $$2}'); \
		out=$$(go test -cover -count=1 "$$pkg"); \
		printf "%s\n" "$$out"; \
		pct=$$(printf "%s\n" "$$out" | awk -F'coverage: |% of statements' '/coverage:/ {print $$2}'); \
		awk -v p="$$pct" -v t="$$threshold" 'BEGIN { exit (p+0 >= t+0) ? 0 : 1 }' \
			|| { printf "coverage gate failed: %s %s%% < %s%%\n" "$$pkg" "$$pct" "$$threshold"; exit 1; }; \
	done

model-install-e2e: ## Attended install loop for enabled model weights
	@set -eu; \
	if ! command -v jq >/dev/null 2>&1; then \
		printf "SKIP model-install-e2e: jq is required to inspect catalog JSON.\n"; \
		exit 0; \
	fi; \
	if [ "$${IMAGE_TOOLS_ALLOW_MODEL_DOWNLOADS:-}" != "1" ]; then \
		printf "SKIP model-install-e2e: set IMAGE_TOOLS_ALLOW_MODEL_DOWNLOADS=1 to allow governed model downloads.\n"; \
		exit 0; \
	fi; \
	image-tools models doctor --json | jq -e '.ok == true' >/dev/null; \
	wait_timeout="$${IMAGE_TOOLS_INSTALL_WAIT_TIMEOUT:-30m}"; \
	ids=$$(image-tools models list --json | jq -r '.models[] | select(.enabled == true) | select((.size_mb_approx // 0) > 0) | .id'); \
	count=$$(printf "%s\n" "$$ids" | sed '/^$$/d' | wc -l | tr -d ' '); \
	if [ "$$count" = "0" ]; then \
		printf "Model install E2E passed: no enabled weight-backed models require install.\n"; \
		exit 0; \
	fi; \
	printf "Model install E2E starting for %s enabled weight-backed model(s).\n" "$$count"; \
	for id in $$ids; do \
		printf "Installing/verifying model %s\n" "$$id"; \
		if ! IMAGE_TOOLS_HTTP_TIMEOUT="$$wait_timeout" image-tools models install "$$id" --wait; then \
			if image-tools models get "$$id" --json | jq -e '.model.install.installed == true' >/dev/null; then \
				printf "Install wait disconnected after %s completed; continuing.\n" "$$id"; \
			else \
				printf "model-install-e2e failed: %s did not install successfully.\n" "$$id"; \
				exit 1; \
			fi; \
		fi; \
	done; \
	printf "Model install E2E passed for %s enabled weight-backed model(s).\n" "$$count"

headless-ai-e2e: ## Run no-download AI/analyze CLI E2E checks
	@set -eu; \
	tmp=$$(mktemp -d); \
	trap 'rm -rf "$$tmp"' EXIT; \
	img="$$tmp/input.png"; \
	naturalized="$$tmp/naturalized.png"; \
	normal="$$tmp/normal.png"; \
	python3 -c 'import struct,sys,zlib; w,h=32,24; raw=b"".join(bytes([0])+b"".join(bytes(((x*7+y*3)%256,(x*5)%256,(y*11)%256)) for x in range(w)) for y in range(h)); c=lambda k,d: struct.pack(">I",len(d))+k+d+struct.pack(">I",zlib.crc32(k+d)&0xffffffff); open(sys.argv[1],"wb").write(b"\x89PNG\r\n\x1a\n"+c(b"IHDR",struct.pack(">IIBBBBB",w,h,8,2,0,0,0))+c(b"IDAT",zlib.compress(raw))+c(b"IEND",b""))' "$$img"; \
	image-tools analyze probe "$$img" --json >/dev/null; \
	image-tools analyze quality "$$img" --json >/dev/null; \
	image-tools analyze duplicate "$$img" --json >/dev/null; \
	image-tools ai naturalize "$$img" --wait --out "$$naturalized" >/dev/null; \
	test -s "$$naturalized" || { printf "headless-ai-e2e failed: naturalize output is empty\n"; exit 1; }; \
	image-tools ai normal-map "$$img" --wait --out "$$normal" >/dev/null; \
	test -s "$$normal" || { printf "headless-ai-e2e failed: normal-map output is empty\n"; exit 1; }; \
	printf "Headless AI E2E passed: probe, quality, duplicate, naturalize, normal-map\n"

gpu-e2e: ## Attended GPU text-to-image E2E when backend + model are provisioned
	@set -eu; \
	if ! command -v jq >/dev/null 2>&1; then \
		printf "SKIP gpu-e2e: jq is required to inspect doctor/select JSON.\n"; \
		exit 0; \
	fi; \
	selection=$$(image-tools models select text_to_image --json); \
	model=$$(printf "%s" "$$selection" | jq -r '.model.id'); \
	if ! printf "%s" "$$selection" | jq -e '.gpu_viable == true' >/dev/null; then \
		reason=$$(printf "%s" "$$selection" | jq -r '.reason // "not GPU viable"'); \
		warnings=$$(printf "%s" "$$selection" | jq -r '(.warnings // []) | join("; ")'); \
		printf "SKIP gpu-e2e: text_to_image is not GPU-viable on this host. %s %s\n" "$$reason" "$$warnings"; \
		exit 0; \
	fi; \
	backends_json=$$(image-tools backends doctor --json 2>/dev/null || true); \
	if ! printf "%s" "$$backends_json" | jq -e '.backends[] | select(.name == "stable-diffusion.cpp") | select((.operations // []) | index("text_to_image")) | .available == true' >/dev/null; then \
		detail=$$(printf "%s" "$$backends_json" | jq -r '.backends[] | select(.name == "stable-diffusion.cpp") | .detail' | head -1); \
		provision=$$(printf "%s" "$$backends_json" | jq -r '.backends[] | select(.name == "stable-diffusion.cpp") | .provision' | head -1); \
		printf "SKIP gpu-e2e: stable-diffusion.cpp is not provisioned for text_to_image. %s. %s\n" "$$detail" "$$provision"; \
		exit 0; \
	fi; \
	if ! image-tools models get "$$model" --json | jq -e '.model.install.installed == true' >/dev/null; then \
		printf "Model %s is not installed; submitting governed install job...\n" "$$model"; \
		image-tools models install "$$model" --wait; \
	fi; \
	out=$$(mktemp --suffix=.png); \
	printf "Running GPU E2E: model=%s output=%s\n" "$$model" "$$out"; \
	image-tools ai generate --model "$$model" --prompt "a small red cube on a neutral tabletop" --negative "text, watermark, logo" --width 256 --height 256 --steps 8 --cfg-scale 5 --seed 42 --wait --out "$$out"; \
	test -s "$$out" || { printf "gpu-e2e failed: %s is empty\n" "$$out"; exit 1; }; \
	printf "GPU E2E produced %s\n" "$$out"

check: fmt lint test ## Run full quality gates

setup: ## Run scenario setup lifecycle
	@vrooli scenario setup "$(SCENARIO_NAME)"

temporal-models: ## Verify formal flow artifacts via the flow-verifier scenario CLI
	@flow-verifier verify check --root "$(CURDIR)"

open: ## Open the scenario in a browser
	@vrooli scenario open "$(SCENARIO_NAME)"

orient: ## Show generated-scenario orientation progress
	@vrooli scenario orient "$(SCENARIO_NAME)"

endpoints: ## Regenerate .vrooli/endpoints.json from API modules (validated against cli/manifest.json)
	@cd api && go run ./cmd/gen-endpoints --output ../.vrooli/endpoints.json

run: start

dev: start
