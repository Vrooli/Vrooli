#!/bin/bash
# Main test runner for tech-tree-designer
# Orchestrates all test phases in the correct order

set -e

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCENARIO_NAME="$(basename "$SCENARIO_DIR")"

echo -e "${BLUE}🧪 Running test suite for ${SCENARIO_NAME}${NC}"
echo ""

# Track overall results
TOTAL_PHASES=0
PASSED_PHASES=0
FAILED_PHASES=0

run_phase() {
    local phase_name="$1"
    local phase_script="$SCENARIO_DIR/test/phases/${phase_name}.sh"

    TOTAL_PHASES=$((TOTAL_PHASES + 1))

    echo -e "${BLUE}📋 Running phase: ${phase_name}${NC}"

    if [ ! -f "$phase_script" ]; then
        echo -e "${YELLOW}⚠️  Phase script not found: ${phase_script}${NC}"
        echo ""
        return 0
    fi

    if bash "$phase_script"; then
        echo -e "${GREEN}✅ Phase passed: ${phase_name}${NC}"
        PASSED_PHASES=$((PASSED_PHASES + 1))
        echo ""
        return 0
    else
        echo -e "${RED}❌ Phase failed: ${phase_name}${NC}"
        FAILED_PHASES=$((FAILED_PHASES + 1))
        echo ""
        return 1
    fi
}

# Run test phases in order
# Order matters: dependencies -> structure -> unit -> integration -> business -> performance

run_phase "test-dependencies" || true
run_phase "test-structure" || true
run_phase "test-unit" || true
run_phase "test-integration" || true
run_phase "test-business" || true
run_phase "test-performance" || true

# Print summary
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}📊 Test Summary for ${SCENARIO_NAME}${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "  Total phases:  ${TOTAL_PHASES}"
echo -e "  ${GREEN}Passed:        ${PASSED_PHASES}${NC}"
echo -e "  ${RED}Failed:        ${FAILED_PHASES}${NC}"
echo ""

if [ $FAILED_PHASES -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    echo ""
    exit 0
else
    echo -e "${RED}❌ Some tests failed${NC}"
    echo ""
    exit 1
fi
