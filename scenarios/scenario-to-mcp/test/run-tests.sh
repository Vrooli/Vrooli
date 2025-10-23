#!/usr/bin/env bash
# Unified test runner for scenario-to-mcp
# Runs all test phases in sequence with proper error handling

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
SCENARIO_NAME="scenario-to-mcp"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results tracking
TOTAL_PHASES=0
PASSED_PHASES=0
FAILED_PHASES=()

echo -e "${BLUE}🧪 Running test suite for ${SCENARIO_NAME}${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Function to run a test phase
run_phase() {
    local phase_name=$1
    local phase_script=$2

    TOTAL_PHASES=$((TOTAL_PHASES + 1))

    echo ""
    echo -e "${BLUE}▶ Running ${phase_name}...${NC}"

    if [ ! -f "$phase_script" ]; then
        echo -e "${YELLOW}⚠ Skipping ${phase_name} (script not found)${NC}"
        return 0
    fi

    if bash "$phase_script"; then
        echo -e "${GREEN}✓ ${phase_name} passed${NC}"
        PASSED_PHASES=$((PASSED_PHASES + 1))
        return 0
    else
        echo -e "${RED}✗ ${phase_name} failed${NC}"
        FAILED_PHASES+=("$phase_name")
        return 1
    fi
}

# Run all test phases in order
run_phase "Structure Tests" "$SCRIPT_DIR/phases/test-structure.sh" || true
run_phase "Dependencies Tests" "$SCRIPT_DIR/phases/test-dependencies.sh" || true
run_phase "Unit Tests" "$SCRIPT_DIR/phases/test-unit.sh" || true
run_phase "Integration Tests" "$SCRIPT_DIR/phases/test-integration.sh" || true
run_phase "Business Logic Tests" "$SCRIPT_DIR/phases/test-business.sh" || true
run_phase "Performance Tests" "$SCRIPT_DIR/phases/test-performance.sh" || true

# Print summary
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}📊 Test Summary${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "Total Phases:  ${TOTAL_PHASES}"
echo -e "${GREEN}Passed:        ${PASSED_PHASES}${NC}"

if [ ${#FAILED_PHASES[@]} -gt 0 ]; then
    echo -e "${RED}Failed:        ${#FAILED_PHASES[@]}${NC}"
    echo ""
    echo -e "${RED}Failed phases:${NC}"
    for phase in "${FAILED_PHASES[@]}"; do
        echo -e "${RED}  - ${phase}${NC}"
    done
    echo ""
    exit 1
else
    echo ""
    echo -e "${GREEN}✓ All test phases passed!${NC}"
    echo ""
    exit 0
fi
