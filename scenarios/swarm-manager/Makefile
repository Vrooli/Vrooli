.PHONY: help setup start stop restart status logs test open run dev fmt fmt-go fmt-ui lint lint-go lint-ui check

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
	@printf "  make open                       Open the scenario in a browser\n"
	@printf "  make run                        Alias for make start\n"
	@printf "  make dev                        Alias for make start\n"
	@printf "\n"
	@printf "For scenario-specific commands, prefer the scenario CLI or 'vrooli scenario ...'.\n"

setup: ## Run scenario setup lifecycle
	@vrooli scenario setup "$(SCENARIO_NAME)"

start: ## Start the scenario
	@vrooli scenario start "$(SCENARIO_NAME)"

stop: ## Stop the scenario
	@vrooli scenario stop "$(SCENARIO_NAME)"

restart: ## Restart the scenario
	@vrooli scenario restart "$(SCENARIO_NAME)"

status: ## Show scenario runtime status
	@vrooli scenario status "$(SCENARIO_NAME)"

logs: ## Show scenario logs
	@vrooli scenario logs "$(SCENARIO_NAME)"

test: ## Run scenario tests
	@vrooli scenario test "$(SCENARIO_NAME)"

open: ## Open the scenario in a browser
	@vrooli scenario open "$(SCENARIO_NAME)"

fmt: fmt-go fmt-ui ## Format all code

fmt-go: ## Format Go code
	@for dir in api cli; do \
		if [ -d $$dir ] && find $$dir -name "*.go" | head -1 | grep -q .; then \
			echo "Formatting Go code in $$dir..."; \
			if command -v gofumpt >/dev/null 2>&1; then \
				cd $$dir && gofumpt -w . && cd ..; \
			elif command -v gofmt >/dev/null 2>&1; then \
				cd $$dir && gofmt -w . && cd ..; \
			fi; \
		fi; \
	done

fmt-ui: ## Format UI code
	@if [ -f ui/package.json ]; then \
		echo "Formatting UI code..."; \
		cd ui && pnpm run lint:fix; \
	fi

lint: lint-go lint-ui ## Lint all code

lint-go: ## Lint Go code
	@for dir in api cli; do \
		if [ -d $$dir ] && find $$dir -name "*.go" | head -1 | grep -q .; then \
			echo "Linting Go code in $$dir..."; \
			if command -v golangci-lint >/dev/null 2>&1; then \
				cd $$dir && golangci-lint run ./... && cd ..; \
			elif command -v go >/dev/null 2>&1; then \
				cd $$dir && go vet ./... && cd ..; \
			fi; \
		fi; \
	done

lint-ui: ## Lint UI code
	@if [ -f ui/package.json ]; then \
		echo "Linting UI code..."; \
		cd ui && pnpm run lint && pnpm run type-check; \
	fi

check: fmt lint test ## Run full quality gates

run: start

dev: start
