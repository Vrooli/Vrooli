#!/bin/bash
# Main test runner for Ultralytics YOLO resource

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASES_DIR="${SCRIPT_DIR}/phases"

# Test phase to run
PHASE="${1:-all}"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

#######################################
# Run a specific test phase
#######################################
run_phase() {
    local phase="$1"
    local script="${PHASES_DIR}/test-${phase}.sh"
    
    if [[ ! -f "$script" ]]; then
        echo -e "${RED}Error: Test phase '$phase' not found${NC}"
        return 1
    fi
    
    echo -e "${YELLOW}Running $phase tests...${NC}"
    
    if bash "$script"; then
        echo -e "${GREEN}✓ $phase tests passed${NC}"
        return 0
    else
        echo -e "${RED}✗ $phase tests failed${NC}"
        return 1
    fi
}

#######################################
# Main test execution
#######################################
main() {
    echo "Ultralytics YOLO Test Suite"
    echo "==========================="
    
    local failed=0
    
    case "$PHASE" in
        smoke)
            run_phase smoke || ((failed++))
            ;;
        integration)
            run_phase integration || ((failed++))
            ;;
        unit)
            run_phase unit || ((failed++))
            ;;
        all)
            for phase in smoke unit integration; do
                run_phase "$phase" || ((failed++))
                echo ""
            done
            ;;
        *)
            echo -e "${RED}Unknown test phase: $PHASE${NC}"
            echo "Available phases: smoke, integration, unit, all"
            exit 1
            ;;
    esac
    
    # Final summary
    echo ""
    echo "==========================="
    if [[ $failed -eq 0 ]]; then
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}$failed test phase(s) failed${NC}"
        exit 1
    fi
}

# Execute main function
main