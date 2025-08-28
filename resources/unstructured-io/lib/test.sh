#!/usr/bin/env bash
# Unstructured.io Resource Testing Functions (Resource Validation, not Document Processing)
# Document processing functionality moved to lib/process.sh

# ==================== RESOURCE VALIDATION FUNCTIONS ====================
# These test Unstructured.io as a resource (health, connectivity, functionality)

# Quick smoke test - validate Unstructured.io resource health
unstructured_io::test::smoke() {
    local test_script="${UNSTRUCTURED_IO_CLI_DIR}/test/integration-test.sh"
    if [[ -f "$test_script" ]]; then
        bash "$test_script" smoke "$@"
    else
        # Fallback to basic status check
        log::info "Running smoke test..."
        if unstructured_io::status >/dev/null 2>&1; then
            log::success "Smoke test passed - service status OK"
            return 0
        else
            log::error "Smoke test failed - service status check failed"
            return 1
        fi
    fi
}

# Full integration test - validate Unstructured.io end-to-end functionality  
unstructured_io::test::integration() {
    local test_script="${UNSTRUCTURED_IO_CLI_DIR}/test/integration-test.sh"
    if [[ -f "$test_script" ]]; then
        bash "$test_script" integration "$@"
    else
        log::info "Running integration test..."
        
        # Test 1: Check if service is installed
        if ! unstructured_io::container_exists; then
            log::error "Integration test failed - service not installed"
            return 1
        fi
        
        # Test 2: Check if service can start
        if ! unstructured_io::container_running; then
            log::info "Starting service for integration test..."
            if ! unstructured_io::start; then
                log::error "Integration test failed - could not start service"
                return 1
            fi
        fi
        
        # Test 3: Health check
        local max_wait=30
        local wait_time=0
        while [[ $wait_time -lt $max_wait ]]; do
            if unstructured_io::check_health >/dev/null 2>&1; then
                break
            fi
            sleep 2
            ((wait_time += 2))
        done
        
        if [[ $wait_time -ge $max_wait ]]; then
            log::error "Integration test failed - health check timeout"
            return 1
        fi
        
        log::success "Integration test passed"
        return 0
    fi
}

# Unit tests - validate Unstructured.io library functions
unstructured_io::test::unit() {
    log::info "Running unit tests..."
    
    # Test core functions exist
    if ! command -v unstructured_io::status >/dev/null 2>&1; then
        log::error "Unit test failed - unstructured_io::status function missing"
        return 1
    fi
    
    if ! command -v unstructured_io::container_exists >/dev/null 2>&1; then
        log::error "Unit test failed - unstructured_io::container_exists function missing"
        return 1
    fi
    
    if ! command -v unstructured_io::install >/dev/null 2>&1; then
        log::error "Unit test failed - unstructured_io::install function missing"
        return 1
    fi
    
    log::success "Unit tests passed - all required functions available"
    return 0
}

# Run all resource tests
unstructured_io::test::all() {
    log::info "Running all Unstructured.io tests..."
    
    local tests_passed=0
    local tests_total=3
    
    # Run smoke test
    log::info "Running smoke test..."
    if unstructured_io::test::smoke "$@"; then
        ((tests_passed++))
    fi
    
    # Run unit tests  
    log::info "Running unit tests..."
    if unstructured_io::test::unit "$@"; then
        ((tests_passed++))
    fi
    
    # Run integration test
    log::info "Running integration tests..."
    if unstructured_io::test::integration "$@"; then
        ((tests_passed++))
    fi
    
    log::info "Tests completed: $tests_passed/$tests_total passed"
    
    if [[ $tests_passed -eq $tests_total ]]; then
        log::success "All tests passed"
        return 0
    else
        log::error "Some tests failed"
        return 1
    fi
}

# ==================== LEGACY DOCUMENT PROCESSING FUNCTIONS ====================
# DEPRECATED: These functions moved to lib/process.sh
# Kept for backward compatibility with deprecation warnings

# DEPRECATED: Use content execute instead
unstructured_io::test::process() {
    log::warning "DEPRECATED: unstructured_io::test::process is deprecated"
    log::warning "Use 'resource-unstructured-io content execute' instead"
    log::warning "This function will be removed in v3.0 (December 2025)"
    
    # Delegate to new content system
    unstructured_io::process_document "$@"
}

# DEPRECATED: Use content list instead
unstructured_io::test::list() {
    log::warning "DEPRECATED: unstructured_io::test::list is deprecated"
    log::warning "Use 'resource-unstructured-io content list' instead"
    log::warning "This function will be removed in v3.0 (December 2025)"
    
    # Delegate to new content system
    unstructured_io::content::list "$@"
}