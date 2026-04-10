.PHONY: help build install test clean setup dev develop deploy status scenarios resources lifecycle-build validate-week0-week1

.DEFAULT_GOAL := help

BUILD_DIR := .vrooli/build
INSTALL_DIR := $(HOME)/.vrooli/bin
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILDINFO_PKG := github.com/vrooli/vrooli/internal/buildinfo
VROOLI_API_FINGERPRINT := $(shell go run ./cmd/vrooli-buildmeta --root . cmd/vrooli-api internal)
VROOLI_CLI_FINGERPRINT := $(shell go run ./cmd/vrooli-buildmeta --root . cmd/vrooli internal)
COMMON_LDFLAGS := -s -w \
	-X $(BUILDINFO_PKG).GitCommit=$(GIT_COMMIT) \
	-X $(BUILDINFO_PKG).BuildTime=$(BUILD_TIME)
VROOLI_API_LDFLAGS := $(COMMON_LDFLAGS) -X $(BUILDINFO_PKG).Fingerprint=$(VROOLI_API_FINGERPRINT)
VROOLI_CLI_LDFLAGS := $(COMMON_LDFLAGS) -X $(BUILDINFO_PKG).Fingerprint=$(VROOLI_CLI_FINGERPRINT)

help: ## Show available project-level targets
	@printf "Vrooli project-level Go targets\n\n"
	@printf "  make build          Build project-level Go binaries into %s\n" "$(BUILD_DIR)"
	@printf "  make install        Install project-level Go binaries into %s\n" "$(INSTALL_DIR)"
	@printf "  make test           Run project-level Go tests\n"
	@printf "  make clean          Remove project-level Go build artifacts\n"
	@printf "  make validate-week0-week1 Run the repeatable Week 0/1 acceptance suite\n"
	@printf "\nCompatibility helpers\n"
	@printf "  make setup          Run the existing setup workflow\n"
	@printf "  make dev            Start the existing development workflow\n"
	@printf "  make lifecycle-build Run the existing Bash/CLI build phase\n"

build: ## Build project-level Go binaries into .vrooli/build
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -trimpath -ldflags "$(VROOLI_API_LDFLAGS)" -o $(BUILD_DIR)/vrooli-api ./cmd/vrooli-api
	CGO_ENABLED=0 go build -trimpath -ldflags "$(VROOLI_CLI_LDFLAGS)" -o $(BUILD_DIR)/vrooli ./cmd/vrooli

install: build ## Install project-level Go binaries into ~/.vrooli/bin
	@mkdir -p $(INSTALL_DIR)
	install -m 0755 $(BUILD_DIR)/vrooli-api $(INSTALL_DIR)/vrooli-api
	install -m 0755 $(BUILD_DIR)/vrooli $(INSTALL_DIR)/vrooli

test: ## Run project-level Go tests
	go test ./internal/...
	go test ./cmd/vrooli-buildmeta
	go test ./cmd/vrooli
	go test -tags testing ./cmd/vrooli-api

validate-week0-week1: ## Run the repeatable Week 0/1 acceptance suite
	$(MAKE) clean
	$(MAKE) build
	$(MAKE) install
	$(MAKE) test
	~/.vrooli/bin/vrooli --version
	~/.vrooli/bin/vrooli info --list
	tmp_go=$$(mktemp); \
	tmp_bash=$$(mktemp); \
	~/.vrooli/bin/vrooli scenario list > "$$tmp_go"; \
	VROOLI_FORCE_BASH=1 ~/.vrooli/bin/vrooli scenario list > "$$tmp_bash"; \
	diff -u "$$tmp_go" "$$tmp_bash"; \
	rm -f "$$tmp_go" "$$tmp_bash"
	tmp_home=$$(mktemp -d); \
	touch "$$tmp_home/.bashrc"; \
	mkdir -p "$$tmp_home/.local/bin" "$$tmp_home/.vrooli/bin"; \
	ln -s "$$(pwd)/cli/vrooli" "$$tmp_home/.local/bin/vrooli"; \
	HOME="$$tmp_home" PATH="/usr/bin:/bin:$$PATH" ./cli/install.sh --force > /tmp/vrooli-install-smoke.log; \
	test -x "$$tmp_home/.vrooli/bin/vrooli"; \
	test ! -e "$$tmp_home/.local/bin/vrooli"; \
	grep -Fq "export PATH=\"$$tmp_home/.vrooli/bin:\$$PATH\"" "$$tmp_home/.bashrc"; \
	rm -rf "$$tmp_home"
	tmp_source=$$(mktemp); \
	cp cmd/vrooli/main.go "$$tmp_source"; \
	trap 'cp "$$tmp_source" cmd/vrooli/main.go; rm -f "$$tmp_source"; $(MAKE) install > /tmp/vrooli-restore.log' EXIT; \
	printf '\n// validation marker: stale-check smoke\n' >> cmd/vrooli/main.go; \
	before_sum=$$(sha256sum ~/.vrooli/bin/vrooli | awk '{print $$1}'); \
	~/.vrooli/bin/vrooli scenario list > /tmp/vrooli-stale-check-1.txt; \
	after_sum=$$(sha256sum ~/.vrooli/bin/vrooli | awk '{print $$1}'); \
	test "$$before_sum" != "$$after_sum"; \
	~/.vrooli/bin/vrooli scenario list > /tmp/vrooli-stale-check-2.txt; \
	final_sum=$$(sha256sum ~/.vrooli/bin/vrooli | awk '{print $$1}'); \
	test "$$after_sum" = "$$final_sum"; \
	cp "$$tmp_source" cmd/vrooli/main.go; \
	rm -f "$$tmp_source"; \
	trap - EXIT; \
	$(MAKE) install > /tmp/vrooli-restore.log
	tmpdir=$$(mktemp -d); \
	logfile="$$tmpdir/api.log"; \
	VROOLI_API_PORT=18092 APP_ROOT="$$(pwd)" VROOLI_ROOT="$$(pwd)" VROOLI_SOURCE_ROOT="$$(pwd)" api/start.sh > "$$logfile" 2>> "$$logfile" & \
	pid=$$!; \
	ok=0; \
	for attempt in $$(seq 1 20); do \
		if curl -fsS "http://127.0.0.1:18092/health" > /dev/null 2> /dev/null; then \
			ok=1; \
			break; \
		fi; \
		sleep 1; \
	done; \
	kill "$$pid" > /dev/null 2> /dev/null || true; \
	wait "$$pid" > /dev/null 2> /dev/null || true; \
	test "$$ok" = "1"; \
	rm -rf "$$tmpdir"

clean: ## Remove project-level Go build artifacts
	rm -rf $(BUILD_DIR)

setup: ## Run the existing setup workflow
	./scripts/manage.sh setup

dev: ## Start the existing development workflow
	vrooli develop

develop: dev ## Alias for make dev

deploy: ## Run the existing deploy workflow
	vrooli deploy

status: ## Show current Vrooli status
	vrooli status

scenarios: ## List scenarios through the existing CLI
	vrooli scenario list

resources: ## Show resource status through the existing CLI
	vrooli resource status

lifecycle-build: ## Run the existing Bash/CLI build phase
	vrooli build
