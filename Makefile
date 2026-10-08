.PHONY: help setup dev develop build install verify-supervisory verify-supervisory-build status stop fmt lint fmt-ts format-ts lint-ts type test debt cross-compile check-platforms vrooli-dist vrooli-dist-all capability-conformance acquisition-schema-check fleet-contract-check lint-portability lint-library-contracts lint-artifact-paths check hygiene fmt-packages lint-packages type-packages test-packages check-packages clean

.DEFAULT_GOAL := help

# The toolchain floor (GOFLAGS -p width, GOMAXPROCS, pnpm concurrency) and the
# root wildcard guard; every scenario Makefile includes the same file.
include mk/toolchain.mk

BUILD_DIR := .vrooli/build
INSTALL_DIR := $(HOME)/.vrooli/bin
SOURCE_ROOT_POINTER := $(HOME)/.vrooli/source-root
VROOLI := go run ./cmd/vrooli
SETUP_ARGS ?=
DEVELOP_ARGS ?=
help: ## Show the supported repo-level entrypoints
	@awk 'BEGIN {FS = ":.*## "; print "Vrooli project entrypoints\n"} /^[a-zA-Z0-9_.-]+:.*## / {printf "  make %-28s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Bootstrap and run project setup
	@if command -v vrooli >/dev/null 2>&1; then \
		VROOLI_SOURCE_ROOT="$(CURDIR)" vrooli setup $(SETUP_ARGS); \
	else \
		$(VROOLI) setup $(SETUP_ARGS); \
	fi

dev: develop ## Alias for make develop

develop: ## Start the development stack
	@$(VROOLI) develop $(DEVELOP_ARGS)

build: require-no-root-wildcard ## Build project-level binaries via the CLI
	@$(VROOLI) build

install: build verify-supervisory-build ## Install project-level binaries into ~/.vrooli/bin (refuses while a supervisory module does not build)
	@mkdir -p "$(INSTALL_DIR)"
	@go run ./cmd/vrooli-atomic-install "$(BUILD_DIR)/vrooli-api" "$(INSTALL_DIR)/vrooli-api"
	@go run ./cmd/vrooli-atomic-install "$(BUILD_DIR)/vrooli" "$(INSTALL_DIR)/vrooli"
	@go run ./cmd/vrooli-atomic-install "$(BUILD_DIR)/vrooli-agent-launcher" "$(INSTALL_DIR)/vrooli-agent-launcher"
	@go run ./cmd/vrooli-atomic-install "$(BUILD_DIR)/vrooli-policy-runner" "$(INSTALL_DIR)/vrooli-policy-runner"
	@printf '%s\n' "$(CURDIR)" > "$(SOURCE_ROOT_POINTER).new"
	@mv -f "$(SOURCE_ROOT_POINTER).new" "$(SOURCE_ROOT_POINTER)"

SUPERVISORY_MODULES := $(shell grep -v '^\#' tools/supervisory-modules.txt | grep -v '^$$')
SUPERVISORY_ROOT_PKGS := ./cmd/vrooli ./cmd/vrooli-watchdog ./internal/runtimesupervisor/... ./internal/safeguards/... ./internal/cli/rootcli/...
SUPERVISORY_TARGETS := darwin/amd64 darwin/arm64 windows/amd64 linux/arm64

verify-supervisory-build: ## Build and vet every module on the boot-recovery path (fast gate used by make install)
	@set -e; \
	echo "verify-supervisory: root packages"; \
	GOWORK=off go build -o /dev/null $(SUPERVISORY_ROOT_PKGS) && GOWORK=off go vet $(SUPERVISORY_ROOT_PKGS); \
	for m in $(SUPERVISORY_MODULES); do \
		echo "verify-supervisory: $$m"; \
		(cd "$$m" && GOWORK=off go build -o /dev/null ./... && GOWORK=off go vet ./...) || { echo "verify-supervisory: $$m does not build; refusing to continue" >&2; exit 1; }; \
	done

verify-supervisory: verify-supervisory-build ## Full boot-recovery gate: tests, cross-compilation, bootstrap script tests, invoker registry
	@set -e; \
	for m in $(SUPERVISORY_MODULES); do \
		echo "verify-supervisory: test $$m"; \
		(cd "$$m" && GOWORK=off go test ./...) || exit 1; \
		for t in $(SUPERVISORY_TARGETS); do \
			(cd "$$m" && GOWORK=off GOOS=$${t%/*} GOARCH=$${t#*/} go build -o /dev/null ./...) || { echo "verify-supervisory: $$m fails to cross-compile for $$t" >&2; exit 1; }; \
		done; \
	done; \
	for t in $(SUPERVISORY_TARGETS); do \
		GOWORK=off GOOS=$${t%/*} GOARCH=$${t#*/} go build -o /dev/null ./cmd/vrooli-watchdog ./internal/runtimesupervisor/... ./internal/safeguards/... || exit 1; \
	done; \
	bash scenarios/vrooli-bridge/bootstrap/bootstrap_test.sh >/dev/null; \
	GOWORK=off go test ./internal/cli/rootcli/...

status: ## Show project status
	@$(VROOLI) status

stop: ## Stop project services
	@$(VROOLI) stop

fmt: ## Format project-level Go code
	@if command -v gofumpt >/dev/null; then \
		gofumpt -w ./cmd ./internal; \
	elif command -v gofmt >/dev/null; then \
		gofmt -w ./cmd ./internal; \
	else \
		echo "Neither gofumpt nor gofmt is available"; \
		exit 1; \
	fi

lint: ## Lint project-level Go code
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run ./cmd/... ./internal/...; \
	else \
		echo "golangci-lint not installed; falling back to go vet"; \
		go vet ./cmd/... ./internal/...; \
	fi

type: require-no-root-wildcard ## Compile-check project-level Go packages without running tests
	@go test -run '^$$' ./cmd/... ./internal/...
	@go test -run '^$$' -tags testing ./cmd/vrooli-api

ONBOARDING_TS_FILES := src/App.tsx src/App.test.tsx src/types.ts src/lib/api.ts src/lib/api.surface.test.ts src/hooks/useWizardState.ts src/hooks/useWizardState.test.tsx src/components/wizard/WizardShell.tsx src/components/wizard/WizardShell.test.tsx src/components/wizard/stepRegistry.tsx src/components/wizard/stepRegistry.test.ts

fmt-ts: ## Format the TypeScript files touched by the onboarding contract work
	@pnpm --dir scenarios/vrooli-onboarding/ui exec prettier --write $(ONBOARDING_TS_FILES)

format-ts: fmt-ts ## Alias for format-ts

lint-ts: ## Check formatting for the onboarding UI files touched by this work
	@pnpm --dir scenarios/vrooli-onboarding/ui exec prettier --check $(ONBOARDING_TS_FILES)

cross-compile: ## Cross-compile every repository Go module discovered from repo-contract target roots.
	@set -eu; \
	for target in windows/amd64 windows/arm64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64; do \
		os=$${target%/*}; arch=$${target#*/}; \
		echo "go vet $${os}/$${arch}"; \
		GOOS=$${os} GOARCH=$${arch} GOWORK=off go vet ./internal/lifecycle/... ./internal/packagegov/...; \
		(cd packages/cli-core && GOOS=$${os} GOARCH=$${arch} GOWORK=off go vet ./...); \
		(cd packages/api-core && GOOS=$${os} GOARCH=$${arch} GOWORK=off go vet ./staleness/... ./preflight/...); \
		done

check-platforms: cross-compile capability-conformance ## Run the scheduled cross-platform compilation and vet gate.

generate-tuning-docs: ## Regenerate the operator-visible tuning lever reference.
	@go run ./cmd/vrooli-tuning-docs --root .

verify-tuning-docs: ## Fail when the generated tuning lever reference has drifted.
	@go run ./cmd/vrooli-tuning-docs --root . --check

vrooli-dist: ## Build a prebuilt vrooli CLI + .fp sidecar (GOOS=<os> GOARCH=<arch> [OUT=<path>] [VERSION=<tag>])
	@test -n "$(GOOS)" || { echo "GOOS is required" >&2; exit 2; }
	@test -n "$(GOARCH)" || { echo "GOARCH is required" >&2; exit 2; }
	@go run ./cmd/vrooli-dist --goos "$(GOOS)" --goarch "$(GOARCH)" --output "$(if $(OUT),$(OUT),dist/vrooli_$(GOOS)_$(GOARCH)$(if $(filter windows,$(GOOS)),.exe,))" --version "$(VERSION)"

vrooli-dist-all: ## Build the complete supported prebuilt CLI matrix into dist/
	@go run ./cmd/vrooli-dist --all --out-dir dist --version "$(VERSION)"

test: ## Run project-level Go tests
	@go test ./internal/...
	@go test ./cmd/vrooli-buildmeta
	@go test ./cmd/vrooli
	@go test -tags testing ./cmd/vrooli-api
	@cd cmd/vrooli-policy-runner && go test ./...

capability-conformance: ## Cross-compile every declared platform claim
	@$(VROOLI) capability conformance

acquisition-schema-check: ## Verify the generated acquisition contract is current
	@cd packages/binaryfetch && go test ./... -run '^TestAcquisitionSchemaDrift$$' -count=1

fleet-contract-check: ## Verify resource claims against the declared acquisition contract
	@go test ./internal/resources -run '^TestFleetContractRepository$$' -count=1

lint-portability: ## Run platform portability AST rules and enforce dated findings
	@set -eu; \
	command -v ast-grep >/dev/null 2>&1 || { echo 'ast-grep is required; install the governed tool described by internal/tools/ast-grep/tool.json' >&2; exit 1; }; \
	work_dir=$$(mktemp -d); \
	ast-grep scan --config sgconfig.yml --rule .ast-grep/rules/no-unguarded-shell-out.yml --globs '!**/*_test.go' --json > "$$work_dir/shell.json"; \
	ast-grep scan --config sgconfig.yml --rule .ast-grep/rules/no-unguarded-kernel-fs.yml --globs '!**/*_test.go' --globs '!**/*_linux.go' --globs '!**/*_darwin.go' --globs '!**/*_windows.go' --globs '!**/*_unix.go' --globs '!**/*_other.go' --json > "$$work_dir/kernel.json"; \
	ast-grep scan --config sgconfig.yml --rule .ast-grep/rules/no-private-retention-walk.yml --globs '!**/*_test.go'; \
	ast-grep scan --config sgconfig.yml --rule .ast-grep/rules/no-raw-toolchain-spawn.yml --globs '!**/*_test.go' --json > "$$work_dir/toolchain.json"; \
	if [ "$$(jq 'length' "$$work_dir/toolchain.json")" != 0 ]; then jq -r '.[] | "raw toolchain spawn outside the floor: \(.file):\(.range.start.line)"' "$$work_dir/toolchain.json" >&2; exit 1; fi; \
	jq -e 'all(.entries[]; .rule != "no-raw-toolchain-spawn")' .vrooli/portability-lint-allowlist.json >/dev/null || { echo 'no-raw-toolchain-spawn admits no allowlist entries; migrate the site onto envkit.Toolchain or shell.Command' >&2; exit 1; }; \
	jq -e 'all(.entries[]; (.reason | length > 0) and (.review_by | test("^[0-9]{4}-[0-9]{2}-[0-9]{2}$$")))' .vrooli/portability-lint-allowlist.json >/dev/null; \
	for rule_file in shell kernel; do \
		rule=$$(if [ "$$rule_file" = shell ]; then echo no-unguarded-shell-out; else echo no-unguarded-kernel-fs; fi); \
		jq -c '.[]' "$$work_dir/$$rule_file.json" | while read -r finding; do \
			path=$$(printf '%s' "$$finding" | jq -r '.file'); line=$$(printf '%s' "$$finding" | jq -r '.range.start.line'); \
			jq -e --arg rule "$$rule" --arg path "$$path" --arg line "$$line" 'any(.entries[]; .rule == $$rule and .path == $$path and .line == ($$line | tonumber))' .vrooli/portability-lint-allowlist.json >/dev/null || { echo "unallowlisted portability finding: $$rule $$path:$$line" >&2; exit 1; }; \
		done; \
	done

lint-library-contracts: ## Enforce shared-library source contracts (no bundler-injected environment)
	@set -eu; \
	command -v ast-grep >/dev/null 2>&1 || { echo 'ast-grep is required; install the governed tool described by internal/tools/ast-grep/tool.json' >&2; exit 1; }; \
	ast-grep scan --config sgconfig.yml --rule .ast-grep/rules/no-bundler-env-in-shared-library.yml

lint-artifact-paths: ## Reject hardcoded generated-artifact locations (see docs/reference/storage-retention.md)
	@set -eu; \
	command -v ast-grep >/dev/null 2>&1 || { echo 'ast-grep is required; install the governed tool described by internal/tools/ast-grep/tool.json' >&2; exit 1; }; \
	ast-grep scan --config sgconfig.yml --rule .ast-grep/rules/no-artifact-path-literal.yml --globs '!**/*_test.go'

check: lint type test check-packages acquisition-schema-check fleet-contract-check lint-portability lint-library-contracts lint-artifact-paths ## Run lint, type, and test quality gates (core + packages)

hygiene: ## Run repository hygiene checks
	@if command -v vrooli >/dev/null 2>&1; then \
		VROOLI_SOURCE_ROOT="$(CURDIR)" vrooli hygiene; \
	else \
		$(VROOLI) hygiene; \
	fi
	@go run ./internal/deployability/cmd/platform-support --check

fmt-packages: ## Format Go code in packages/*
	@vrooli package list --json | jq -r '.packages[] | select(.manifest.package.language == "go") | .root_path' | xargs -r -n1 gofumpt -w

lint-packages: ## Lint Go code in packages/*
	@vrooli package list --json | jq -r '.packages[] | select(.manifest.package.language == "go") | .root_path' | xargs -r -n1 sh -c 'cd "$$0" && golangci-lint run ./...'

type-packages: ## Compile-check Go packages in packages/*
	@vrooli package list --json | jq -r '.packages[] | select(.manifest.package.language == "go") | .root_path' | xargs -r -n1 sh -c 'cd "$$0" && go test -run "^$$" ./...'

test-packages: ## Run Go tests in packages/*
	@vrooli package list --json | jq -r '.packages[].name' | xargs -r -n1 vrooli package test

check-packages: ## Run quality gates across packages/*
	@$(MAKE) fmt-packages lint-packages type-packages test-packages

clean: ## Clean build artifacts via the CLI
	@$(VROOLI) clean
