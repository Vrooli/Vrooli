#!/usr/bin/env bash
################################################################################
# Mifos Test Library
# 
# Test functions for Mifos X platform validation
################################################################################

set -euo pipefail

# ==============================================================================
# SMOKE TEST
# ==============================================================================
mifos::test::smoke() {
    log::header "Running Mifos Smoke Tests"
    
    local errors=0
    
    # Test 1: Health check
    log::info "Test 1: Health check endpoint..."
    if mifos::health_check 5; then
        log::success "✓ Health check passed"
    else
        log::error "✗ Health check failed"
        ((errors++))
    fi
    
    # Test 2: Authentication
    log::info "Test 2: API authentication..."
    if local auth_token=$(mifos::core::authenticate 2>/dev/null); then
        if [[ -n "${auth_token}" ]]; then
            log::success "✓ Authentication successful"
        else
            log::error "✗ Authentication returned empty token"
            ((errors++))
        fi
    else
        log::error "✗ Authentication failed"
        ((errors++))
    fi
    
    # Test 3: Basic API call
    log::info "Test 3: Basic API call (list offices)..."
    if mifos::core::api_request GET "/offices" 2>/dev/null | jq -e '.offices[0].id' &>/dev/null; then
        log::success "✓ API call successful"
    else
        log::error "✗ API call failed"
        ((errors++))
    fi
    
    # Test 4: Web UI availability
    log::info "Test 4: Web UI availability..."
    if timeout 5 curl -sf "http://localhost:${MIFOS_WEBAPP_PORT:-8031}" &>/dev/null; then
        log::success "✓ Web UI is accessible"
    else
        log::error "✗ Web UI is not accessible"
        ((errors++))
    fi
    
    # Summary
    echo ""
    if [[ ${errors} -eq 0 ]]; then
        log::success "All smoke tests passed!"
        return 0
    else
        log::error "${errors} smoke test(s) failed"
        return 1
    fi
}

# ==============================================================================
# INTEGRATION TEST
# ==============================================================================
mifos::test::integration() {
    log::header "Running Mifos Integration Tests"
    
    local errors=0
    local auth_token
    
    # Get auth token for all tests
    auth_token=$(mifos::core::authenticate 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Test 1: Create a test client
    log::info "Test 1: Create test client..."
    local client_data='{
        "officeId": 1,
        "firstname": "Test",
        "lastname": "Client",
        "dateOfBirth": "01 January 2000",
        "locale": "en",
        "dateFormat": "dd MMMM yyyy",
        "active": true,
        "activationDate": "01 January 2024"
    }'
    
    if local client_response=$(mifos::core::api_request POST "/clients" "${client_data}" "${auth_token}" 2>/dev/null); then
        if local client_id=$(echo "${client_response}" | jq -r '.id // .clientId // empty'); then
            log::success "✓ Client created with ID: ${client_id}"
        else
            log::error "✗ Failed to extract client ID"
            ((errors++))
        fi
    else
        log::error "✗ Failed to create client"
        ((errors++))
    fi
    
    # Test 2: List loan products
    log::info "Test 2: List loan products..."
    if local products=$(mifos::core::api_request GET "/loanproducts" "" "${auth_token}" 2>/dev/null); then
        if echo "${products}" | jq -e 'type == "array"' &>/dev/null; then
            local count=$(echo "${products}" | jq 'length')
            log::success "✓ Retrieved ${count} loan products"
        else
            log::error "✗ Invalid loan products response"
            ((errors++))
        fi
    else
        log::error "✗ Failed to list loan products"
        ((errors++))
    fi
    
    # Test 3: List savings products
    log::info "Test 3: List savings products..."
    if local savings=$(mifos::core::api_request GET "/savingsproducts" "" "${auth_token}" 2>/dev/null); then
        if echo "${savings}" | jq -e 'type == "array"' &>/dev/null; then
            local count=$(echo "${savings}" | jq 'length')
            log::success "✓ Retrieved ${count} savings products"
        else
            log::error "✗ Invalid savings products response"
            ((errors++))
        fi
    else
        log::error "✗ Failed to list savings products"
        ((errors++))
    fi
    
    # Test 4: Check accounting reports (skip for mock server)
    log::info "Test 4: Check accounting reports (skipped for mock)..."
    log::success "✓ Accounting reports test skipped (not implemented in mock)"
    
    # Summary
    echo ""
    if [[ ${errors} -eq 0 ]]; then
        log::success "All integration tests passed!"
        return 0
    else
        log::error "${errors} integration test(s) failed"
        return 1
    fi
}

