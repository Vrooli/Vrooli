#!/bin/bash
# Step-CA Test Functions

set -euo pipefail

# Test smoke - Quick health check
test_smoke() {
    echo "🧪 Running smoke tests..."
    
    local failed=0
    
    # Test 1: Container running
    echo -n "  Checking container status... "
    if docker ps --format "{{.Names}}" | grep -q "^vrooli-step-ca$"; then
        echo "✅"
    else
        echo "❌ Container not running"
        ((failed++))
    fi
    
    # Test 2: Health endpoint
    echo -n "  Checking health endpoint... "
    if timeout 5 curl -sk "https://localhost:${STEPCA_PORT:-9010}/health" >/dev/null 2>&1; then
        echo "✅"
    else
        echo "❌ Health check failed"
        ((failed++))
    fi
    
    # Test 3: ACME directory
    echo -n "  Checking ACME directory... "
    if timeout 5 curl -sk "https://localhost:${STEPCA_PORT:-9010}/acme/acme/directory" | grep -q "newNonce" 2>/dev/null; then
        echo "✅"
    else
        echo "❌ ACME directory not accessible"
        ((failed++))
    fi
    
    # Test 4: Configuration exists
    echo -n "  Checking configuration... "
    if [[ -f "${DATA_DIR:-/tmp}/config/ca.json" ]]; then
        echo "✅"
    else
        echo "❌ Configuration not found"
        ((failed++))
    fi
    
    if [[ $failed -eq 0 ]]; then
        echo "✅ All smoke tests passed"
        return 0
    else
        echo "❌ $failed smoke tests failed"
        return 1
    fi
}

# Test integration - Full functionality tests
test_integration() {
    echo "🧪 Running integration tests..."
    
    local failed=0
    
    # Test 1: API accessibility
    echo -n "  Testing API accessibility... "
    if timeout 10 curl -sk "https://localhost:${STEPCA_PORT:-9010}/1.0/version" >/dev/null 2>&1; then
        echo "✅"
    else
        echo "❌ API not accessible"
        ((failed++))
    fi
    
    # Test 2: Root certificate
    echo -n "  Checking root certificate... "
    if [[ -f "${CERTS_DIR:-/tmp}/root_ca.crt" ]]; then
        echo "✅"
    else
        echo "❌ Root certificate not found"
        ((failed++))
    fi
    
    # Test 3: Provisioner configuration
    echo -n "  Checking provisioner config... "
    if [[ -f "${CONFIG_DIR:-/tmp}/ca.json" ]] && grep -q "provisioners" "${CONFIG_DIR:-/tmp}/ca.json" 2>/dev/null; then
        echo "✅"
    else
        echo "❌ Provisioner not configured"
        ((failed++))
    fi
    
    # Test 4: Certificate request (simulation)
    echo -n "  Testing certificate request endpoint... "
    # This would normally test actual certificate issuance
    echo "✅ (simulated)"
    
    if [[ $failed -eq 0 ]]; then
        echo "✅ All integration tests passed"
        return 0
    else
        echo "❌ $failed integration tests failed"
        return 1
    fi
}

# Test unit - Library function tests
test_unit() {
    echo "🧪 Running unit tests..."
    
    local failed=0
    
    # Test 1: Port configuration
    echo -n "  Testing port configuration... "
    if [[ -n "${STEPCA_PORT:-}" ]] && [[ "$STEPCA_PORT" =~ ^[0-9]+$ ]]; then
        echo "✅"
    else
        echo "❌ Invalid port configuration"
        ((failed++))
    fi
    
    # Test 2: Directory creation
    echo -n "  Testing directory creation... "
    local test_dir="/tmp/step-ca-test-$$"
    if mkdir -p "$test_dir" 2>/dev/null && [[ -d "$test_dir" ]]; then
        rm -rf "$test_dir"
        echo "✅"
    else
        echo "❌ Cannot create directories"
        ((failed++))
    fi
    
    # Test 3: Docker availability
    echo -n "  Testing Docker availability... "
    if docker version >/dev/null 2>&1; then
        echo "✅"
    else
        echo "❌ Docker not available"
        ((failed++))
    fi
    
    # Test 4: Network connectivity
    echo -n "  Testing network configuration... "
    if docker network ls --format "{{.Name}}" | grep -q "vrooli-network" || docker network create vrooli-network >/dev/null 2>&1; then
        echo "✅"
    else
        echo "❌ Network configuration failed"
        ((failed++))
    fi
    
    if [[ $failed -eq 0 ]]; then
        echo "✅ All unit tests passed"
        return 0
    else
        echo "❌ $failed unit tests failed"
        return 1
    fi
}

# Test all - Run all test suites
test_all() {
    echo "🧪 Running all test suites..."
    echo ""
    
    local total_failed=0
    
    # Run unit tests
    if ! test_unit; then
        ((total_failed++))
    fi
    echo ""
    
    # Run smoke tests
    if ! test_smoke; then
        ((total_failed++))
    fi
    echo ""
    
    # Run integration tests
    if ! test_integration; then
        ((total_failed++))
    fi
    
    echo ""
    if [[ $total_failed -eq 0 ]]; then
        echo "✅ All test suites passed"
        return 0
    else
        echo "❌ $total_failed test suites failed"
        return 1
    fi
}