#!/usr/bin/env bash
################################################################################
# Node-RED Test Runner - v2.0 Contract Compliant
# 
# Main test orchestrator for Node-RED resource validation
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NODE_RED_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${NODE_RED_DIR}/../.." && pwd)"

# Source common utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"

# Test configuration
TEST_TIMEOUT="${TEST_TIMEOUT:-600}"
NODE_RED_PORT="${NODE_RED_PORT:-1880}"
NODE_RED_CONTAINER="${NODE_RED_CONTAINER_NAME:-node-red}"

# Test phases
declare -A TEST_PHASES=(
    ["smoke"]="Quick health and connectivity validation"
    ["integration"]="End-to-end workflow functionality testing"
    ["unit"]="Library function unit tests"
    ["all"]="Run all test phases sequentially"
)

# Test execution
run_test_phase() {
    local phase="$1"
    local test_script="${SCRIPT_DIR}/phases/test-${phase}.sh"
    
    if [[ ! -f "$test_script" ]]; then
        log::error "Test phase script not found: $test_script"
        return 1
    fi
    
    log::info "Running ${phase} tests..."
    
    if timeout "${TEST_TIMEOUT}" bash "$test_script"; then
        log::success "${phase} tests passed"
        return 0
    else
        log::error "${phase} tests failed"
        return 1
    fi
}

# Main test runner
main() {
    local phase="${1:-all}"
    
    # Validate phase
    if [[ ! "${TEST_PHASES[$phase]+exists}" ]]; then
        log::error "Unknown test phase: $phase"
        log::info "Available phases: ${!TEST_PHASES[*]}"
        exit 1
    fi
    
    log::header "Node-RED Test Suite - Phase: $phase"
    
    # Check if Node-RED is running
    if ! docker ps --format "{{.Names}}" | grep -q "^${NODE_RED_CONTAINER}$"; then
        log::warning "Node-RED is not running. Starting it for tests..."
        "${NODE_RED_DIR}/cli.sh" manage start --wait || {
            log::error "Failed to start Node-RED for testing"
            exit 1
        }
    fi
    
    # Run tests based on phase
    local failed=0
    
    if [[ "$phase" == "all" ]]; then
        for test_phase in smoke integration unit; do
            if ! run_test_phase "$test_phase"; then
                ((failed++))
            fi
        done
    else
        if ! run_test_phase "$phase"; then
            ((failed++))
        fi
    fi
    
    # Report results
    if [[ $failed -eq 0 ]]; then
        log::success "All tests passed!"
        exit 0
    else
        log::error "$failed test phase(s) failed"
        exit 1
    fi
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi