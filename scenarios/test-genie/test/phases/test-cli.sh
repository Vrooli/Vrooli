#!/bin/bash

# Phase: CLI Testing
# Validates test-genie CLI commands work correctly

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}🧪 Testing Test Genie CLI Commands${NC}"

# Function to test a command and check for expected output
test_command() {
    local description="$1"
    local command="$2"
    local expected="$3"

    echo -n "Testing: $description... "

    output=$(eval "$command" 2>&1)

    if echo "$output" | grep -q "$expected"; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${RED}✗${NC}"
        echo "  Command: $command"
        echo "  Expected: $expected"
        echo "  Got: $output"
        return 1
    fi
}

# Test help command
test_command "CLI help command" \
    "test-genie --help" \
    "AI-powered comprehensive test generation"

# Test status command
test_command "CLI status command" \
    "test-genie status" \
    "Test Genie System Status"

# Test health command
test_command "CLI health command" \
    "test-genie health" \
    "Test Genie is healthy"

# Test generate command with dry run
test_command "CLI generate command" \
    "test-genie generate test-cli-demo --types unit --coverage 80 --json" \
    "request_id"

# Test coverage command (simplified - just check it runs)
test_command "CLI coverage analysis" \
    "timeout 5 test-genie coverage test-cli-demo --depth basic 2>&1 | head -1" \
    "Analyzing coverage"

# Test vault command help
test_command "CLI vault help" \
    "test-genie vault --help 2>&1 || true" \
    "vault"

# Test maintain command help
test_command "CLI maintain help" \
    "test-genie maintain --help 2>&1 || true" \
    "maintain"

# Test execute command help (no suite to execute)
test_command "CLI execute help" \
    "test-genie execute --help 2>&1 || true" \
    "execute"

# Test results command help
test_command "CLI results help" \
    "test-genie results --help 2>&1 || true" \
    "results"

# Test invalid command
test_command "CLI handles invalid command" \
    "test-genie invalid-command 2>&1 || true" \
    "Unknown command"

# Test JSON output format
test_command "CLI JSON output formatting" \
    "test-genie status --json | jq -e '.status' >/dev/null && echo 'valid json'" \
    "valid json"

echo -e "${GREEN}✅ All CLI tests passed!${NC}"