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

.PHONY: help start run stop restart test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui coverage-model-lifecycle model-install-e2e headless-ai-e2e gpu-e2e gpu-e2e-import gpu-e2e-lora gpu-e2e-controlnet gpu-e2e-ipadapter gpu-e2e-conditioning conditioning-freshness check setup open orient temporal-models endpoints backends-doc

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

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

fmt: fmt-go fmt-ui ## Format all code

lint: lint-go lint-ui ## Lint all code

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

# =============================================================================
# Conditioning-adapter attended GPU E2E (plan Phase 7). Each target proves one
# adapter kind on SD1.5 @ 512 with sequential offload (fits ~7-8 GB). They are
# NOT part of `make test` / the default `vrooli scenario test` run. They mirror
# gpu-e2e's skip-gracefully discipline: a missing prerequisite SKIPs (exit 0)
# with a precise reason rather than failing.
#
# Bootstrap note: an adapter ships Ready=false (no vaporware). The resolver
# refuses a conditioned submit until Ready=true, so these targets SKIP with the
# proving procedure when the adapter is not yet Ready — the operator proves the
# sidecar path once (see docs/internal/TESTING.md), flips Ready in
# api/internal/adapters/adapters.seed.json, then re-runs the target as the
# durable regression guard.
# =============================================================================
GPU_E2E_BASE_MODEL ?= sd-1.5
GPU_E2E_W ?= 512
GPU_E2E_H ?= 512

define adapter_e2e_preamble
	if ! command -v jq >/dev/null 2>&1; then \
		printf "SKIP %s: jq is required to inspect adapter/select JSON.\n" "$(1)"; \
		exit 0; \
	fi; \
	sel=$$(image-tools models select image_to_image --override "$(GPU_E2E_BASE_MODEL)" --json 2>/dev/null || true); \
	if ! printf "%s" "$$sel" | jq -e '.gpu_viable == true' >/dev/null 2>&1; then \
		printf "SKIP %s: base model %s is not GPU-viable on this host.\n" "$(1)" "$(GPU_E2E_BASE_MODEL)"; \
		exit 0; \
	fi; \
	ad=$$(image-tools adapters get "$(2)" --json 2>/dev/null || true); \
	if [ -z "$$ad" ]; then \
		printf "SKIP %s: adapter %s not found in the catalog.\n" "$(1)" "$(2)"; \
		exit 0; \
	fi; \
	if ! printf "%s" "$$ad" | jq -e '(.adapter.install.installed // .install.installed // false) == true' >/dev/null 2>&1; then \
		printf "SKIP %s: adapter %s is not installed (run: image-tools adapters install %s).\n" "$(1)" "$(2)" "$(2)"; \
		exit 0; \
	fi; \
	if ! printf "%s" "$$ad" | jq -e '(.adapter.ready // .ready // false) == true' >/dev/null 2>&1; then \
		printf "SKIP %s: adapter %s is Ready=false (bootstrap). Prove the sidecar path per docs/internal/TESTING.md, flip Ready in adapters.seed.json, then re-run.\n" "$(1)" "$(2)"; \
		exit 0; \
	fi
endef

