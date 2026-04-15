.PHONY: help setup dev develop build install status stop info test validate clean

.DEFAULT_GOAL := help

BUILD_DIR := .vrooli/build
INSTALL_DIR := $(HOME)/.vrooli/bin
VROOLI := go run ./cmd/vrooli --no-stale-check
SETUP_ARGS ?=
DEVELOP_ARGS ?=

help: ## Show the supported repo-level entrypoints
	@printf "Vrooli project entrypoints\n\n"
	@printf "  make setup                      Bootstrap and run project setup\n"
	@printf "  make develop                    Start the development stack\n"
	@printf "  make build                      Build project-level binaries via the CLI\n"
	@printf "  make install                    Install project-level binaries to %s\n" "$(INSTALL_DIR)"
	@printf "  make status                     Show project status\n"
	@printf "  make stop                       Stop project services\n"
	@printf "  make info                       Show the canonical project briefing\n"
	@printf "  make test                       Run project-level Go tests\n"
	@printf "  make clean                      Clean build artifacts via the CLI\n"

setup: ## Bootstrap and run project setup
	@$(VROOLI) setup $(SETUP_ARGS)

dev: develop ## Alias for make develop

develop: ## Start the development stack
	@$(VROOLI) develop $(DEVELOP_ARGS)

build: ## Build project-level binaries via the CLI
	@$(VROOLI) build

install: build ## Install project-level binaries into ~/.vrooli/bin
	@mkdir -p "$(INSTALL_DIR)"
	@install -m 0755 "$(BUILD_DIR)/vrooli-api" "$(INSTALL_DIR)/vrooli-api"
	@install -m 0755 "$(BUILD_DIR)/vrooli" "$(INSTALL_DIR)/vrooli"

status: ## Show project status
	@$(VROOLI) status

stop: ## Stop project services
	@$(VROOLI) stop

info: ## Show the canonical project briefing
	@$(VROOLI) info

test: ## Run project-level Go tests
	@go test ./internal/...
	@go test ./cmd/vrooli-buildmeta
	@go test ./cmd/vrooli
	@go test -tags testing ./cmd/vrooli-api

clean: ## Clean build artifacts via the CLI
	@$(VROOLI) clean
