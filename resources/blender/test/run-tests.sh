#!/usr/bin/env bash
################################################################################
# Blender Test Runner - v2.0 Universal Contract Compliant
# 
# Main test orchestrator for Blender resource validation
# Delegates to phase-specific test scripts
#
# Usage:
#   ./run-tests.sh [phase]
#   Phases: smoke, unit, integration, all
#
################################################################################

set -euo pipefail

# Resolve paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"
PHASES_DIR="${SCRIPT_DIR}/phases"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"

# Test phase to run
PHASE="${1:-all}"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to run a test phase
run_phase() {
    local phase="$1"
    local script="${PHASES_DIR}/test-${phase}.sh"
    
    if [[ ! -f "$script" ]]; then
        echo -e "${YELLOW}[SKIP]${NC} Test phase '${phase}' not found"
        return 2  # Not available
    fi
    
    echo -e "${GREEN}[START]${NC} Running ${phase} tests..."
    
    if bash "$script"; then
        echo -e "${GREEN}[PASS]${NC} ${phase} tests completed successfully"
        return 0
    else
        echo -e "${RED}[FAIL]${NC} ${phase} tests failed"
        return 1
    fi
}

# Main test execution
main() {
    local exit_code=0
    
    echo "======================================"
    echo "Blender Resource Test Suite"
    echo "Phase: ${PHASE}"
    echo "======================================"
    
    case "$PHASE" in
        smoke)
            run_phase smoke || exit_code=$?
            ;;
        unit)
            run_phase unit || exit_code=$?
            ;;
        integration)
            run_phase integration || exit_code=$?
            ;;
        all)
            # Run all phases in order
            local any_failed=false
            
            for phase in smoke unit integration; do
                if ! run_phase "$phase"; then
                    any_failed=true
                fi
            done
            
            if [[ "$any_failed" == true ]]; then
                exit_code=1
            fi
            ;;
        *)
            echo -e "${RED}[ERROR]${NC} Unknown test phase: ${PHASE}"
            echo "Valid phases: smoke, unit, integration, all"
            exit 1
            ;;
    esac
    
    echo "======================================"
    if [[ $exit_code -eq 0 ]]; then
        echo -e "${GREEN}[SUCCESS]${NC} All tests passed"
    else
        echo -e "${RED}[FAILURE]${NC} Some tests failed"
    fi
    echo "======================================"
    
    exit $exit_code
}

main "$@"