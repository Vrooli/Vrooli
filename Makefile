.PHONY: help setup dev develop build install status stop fmt lint type test check hygiene fmt-packages lint-packages type-packages test-packages check-packages clean

.DEFAULT_GOAL := help

BUILD_DIR := .vrooli/build
INSTALL_DIR := $(HOME)/.vrooli/bin
SOURCE_ROOT_POINTER := $(HOME)/.vrooli/source-root
VROOLI := go run ./cmd/vrooli --no-stale-check
SETUP_ARGS ?=
DEVELOP_ARGS ?=
PACKAGE_DIRS := packages/api-core packages/cli-core packages/repo-contract-go

help: ## Show the supported repo-level entrypoints
	@printf "Vrooli project entrypoints\n\n"
	@printf "  make setup                      Bootstrap and run project setup\n"
	@printf "  make develop                    Start the development stack\n"
	@printf "  make build                      Build project-level binaries via the CLI\n"
	@printf "  make install                    Install project-level binaries to %s\n" "$(INSTALL_DIR)"
	@printf "  make status                     Show project status\n"
	@printf "  make stop                       Stop project services\n"
	@printf "  make fmt                        Format project-level Go code\n"
	@printf "  make lint                       Lint project-level Go code\n"
	@printf "  make type                       Compile-check project-level Go packages\n"
	@printf "  make test                       Run project-level Go tests\n"
	@printf "  make check                      Run lint, type, and test quality gates (core + packages)\n"
	@printf "  make hygiene                    Run repository hygiene checks\n"
	@printf "  make fmt-packages               Format Go code in packages/*\n"
	@printf "  make lint-packages              Lint Go code in packages/*\n"
	@printf "  make type-packages              Compile-check Go packages in packages/*\n"
	@printf "  make test-packages              Run Go tests in packages/*\n"
	@printf "  make check-packages             Run quality gates across packages/*\n"
	@printf "  make clean                      Clean build artifacts via the CLI\n"

setup: ## Bootstrap and run project setup
	@if command -v vrooli >/dev/null 2>&1; then \
		VROOLI_SOURCE_ROOT="$(CURDIR)" vrooli --no-stale-check setup $(SETUP_ARGS); \
	else \
		$(VROOLI) setup $(SETUP_ARGS); \
	fi

dev: develop ## Alias for make develop

develop: ## Start the development stack
	@$(VROOLI) develop $(DEVELOP_ARGS)

build: ## Build project-level binaries via the CLI
	@$(VROOLI) build

# atomic_install — copy SRC ($1) to DST ($2) crash-safely.
#
# Why this dance instead of a single `install -m 0755 SRC DST`:
#   GNU `install` opens DST with O_TRUNC, writes, closes — no fsync, no
#   rename. On ext4 (data=ordered, the Ubuntu default) the inode size+mode
#   update can land in the journal before the data blocks reach the platter.
#   If the box loses power inside that ~5–30 s window, DST is left as a
#   zero-filled stub with the executable bit still set: PATH resolves it,
#   but exec(2) returns ENOEXEC ("Exec format error"). We hit exactly this
#   on 2026-05-07 — an in-flight `vrooli` install was caught by a crash and
#   the binary became unrecoverable, breaking every shell that relied on it.
#
# Crash-safe sequence:
#   1. install --> "$2.new". Writing a *different* path means DST itself is
#      never opened with O_TRUNC; if we crash here, the previous good DST
#      is still on disk and still runnable.
#   2. sync -f "$2.new". Flushes the filesystem holding "$2.new", forcing
#      the data blocks to stable storage BEFORE the rename makes them
#      visible under DST's name.
#   3. mv -f --> rename(2), atomic within one filesystem: DST points at the
#      old inode or the fully-written new inode, never a partial state.
#   4. sync -f "$2". Persists the directory entry change itself so a crash
#      after rename(2) can't roll the entry back or leave it dangling.
define atomic_install
	install -m 0755 "$(1)" "$(2).new" && \
	sync -f "$(2).new" && \
	mv -f "$(2).new" "$(2)" && \
	sync -f "$(2)"
endef

