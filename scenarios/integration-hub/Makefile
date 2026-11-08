.PHONY: help setup start stop restart status logs test build fmt
.DEFAULT_GOAL := help
SCENARIO_NAME := $(notdir $(CURDIR))

SCENARIO_CUSTOM_TARGETS += build
# Shared toolchain floor (mk/toolchain.mk) and build targets (mk/scenario.mk); a custom body for one of them is declared in SCENARIO_CUSTOM_TARGETS.
-include ../../mk/scenario.mk

help:
	@printf "Vrooli scenario entrypoints for %s\n" "$(SCENARIO_NAME)"
	@printf "  make setup|start|stop|restart|status|logs|test\n"
setup:
	@vrooli scenario setup "$(SCENARIO_NAME)"
start:
	@vrooli scenario start "$(SCENARIO_NAME)"
stop:
	@vrooli scenario stop "$(SCENARIO_NAME)"
restart:
	@vrooli scenario restart "$(SCENARIO_NAME)"
status:
	@vrooli scenario status "$(SCENARIO_NAME)"
logs:
	@vrooli scenario logs "$(SCENARIO_NAME)" --tail 50
test:
	@vrooli scenario test "$(SCENARIO_NAME)"
build:
	@vrooli package build "$(SCENARIO_NAME)"
fmt: fmt-go
