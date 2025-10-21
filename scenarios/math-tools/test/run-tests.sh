#!/bin/bash
# Math Tools - Comprehensive Test Suite
#
# This script orchestrates all testing phases for the math-tools scenario

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

# Test phase tracking
PASSED_PHASES=0
FAILED_PHASES=0
TOTAL_PHASES=0

# Run a test phase
run_phase() {
    local phase_name="$1"
    local phase_script="$2"

    TOTAL_PHASES=$((TOTAL_PHASES + 1))

    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Running Phase: ${phase_name}${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

    if [[ -f "$phase_script" ]]; then
        if bash "$phase_script"; then
            echo -e "\n${GREEN}✓ Phase ${phase_name} PASSED${NC}"
            PASSED_PHASES=$((PASSED_PHASES + 1))
            return 0
        else
            echo -e "\n${RED}✗ Phase ${phase_name} FAILED${NC}"
            FAILED_PHASES=$((FAILED_PHASES + 1))
            return 1
        fi
    else
        echo -e "${YELLOW}⚠ Phase ${phase_name} SKIPPED (script not found: $phase_script)${NC}"
        return 0
    fi
}

# Main test execution
main() {
    echo -e "${BLUE}╔════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║     Math Tools - Test Suite Runner        ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════╝${NC}"

    # Change to scenario directory
    cd "$SCENARIO_DIR"

    # Run test phases in order
    run_phase "Structure Validation" "$SCRIPT_DIR/phases/test-structure.sh" || true
    run_phase "Dependency Check" "$SCRIPT_DIR/phases/test-dependencies.sh" || true
    run_phase "Unit Tests" "$SCRIPT_DIR/phases/test-unit.sh" || true
    run_phase "Integration Tests" "$SCRIPT_DIR/phases/test-integration.sh" || true
    run_phase "Business Logic Tests" "$SCRIPT_DIR/phases/test-business.sh" || true
    run_phase "Performance Tests" "$SCRIPT_DIR/phases/test-performance.sh" || true

    # Summary
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Test Suite Summary${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "Total Phases:  ${TOTAL_PHASES}"
    echo -e "${GREEN}Passed:       ${PASSED_PHASES}${NC}"
    echo -e "${RED}Failed:       ${FAILED_PHASES}${NC}"

    if [[ $FAILED_PHASES -eq 0 ]]; then
        echo -e "\n${GREEN}✓ All test phases passed!${NC}\n"
        exit 0
    else
        echo -e "\n${RED}✗ Some test phases failed${NC}\n"
        exit 1
    fi
}

main "$@"
