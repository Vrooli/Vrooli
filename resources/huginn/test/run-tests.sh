#!/usr/bin/env bash
################################################################################
# Huginn Test Runner - v2.0 Universal Contract Compliant
# 
# Main test orchestrator for Huginn resource testing
################################################################################

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/config/defaults.sh"

# Test configuration
TEST_PHASES_DIR="${SCRIPT_DIR}/phases"
TEST_RESULTS_FILE="${SCRIPT_DIR}/test-results.txt"
TEST_LOG_FILE="${SCRIPT_DIR}/test.log"

# Initialize test environment
test::init() {
    echo "=== Huginn Test Suite ===" | tee "${TEST_LOG_FILE}"
    echo "Starting at: $(date)" | tee -a "${TEST_LOG_FILE}"
    > "${TEST_RESULTS_FILE}"
}

# Run a test phase
test::run_phase() {
    local phase="${1}"
    local phase_script="${TEST_PHASES_DIR}/test-${phase}.sh"
    
    if [[ ! -f "${phase_script}" ]]; then
        echo "⚠️  Test phase '${phase}' not found at ${phase_script}" | tee -a "${TEST_LOG_FILE}"
        return 1
    fi
    
    echo "" | tee -a "${TEST_LOG_FILE}"
    echo "📋 Running ${phase} tests..." | tee -a "${TEST_LOG_FILE}"
    echo "---" | tee -a "${TEST_LOG_FILE}"
    
    local start_time=$(date +%s)
    
    if bash "${phase_script}" 2>&1 | tee -a "${TEST_LOG_FILE}"; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo "✅ ${phase} tests passed (${duration}s)" | tee -a "${TEST_RESULTS_FILE}"
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo "❌ ${phase} tests failed (${duration}s)" | tee -a "${TEST_RESULTS_FILE}"
        return 1
    fi
}

# Run all test phases
test::run_all() {
    local failed=0
    
    for phase in smoke integration unit; do
        if ! test::run_phase "${phase}"; then
            ((failed++))
        fi
    done
    
    return ${failed}
}

# Main test execution
main() {
    local test_type="${1:-all}"
    
    test::init
    
    case "${test_type}" in
        smoke|integration|unit)
            test::run_phase "${test_type}"
            ;;
        all)
            test::run_all
            ;;
        *)
            echo "❌ Unknown test type: ${test_type}"
            echo "   Valid options: smoke, integration, unit, all"
            exit 1
            ;;
    esac
    
    local exit_code=$?
    
    echo "" | tee -a "${TEST_LOG_FILE}"
    echo "=== Test Summary ===" | tee -a "${TEST_LOG_FILE}"
    cat "${TEST_RESULTS_FILE}" | tee -a "${TEST_LOG_FILE}"
    echo "Completed at: $(date)" | tee -a "${TEST_LOG_FILE}"
    
    exit ${exit_code}
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi