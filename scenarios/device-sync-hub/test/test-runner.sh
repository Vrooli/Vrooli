#!/usr/bin/env bash

# Test Runner for Device Sync Hub
# Executes phased testing based on the argument provided
set -euo pipefail

# Test environment
export SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export TEST_DIR="$SCENARIO_DIR/test"
export PHASES_DIR="$TEST_DIR/phases"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default phase
PHASE="${1:-all}"

# Helper functions
run_phase() {
    local phase_name="$1"
    local phase_script="$PHASES_DIR/test-${phase_name}.sh"
    
    if [[ ! -f "$phase_script" ]]; then
        echo -e "${RED}Error: Test phase '$phase_name' not found${NC}"
        return 1
    fi
    
    echo -e "${BLUE}Running $phase_name tests...${NC}"
    if bash "$phase_script"; then
        echo -e "${GREEN}✓ $phase_name tests passed${NC}"
        return 0
    else
        echo -e "${RED}✗ $phase_name tests failed${NC}"
        return 1
    fi
}

# Main execution
case "$PHASE" in
    smoke)
        run_phase "smoke"
        ;;
    unit)
        run_phase "unit"
        ;;
    integration)
        run_phase "integration"
        ;;
    all)
        echo -e "${GREEN}=== Running All Test Phases ===${NC}"
        
        # Track overall status
        ALL_PASSED=true
        
        # Run each phase
        for phase in smoke unit integration; do
            if ! run_phase "$phase"; then
                ALL_PASSED=false
            fi
            echo ""
        done
        
        # Final result
        if $ALL_PASSED; then
            echo -e "${GREEN}=== All tests passed! ===${NC}"
            exit 0
        else
            echo -e "${RED}=== Some tests failed ===${NC}"
            exit 1
        fi
        ;;
    *)
        echo "Usage: $0 [smoke|unit|integration|all]"
        echo ""
        echo "Phases:"
        echo "  smoke       - Quick health checks"
        echo "  unit        - Component validation"
        echo "  integration - Full integration tests"
        echo "  all         - Run all test phases (default)"
        exit 1
        ;;
esac