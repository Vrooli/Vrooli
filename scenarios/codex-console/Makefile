# Codex Console Scenario Makefile
#
# Use these commands to interact with the scenario through the Vrooli lifecycle system.
# Never run binaries directly; always use `make` targets or `vrooli scenario ...` commands.

.PHONY: help start stop dev test logs status clean build fmt fmt-go fmt-ui lint lint-go lint-ui check

.DEFAULT_GOAL := help
SCENARIO_NAME := $(notdir $(CURDIR))

help:
	@echo "\033[1;36mCodex Console Scenario Commands\033[0m"
	@echo "  make start   - Start services via lifecycle"
	@echo "  make stop    - Stop services"
	@echo "  make dev     - Start in development mode"
	@echo "  make test    - Run scenario tests"
	@echo "  make logs    - Tail scenario logs"
	@echo "  make clean   - Clean build artefacts"

start:
	@echo "\033[1;34mStarting $(SCENARIO_NAME) via lifecycle...\033[0m"
	@vrooli scenario start $(SCENARIO_NAME)

stop:
	@echo "\033[1;33mStopping $(SCENARIO_NAME)...\033[0m"
	@vrooli scenario stop $(SCENARIO_NAME)

dev:
	@echo "\033[1;34mStarting $(SCENARIO_NAME) in development mode...\033[0m"
	@vrooli scenario start $(SCENARIO_NAME) --dev

test:
	@echo "\033[1;34mRunning tests for $(SCENARIO_NAME)...\033[0m"
	@vrooli scenario test $(SCENARIO_NAME)

logs:
	@vrooli scenario logs $(SCENARIO_NAME) --tail 100

status:
	@vrooli scenario status $(SCENARIO_NAME)

clean:
	@echo "\033[1;33mRemoving build artefacts...\033[0m"
	@rm -f api/codex-console-api
	@rm -rf ui/dist
	@echo "\033[1;32mClean complete\033[0m"

fmt: fmt-go fmt-ui

fmt-go:
	@if [ -d api ]; then \
		cd api && gofmt -w .; \
	fi

fmt-ui:
	@if [ -d ui ]; then \
		cd ui && pnpm exec prettier --write "src/**/*.{ts,tsx,css}"; \
	fi

lint: lint-go lint-ui

lint-go:
	@if [ -d api ]; then \
		cd api && go vet ./... || true; \
	fi

lint-ui:
	@if [ -d ui ]; then \
		cd ui && pnpm run lint || true; \
	fi

check: fmt lint test

build:
	@echo "\033[1;34mBuilding $(SCENARIO_NAME)...\033[0m"
	@if [ -d api ]; then cd api && go build -o codex-console-api .; fi
	@if [ -d ui ]; then cd ui && pnpm run build; fi
