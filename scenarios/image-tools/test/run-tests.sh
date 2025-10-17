#!/bin/bash
# Image Tools - Comprehensive Test Runner
# Runs all test phases in sequence

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"
SCENARIO_NAME="image-tools"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test phase tracking
TOTAL_PHASES=0
PASSED_PHASES=0
FAILED_PHASES=0
SKIPPED_PHASES=0

# Function to run a test phase
run_phase() {
    local phase_name=$1
    local phase_script=$2

    TOTAL_PHASES=$((TOTAL_PHASES + 1))

    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}Running Phase: ${phase_name}${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"

    if [ ! -f "$phase_script" ]; then
        echo -e "${YELLOW}⚠️  Skipped: ${phase_name} (script not found)${NC}"
        SKIPPED_PHASES=$((SKIPPED_PHASES + 1))
        return 0
    fi

    if bash "$phase_script"; then
        echo -e "${GREEN}✅ Passed: ${phase_name}${NC}"
        PASSED_PHASES=$((PASSED_PHASES + 1))
        return 0
    else
        echo -e "${RED}❌ Failed: ${phase_name}${NC}"
        FAILED_PHASES=$((FAILED_PHASES + 1))
        return 1
    fi
}

# Main test execution
echo -e "${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║         Image Tools - Comprehensive Test Suite           ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"

cd "$SCENARIO_DIR"

# Phase 1: Dependencies
run_phase "Dependencies Check" "$SCRIPT_DIR/phases/test-dependencies.sh" || true

# Phase 2: Structure Validation
run_phase "Structure Validation" "$SCRIPT_DIR/phases/test-structure.sh" || true

# Phase 3: Unit Tests
run_phase "Unit Tests" "$SCRIPT_DIR/phases/test-unit.sh" || true

# Phase 4: Integration Tests
run_phase "Integration Tests" "$SCRIPT_DIR/phases/test-integration.sh" || true

# Phase 5: Business Logic Tests
run_phase "Business Logic Tests" "$SCRIPT_DIR/phases/test-business.sh" || true

# Phase 6: Performance Tests
run_phase "Performance Tests" "$SCRIPT_DIR/phases/test-performance.sh" || true

# Phase 7: Smoke Tests
run_phase "Smoke Tests" "$SCRIPT_DIR/phases/test-smoke.sh" || true

# Test Summary
echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "Total Phases:   ${TOTAL_PHASES}"
echo -e "${GREEN}Passed:         ${PASSED_PHASES}${NC}"
echo -e "${RED}Failed:         ${FAILED_PHASES}${NC}"
echo -e "${YELLOW}Skipped:        ${SKIPPED_PHASES}${NC}"
echo ""

if [ $FAILED_PHASES -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed. See output above for details.${NC}"
    exit 1
fi
