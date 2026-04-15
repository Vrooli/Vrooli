#!/usr/bin/env bash
################################################################################
# Home Assistant Smoke Tests
# Quick validation that service is running and responsive
# Must complete in < 30 seconds
################################################################################

set -euo pipefail

# Get directories
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
TEST_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${TEST_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"

# Source dependencies
source "${REPO_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/lib/test.sh"

# Set timeout for smoke tests
export TEST_TIMEOUT=30

#######################################
# Main smoke test execution
#######################################
main() {
    local start_time=$(date +%s)
    
    log::header "Home Assistant Smoke Tests"
    log::info "Maximum duration: ${TEST_TIMEOUT}s"
    
    # Run smoke tests via test library
    if home_assistant::test::smoke; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        if [[ $duration -gt $TEST_TIMEOUT ]]; then
            log::error "Smoke tests took too long: ${duration}s (limit: ${TEST_TIMEOUT}s)"
            exit 1
        fi
        
        log::success "Smoke tests completed in ${duration}s"
        exit 0
    else
        log::error "Smoke tests failed"
        exit 1
    fi
}

# Run with timeout enforcement
timeout "$TEST_TIMEOUT" bash -c "$(declare -f main); main" || {
    exit_code=$?
    if [[ $exit_code -eq 124 ]]; then
        log::error "Smoke tests timed out after ${TEST_TIMEOUT}s"
    fi
    exit $exit_code
}
