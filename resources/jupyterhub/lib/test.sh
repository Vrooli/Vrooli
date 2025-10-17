#!/bin/bash
# JupyterHub Test Library Functions

set -euo pipefail

# Source core library
source "${LIB_DIR}/core.sh"

# Test smoke - Quick health validation (<30s)
test_smoke() {
    log_info "Running smoke tests..."
    
    local failed=0
    
    # Test 1: Service is running
    echo -n "Testing service status... "
    if [[ "$(get_service_status)" == "running" ]]; then
        echo "✅ PASS"
    else
        echo "❌ FAIL - Service not running"
        ((failed++))
    fi
    
    # Test 2: Health check responds
    echo -n "Testing health endpoint... "
    if timeout 5 curl -sf "http://localhost:${JUPYTERHUB_PORT}/hub/health" &>/dev/null; then
        echo "✅ PASS"
    else
        echo "❌ FAIL - Health check failed"
        ((failed++))
    fi
    
    # Test 3: Hub API accessible
    echo -n "Testing hub API... "
    if timeout 5 curl -sf "http://localhost:${JUPYTERHUB_API_PORT:-8001}/hub/api" &>/dev/null; then
        echo "✅ PASS"
    else
        echo "❌ FAIL - API not accessible"
        ((failed++))
    fi
    
    # Test 4: Proxy running
    echo -n "Testing proxy... "
    if timeout 5 curl -sf "http://localhost:${JUPYTERHUB_PROXY_PORT:-8081}/api/routes" &>/dev/null; then
        echo "✅ PASS"
    else
        echo "⚠️  WARN - Proxy API not accessible (may be normal)"
    fi
    
    # Test 5: Data directories exist
    echo -n "Testing data directories... "
    if [[ -d "${JUPYTERHUB_DATA_DIR}" ]]; then
        echo "✅ PASS"
    else
        echo "❌ FAIL - Data directory missing"
        ((failed++))
    fi
    
    if [[ $failed -eq 0 ]]; then
        log_info "✅ All smoke tests passed"
        return 0
    else
        log_error "$failed smoke tests failed"
        return 1
    fi
}

# Test integration - End-to-end functionality
test_integration() {
    log_info "Running integration tests..."
    
    local failed=0
    
    # Test 1: Hub web interface responds
    echo -n "Testing web interface... "
    if timeout 10 curl -sf "http://localhost:${JUPYTERHUB_PORT}" | grep -q "JupyterHub" &>/dev/null; then
        echo "✅ PASS"
    else
        echo "❌ FAIL - Web interface not responding"
        ((failed++))
    fi
    
    # Test 2: Login page accessible
    echo -n "Testing login page... "
    if timeout 10 curl -sf "http://localhost:${JUPYTERHUB_PORT}/hub/login" &>/dev/null; then
        echo "✅ PASS"
    else
        echo "❌ FAIL - Login page not accessible"
        ((failed++))
    fi
    
    # Test 3: Static assets served
    echo -n "Testing static assets... "
    if timeout 5 curl -sf "http://localhost:${JUPYTERHUB_PORT}/hub/static/css/style.min.css" &>/dev/null; then
        echo "✅ PASS"
    else
        echo "⚠️  WARN - Static assets not found (may be normal in minimal setup)"
    fi
    
    # Test 4: Database connectivity
    echo -n "Testing database... "
    if docker exec ${JUPYTERHUB_CONTAINER_NAME} test -f /data/jupyterhub.sqlite 2>/dev/null; then
        echo "✅ PASS"
    else
        echo "⚠️  WARN - Database file not found (may be using PostgreSQL)"
    fi
    
    # Test 5: Docker connectivity
    echo -n "Testing Docker access... "
    if docker exec ${JUPYTERHUB_CONTAINER_NAME} docker version &>/dev/null 2>&1; then
        echo "✅ PASS"
    else
        echo "⚠️  WARN - Docker not accessible from container"
    fi
    
    if [[ $failed -eq 0 ]]; then
        log_info "✅ All integration tests passed"
        return 0
    else
        log_error "$failed integration tests failed"
        return 1
    fi
}

# Test unit - Library function tests
test_unit() {
    log_info "Running unit tests..."
    
    local failed=0
    
    # Test 1: Configuration loading
    echo -n "Testing configuration... "
    if [[ -n "${JUPYTERHUB_PORT}" ]]; then
        echo "✅ PASS"
    else
        echo "❌ FAIL - Configuration not loaded"
        ((failed++))
    fi
    
    # Test 2: Logging functions
    echo -n "Testing logging... "
    if log_info "Test" &>/dev/null && log_error "Test" &>/dev/null; then
        echo "✅ PASS"
    else
        echo "❌ FAIL - Logging functions failed"
        ((failed++))
    fi
    
    # Test 3: Status detection
    echo -n "Testing status detection... "
    local status=$(get_service_status)
    if [[ "$status" == "running" ]] || [[ "$status" == "stopped" ]]; then
        echo "✅ PASS"
    else
        echo "❌ FAIL - Invalid status: $status"
        ((failed++))
    fi
    
    # Test 4: Token generation
    echo -n "Testing token generation... "
    if command -v openssl &>/dev/null; then
        local token=$(openssl rand -hex 32)
        if [[ ${#token} -eq 64 ]]; then
            echo "✅ PASS"
        else
            echo "❌ FAIL - Invalid token length"
            ((failed++))
        fi
    else
        echo "⚠️  SKIP - OpenSSL not available"
    fi
    
    if [[ $failed -eq 0 ]]; then
        log_info "✅ All unit tests passed"
        return 0
    else
        log_error "$failed unit tests failed"
        return 1
    fi
}

# Test all - Run all test suites
test_all() {
    log_info "Running all test suites..."
    
    local failed=0
    
    # Run smoke tests
    echo ""
    echo "━━━ SMOKE TESTS ━━━"
    if ! test_smoke; then
        ((failed++))
    fi
    
    # Run unit tests
    echo ""
    echo "━━━ UNIT TESTS ━━━"
    if ! test_unit; then
        ((failed++))
    fi
    
    # Run integration tests
    echo ""
    echo "━━━ INTEGRATION TESTS ━━━"
    if ! test_integration; then
        ((failed++))
    fi
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    if [[ $failed -eq 0 ]]; then
        log_info "✅ All test suites passed"
        return 0
    else
        log_error "$failed test suites failed"
        return 1
    fi
}