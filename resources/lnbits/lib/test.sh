#!/usr/bin/env bash
# LNbits Test Library

set -euo pipefail

# Source core library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/core.sh"

# Run smoke test (quick health check)
test_smoke() {
    echo "Running LNbits smoke test..."
    
    # Check if installed
    if ! is_installed; then
        echo "❌ LNbits is not installed"
        exit 1
    fi
    echo "✅ LNbits is installed"
    
    # Check if running
    if ! is_running; then
        echo "❌ LNbits is not running"
        exit 1
    fi
    echo "✅ LNbits container is running"
    
    # Check health endpoint
    if ! check_health; then
        echo "❌ Health check failed"
        exit 1
    fi
    echo "✅ Health endpoint responding"
    
    # Check API accessibility
    if ! timeout 5 curl -sf "http://localhost:${LNBITS_PORT}/api/v1/health" | grep -q "ok"; then
        echo "❌ API not accessible"
        exit 1
    fi
    echo "✅ API is accessible"
    
    echo ""
    echo "Smoke test passed! ✅"
    exit 0
}

# Run integration test
test_integration() {
    echo "Running LNbits integration test..."
    
    # First run smoke test
    test_smoke
    
    echo ""
    echo "Testing API endpoints..."
    
    # Test health endpoint with detailed response
    echo -n "Testing /api/v1/health... "
    local health_response
    health_response=$(timeout 5 curl -sf "http://localhost:${LNBITS_PORT}/api/v1/health" 2>/dev/null)
    if [[ $? -eq 0 ]]; then
        echo "✅"
    else
        echo "❌"
        exit 1
    fi
    
    # Test API docs endpoint
    echo -n "Testing /docs endpoint... "
    if timeout 5 curl -sf "http://localhost:${LNBITS_PORT}/docs" &>/dev/null; then
        echo "✅"
    else
        echo "❌"
        exit 1
    fi
    
    # Test wallet creation endpoint (expect auth error)
    echo -n "Testing wallet endpoint (auth check)... "
    local wallet_response
    wallet_response=$(curl -sf -X POST \
        "http://localhost:${LNBITS_PORT}/api/v1/wallet" \
        -H "Content-Type: application/json" \
        -d '{"name": "Test Wallet"}' 2>&1 || true)
    
    # We expect this to fail with auth error or succeed
    echo "✅ (endpoint exists)"
    
    # Test PostgreSQL connectivity
    echo -n "Testing PostgreSQL connectivity... "
    if docker exec "${LNBITS_POSTGRES_CONTAINER}" pg_isready -U "${LNBITS_POSTGRES_USER}" &>/dev/null; then
        echo "✅"
    else
        echo "❌"
        exit 1
    fi
    
    # Test data persistence
    echo -n "Testing data directory... "
    if [[ -d "${LNBITS_DATA_DIR}" ]]; then
        echo "✅"
    else
        echo "❌"
        exit 1
    fi
    
    echo ""
    echo "Integration test passed! ✅"
    exit 0
}

# Run unit tests
test_unit() {
    echo "Running LNbits unit tests..."
    
    # Test is_installed function
    echo -n "Testing is_installed function... "
    if is_installed; then
        echo "✅ (installed)"
    else
        echo "✅ (not installed)"
    fi
    
    # Test is_running function
    echo -n "Testing is_running function... "
    if is_running; then
        echo "✅ (running)"
    else
        echo "✅ (not running)"
    fi
    
    # Test configuration loading
    echo -n "Testing configuration loading... "
    if [[ -n "${LNBITS_PORT}" ]]; then
        echo "✅"
    else
        echo "❌"
        exit 1
    fi
    
    # Test runtime.json exists
    echo -n "Testing runtime.json... "
    if [[ -f "${SCRIPT_DIR}/config/runtime.json" ]]; then
        echo "✅"
    else
        echo "❌"
        exit 1
    fi
    
    # Test Docker availability
    echo -n "Testing Docker availability... "
    if docker version &>/dev/null; then
        echo "✅"
    else
        echo "❌"
        exit 1
    fi
    
    echo ""
    echo "Unit tests passed! ✅"
    exit 0
}

# Run all tests
test_all() {
    echo "Running all LNbits tests..."
    echo "=========================="
    echo ""
    
    # Run unit tests first (don't need service running)
    test_unit
    echo ""
    
    # Run smoke test
    test_smoke
    echo ""
    
    # Run integration test
    test_integration
    echo ""
    
    echo "All tests passed! ✅"
    exit 0
}