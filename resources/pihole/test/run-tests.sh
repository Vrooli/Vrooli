#!/usr/bin/env bash
# Pi-hole Test Runner - Main test orchestrator
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Source test phases
TEST_PHASES_DIR="${SCRIPT_DIR}/phases"

# Run specified test phase
run_test_phase() {
    local phase="${1:-all}"
    
    case "$phase" in
        smoke)
            bash "${TEST_PHASES_DIR}/test-smoke.sh"
            ;;
        integration)
            bash "${TEST_PHASES_DIR}/test-integration.sh"
            ;;
        unit)
            bash "${TEST_PHASES_DIR}/test-unit.sh"
            ;;
        all)
            local failed=0
            
            echo "Running all test phases..."
            echo "=========================="
            
            # Run each phase
            for test_phase in smoke integration unit; do
                echo ""
                echo "Phase: $test_phase"
                echo "----------------"
                if bash "${TEST_PHASES_DIR}/test-${test_phase}.sh"; then
                    echo "✓ $test_phase tests passed"
                else
                    echo "✗ $test_phase tests failed"
                    ((failed++))
                fi
            done
            
            echo ""
            echo "=========================="
            if [[ $failed -eq 0 ]]; then
                echo "All test phases passed!"
                exit 0
            else
                echo "Failed $failed test phases"
                exit 1
            fi
            ;;
        *)
            echo "Error: Unknown test phase: $phase" >&2
            echo "Valid phases: smoke, integration, unit, all" >&2
            exit 1
            ;;
    esac
}

# Main execution
main() {
    local phase="${1:-all}"
    run_test_phase "$phase"
}

main "$@"