install: build ## Install project-level binaries into ~/.vrooli/bin
	@mkdir -p "$(INSTALL_DIR)"
	@$(call atomic_install,$(BUILD_DIR)/vrooli-api,$(INSTALL_DIR)/vrooli-api)
	@$(call atomic_install,$(BUILD_DIR)/vrooli,$(INSTALL_DIR)/vrooli)
	@$(call atomic_install,$(BUILD_DIR)/vrooli-policy-runner,$(INSTALL_DIR)/vrooli-policy-runner)
	@printf '%s\n' "$(CURDIR)" > "$(SOURCE_ROOT_POINTER).new"
	@mv -f "$(SOURCE_ROOT_POINTER).new" "$(SOURCE_ROOT_POINTER)"

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

type: ## Compile-check project-level Go packages without running tests
	@go test -run '^$$' ./cmd/... ./internal/...
	@go test -run '^$$' -tags testing ./cmd/vrooli-api

cross-compile: ## Cross-compile every repository Go module; modules under node_modules/vendor are excluded by discovery with that reason in this target.
	@set -eu; \
	modules=$$(find . -name go.mod -not -path '*/node_modules/*' -not -path '*/vendor/*' -print | sort); \
	for mod in $$modules; do \
		dir=$${mod%/go.mod}; \
		case "$$dir" in \
			./templates/*) echo "==> SKIP cross-compile $$dir (darwin/arm64): source template contains generation placeholders and is not a buildable module until instantiated" ;; \
			./scenarios/flow-verifier/api) echo "==> SKIP cross-compile $$dir (darwin/arm64): repository template retains {{SCENARIO_ID}} imports and missing generated channelmanager/forest packages; module is not a buildable scenario artifact" ;; \
			./scenarios/go-code-graph/bas/fixtures/go-cycles) echo "==> SKIP cross-compile $$dir (darwin/arm64): intentional import-cycle fixture for graph detection tests, not a buildable module" ;; \
			*) echo "==> cross-compile $$dir (darwin/arm64)"; \
				(cd "$$dir" && GOFLAGS=-mod=mod GOOS=darwin GOARCH=arm64 go build ./...) ;; \
		esac; \
		case "$$dir" in \
			./templates/*) echo "==> SKIP cross-compile $$dir (windows/amd64): source template contains generation placeholders and is not a buildable module until instantiated" ;; \
			./scenarios/api-library/api) echo "==> SKIP cross-compile $$dir (windows/amd64): pinned github.com/redis/go-redis/v9 fails upstream with undefined errUnexpectedRead" ;; \
			./scenarios/flow-verifier/api) echo "==> SKIP cross-compile $$dir (windows/amd64): repository template retains {{SCENARIO_ID}} imports and missing generated channelmanager/forest packages; module is not a buildable scenario artifact" ;; \
			./scenarios/go-code-graph/bas/fixtures/go-cycles) echo "==> SKIP cross-compile $$dir (windows/amd64): intentional import-cycle fixture for graph detection tests, not a buildable module" ;; \
			*) echo "==> cross-compile $$dir (windows/amd64)"; \
				(cd "$$dir" && GOFLAGS=-mod=mod GOOS=windows GOARCH=amd64 go build ./...) ;; \
		esac; \
		done

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

check: lint type cross-compile test check-packages ## Run lint, type, cross-compile, and test quality gates (core + packages)

hygiene: ## Run repository hygiene checks
	@$(VROOLI) hygiene

fmt-packages: ## Format Go code in packages/*
	@for dir in $(PACKAGE_DIRS); do \
		printf "==> fmt %s\n" "$$dir"; \
		$(MAKE) -C "$$dir" fmt || exit $$?; \
	done

lint-packages: ## Lint Go code in packages/*
	@for dir in $(PACKAGE_DIRS); do \
		printf "==> lint %s\n" "$$dir"; \
		$(MAKE) -C "$$dir" lint || exit $$?; \
	done

type-packages: ## Compile-check Go packages in packages/*
	@for dir in $(PACKAGE_DIRS); do \
		printf "==> type %s\n" "$$dir"; \
		$(MAKE) -C "$$dir" type || exit $$?; \
	done

test-packages: ## Run Go tests in packages/*
	@for dir in $(PACKAGE_DIRS); do \
		printf "==> test %s\n" "$$dir"; \
		$(MAKE) -C "$$dir" test || exit $$?; \
	done
	@printf "==> test packages/audio-capture-browser\n"
	@vrooli package test audio-capture-browser

check-packages: ## Run quality gates across packages/*
	@for dir in $(PACKAGE_DIRS); do \
		printf "==> check %s\n" "$$dir"; \
		$(MAKE) -C "$$dir" check || exit $$?; \
	done

clean: ## Clean build artifacts via the CLI
	@$(VROOLI) clean
