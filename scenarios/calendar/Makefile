.PHONY: help setup start stop restart status logs test open run dev

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
	@printf "For scenario-specific commands, prefer the scenario CLI or 'vrooli scenario ...'.\\n"\n\nsetup: ## Run scenario setup lifecycle\n\t@vrooli scenario setup "$(SCENARIO_NAME)"\n\nstart: ## Start the scenario\n\t@vrooli scenario start "$(SCENARIO_NAME)"\n\nstop: ## Stop the scenario\n\t@vrooli scenario stop "$(SCENARIO_NAME)"\n\nrestart: ## Restart the scenario\n\t@vrooli scenario restart "$(SCENARIO_NAME)"\n\nstatus: ## Show scenario runtime status\n\t@vrooli scenario status "$(SCENARIO_NAME)"\n\nlogs: ## Show scenario logs\n\t@vrooli scenario logs "$(SCENARIO_NAME)"\n\ntest: ## Run scenario tests\n\t@vrooli scenario test "$(SCENARIO_NAME)"\n\nopen: ## Open the scenario in a browser\n\t@vrooli scenario open "$(SCENARIO_NAME)"\n\nrun: start\n\ndev: start\n