#!/usr/bin/env bash
################################################################################
# OWASP ZAP Resource - Test Library Functions
################################################################################

# Source core functions
source "${ZAP_CLI_DIR}/lib/core.sh"

################################################################################
# Test Command Handler
################################################################################

zap::test() {
    local test_type="${1:-all}"
    
    case "${test_type}" in
        smoke)
            zap::test_smoke
            ;;
        integration)
            zap::test_integration
            ;;
        unit)
            zap::test_unit
            ;;
        all)
            zap::test_all
            ;;
        *)
            log_error "Unknown test type: ${test_type}"
            echo "Available test types: smoke, integration, unit, all"
            return 1
            ;;
    esac
}

################################################################################
# Test Implementations
################################################################################

zap::test_smoke() {
    log_info "Running smoke tests..."
    
    # Test 1: Check if ZAP is running
    log_info "Test 1: Checking if ZAP is running..."
    if zap::is_running; then
        log_success "✓ ZAP container is running"
    else
        log_warn "ZAP is not running, attempting to start..."
        if zap::start true; then
            log_success "✓ ZAP started successfully"
        else
            log_error "✗ Failed to start ZAP"
            return 1
        fi
    fi
    
    # Test 2: Health check
    log_info "Test 2: Performing health check..."
    if zap::health_check; then
        log_success "✓ Health check passed"
    else
        log_error "✗ Health check failed"
        return 1
    fi
    
    # Test 3: API accessibility
    log_info "Test 3: Testing API accessibility..."
    local version=$(zap::get_version)
    if [[ -n "${version}" ]]; then
        log_success "✓ API accessible (ZAP version: ${version})"
    else
        log_error "✗ API not accessible"
        return 1
    fi
    
    log_success "All smoke tests passed!"
    return 0
}

zap::test_integration() {
    log_info "Running integration tests..."
    
    # Ensure ZAP is running
    if ! zap::is_running; then
        log_info "Starting ZAP for integration tests..."
        zap::start true || return 1
    fi
    
    # Test 1: Add a target URL
    log_info "Test 1: Adding target URL..."
    if zap::content_add "http://example.com"; then
        log_success "✓ Target added successfully"
    else
        log_error "✗ Failed to add target"
        return 1
    fi
    
    # Test 2: Start a spider scan
    log_info "Test 2: Starting spider scan..."
    if zap::content_execute "http://example.com" "spider"; then
        log_success "✓ Spider scan started"
    else
        log_error "✗ Failed to start spider scan"
        return 1
    fi
    
    # Test 3: Retrieve alerts
    log_info "Test 3: Retrieving scan results..."
    local alerts=$(zap::content_list json)
    if [[ -n "${alerts}" ]]; then
        log_success "✓ Retrieved scan results"
    else
        log_warn "⚠ No alerts found (this may be normal)"
    fi
    
    # Test 4: Generate report
    log_info "Test 4: Generating HTML report..."
    local report=$(zap::content_get html)
    if [[ "${report}" == *"<html"* ]]; then
        log_success "✓ HTML report generated"
    else
        log_error "✗ Failed to generate HTML report"
        return 1
    fi
    
    log_success "All integration tests passed!"
    return 0
}

zap::test_unit() {
    log_info "Running unit tests..."
    
    # Test 1: Configuration loading
    log_info "Test 1: Testing configuration..."
    if [[ -n "${ZAP_API_PORT}" ]] && [[ "${ZAP_API_PORT}" -eq 8180 ]]; then
        log_success "✓ Configuration loaded correctly"
    else
        log_error "✗ Configuration not loaded"
        return 1
    fi
    
    # Test 2: Directory creation
    log_info "Test 2: Testing directory creation..."
    local test_dir="/tmp/zap-test-$$"
    ZAP_DATA_DIR="${test_dir}" 
    mkdir -p "${ZAP_DATA_DIR}/sessions" "${ZAP_DATA_DIR}/reports"
    if [[ -d "${test_dir}/sessions" ]] && [[ -d "${test_dir}/reports" ]]; then
        log_success "✓ Directories created successfully"
        rm -rf "${test_dir}"
    else
        log_error "✗ Failed to create directories"
        rm -rf "${test_dir}"
        return 1
    fi
    
    # Test 3: API key generation
    log_info "Test 3: Testing API key generation..."
    local test_key=$(openssl rand -hex 32 2>/dev/null || cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1)
    if [[ ${#test_key} -eq 32 ]]; then
        log_success "✓ API key generation works"
    else
        log_error "✗ API key generation failed"
        return 1
    fi
    
    log_success "All unit tests passed!"
    return 0
}

zap::test_all() {
    log_info "Running all tests..."
    
    local failed=0
    
    # Run unit tests
    if zap::test_unit; then
        log_success "Unit tests: PASSED"
    else
        log_error "Unit tests: FAILED"
        ((failed++))
    fi
    
    # Run smoke tests
    if zap::test_smoke; then
        log_success "Smoke tests: PASSED"
    else
        log_error "Smoke tests: FAILED"
        ((failed++))
    fi
    
    # Run integration tests
    if zap::test_integration; then
        log_success "Integration tests: PASSED"
    else
        log_error "Integration tests: FAILED"
        ((failed++))
    fi
    
    if [[ ${failed} -eq 0 ]]; then
        log_success "All test suites passed!"
        return 0
    else
        log_error "${failed} test suite(s) failed"
        return 1
    fi
}

################################################################################
# Export Functions
################################################################################

export -f zap::test
export -f zap::test_smoke
export -f zap::test_integration
export -f zap::test_unit
export -f zap::test_all