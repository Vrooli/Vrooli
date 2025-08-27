#!/usr/bin/env bash
# SimPy test functions - focused on resource health, not business functionality

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SIMPY_TEST_DIR="${APP_ROOT}/resources/simpy/lib"

# Source dependencies
# shellcheck disable=SC1091
source "${SIMPY_TEST_DIR}/core.sh"

#######################################
# Smoke test - basic health check
#######################################
simpy::test::smoke() {
    log::info "Running SimPy smoke test..."
    
    # Check if installed
    if ! simpy::is_installed; then
        log::error "SimPy is not installed"
        return 1
    fi
    
    # Check if service can start
    local was_running=false
    if simpy::is_running; then
        was_running=true
        log::info "SimPy service already running"
    else
        log::info "Starting SimPy service for test..."
        if ! simpy::start; then
            log::error "Failed to start SimPy service"
            return 1
        fi
    fi
    
    # Test connection
    if ! simpy::test_connection; then
        log::error "SimPy service is not responding"
        return 1
    fi
    
    # Test basic API endpoints
    local base_url="http://${SIMPY_HOST}:${SIMPY_PORT}"
    
    # Test health endpoint
    if ! curl -s -f "$base_url/health" >/dev/null; then
        log::error "Health endpoint not responding"
        return 1
    fi
    
    # Test version endpoint
    if ! curl -s -f "$base_url/version" >/dev/null; then
        log::error "Version endpoint not responding"
        return 1
    fi
    
    # Clean up if we started the service for testing
    if [[ "$was_running" == "false" ]]; then
        log::info "Stopping SimPy service after test..."
        simpy::stop || true
    fi
    
    log::success "SimPy smoke test passed"
    return 0
}

#######################################
# Integration test - test with other services
#######################################
simpy::test::integration() {
    log::info "Running SimPy integration test..."
    
    # Smoke test first
    if ! simpy::test::smoke; then
        log::error "Smoke test failed, skipping integration test"
        return 1
    fi
    
    # Start service if not running
    local was_running=false
    if simpy::is_running; then
        was_running=true
    else
        if ! simpy::start; then
            log::error "Failed to start SimPy service for integration test"
            return 1
        fi
    fi
    
    # Test simulation execution via API
    local test_script='{"code": "import simpy\nprint({\"test\": \"success\", \"simpy_version\": simpy.__version__})"}'
    local response
    
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$test_script" \
        "http://${SIMPY_HOST}:${SIMPY_PORT}/simulate")
    
    if [[ -z "$response" ]]; then
        log::error "No response from simulation API"
        return 1
    fi
    
    # Check if simulation was successful
    if ! echo "$response" | grep -q '"success": true'; then
        log::error "Test simulation failed"
        log::error "Response: $response"
        return 1
    fi
    
    # Clean up if we started the service
    if [[ "$was_running" == "false" ]]; then
        simpy::stop || true
    fi
    
    log::success "SimPy integration test passed"
    return 0
}

#######################################
# Unit test - test individual functions
#######################################
simpy::test::unit() {
    log::info "Running SimPy unit tests..."
    
    # Test basic function availability
    if ! command -v python3 >/dev/null; then
        log::error "Python3 not available"
        return 1
    fi
    
    # Test SimPy import
    if ! python3 -c "import simpy; print('SimPy version:', simpy.__version__)" 2>/dev/null; then
        log::error "SimPy module not importable"
        return 1
    fi
    
    # Test configuration functions
    simpy::export_config
    
    # Test directory creation
    if ! mkdir -p "$SIMPY_DATA_DIR/test_dir" 2>/dev/null; then
        log::error "Cannot create test directory"
        return 1
    fi
    rm -rf "$SIMPY_DATA_DIR/test_dir"
    
    # Test version function
    local version
    version=$(simpy::get_version)
    if [[ "$version" == "not_installed" ]]; then
        log::error "SimPy version check failed"
        return 1
    fi
    
    log::success "SimPy unit tests passed"
    return 0
}

#######################################
# Run all tests
#######################################
simpy::test::all() {
    log::info "Running all SimPy tests..."
    
    local failed=0
    
    # Unit tests
    if ! simpy::test::unit; then
        log::error "Unit tests failed"
        ((failed++))
    fi
    
    # Smoke tests
    if ! simpy::test::smoke; then
        log::error "Smoke tests failed"
        ((failed++))
    fi
    
    # Integration tests
    if ! simpy::test::integration; then
        log::error "Integration tests failed"
        ((failed++))
    fi
    
    if [[ $failed -eq 0 ]]; then
        log::success "All SimPy tests passed"
        return 0
    else
        log::error "$failed test suite(s) failed"
        return 1
    fi
}