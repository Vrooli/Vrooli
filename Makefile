.PHONY: help build install test validate clean setup dev develop deploy status scenarios resources lifecycle-build validate-repo-contract validate-phase6-host-setup-cleanup validate-live-develop-smoke validate-week0-week1 validate-week3-live validate-week5-cross validate-week6-slice

.DEFAULT_GOAL := help

BUILD_DIR := .vrooli/build
INSTALL_DIR = $(HOME)/.vrooli/bin
VROOLI_BIN = $(INSTALL_DIR)/vrooli
SETUP_ARGS ?=
DEV_ARGS ?=
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILDINFO_PKG := github.com/vrooli/vrooli/internal/buildinfo
COMMON_LDFLAGS := -s -w \
	-X $(BUILDINFO_PKG).GitCommit=$(GIT_COMMIT) \
	-X $(BUILDINFO_PKG).BuildTime=$(BUILD_TIME)

help: ## Show available project-level targets
	@printf "Vrooli project-level Go targets\n\n"
	@printf "  make build          Build project-level Go binaries into %s\n" "$(BUILD_DIR)"
	@printf "  make install        Install project-level Go binaries into %s\n" "$(INSTALL_DIR)"
	@printf "  make test           Run project-level Go tests\n"
	@printf "  make validate-repo-contract Validate the repo contract schema, data, and drift checks\n"
	@printf "  make validate-phase6-host-setup-cleanup Validate deleted host-setup surfaces stay deleted\n"
	@printf "  make validate       Run the retained project-level validation suite\n"
	@printf "  make clean          Remove project-level Go build artifacts\n"
	@printf "\nProject helpers\n"
	@printf "  make setup          Bootstrap the Go CLI and run native setup\n"
	@printf "  make dev            Start the native development workflow\n"
	@printf "  make lifecycle-build Run the native project build command via the installed CLI\n"
	@printf "\nFocused validation slices\n"
	@printf "  make validate-week0-week1 Run installed-binary and stale-check smokes\n"
	@printf "  make validate-week3-live Run live scenario lifecycle validation\n"
	@printf "  make validate-week5-cross Run cross-compilation validation\n"
	@printf "  make validate-week6-slice Run native project command validation\n"

build: ## Build project-level Go binaries into .vrooli/build
	@mkdir -p $(BUILD_DIR)
	VROOLI_API_FINGERPRINT="$$(go run ./cmd/vrooli-buildmeta --root . cmd/vrooli-api internal)"; \
	CGO_ENABLED=0 go build -trimpath -ldflags "$(COMMON_LDFLAGS) -X $(BUILDINFO_PKG).Fingerprint=$$VROOLI_API_FINGERPRINT" -o $(BUILD_DIR)/vrooli-api ./cmd/vrooli-api
	VROOLI_CLI_FINGERPRINT="$$(go run ./cmd/vrooli-buildmeta --root . cmd/vrooli internal)"; \
	CGO_ENABLED=0 go build -trimpath -ldflags "$(COMMON_LDFLAGS) -X $(BUILDINFO_PKG).Fingerprint=$$VROOLI_CLI_FINGERPRINT" -o $(BUILD_DIR)/vrooli ./cmd/vrooli

install: build ## Install project-level Go binaries into ~/.vrooli/bin
	@mkdir -p $(INSTALL_DIR)
	install -m 0755 $(BUILD_DIR)/vrooli-api $(INSTALL_DIR)/vrooli-api
	install -m 0755 $(BUILD_DIR)/vrooli $(INSTALL_DIR)/vrooli

test: ## Run project-level Go tests
	$(MAKE) validate-repo-contract
	go test ./internal/...
	go test ./cmd/vrooli-buildmeta
	go test ./cmd/vrooli
	go test -tags testing ./cmd/vrooli-api

validate: ## Run the retained project-level validation suite
	$(MAKE) test
	$(MAKE) validate-phase6-host-setup-cleanup
	$(MAKE) validate-week0-week1
	$(MAKE) validate-week3-live
	$(MAKE) validate-week5-cross
	$(MAKE) validate-week6-slice

