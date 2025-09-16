#!/bin/bash

# Open Data Cube Test Runner
# Main test orchestration script

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test phase to run
PHASE="${1:-all}"

# Function to run a test phase
run_test_phase() {
    local phase="$1"
    local script="${SCRIPT_DIR}/phases/test-${phase}.sh"
    
    if [[ ! -f "$script" ]]; then
        echo -e "${YELLOW}Warning: Test script $script not found${NC}"
        return 1
    fi
    
    echo -e "${GREEN}Running $phase tests...${NC}"
    
    if bash "$script"; then
        echo -e "${GREEN}✓ $phase tests passed${NC}"
        return 0
    else
        echo -e "${RED}✗ $phase tests failed${NC}"
        return 1
    fi
}

# Main test execution
main() {
    echo "========================================="
    echo "Open Data Cube Resource Test Suite"
    echo "========================================="
    echo ""
    
    local exit_code=0
    
    case "$PHASE" in
        smoke)
            run_test_phase "smoke" || exit_code=$?
            ;;
        integration)
            run_test_phase "integration" || exit_code=$?
            ;;
        unit)
            run_test_phase "unit" || exit_code=$?
            ;;
        all)
            # Run all test phases in sequence
            for phase in smoke integration unit; do
                echo ""
                run_test_phase "$phase" || exit_code=$?
            done
            ;;
        *)
            echo -e "${RED}Unknown test phase: $PHASE${NC}"
            echo "Usage: $0 [smoke|integration|unit|all]"
            exit 1
            ;;
    esac
    
    echo ""
    echo "========================================="
    
    if [[ $exit_code -eq 0 ]]; then
        echo -e "${GREEN}All tests completed successfully${NC}"
    else
        echo -e "${RED}Some tests failed${NC}"
    fi
    
    echo "========================================="
    
    exit $exit_code
}

# Run main function
main