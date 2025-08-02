#!/usr/bin/env bash
# Validation Test Runner for BATS Testing Infrastructure
# Run this script to validate that the entire infrastructure works correctly

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FIXTURES_DIR="$(dirname "$SCRIPT_DIR")"

echo -e "${BLUE}=== BATS Testing Infrastructure Validation ===${NC}"
echo "Running comprehensive validation tests..."
echo

# Check if bats is available
if ! command -v bats >/dev/null 2>&1; then
    echo -e "${RED}ERROR: bats command not found${NC}"
    echo "Please install BATS testing framework:"
    echo "  - Ubuntu/Debian: sudo apt install bats"
    echo "  - macOS: brew install bats-core"
    echo "  - Or from source: https://github.com/bats-core/bats-core"
    exit 1
fi

echo -e "${BLUE}Using bats version:${NC}"
bats --version
echo

# Function to run a test file and report results
run_test_file() {
    local test_file="$1"
    local test_name="$(basename "$test_file" .bats)"
    
    echo -e "${BLUE}Running $test_name...${NC}"
    
    if bats "$test_file"; then
        echo -e "${GREEN}✓ $test_name passed${NC}"
        return 0
    else
        echo -e "${RED}✗ $test_name failed${NC}"
        return 1
    fi
}

# Track results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Find all validation test files
validation_tests=(
    "$SCRIPT_DIR/test-infrastructure.bats"
    "$SCRIPT_DIR/test-assertions.bats"
    "$SCRIPT_DIR/test-mock-registry.bats"
)

# Run each validation test
for test_file in "${validation_tests[@]}"; do
    if [[ -f "$test_file" ]]; then
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
        echo "----------------------------------------"
        
        if run_test_file "$test_file"; then
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
        echo
    else
        echo -e "${YELLOW}Warning: Test file not found: $test_file${NC}"
    fi
done

# Summary
echo "========================================"
echo -e "${BLUE}Validation Summary:${NC}"
echo "Total test suites: $TOTAL_TESTS"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"

if [[ $FAILED_TESTS -eq 0 ]]; then
    echo
    echo -e "${GREEN}🎉 All validation tests passed!${NC}"
    echo -e "${GREEN}The BATS testing infrastructure is working correctly.${NC}"
    exit 0
else
    echo
    echo -e "${RED}❌ Some validation tests failed.${NC}"
    echo -e "${RED}Please review the output above and fix any issues.${NC}"
    exit 1
fi