validate-repo-contract: ## Validate the repo contract schema, data, and drift checks
	python3 .vrooli/schemas/validate-repo-contract.py
	go test ./internal/repocontract

validate-phase6-host-setup-cleanup: ## Validate deleted host-setup surfaces stay deleted
	go test ./internal/setup -run TestRepoRemovesLegacyHostSetupSurfaces -count=1
	! rg -n 'scripts/lib/setup\.sh|scripts/lib/setup-conditions|scripts/lib/deps/(ajv|ast-grep|bats|js-yaml|lychee|shellcheck)\.sh' internal/lifecycle/setup.go cmd .vrooli scripts/README.md docs/GETTING_STARTED.md docs/devops/README.md scenarios/scenario-to-cloud/api/bundling_rules_test.go

validate-week0-week1: ## Run installed-binary and stale-check smoke validation
	$(MAKE) build
	$(MAKE) install
	VROOLI_ROOT="$$(pwd)" VROOLI_SOURCE_ROOT="$$(pwd)" ~/.vrooli/bin/vrooli --version
	VROOLI_ROOT="$$(pwd)" VROOLI_SOURCE_ROOT="$$(pwd)" ~/.vrooli/bin/vrooli info --list
	tmp_home=$$(mktemp -d); \
	HOME="$$tmp_home" $(MAKE) install > /tmp/vrooli-install-smoke.log; \
	test -x "$$tmp_home/.vrooli/bin/vrooli"; \
	test -x "$$tmp_home/.vrooli/bin/vrooli-api"; \
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
	$(MAKE) validate-live-develop-smoke

validate-live-develop-smoke: ## Run the live repo-root vrooli develop smoke
	tmpdir=$$(mktemp -d); \
	logfile="$$tmpdir/develop.log"; \
	repo_root="$$(pwd)"; \
	api_port=18093; \
	pid=""; \
	ok=0; \
	cleanup() { \
		if [ -n "$$pid" ]; then \
			kill "$$pid" > /dev/null 2> /dev/null || true; \
			wait "$$pid" > /dev/null 2> /dev/null || true; \
		fi; \
		if command -v lsof > /dev/null 2> /dev/null; then \
			api_pids=$$(lsof -tiTCP:$$api_port -sTCP:LISTEN 2> /dev/null || true); \
			for api_pid in $$api_pids; do \
				kill "$$api_pid" > /dev/null 2> /dev/null || true; \
			done; \
		fi; \
		( cd scenarios/vrooli-orchestrator && VROOLI_ROOT="$$repo_root" VROOLI_SOURCE_ROOT="$$repo_root" make stop > /tmp/vrooli-orchestrator-stop.log 2> /dev/null ) || true; \
		rm -rf "$$tmpdir"; \
	}; \
	trap cleanup EXIT; \
	VROOLI_API_PORT=$$api_port APP_ROOT="$$(pwd)" VROOLI_ROOT="$$(pwd)" VROOLI_SOURCE_ROOT="$$(pwd)" ~/.vrooli/bin/vrooli develop > "$$logfile" 2>> "$$logfile" & \
	pid=$$!; \
	for attempt in $$(seq 1 90); do \
		if curl -fsS "http://127.0.0.1:$$api_port/health" > /dev/null 2> /dev/null; then \
			ok=1; \
			break; \
		fi; \
		if ! kill -0 "$$pid" > /dev/null 2> /dev/null; then \
			tail -n 80 "$$logfile"; \
			exit 1; \
		fi; \
		sleep 2; \
	done; \
	if [ "$$ok" != "1" ]; then \
		tail -n 80 "$$logfile"; \
		exit 1; \
	fi; \
	trap - EXIT; \
	cleanup