gpu-e2e-import: ## Attended GPU E2E: import an HF SDXL/SD1.5 base by repo id, generate, verify derived ops
	@set -eu; \
	if ! command -v jq >/dev/null 2>&1; then printf "SKIP gpu-e2e-import: jq required.\n"; exit 0; fi; \
	if [ -z "$${IMAGE_TOOLS_E2E_IMPORT_REPO:-}" ]; then \
		printf "SKIP gpu-e2e-import: set IMAGE_TOOLS_E2E_IMPORT_REPO=<hf-repo-id> (e.g. runwayml/stable-diffusion-v1-5) to run.\n"; \
		exit 0; \
	fi; \
	repo="$$IMAGE_TOOLS_E2E_IMPORT_REPO"; \
	id="$${IMAGE_TOOLS_E2E_IMPORT_ID:-imported-e2e}"; \
	if image-tools models get "$$id" --json 2>/dev/null | jq -e '.model.install.installed == true' >/dev/null 2>&1; then \
		printf "Reusing already-imported %s (idempotent regression guard).\n" "$$id"; \
	else \
		printf "Inspecting %s...\n" "$$repo"; \
		insp=$$(image-tools models inspect "$$repo" --json); \
		arch=$$(printf "%s" "$$insp" | jq -r '.architecture.architecture // .architecture // empty'); \
		[ -n "$$arch" ] || { printf "gpu-e2e-import failed: could not infer architecture for %s\n" "$$repo"; exit 1; }; \
		printf "Inferred architecture: %s — importing as %s\n" "$$arch" "$$id"; \
		image-tools models import "$$repo" --id "$$id" --architecture "$$arch" --json >/dev/null; \
		image-tools models install "$$id" --wait >/dev/null 2>&1 || true; \
		for _ in $$(seq 1 120); do \
			if image-tools models get "$$id" --json 2>/dev/null | jq -e '.model.install.installed == true' >/dev/null 2>&1; then break; fi; \
			sleep 10; \
		done; \
		image-tools models get "$$id" --json | jq -e '.model.install.installed == true' >/dev/null || { printf "gpu-e2e-import failed: %s did not install\n" "$$id"; exit 1; }; \
	fi; \
	image-tools models enable "$$id" >/dev/null; \
	out=$$(mktemp --suffix=.png); \
	image-tools ai generate --model "$$id" --prompt "a small red cube on a neutral tabletop" --width "$(GPU_E2E_W)" --height "$(GPU_E2E_H)" --steps 8 --seed 42 --wait --out "$$out"; \
	test -s "$$out" || { printf "gpu-e2e-import failed: output empty\n"; exit 1; }; \
	image-tools models get "$$id" --json | jq -e '.model.operations | index("image_to_image")' >/dev/null || { printf "gpu-e2e-import failed: derived image_to_image not offered for %s\n" "$$id"; exit 1; }; \
	printf "gpu-e2e-import produced %s and %s offers derived img2img\n" "$$out" "$$id"

gpu-e2e-lora: ## Attended GPU E2E: SD1.5 + LCM-LoRA @ 512
	@set -eu; \
	$(call adapter_e2e_preamble,gpu-e2e-lora,lcm-lora-sdv1-5); \
	out=$$(mktemp --suffix=.png); \
	image-tools ai generate --model "$(GPU_E2E_BASE_MODEL)" --prompt "a serene mountain lake at dawn" --width "$(GPU_E2E_W)" --height "$(GPU_E2E_H)" --steps 8 --seed 42 --lora "lcm-lora-sdv1-5:1.0" --wait --out "$$out"; \
	test -s "$$out" || { printf "gpu-e2e-lora failed: output empty\n"; exit 1; }; \
	printf "gpu-e2e-lora produced %s\n" "$$out"

gpu-e2e-controlnet: ## Attended GPU E2E: SD1.5 + Canny ControlNet @ 512 (needs a control image blob key)
	@set -eu; \
	$(call adapter_e2e_preamble,gpu-e2e-controlnet,controlnet-canny-sd15); \
	if [ -z "$${IMAGE_TOOLS_E2E_CONTROL_IMAGE_KEY:-}" ]; then \
		printf "SKIP gpu-e2e-controlnet: set IMAGE_TOOLS_E2E_CONTROL_IMAGE_KEY=<blob-key> (a pre-made canny map, or the output of the canny op) to run.\n"; \
		exit 0; \
	fi; \
	out=$$(mktemp --suffix=.png); \
	image-tools ai generate --model "$(GPU_E2E_BASE_MODEL)" --prompt "a stone castle on a hill" --width "$(GPU_E2E_W)" --height "$(GPU_E2E_H)" --steps 20 --seed 42 --controlnet "controlnet-canny-sd15:1.0:$$IMAGE_TOOLS_E2E_CONTROL_IMAGE_KEY" --wait --out "$$out"; \
	test -s "$$out" || { printf "gpu-e2e-controlnet failed: output empty\n"; exit 1; }; \
	printf "gpu-e2e-controlnet produced %s\n" "$$out"

