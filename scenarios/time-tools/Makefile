# Time Tools Scenario Makefile
# Comprehensive temporal operations and scheduling platform

SCENARIO_NAME := time-tools

.PHONY: help run test logs stop status clean

# Default target
help:
	@echo "Time Tools Scenario Management"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  run     - Start the time-tools scenario"
	@echo "  test    - Run scenario tests"
	@echo "  logs    - View scenario logs"
	@echo "  stop    - Stop the scenario"
	@echo "  status  - Check scenario status"
	@echo "  clean   - Clean up scenario resources"

# Start the scenario
run:
	@echo "Starting $(SCENARIO_NAME) scenario..."
	vrooli scenario run $(SCENARIO_NAME)

# Run tests
test:
	@echo "Testing $(SCENARIO_NAME) scenario..."
	vrooli scenario test $(SCENARIO_NAME)

# View logs
logs:
	@echo "Viewing $(SCENARIO_NAME) logs..."
	vrooli scenario logs $(SCENARIO_NAME)

# Stop the scenario
stop:
	@echo "Stopping $(SCENARIO_NAME) scenario..."
	vrooli scenario stop $(SCENARIO_NAME)

# Check status
status:
	@echo "Checking $(SCENARIO_NAME) status..."
	vrooli scenario status $(SCENARIO_NAME)

# Clean up resources
clean:
	@echo "Cleaning $(SCENARIO_NAME) resources..."
	vrooli scenario stop $(SCENARIO_NAME) || true
	rm -f api/time-tools-api
	rm -f api/test-build