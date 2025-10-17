#!/usr/bin/env bash
################################################################################
# Home Assistant Integration Tests
# End-to-end functionality validation
# Must complete in < 120 seconds
################################################################################

set -euo pipefail

# Get directories
PHASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$PHASE_DIR/.." && pwd)"
RESOURCE_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/lib/test.sh"

# Set timeout for integration tests
export TEST_TIMEOUT=120

#######################################
# Main integration test execution
#######################################
main() {
    local start_time=$(date +%s)
    
    log::header "Home Assistant Integration Tests"
    log::info "Maximum duration: ${TEST_TIMEOUT}s"
    
    # Run integration tests via test library
    if home_assistant::test::integration; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        if [[ $duration -gt $TEST_TIMEOUT ]]; then
            log::error "Integration tests took too long: ${duration}s (limit: ${TEST_TIMEOUT}s)"
            exit 1
        fi
        
        log::success "Integration tests completed in ${duration}s"
        exit 0
    else
        log::error "Integration tests failed"
        exit 1
    fi
}

# Run with timeout enforcement
timeout "$TEST_TIMEOUT" bash -c "$(declare -f main); main" || {
    exit_code=$?
    if [[ $exit_code -eq 124 ]]; then
        log::error "Integration tests timed out after ${TEST_TIMEOUT}s"
    fi
    exit $exit_code
}