validate-week3-live: ## Run live Week 3 native scenario smokes
	$(MAKE) build
	$(MAKE) install
	@set -eu; \
	repo_root="$$(pwd)"; \
	live_dir="$$(mktemp -d)"; \
	cleanup() { \
		rm -rf "$$live_dir"; \
	}; \
	trap cleanup EXIT; \
	require_scenario() { \
		scenario="$$1"; \
		api_binary="$$2"; \
		cli_name="$$3"; \
		test -x "$$repo_root/scenarios/$$scenario/api/$$api_binary"; \
		test -f "$$repo_root/scenarios/$$scenario/ui/dist/index.html"; \
		command -v "$$cli_name" > /dev/null 2> /dev/null; \
	}; \
	snapshot_records() { \
		home="$$1"; \
		scenario="$$2"; \
		out="$$3"; \
		process_dir="$$home/.vrooli/processes/scenarios/$$scenario"; \
		if [ -d "$$process_dir" ]; then \
			find "$$process_dir" -maxdepth 1 -name '*.json' ! -name 'degraded.json' -print | sort | while IFS= read -r file; do \
				jq -c -S '{step,phase,scenario,status,working_dir,command,has_port:(.port > 0)}' "$$file"; \
			done | jq -s 'sort_by(.step)' > "$$out"; \
		else \
			printf '[]\n' > "$$out"; \
		fi; \
	}; \
	snapshot_locks() { \
		home="$$1"; \
		out="$$2"; \
		state_dir="$$home/.vrooli/state/scenarios"; \
		if [ -d "$$state_dir" ]; then \
			find "$$state_dir" -maxdepth 1 -name '.port_*.lock' -print | sort | while IFS= read -r file; do \
				owner="$$(cut -d: -f1 "$$file")"; \
				jq -nc --arg owner "$$owner" '{owner:$$owner}'; \
			done | jq -s 'sort_by(.owner)' > "$$out"; \
		else \
			printf '[]\n' > "$$out"; \
		fi; \
	}; \
	write_meta() { \
		home="$$1"; \
		scenario="$$2"; \
		expect_db="$$3"; \
		out="$$4"; \
		lifecycle_log="$$home/.vrooli/logs/$$scenario.log"; \
		api_log="$$home/.vrooli/logs/scenarios/$$scenario/vrooli.develop.$$scenario.start-api.log"; \
		ui_log="$$home/.vrooli/logs/scenarios/$$scenario/vrooli.develop.$$scenario.start-ui.log"; \
		ensure_db=false; \
		start_api=false; \
		start_ui=false; \
		api_log_exists=false; \
		ui_log_exists=false; \
		api_bak_exists=false; \
		ui_bak_exists=false; \
		if [ -f "$$lifecycle_log" ] && [ "$$expect_db" = "true" ] && grep -Fq "Ensuring database exists:" "$$lifecycle_log"; then \
			ensure_db=true; \
		fi; \
		if [ -f "$$lifecycle_log" ] && grep -Fq "start-api" "$$lifecycle_log"; then \
			start_api=true; \
		fi; \
		if [ -f "$$lifecycle_log" ] && grep -Fq "start-ui" "$$lifecycle_log"; then \
			start_ui=true; \
		fi; \
		if [ -f "$$api_log" ]; then \
			api_log_exists=true; \
		fi; \
		if [ -f "$$ui_log" ]; then \
			ui_log_exists=true; \
		fi; \
		if [ -f "$$api_log.bak" ]; then \
			api_bak_exists=true; \
		fi; \
		if [ -f "$$ui_log.bak" ]; then \
			ui_bak_exists=true; \
		fi; \
		printf 'ensure_db=%s\nstart_api=%s\nstart_ui=%s\napi_log=%s\nui_log=%s\napi_bak=%s\nui_bak=%s\n' \
			"$$ensure_db" "$$start_api" "$$start_ui" "$$api_log_exists" "$$ui_log_exists" "$$api_bak_exists" "$$ui_bak_exists" \
			> "$$out"; \
	}; \
	stop_scenario() { \
		scenario="$$1"; \
		home="$$2"; \
		HOME="$$home" VROOLI_ROOT="$$repo_root" VROOLI_SOURCE_ROOT="$$repo_root" ~/.vrooli/bin/vrooli --no-stale-check scenario stop "$$scenario" > /dev/null 2> /dev/null || true; \
	}; \
	run_scenario() { \
		scenario="$$1"; \
		expect_db="$$2"; \
		prefix="$$3"; \
		home="$$4"; \
		mkdir -p "$$home"; \
		stop_scenario "$$scenario" "$$home"; \
		HOME="$$home" VROOLI_ROOT="$$repo_root" VROOLI_SOURCE_ROOT="$$repo_root" ~/.vrooli/bin/vrooli --no-stale-check scenario start "$$scenario" > "$$prefix.start.out"; \
		snapshot_records "$$home" "$$scenario" "$$prefix.start.records.json"; \
		snapshot_locks "$$home" "$$prefix.start.locks.json"; \
		jq -e 'length > 0 and all(.[]; .has_port == true)' "$$prefix.start.records.json" > /dev/null; \
		jq -e --arg scenario "$$scenario" 'length > 0 and all(.[]; .owner == $$scenario)' "$$prefix.start.locks.json" > /dev/null; \
		write_meta "$$home" "$$scenario" "$$expect_db" "$$prefix.start.meta"; \
		HOME="$$home" VROOLI_ROOT="$$repo_root" VROOLI_SOURCE_ROOT="$$repo_root" ~/.vrooli/bin/vrooli --no-stale-check scenario restart "$$scenario" > "$$prefix.restart.out"; \
		snapshot_records "$$home" "$$scenario" "$$prefix.restart.records.json"; \
		snapshot_locks "$$home" "$$scenario" "$$prefix.restart.locks.json"; \
		jq -e 'length > 0 and all(.[]; .has_port == true)' "$$prefix.restart.records.json" > /dev/null; \
		jq -e --arg scenario "$$scenario" 'length > 0 and all(.[]; .owner == $$scenario)' "$$prefix.restart.locks.json" > /dev/null; \
		write_meta "$$home" "$$scenario" "$$expect_db" "$$prefix.restart.meta"; \
		HOME="$$home" VROOLI_ROOT="$$repo_root" VROOLI_SOURCE_ROOT="$$repo_root" ~/.vrooli/bin/vrooli --no-stale-check scenario stop "$$scenario" > "$$prefix.stop.out"; \
		process_dir="$$home/.vrooli/processes/scenarios/$$scenario"; \
		if [ -d "$$process_dir" ] && find "$$process_dir" -maxdepth 1 \( -name '*.json' -o -name '*.pid' \) -print | grep -q .; then \
			echo "process cleanup failed for $$scenario" >&2; \
			exit 1; \
		fi; \
		state_dir="$$home/.vrooli/state/scenarios"; \
		if [ -d "$$state_dir" ] && find "$$state_dir" -maxdepth 1 -name '.port_*.lock' -print | grep -q .; then \
			echo "port lock cleanup failed for $$scenario" >&2; \
			exit 1; \
		fi; \
	}; \
	require_scenario "test-genie" "test-genie-api" "test-genie"; \
	require_scenario "reference-react-vite" "reference-react-vite-api" "reference-react-vite"; \
	run_scenario "test-genie" "false" "$$live_dir/test-genie" "$$live_dir/test-genie-home"; \
	run_scenario "reference-react-vite" "true" "$$live_dir/reference-react-vite" "$$live_dir/reference-react-vite-home"; \
	echo "Week 3 live native validation passed"; \
	trap - EXIT; \
	cleanup

