#!/bin/bash
# VirusTotal Resource Test Runner
# Main test orchestrator following v2.0 contract

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"
PHASES_DIR="${SCRIPT_DIR}/phases"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test phase to run (default: all)
TEST_PHASE="${1:-all}"

# Function to run a test phase
run_phase() {
    local phase="$1"
    local script="${PHASES_DIR}/test-${phase}.sh"
    
    if [[ ! -f "$script" ]]; then
        echo -e "${YELLOW}Warning: Test phase '${phase}' not found at ${script}${NC}"
        return 1
    fi
    
    echo -e "${GREEN}Running ${phase} tests...${NC}"
    
    if bash "$script"; then
        echo -e "${GREEN}✓ ${phase} tests passed${NC}"
        return 0
    else
        echo -e "${RED}✗ ${phase} tests failed${NC}"
        return 1
    fi
}

# Main test execution
main() {
    echo "VirusTotal Resource Test Suite"
    echo "=============================="
    echo "Phase: ${TEST_PHASE}"
    echo ""
    
    local exit_code=0
    
    case "${TEST_PHASE}" in
        smoke)
            run_phase "smoke" || exit_code=$?
            ;;
        integration)
            run_phase "integration" || exit_code=$?
            ;;
        unit)
            run_phase "unit" || exit_code=$?
            ;;
        all)
            # Run all phases in sequence
            for phase in smoke unit integration; do
                echo ""
                if ! run_phase "$phase"; then
                    exit_code=1
                    # Continue running other tests even if one fails
                fi
            done
            ;;
        *)
            echo -e "${RED}Error: Unknown test phase '${TEST_PHASE}'${NC}"
            echo "Valid phases: smoke, integration, unit, all"
            exit 1
            ;;
    esac
    
    echo ""
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}All tests completed successfully!${NC}"
    else
        echo -e "${RED}Some tests failed. Please review the output above.${NC}"
    fi
    
    exit $exit_code
}

# Run main function
main