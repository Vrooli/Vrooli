#!/bin/bash

# Test script for Core Debugger scenario
set -euo pipefail

echo "Testing Core Debugger scenario..."
echo

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test function
run_test() {
    local name="$1"
    local command="$2"
    
    printf "Testing: %s... " "$name"
    if eval "$command" > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
    fi
}

# Structure tests
run_test "PRD exists" "test -f PRD.md"
run_test "Service.json exists" "test -f .vrooli/service.json"
run_test "CLI exists and is executable" "test -x cli/core-debugger"
run_test "API source exists" "test -f api/main.go"
run_test "UI exists" "test -f ui/index.html"
run_test "Components registry exists" "test -f data/components.json"
run_test "Workarounds database exists" "test -f data/workarounds/common.json"

# CLI tests
run_test "CLI help works" "./cli/core-debugger help | grep -q 'Core Debugger'"
run_test "CLI version works" "./cli/core-debugger version | grep -q 'v1.0.0'"
run_test "CLI status works" "./cli/core-debugger status | grep -q 'Core Debugger Status'"

# Data structure tests
run_test "Issues directory exists" "test -d data/issues"
run_test "Health directory exists" "test -d data/health"
run_test "Logs directory exists" "test -d data/logs"

# JSON validation tests
run_test "Components JSON is valid" "jq '.' data/components.json"
run_test "Workarounds JSON is valid" "jq '.' data/workarounds/common.json"
run_test "Service.json is valid" "jq '.' .vrooli/service.json"

# API build test
echo
echo "Building API (this may take a moment)..."
if (cd api && go mod download && go build -o test-build main.go && rm test-build); then
    echo -e "${GREEN}✓ API builds successfully${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ API build failed${NC}"
    ((TESTS_FAILED++))
fi

# Summary
echo
echo "================================"
echo "Test Results:"
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
echo "================================"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    echo
    echo "You can now run the scenario with:"
    echo "  vrooli scenario setup core-debugger"
    echo "  vrooli scenario run core-debugger"
    exit 0
else
    echo -e "${RED}Some tests failed. Please review and fix.${NC}"
    exit 1
fi