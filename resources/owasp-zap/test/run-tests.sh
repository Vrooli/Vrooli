#!/usr/bin/env bash
################################################################################
# OWASP ZAP Resource - Test Runner
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ZAP_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Source test phases
TEST_PHASES_DIR="${SCRIPT_DIR}/phases"

# Run test phases in order
run_test_phase() {
    local phase_script="$1"
    local phase_name="$(basename "${phase_script}" .sh | sed 's/test-//')"
    
    echo "========================================="
    echo "Running ${phase_name} tests..."
    echo "========================================="
    
    if bash "${phase_script}"; then
        echo "✓ ${phase_name} tests passed"
        return 0
    else
        echo "✗ ${phase_name} tests failed"
        return 1
    fi
}

# Main test execution
main() {
    local test_type="${1:-all}"
    local failed=0
    
    case "${test_type}" in
        smoke)
            run_test_phase "${TEST_PHASES_DIR}/test-smoke.sh" || ((failed++))
            ;;
        integration)
            run_test_phase "${TEST_PHASES_DIR}/test-integration.sh" || ((failed++))
            ;;
        unit)
            run_test_phase "${TEST_PHASES_DIR}/test-unit.sh" || ((failed++))
            ;;
        all)
            run_test_phase "${TEST_PHASES_DIR}/test-unit.sh" || ((failed++))
            run_test_phase "${TEST_PHASES_DIR}/test-smoke.sh" || ((failed++))
            run_test_phase "${TEST_PHASES_DIR}/test-integration.sh" || ((failed++))
            ;;
        *)
            echo "Unknown test type: ${test_type}"
            echo "Usage: $0 [smoke|integration|unit|all]"
            exit 1
            ;;
    esac
    
    if [[ ${failed} -eq 0 ]]; then
        echo "========================================="
        echo "All tests passed successfully!"
        echo "========================================="
        exit 0
    else
        echo "========================================="
        echo "${failed} test phase(s) failed"
        echo "========================================="
        exit 1
    fi
}

main "$@"