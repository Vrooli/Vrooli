#!/usr/bin/env bash
# OpenTripPlanner Test Runner

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASE="${1:-all}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

run_test_phase() {
    local phase_name="$1"
    local phase_script="${SCRIPT_DIR}/phases/${phase_name}"
    
    if [[ ! -f "$phase_script" ]]; then
        echo -e "${YELLOW}Skipping ${phase_name} (not implemented)${NC}"
        return 0
    fi
    
    echo -e "\n${GREEN}Running ${phase_name}...${NC}"
    
    if bash "$phase_script"; then
        echo -e "${GREEN}✓ ${phase_name} passed${NC}"
        return 0
    else
        echo -e "${RED}✗ ${phase_name} failed${NC}"
        return 1
    fi
}

main() {
    local exit_code=0
    
    echo "OpenTripPlanner Test Suite"
    echo "========================="
    
    case "$PHASE" in
        all)
            run_test_phase "test-smoke.sh" || exit_code=1
            run_test_phase "test-integration.sh" || exit_code=1
            run_test_phase "test-unit.sh" || exit_code=1
            ;;
        smoke)
            run_test_phase "test-smoke.sh" || exit_code=1
            ;;
        integration)
            run_test_phase "test-integration.sh" || exit_code=1
            ;;
        unit)
            run_test_phase "test-unit.sh" || exit_code=1
            ;;
        *)
            echo "Unknown test phase: $PHASE"
            echo "Usage: $0 [all|smoke|integration|unit]"
            exit 1
            ;;
    esac
    
    echo ""
    if [[ $exit_code -eq 0 ]]; then
        echo -e "${GREEN}All tests passed!${NC}"
    else
        echo -e "${RED}Some tests failed${NC}"
    fi
    
    exit $exit_code
}

main "$@"