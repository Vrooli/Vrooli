#!/bin/bash
# Device Sync Hub Test Runner
# Orchestrates all test phases

set -euo pipefail

SCENARIO_ROOT="${PWD}"
TEST_DIR="$SCENARIO_ROOT/test/phases"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${BOLD}=== Device Sync Hub Test Suite ===${NC}"
echo ""

# Track results
total_phases=0
passed_phases=0
failed_phases=0

# Run test phase helper
run_phase() {
    local phase_name="$1"
    local phase_script="$TEST_DIR/$phase_name.sh"

    total_phases=$((total_phases + 1))

    echo -e "${BLUE}[INFO]${NC} Running phase: $phase_name"

    if [ ! -f "$phase_script" ]; then
        echo -e "${YELLOW}[SKIP]${NC} Phase script not found: $phase_script"
        return 0
    fi

    if [ ! -x "$phase_script" ]; then
        chmod +x "$phase_script"
    fi

    if "$phase_script"; then
        echo -e "${GREEN}[PASS]${NC} Phase completed: $phase_name"
        passed_phases=$((passed_phases + 1))
        return 0
    else
        echo -e "${RED}[FAIL]${NC} Phase failed: $phase_name"
        failed_phases=$((failed_phases + 1))
        return 1
    fi
}

# Run test phases in recommended order
run_phase "test-structure" || true
run_phase "test-dependencies" || true
run_phase "test-unit" || true
run_phase "test-smoke" || true
run_phase "test-integration" || true
run_phase "test-business" || true
run_phase "test-performance" || true

# Summary
echo ""
echo -e "${BOLD}=== Test Summary ===${NC}"
echo -e "Total phases:  $total_phases"
echo -e "${GREEN}Passed:${NC}        $passed_phases"
echo -e "${RED}Failed:${NC}        $failed_phases"
echo ""

if [ "$failed_phases" -eq 0 ]; then
    echo -e "${GREEN}${BOLD}✓ All tests passed${NC}"
    exit 0
else
    echo -e "${RED}${BOLD}✗ Some tests failed${NC}"
    exit 1
fi
