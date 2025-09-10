#!/usr/bin/env bash
# DeepStack Resource - Main Test Runner

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"

# Test phase to run
PHASE="${1:-all}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Run test phase
run_phase() {
    local phase="$1"
    local script="${SCRIPT_DIR}/phases/test-${phase}.sh"
    
    if [[ ! -f "$script" ]]; then
        echo -e "${RED}✗ Test phase script not found: $script${NC}" >&2
        return 1
    fi
    
    echo -e "${YELLOW}Running $phase tests...${NC}"
    echo "======================================"
    
    if bash "$script"; then
        echo -e "${GREEN}✓ $phase tests passed${NC}"
        return 0
    else
        echo -e "${RED}✗ $phase tests failed${NC}" >&2
        return 1
    fi
}

# Main test execution
main() {
    echo "DeepStack Resource Test Suite"
    echo "============================="
    echo ""
    
    local failures=0
    
    case "$PHASE" in
        all)
            # Run all test phases
            run_phase "smoke" || ((failures++))
            echo ""
            run_phase "integration" || ((failures++))
            echo ""
            run_phase "unit" || ((failures++))
            ;;
        smoke|integration|unit)
            # Run specific phase
            run_phase "$PHASE" || ((failures++))
            ;;
        *)
            echo -e "${RED}Error: Unknown test phase: $PHASE${NC}" >&2
            echo "Valid phases: all, smoke, integration, unit" >&2
            exit 1
            ;;
    esac
    
    echo ""
    echo "======================================"
    if [[ $failures -eq 0 ]]; then
        echo -e "${GREEN}✓ All tests passed successfully${NC}"
        exit 0
    else
        echo -e "${RED}✗ Tests failed with $failures error(s)${NC}" >&2
        exit 1
    fi
}

# Run main function
main