gpu-e2e-ipadapter: ## Attended GPU E2E: SD1.5 + IP-Adapter @ 512 (needs a reference image blob key)
	@set -eu; \
	$(call adapter_e2e_preamble,gpu-e2e-ipadapter,ip-adapter-sd15); \
	if [ -z "$${IMAGE_TOOLS_E2E_REF_IMAGE_KEY:-}" ]; then \
		printf "SKIP gpu-e2e-ipadapter: set IMAGE_TOOLS_E2E_REF_IMAGE_KEY=<blob-key> (the reference image) to run.\n"; \
		exit 0; \
	fi; \
	out=$$(mktemp --suffix=.png); \
	image-tools ai generate --model "$(GPU_E2E_BASE_MODEL)" --prompt "in a sunlit meadow" --width "$(GPU_E2E_W)" --height "$(GPU_E2E_H)" --steps 20 --seed 42 --ip-adapter "ip-adapter-sd15:0.6:$$IMAGE_TOOLS_E2E_REF_IMAGE_KEY" --wait --out "$$out"; \
	test -s "$$out" || { printf "gpu-e2e-ipadapter failed: output empty\n"; exit 1; }; \
	printf "gpu-e2e-ipadapter produced %s\n" "$$out"

gpu-e2e-conditioning: gpu-e2e-lora gpu-e2e-controlnet gpu-e2e-ipadapter ## Umbrella: all conditioning-adapter attended GPU E2E targets

conditioning-freshness: ## Check manual GPU-e2e evidence freshness (valid_for_days window) in the conditioning/import requirement modules
	@set -eu; \
	if ! command -v jq >/dev/null 2>&1; then printf "SKIP conditioning-freshness: jq required.\n"; exit 0; fi; \
	now=$$(date +%s); stale=0; pending=0; fresh=0; \
	tmp=$$(mktemp); trap 'rm -f "$$tmp"' EXIT; \
	for mod in requirements/35-conditioning-adapters/module.json requirements/36-byo-import/module.json; do \
		jq -r '.requirements[] | . as $$r | (.validation // [])[] | select(.type=="manual" and (.valid_for_days != null)) | [$$r.id, (if (.last_validated_at // "") == "" then "-" else .last_validated_at end), (.valid_for_days|tostring)] | join("|")' "$$mod" >> "$$tmp"; \
	done; \
	while IFS="|" read -r rid ts days; do \
		[ -n "$$rid" ] || continue; \
		if [ "$$ts" = "-" ] || [ -z "$$ts" ] || [ "$$ts" = "null" ]; then \
			printf "PENDING  %-22s (never proven; adapter stays Ready=false)\n" "$$rid"; pending=$$((pending+1)); continue; \
		fi; \
		vts=$$(date -d "$$ts" +%s 2>/dev/null || echo 0); \
		deadline=$$((vts + days*86400)); \
		if [ "$$now" -gt "$$deadline" ]; then \
			printf "STALE    %-22s last=%s window=%sd — re-run the attended GPU e2e\n" "$$rid" "$$ts" "$$days"; stale=$$((stale+1)); \
		else \
			printf "FRESH    %-22s last=%s window=%sd\n" "$$rid" "$$ts" "$$days"; fresh=$$((fresh+1)); \
		fi; \
	done < "$$tmp"; \
	printf "conditioning-freshness: %s fresh, %s pending, %s stale\n" "$$fresh" "$$pending" "$$stale"; \
	if [ "$$stale" -gt 0 ]; then printf "conditioning-freshness FAILED: stale manual evidence.\n"; exit 1; fi

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

backends-doc: ## Regenerate the generated host-tool matrix in docs/reference/backends.md from providerSpecs()
	@cd api && UPDATE_BACKENDS_DOC=1 go test ./internal/ai/ -run TestBackendsDocHostToolMatrixUpToDate -count=1 >/dev/null && echo "backends.md host-tool matrix regenerated"

run: start

dev: start
