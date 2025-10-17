#!/usr/bin/env bash
# Splink Test Runner - Main test orchestrator

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Test phase to run
TEST_PHASE="${1:-all}"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Run a test phase
run_phase() {
    local phase="$1"
    local phase_script="${SCRIPT_DIR}/phases/test-${phase}.sh"
    
    if [[ ! -f "$phase_script" ]]; then
        echo -e "${YELLOW}Warning: Test phase '$phase' not found${NC}"
        return 1
    fi
    
    echo -e "${GREEN}Running $phase tests...${NC}"
    if bash "$phase_script"; then
        echo -e "${GREEN}✓ $phase tests passed${NC}"
        return 0
    else
        echo -e "${RED}✗ $phase tests failed${NC}"
        return 1
    fi
}

# Main test execution
main() {
    echo "==================================="
    echo "Splink Resource Test Suite"
    echo "==================================="
    echo ""
    
    local failed=0
    
    case "$TEST_PHASE" in
        smoke)
            run_phase "smoke" || ((failed++))
            ;;
        integration)
            run_phase "integration" || ((failed++))
            ;;
        unit)
            run_phase "unit" || ((failed++))
            ;;
        all)
            run_phase "smoke" || ((failed++))
            echo ""
            run_phase "unit" || ((failed++))
            echo ""
            run_phase "integration" || ((failed++))
            ;;
        *)
            echo -e "${RED}Error: Unknown test phase '$TEST_PHASE'${NC}"
            echo "Available phases: smoke, integration, unit, all"
            exit 1
            ;;
    esac
    
    echo ""
    echo "==================================="
    
    if [[ $failed -eq 0 ]]; then
        echo -e "${GREEN}All tests PASSED ✓${NC}"
        exit 0
    else
        echo -e "${RED}$failed test phase(s) FAILED ✗${NC}"
        exit 1
    fi
}

# Execute main function
main "$@"