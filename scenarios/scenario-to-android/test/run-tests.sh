#!/usr/bin/env bash
# Scenario to Android - Test Runner
#
# This script orchestrates all test phases for scenario-to-android.
# It runs tests in the correct order and aggregates results.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
PHASES_DIR="$SCRIPT_DIR/phases"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Track test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
FAILED_PHASES=()

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}🧪 Scenario to Android - Test Suite${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Function to run a test phase
run_phase() {
    local phase_name="$1"
    local phase_script="$PHASES_DIR/$phase_name"

    if [[ ! -f "$phase_script" ]]; then
        echo -e "${YELLOW}⚠️  Skipping $phase_name (script not found)${NC}"
        return 0
    fi

    echo -e "${BLUE}▶ Running phase: ${phase_name%.sh}${NC}"
    echo ""

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if bash "$phase_script"; then
        echo -e "${GREEN}✓ ${phase_name%.sh} passed${NC}"
        echo ""
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "${RED}✗ ${phase_name%.sh} failed${NC}"
        echo ""
        FAILED_TESTS=$((FAILED_TESTS + 1))
        FAILED_PHASES+=("${phase_name%.sh}")
        return 1
    fi
}

# Run test phases in order
# Phase 1: Structure validation
run_phase "test-structure.sh" || true

# Phase 2: Dependency checks
run_phase "test-dependencies.sh" || true

# Phase 3: Business logic tests
run_phase "test-business.sh" || true

# Phase 4: Unit tests
run_phase "test-unit.sh" || true

# Phase 5: Integration tests
run_phase "test-integration.sh" || true

# Phase 6: Performance tests
run_phase "test-performance.sh" || true

# Print summary
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}📊 Test Summary${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "Total test phases: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"
echo ""

if [[ $FAILED_TESTS -gt 0 ]]; then
    echo -e "${RED}Failed phases:${NC}"
    for phase in "${FAILED_PHASES[@]}"; do
        echo -e "  ${RED}✗${NC} $phase"
    done
    echo ""
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${RED}❌ Test suite failed${NC}"
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    exit 1
else
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}✅ All tests passed!${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    exit 0
fi