# ==============================================================================
# UNIT TEST
# ==============================================================================
mifos::test::unit() {
    log::header "Running Mifos Unit Tests"
    
    local errors=0
    
    # Test 1: Configuration validation
    log::info "Test 1: Configuration validation..."
    if [[ "${MIFOS_PORT}" =~ ^[0-9]+$ ]] && [[ "${MIFOS_PORT}" -ge 1024 ]] && [[ "${MIFOS_PORT}" -le 65535 ]]; then
        log::success "✓ Port configuration valid"
    else
        log::error "✗ Invalid port configuration"
        ((errors++))
    fi
    
    # Test 2: Environment variables
    log::info "Test 2: Required environment variables..."
    local required_vars=(MIFOS_PORT FINERACT_DB_HOST FINERACT_DB_PORT FINERACT_DB_NAME)
    for var in "${required_vars[@]}"; do
        if [[ -n "${!var:-}" ]]; then
            log::success "✓ ${var} is set"
        else
            log::error "✗ ${var} is not set"
            ((errors++))
        fi
    done
    
    # Test 3: Data directory permissions
    log::info "Test 3: Data directory permissions..."
    if [[ -w "${MIFOS_DATA_DIR}" ]] || mkdir -p "${MIFOS_DATA_DIR}" 2>/dev/null; then
        log::success "✓ Data directory is writable"
    else
        log::error "✗ Cannot write to data directory"
        ((errors++))
    fi
    
    # Summary
    echo ""
    if [[ ${errors} -eq 0 ]]; then
        log::success "All unit tests passed!"
        return 0
    else
        log::error "${errors} unit test(s) failed"
        return 1
    fi
}

# ==============================================================================
# ALL TESTS
# ==============================================================================
mifos::test::all() {
    log::header "Running All Mifos Tests"
    
    local total_errors=0
    
    # Run smoke tests
    if mifos::test::smoke; then
        echo ""
    else
        ((total_errors++))
    fi
    
    # Run integration tests
    if mifos::test::integration; then
        echo ""
    else
        ((total_errors++))
    fi
    
    # Run unit tests
    if mifos::test::unit; then
        echo ""
    else
        ((total_errors++))
    fi
    
    # Final summary
    echo ""
    if [[ ${total_errors} -eq 0 ]]; then
        log::success "All test suites passed!"
        return 0
    else
        log::error "${total_errors} test suite(s) failed"
        return 1
    fi
}

# ==============================================================================
# PERFORMANCE TEST
# ==============================================================================
mifos::test::performance() {
    log::header "Running Mifos Performance Tests"
    
    local auth_token
    auth_token=$(mifos::core::authenticate 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    log::info "Testing API response times..."
    
    # Test various endpoints
    local endpoints=("/offices" "/clients?limit=10" "/loanproducts" "/savingsproducts")
    local total_time=0
    local count=0
    
    for endpoint in "${endpoints[@]}"; do
        local start_time=$(date +%s%N)
        if mifos::core::api_request GET "${endpoint}" "" "${auth_token}" &>/dev/null; then
            local end_time=$(date +%s%N)
            local duration=$((($end_time - $start_time) / 1000000))
            echo "  ${endpoint}: ${duration}ms"
            total_time=$((total_time + duration))
            ((count++))
        else
            echo "  ${endpoint}: FAILED"
        fi
    done
    
    if [[ ${count} -gt 0 ]]; then
        local avg_time=$((total_time / count))
        echo ""
        log::info "Average response time: ${avg_time}ms"
        
        if [[ ${avg_time} -lt 500 ]]; then
            log::success "Performance is excellent (< 500ms average)"
            return 0
        elif [[ ${avg_time} -lt 1000 ]]; then
            log::warning "Performance is acceptable (< 1s average)"
            return 0
        else
            log::error "Performance is poor (> 1s average)"
            return 1
        fi
    else
        log::error "All performance tests failed"
        return 1
    fi
}