validate-week5-cross: ## Run the Week 5 cross-compile suite
	tmpdir=$$(mktemp -d); \
	trap 'rm -rf "$$tmpdir"' EXIT; \
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$$tmpdir/vrooli-linux-amd64" ./cmd/vrooli; \
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o "$$tmpdir/vrooli-darwin-amd64" ./cmd/vrooli; \
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o "$$tmpdir/vrooli-darwin-arm64" ./cmd/vrooli; \
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o "$$tmpdir/vrooli-windows-amd64.exe" ./cmd/vrooli; \
	trap - EXIT; \
	rm -rf "$$tmpdir"

validate-scenario-to-cloud-native: ## Validate scenario-to-cloud's native deployment-local CLI contract
	test -z "$$(find scenarios/scenario-to-cloud/api scenarios/scenario-to-cloud/cli -name '*.go' ! -name '*_test.go' -print0 | xargs -0 rg -n 'scripts/lib/setup\.sh' || true)"
	test -z "$$(find scenarios/scenario-to-cloud/api scenarios/scenario-to-cloud/cli -name '*.go' ! -name '*_test.go' -print0 | xargs -0 rg -n 'scripts/manage\.sh' | rg -v 'scenarios/scenario-to-cloud/api/bundle/builder.go' || true)"
	! rg -n 'scripts/manage\.sh|scripts/lib/setup\.sh' \
		scenarios/scenario-to-cloud/docs \
		scenarios/scenario-to-cloud/requirements \
		scenarios/scenario-to-cloud/README.md \
		scenarios/scenario-to-cloud/PRD.md \
		scenarios/deployment-manager/docs/examples/picker-wheel-desktop.md \
		scenarios/scenario-to-desktop/README.md
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOWORK=off go build -trimpath -o /tmp/vrooli-scenario-to-cloud-native-amd64 ./cmd/vrooli
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 GOWORK=off go build -trimpath -o /tmp/vrooli-scenario-to-cloud-native-arm64 ./cmd/vrooli
	rm -f /tmp/vrooli-scenario-to-cloud-native-amd64 /tmp/vrooli-scenario-to-cloud-native-arm64
	cd scenarios/scenario-to-cloud/api && go test ./...
	cd scenarios/scenario-to-cloud/cli && go test ./...

