#!/usr/bin/env bash
################################################################################
# Windmill Test Library - v2.0 Universal Contract Compliant
# 
# Test command implementations that delegate to test runner
################################################################################

# Test runner script location
readonly WINDMILL_TEST_RUNNER="${WINDMILL_CLI_DIR}/test/run-tests.sh"

################################################################################
# Test Command Implementations
################################################################################

windmill::test::smoke() {
    log::info "Running Windmill smoke tests..."
    
    if [[ ! -f "$WINDMILL_TEST_RUNNER" ]]; then
        log::error "Test runner not found at: $WINDMILL_TEST_RUNNER"
        return 1
    fi
    
    bash "$WINDMILL_TEST_RUNNER" smoke
    return $?
}

windmill::test::integration() {
    log::info "Running Windmill integration tests..."
    
    if [[ ! -f "$WINDMILL_TEST_RUNNER" ]]; then
        log::error "Test runner not found at: $WINDMILL_TEST_RUNNER"
        return 1
    fi
    
    bash "$WINDMILL_TEST_RUNNER" integration
    return $?
}

windmill::test::unit() {
    log::info "Running Windmill unit tests..."
    
    if [[ ! -f "$WINDMILL_TEST_RUNNER" ]]; then
        log::error "Test runner not found at: $WINDMILL_TEST_RUNNER"
        return 1
    fi
    
    bash "$WINDMILL_TEST_RUNNER" unit
    return $?
}

windmill::test::all() {
    log::info "Running all Windmill tests..."
    
    if [[ ! -f "$WINDMILL_TEST_RUNNER" ]]; then
        log::error "Test runner not found at: $WINDMILL_TEST_RUNNER"
        return 1
    fi
    
    bash "$WINDMILL_TEST_RUNNER" all
    return $?
}

################################################################################
# Legacy Support (if needed)
################################################################################

# Quick status check (used by smoke test)
windmill::quick_status() {
    log::info "Quick Windmill status check..."
    
    # Check if service is running
    if docker ps --format "table {{.Names}}" | grep -q "${WINDMILL_PROJECT_NAME}"; then
        log::success "Windmill is running"
        
        # Try health endpoint
        if timeout 5 curl -sf "${WINDMILL_BASE_URL}/api/version" &>/dev/null; then
            log::success "Health endpoint responsive"
            return 0
        else
            log::warning "Health endpoint not responding"
            return 1
        fi
    else
        log::error "Windmill is not running"
        return 1
    fi
}