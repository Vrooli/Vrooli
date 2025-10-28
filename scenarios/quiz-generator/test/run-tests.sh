#!/usr/bin/env bash
set -e

# Quiz Generator - Test Runner
# Orchestrates phased testing for the quiz-generator scenario

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(dirname "$SCRIPT_DIR")"
SCENARIO_NAME="quiz-generator"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Track results
TOTAL_PHASES=0
PASSED_PHASES=0
FAILED_PHASES=0
SKIPPED_PHASES=0

echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Quiz Generator Test Suite${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
echo ""

# Discover actual running ports from environment variables
# These are set by the lifecycle system when the scenario starts
export API_PORT="${API_PORT:-16470}"  # Fallback for when not running via lifecycle
export UI_PORT="${UI_PORT:-3251}"     # Fallback for when not running via lifecycle

# Run a test phase
run_phase() {
    local phase_name="$1"
    local phase_script="$2"

    TOTAL_PHASES=$((TOTAL_PHASES + 1))

    if [ ! -f "$phase_script" ]; then
        echo -e "${YELLOW}⊘ ${phase_name}: SKIPPED (script not found)${NC}"
        SKIPPED_PHASES=$((SKIPPED_PHASES + 1))
        return 0
    fi

    if [ ! -x "$phase_script" ]; then
        chmod +x "$phase_script"
    fi

    echo -e "${BLUE}▶ Running ${phase_name}...${NC}"

    if "$phase_script"; then
        echo -e "${GREEN}✓ ${phase_name}: PASSED${NC}"
        PASSED_PHASES=$((PASSED_PHASES + 1))
        return 0
    else
        echo -e "${RED}✗ ${phase_name}: FAILED${NC}"
        FAILED_PHASES=$((FAILED_PHASES + 1))
        return 1
    fi
}

# Test phases in order
cd "$SCENARIO_ROOT"

# Phase 1: Unit Tests
run_phase "Unit Tests" "$SCRIPT_DIR/phases/test-unit.sh" || true

# Phase 2: Smoke Tests
run_phase "Smoke Tests" "$SCRIPT_DIR/phases/test-smoke.sh" || true

# Phase 3: Integration Tests
run_phase "Integration Tests" "$SCRIPT_DIR/phases/test-integration.sh" || true

# Phase 4: Business Logic Tests (if exists)
run_phase "Business Tests" "$SCRIPT_DIR/phases/test-business.sh" || true

# Phase 5: Performance Tests (if exists)
run_phase "Performance Tests" "$SCRIPT_DIR/phases/test-performance.sh" || true

# Summary
echo ""
echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Test Summary${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
echo -e "  Total Phases:   ${TOTAL_PHASES}"
echo -e "  ${GREEN}Passed:         ${PASSED_PHASES}${NC}"
echo -e "  ${RED}Failed:         ${FAILED_PHASES}${NC}"
echo -e "  ${YELLOW}Skipped:        ${SKIPPED_PHASES}${NC}"
echo ""

if [ $FAILED_PHASES -gt 0 ]; then
    echo -e "${RED}❌ Some tests failed${NC}"
    exit 1
elif [ $PASSED_PHASES -eq 0 ]; then
    echo -e "${YELLOW}⚠️  No tests were run${NC}"
    exit 1
else
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
fi
