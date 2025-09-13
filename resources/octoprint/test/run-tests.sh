#!/bin/bash

# OctoPrint Test Runner
# Main test orchestrator for OctoPrint resource

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Run specified test phase
run_test_phase() {
    local phase="${1:-all}"
    local phase_script="${SCRIPT_DIR}/phases/test-${phase}.sh"
    
    if [[ "${phase}" == "all" ]]; then
        # Run all test phases
        local all_passed=true
        
        for test_phase in smoke unit integration; do
            echo ""
            echo "Running ${test_phase} tests..."
            if bash "${SCRIPT_DIR}/phases/test-${test_phase}.sh"; then
                echo "✓ ${test_phase} tests passed"
            else
                echo "✗ ${test_phase} tests failed"
                all_passed=false
            fi
        done
        
        if [[ "${all_passed}" == true ]]; then
            echo ""
            echo "All test phases PASSED"
            exit 0
        else
            echo ""
            echo "Some test phases FAILED"
            exit 1
        fi
    elif [[ -f "${phase_script}" ]]; then
        # Run specific phase
        bash "${phase_script}"
    else
        echo "Error: Unknown test phase '${phase}'"
        echo "Available phases: smoke, unit, integration, all"
        exit 1
    fi
}

# Main execution
main() {
    local phase="${1:-all}"
    
    echo "OctoPrint Resource Test Runner"
    echo "=============================="
    echo "Test phase: ${phase}"
    echo "Port: ${OCTOPRINT_PORT}"
    echo ""
    
    run_test_phase "${phase}"
}

main "$@"