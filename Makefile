.PHONY: help build install test clean setup dev develop deploy status scenarios resources lifecycle-build validate-week0-week1 validate-week2 validate-week0-week2

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
	@printf "  make validate-week2 Run the repeatable Week 2 acceptance suite\n"
	@printf "  make validate-week0-week2 Run the combined Week 0-2 acceptance suite\n"
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
	VROOLI_ROOT="$$(pwd)" VROOLI_SOURCE_ROOT="$$(pwd)" ~/.vrooli/bin/vrooli --version
	VROOLI_ROOT="$$(pwd)" VROOLI_SOURCE_ROOT="$$(pwd)" ~/.vrooli/bin/vrooli info --list
	tmp_go=$$(mktemp); \
	tmp_bash=$$(mktemp); \
	VROOLI_ROOT="$$(pwd)" VROOLI_SOURCE_ROOT="$$(pwd)" ~/.vrooli/bin/vrooli scenario list > "$$tmp_go"; \
	VROOLI_ROOT="$$(pwd)" VROOLI_SOURCE_ROOT="$$(pwd)" VROOLI_FORCE_BASH=1 ~/.vrooli/bin/vrooli scenario list > "$$tmp_bash"; \
	grep -Fq "test-genie" "$$tmp_go"; \
	grep -Fq "test-genie" "$$tmp_bash"; \
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
	VROOLI_ROOT="$$(pwd)" VROOLI_SOURCE_ROOT="$$(pwd)" ~/.vrooli/bin/vrooli scenario list > /tmp/vrooli-stale-check-1.txt; \
	after_sum=$$(sha256sum ~/.vrooli/bin/vrooli | awk '{print $$1}'); \
	test "$$before_sum" != "$$after_sum"; \
	VROOLI_ROOT="$$(pwd)" VROOLI_SOURCE_ROOT="$$(pwd)" ~/.vrooli/bin/vrooli scenario list > /tmp/vrooli-stale-check-2.txt; \
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
	tmpdir=$$(mktemp -d); \
	logfile="$$tmpdir/develop.log"; \
	repo_root="$$(pwd)"; \
	pid=""; \
	ok=0; \
	cleanup() { \
		if [ -n "$$pid" ]; then \
			kill "$$pid" > /dev/null 2> /dev/null || true; \
			wait "$$pid" > /dev/null 2> /dev/null || true; \
		fi; \
		( cd scenarios/vrooli-orchestrator && VROOLI_ROOT="$$repo_root" VROOLI_SOURCE_ROOT="$$repo_root" make stop > /tmp/vrooli-orchestrator-stop.log 2> /dev/null ) || true; \
		rm -rf "$$tmpdir"; \
	}; \
	trap cleanup EXIT; \
	VROOLI_API_PORT=18093 APP_ROOT="$$(pwd)" VROOLI_ROOT="$$(pwd)" VROOLI_SOURCE_ROOT="$$(pwd)" ~/.vrooli/bin/vrooli develop > "$$logfile" 2>> "$$logfile" & \
	pid=$$!; \
	for attempt in $$(seq 1 90); do \
		if curl -fsS "http://127.0.0.1:18093/health" > /dev/null 2> /dev/null; then \
			ok=1; \
			break; \
		fi; \
		sleep 2; \
	done; \
	if [ "$$ok" != "1" ]; then \
		tail -n 80 "$$logfile"; \
		exit 1; \
	fi; \
	trap - EXIT; \
	cleanup

validate-week2: ## Run the repeatable Week 2 acceptance suite
	$(MAKE) build
	$(MAKE) install
	$(MAKE) test
	tmp_root=$$(mktemp -d); \
	tmp_home=$$(mktemp -d); \
	sandbox_root=$$(mktemp -d); \
	api_dir="$$tmp_root/http-api"; \
	ui_dir="$$tmp_root/http-ui"; \
	api_pid=""; \
	ui_pid=""; \
	cleanup() { \
		if [ -n "$$api_pid" ]; then \
			kill "$$api_pid" > /dev/null 2> /dev/null || true; \
			wait "$$api_pid" > /dev/null 2> /dev/null || true; \
		fi; \
		if [ -n "$$ui_pid" ]; then \
			kill "$$ui_pid" > /dev/null 2> /dev/null || true; \
			wait "$$ui_pid" > /dev/null 2> /dev/null || true; \
		fi; \
		rm -rf "$$tmp_root" "$$tmp_home" "$$sandbox_root"; \
	}; \
	trap cleanup EXIT; \
	mkdir -p "$$tmp_root/scenarios/alpha/.vrooli" "$$tmp_root/scenarios/beta/.vrooli" "$$tmp_root/scenarios/_artifacts" "$$api_dir" "$$ui_dir" "$$sandbox_root/.vrooli"; \
	printf 'ok\n' > "$$api_dir/health"; \
	printf 'ok\n' > "$$ui_dir/health"; \
	jq -n \
		'{version:"1.0.0", service:{name:"alpha", displayName:"Alpha", description:"Alpha scenario", version:"0.1.0"}, ports:{api:{env_var:"API_PORT", range:"15000-19999"}, ui:{env_var:"UI_PORT", range:"35000-39999"}, websocket:{env_var:"WS_PORT", range:"25000-29999"}}, lifecycle:{version:"2.0.0", health:{checks:[{name:"api", type:"http", target:"http://127.0.0.1:$${API_PORT}/health", critical:true, timeout:1000},{name:"ui", type:"http", target:"http://127.0.0.1:$${UI_PORT}/health", critical:false, timeout:1000}]}, develop:{description:"Run alpha", steps:[{name:"start-api", run:"python3 -m http.server", background:true},{name:"start-ui", run:"python3 -m http.server", background:true}]}}}' \
		> "$$tmp_root/scenarios/alpha/.vrooli/service.json"; \
	jq -n \
		'{version:"1.0.0", service:{name:"beta", displayName:"Beta", description:"Beta scenario", version:"0.1.0"}, ports:{api:{env_var:"API_PORT", range:"15000-19999"}}, lifecycle:{version:"2.0.0", develop:{description:"Run beta", steps:[{name:"start-api", run:"python3 -m http.server", background:true}]}}}' \
		> "$$tmp_root/scenarios/beta/.vrooli/service.json"; \
	jq -n \
		'{version:"1.0.0", service:{name:"alpha", displayName:"Alpha Sandbox", description:"Sandbox alpha", version:"0.2.0"}, ports:{api:{env_var:"API_PORT", range:"15000-19999"}, ui:{env_var:"UI_PORT", range:"35000-39999"}, websocket:{env_var:"WS_PORT", range:"25000-29999"}}, lifecycle:{version:"2.0.0", develop:{description:"Run alpha from sandbox", steps:[{name:"start-api", run:"python3 -m http.server", background:true},{name:"start-ui", run:"python3 -m http.server", background:true}]}}}' \
		> "$$sandbox_root/.vrooli/service.json"; \
	API_PORT=18080 UI_PORT=38080 WS_PORT=28080 python3 -m http.server 18080 --bind 127.0.0.1 --directory "$$api_dir" > "$$tmp_root/api.log" 2>> "$$tmp_root/api.log" & \
	api_pid=$$!; \
	API_PORT=18080 UI_PORT=38080 WS_PORT=28080 python3 -m http.server 38080 --bind 127.0.0.1 --directory "$$ui_dir" > "$$tmp_root/ui.log" 2>> "$$tmp_root/ui.log" & \
	ui_pid=$$!; \
	ok=0; \
	for attempt in $$(seq 1 20); do \
		if curl -fsS "http://127.0.0.1:18080/health" > /dev/null 2> /dev/null && curl -fsS "http://127.0.0.1:38080/health" > /dev/null 2> /dev/null; then \
			ok=1; \
			break; \
		fi; \
		sleep 1; \
	done; \
	test "$$ok" = "1"; \
	now=$$(date -u +%Y-%m-%dT%H:%M:%SZ); \
	mkdir -p "$$tmp_home/.vrooli/processes/scenarios/alpha"; \
	jq -n --argjson pid "$$api_pid" --arg ts "$$now" \
		'{pid:$$pid, pgid:$$pid, process_id:"vrooli.develop.alpha.start-api", phase:"develop", scenario:"alpha", step:"start-api", command:"python3 -m http.server 18080", working_dir:"/fixture/scenarios/alpha", log_file:"/tmp/alpha-api.log", port:18080, started_at:$$ts, status:"running"}' \
		> "$$tmp_home/.vrooli/processes/scenarios/alpha/start-api.json"; \
	jq -n --argjson pid "$$ui_pid" --arg ts "$$now" \
		'{pid:$$pid, pgid:$$pid, process_id:"vrooli.develop.alpha.start-ui", phase:"develop", scenario:"alpha", step:"start-ui", command:"python3 -m http.server 38080", working_dir:"/fixture/scenarios/alpha", log_file:"/tmp/alpha-ui.log", port:38080, started_at:$$ts, status:"running"}' \
		> "$$tmp_home/.vrooli/processes/scenarios/alpha/start-ui.json"; \
	HOME="$$tmp_home" VROOLI_ROOT="$$tmp_root" ~/.vrooli/bin/vrooli --no-stale-check scenario list --json > "$$tmp_root/list.json"; \
	jq -e '.success == true and .summary.total_scenarios == 2 and .summary.running == 1' "$$tmp_root/list.json" > /dev/null; \
	jq -e '[.scenarios[].name] == ["alpha","beta"]' "$$tmp_root/list.json" > /dev/null; \
	HOME="$$tmp_home" VROOLI_ROOT="$$tmp_root" ~/.vrooli/bin/vrooli --no-stale-check scenario list --json --include-ports > "$$tmp_root/list-ports.json"; \
	jq -e '.scenarios[] | select(.name == "alpha") | .ports | length == 2' "$$tmp_root/list-ports.json" > /dev/null; \
	HOME="$$tmp_home" VROOLI_ROOT="$$tmp_root" VROOLI_SANDBOX_MERGED="$$sandbox_root" VROOLI_SANDBOX_SCOPE="scenarios/alpha" ~/.vrooli/bin/vrooli --no-stale-check scenario info alpha --json > "$$tmp_root/info.json"; \
	jq -e '.scenario.description == "Sandbox alpha" and .scenario.sandbox_redirected == true' "$$tmp_root/info.json" > /dev/null; \
	HOME="$$tmp_home" VROOLI_ROOT="$$tmp_root" VROOLI_API_PORT=1 ~/.vrooli/bin/vrooli --no-stale-check scenario status alpha --json > "$$tmp_root/status.json"; \
	jq -e '.scenario.status == "running" and .scenario.health_status == "healthy" and .runtime.ports.API_PORT == 18080 and .runtime.ports.UI_PORT == 38080 and .runtime.ports.WS_PORT == 28080' "$$tmp_root/status.json" > /dev/null; \
	HOME="$$HOME" VROOLI_ROOT="$$(pwd)" VROOLI_SOURCE_ROOT="$$(pwd)" VROOLI_FORCE_BASH=1 ~/.vrooli/bin/vrooli scenario list > "$$tmp_root/fallback.txt"; \
	grep -Fq "test-genie" "$$tmp_root/fallback.txt"; \
	trap - EXIT; \
	cleanup

validate-week0-week2: ## Run the combined Week 0-2 acceptance suite
	$(MAKE) validate-week0-week1
	$(MAKE) validate-week2

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
