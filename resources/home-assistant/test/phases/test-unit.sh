#!/usr/bin/env bash
################################################################################
# Home Assistant Unit Tests
# Library function validation
# Must complete in < 60 seconds
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

# Set timeout for unit tests
export TEST_TIMEOUT=60

#######################################
# Main unit test execution
#######################################
main() {
    local start_time=$(date +%s)
    
    log::header "Home Assistant Unit Tests"
    log::info "Maximum duration: ${TEST_TIMEOUT}s"
    
    # Run unit tests via test library
    if home_assistant::test::unit; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        if [[ $duration -gt $TEST_TIMEOUT ]]; then
            log::error "Unit tests took too long: ${duration}s (limit: ${TEST_TIMEOUT}s)"
            exit 1
        fi
        
        log::success "Unit tests completed in ${duration}s"
        exit 0
    else
        log::error "Unit tests failed"
        exit 1
    fi
}

# Run with timeout enforcement
timeout "$TEST_TIMEOUT" bash -c "$(declare -f main); main" || {
    exit_code=$?
    if [[ $exit_code -eq 124 ]]; then
        log::error "Unit tests timed out after ${TEST_TIMEOUT}s"
    fi
    exit $exit_code
}