validate-week6-slice: ## Run the expanded Week 6 native command slice
	test ! -d cli
	$(MAKE) build
	$(MAKE) install
	test ! -e cli/vrooli
	test ! -e scripts/manage.sh
	~/.vrooli/bin/vrooli --version > /tmp/vrooli-week6-cli-version.txt
	grep -Fq 'Vrooli CLI v' /tmp/vrooli-week6-cli-version.txt
	! rg -n '"/vrooli/cli/vrooli"|[[:<:]]cli/vrooli[[:>:]]' scenarios cmd internal packages api -g '*.go'
	! rg -n 'scripts/manage\.sh' docs scripts cmd internal api packages \
		-g '!docs/plans/*' \
		-g '!docs/repo-contract.md' \
		-g '!scripts/README.md' \
		-g '!internal/repocontract/**' \
		-g '!internal/repocontractcheck/**' \
		-g '!templates/scenarios/*'
	! rg -n 'cli/commands/clean-commands\.sh|cli/commands/doctor\.sh|cli/commands/resource-commands\.sh|cli/commands/resource-discovery\.sh|cli/commands/status-command\.sh|cli/commands/stop-commands\.sh|cli/lib/arg-parser\.sh|cli/lib/output-formatter\.sh' scripts cmd internal packages api resources -g '!*.md'
	set -e; \
	tmp_home=$$(mktemp -d); \
	tmp_root=$$(mktemp -d); \
	mkdir -p "$$tmp_home/.vrooli/state/scenarios" "$$tmp_root/.vrooli" "$$tmp_root/docs" "$$tmp_root/scenarios/alpha/.vrooli" "$$tmp_root/build" "$$tmp_root/scripts/resources"; \
	printf 'ghost:999999:1\n' > "$$tmp_home/.vrooli/state/scenarios/.port_21234.lock"; \
	printf '%s\n' '{"resource_ports":{},"reserved_ranges":{}}' > "$$tmp_root/scripts/resources/port_registry.json"; \
	printf '%s\n' '{"version":"1.0.0","service":{"name":"vrooli-week6","displayName":"Week 6 Fixture","description":"Week 6 validation fixture","version":"0.1.0"},"lifecycle":{"version":"2.0.0","build":{"description":"build","steps":[{"name":"write-build","run":"mkdir -p build && printf '\''build\n'\'' > build/build.txt"}]},"clean":{"description":"clean","steps":[{"name":"write-clean","run":"mkdir -p build && printf '\''clean\n'\'' > build/clean.txt"}]},"deploy":{"description":"deploy","steps":[{"name":"write-deploy","run":"mkdir -p build && printf '\''deploy\n'\'' > build/deploy.txt"}]},"backup":{"description":"backup","steps":[{"name":"write-backup","run":"mkdir -p build && printf '\''backup\n'\'' > build/backup.txt"}]},"restore":{"description":"restore","steps":[{"name":"write-restore","run":"mkdir -p build && printf '\''restore\n'\'' > build/restore.txt"}]}}}' > "$$tmp_root/.vrooli/service.json"; \
	printf '%s\n' '{"files":["docs/context.md"]}' > "$$tmp_root/.vrooli/info-manifest.json"; \
	printf 'Week 6 context\n' > "$$tmp_root/docs/context.md"; \
	printf '%s\n' '{"version":"1.0.0","service":{"name":"alpha","displayName":"Alpha","description":"Alpha scenario","version":"0.1.0"},"lifecycle":{"version":"2.0.0"}}' > "$$tmp_root/scenarios/alpha/.vrooli/service.json"; \
	now=$$(date -u +%Y-%m-%dT%H:%M:%SZ); \
	mkdir -p "$$tmp_home/.vrooli/processes/scenarios/alpha"; \
	jq -n --argjson pid "$$$$" --arg ts "$$now" '{pid:$$pid,pgid:$$pid,process_id:"vrooli.develop.alpha.start-api",phase:"develop",scenario:"alpha",step:"start-api",command:"sleep 30",working_dir:"/fixture/scenarios/alpha",log_file:"/tmp/alpha.log",port:21234,started_at:$$ts,status:"running"}' > "$$tmp_home/.vrooli/processes/scenarios/alpha/start-api.json"; \
	HOME="$$tmp_home" VROOLI_ROOT="$$tmp_root" VROOLI_SOURCE_ROOT="$$tmp_root" ~/.vrooli/bin/vrooli --no-stale-check info --json > /tmp/vrooli-week6-info.json; \
	grep -Fq '"root": "'$$tmp_root'"' /tmp/vrooli-week6-info.json; \
	HOME="$$tmp_home" VROOLI_ROOT="$$tmp_root" VROOLI_SOURCE_ROOT="$$tmp_root" ~/.vrooli/bin/vrooli --no-stale-check doctor --json > /tmp/vrooli-week6-doctor.json; \
	grep -Fq '"checks"' /tmp/vrooli-week6-doctor.json; \
	HOME="$$tmp_home" VROOLI_ROOT="$$tmp_root" VROOLI_SOURCE_ROOT="$$tmp_root" ~/.vrooli/bin/vrooli --no-stale-check status --scenarios --json > /tmp/vrooli-week6-status.json; \
	grep -Fq '"scenarios_total": 1' /tmp/vrooli-week6-status.json; \
	grep -Fq '"name": "alpha"' /tmp/vrooli-week6-status.json; \
	HOME="$$tmp_home" VROOLI_ROOT="$$tmp_root" VROOLI_SOURCE_ROOT="$$tmp_root" ~/.vrooli/bin/vrooli --no-stale-check orphans --json > /tmp/vrooli-week6-orphans.json; \
	grep -Fq '"orphans"' /tmp/vrooli-week6-orphans.json; \
	HOME="$$tmp_home" VROOLI_ROOT="$$tmp_root" VROOLI_SOURCE_ROOT="$$tmp_root" ~/.vrooli/bin/vrooli --no-stale-check locks --json > /tmp/vrooli-week6-locks-list.json; \
	grep -Fq '"port": 21234' /tmp/vrooli-week6-locks-list.json; \
	HOME="$$tmp_home" VROOLI_ROOT="$$tmp_root" VROOLI_SOURCE_ROOT="$$tmp_root" ~/.vrooli/bin/vrooli --no-stale-check cleanup locks --json > /tmp/vrooli-week6-locks.json; \
	test ! -f "$$tmp_home/.vrooli/state/scenarios/.port_21234.lock"; \
	grep -Fq '"success": true' /tmp/vrooli-week6-locks.json; \
	HOME="$$tmp_home" VROOLI_ROOT="$$tmp_root" VROOLI_SOURCE_ROOT="$$tmp_root" ~/.vrooli/bin/vrooli --no-stale-check diagnose-port 21234 alpha --json > /tmp/vrooli-week6-diagnose.json; \
	grep -Fq '"port": 21234' /tmp/vrooli-week6-diagnose.json; \
	HOME="$$tmp_home" VROOLI_ROOT="$$tmp_root" VROOLI_SOURCE_ROOT="$$tmp_root" ~/.vrooli/bin/vrooli --no-stale-check stop scenarios --json > /tmp/vrooli-week6-stop.json; \
	grep -Fq '"success": true' /tmp/vrooli-week6-stop.json; \
	HOME="$$tmp_home" VROOLI_ROOT="$$tmp_root" VROOLI_SOURCE_ROOT="$$tmp_root" ~/.vrooli/bin/vrooli --no-stale-check build; \
	test "$$(cat "$$tmp_root/build/build.txt")" = "build"; \
	HOME="$$tmp_home" VROOLI_ROOT="$$tmp_root" VROOLI_SOURCE_ROOT="$$tmp_root" ~/.vrooli/bin/vrooli --no-stale-check clean; \
	test "$$(cat "$$tmp_root/build/clean.txt")" = "clean"; \
	HOME="$$tmp_home" VROOLI_ROOT="$$tmp_root" VROOLI_SOURCE_ROOT="$$tmp_root" ~/.vrooli/bin/vrooli --no-stale-check deploy; \
	test "$$(cat "$$tmp_root/build/deploy.txt")" = "deploy"; \
	HOME="$$tmp_home" VROOLI_ROOT="$$tmp_root" VROOLI_SOURCE_ROOT="$$tmp_root" ~/.vrooli/bin/vrooli --no-stale-check backup; \
	test "$$(cat "$$tmp_root/build/backup.txt")" = "backup"; \
	HOME="$$tmp_home" VROOLI_ROOT="$$tmp_root" VROOLI_SOURCE_ROOT="$$tmp_root" ~/.vrooli/bin/vrooli --no-stale-check restore; \
	test "$$(cat "$$tmp_root/build/restore.txt")" = "restore"; \
	rm -rf "$$tmp_home" "$$tmp_root"

clean: ## Remove project-level Go build artifacts
	rm -rf $(BUILD_DIR)

setup: ## Bootstrap the Go CLI and run native setup
	$(MAKE) install
	$(VROOLI_BIN) setup $(SETUP_ARGS)

dev: ## Start the native development workflow
	test -x "$(VROOLI_BIN)"
	$(VROOLI_BIN) develop $(DEV_ARGS)

develop: dev ## Alias for make dev

deploy: ## Run the existing deploy workflow
	test -x "$(VROOLI_BIN)"
	$(VROOLI_BIN) deploy

status: ## Show current Vrooli status
	test -x "$(VROOLI_BIN)"
	$(VROOLI_BIN) status

scenarios: ## List scenarios through the existing CLI
	test -x "$(VROOLI_BIN)"
	$(VROOLI_BIN) scenario list

resources: ## Show resource status through the existing CLI
	test -x "$(VROOLI_BIN)"
	$(VROOLI_BIN) resource status

lifecycle-build: ## Run the native project build command via the installed CLI
	test -x "$(VROOLI_BIN)"
	$(VROOLI_BIN) build
