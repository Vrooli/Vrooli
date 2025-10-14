#!/bin/bash
set -e

# Data Tools Test Runner
# Executes all test phases in order

echo "======================================"
echo "  Data Tools Scenario Test Suite"
echo "======================================"
echo ""

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASES_DIR="$SCRIPT_DIR/phases"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[1;34m'
NC='\033[0m'

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0
FAILED_TESTS=()

# Helper function to run a test phase
run_phase() {
    local phase_name="$1"
    local phase_script="$PHASES_DIR/$phase_name"

    if [ ! -f "$phase_script" ]; then
        echo -e "${YELLOW}⚠️  Skipping $phase_name (not found)${NC}"
        return 0
    fi

    echo -e "${BLUE}Running: $phase_name${NC}"
    if bash "$phase_script"; then
        echo -e "${GREEN}✅ $phase_name passed${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo ""
        return 0
    else
        echo -e "${RED}❌ $phase_name failed${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        FAILED_TESTS+=("$phase_name")
        echo ""
        return 1
    fi
}

# Run all test phases
echo "Starting test execution..."
echo ""

run_phase "test-structure.sh" || true
run_phase "test-dependencies.sh" || true
run_phase "test-unit.sh" || true
run_phase "test-integration.sh" || true
run_phase "test-business.sh" || true
run_phase "test-performance.sh" || true

# Print summary
echo "======================================"
echo "  Test Summary"
echo "======================================"
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"

if [ $TESTS_FAILED -gt 0 ]; then
    echo ""
    echo "Failed tests:"
    for test in "${FAILED_TESTS[@]}"; do
        echo -e "  ${RED}✗${NC} $test"
    done
    echo ""
    exit 1
else
    echo ""
    echo -e "${GREEN}✅ All tests passed!${NC}"
    echo ""
    exit 0
fi
