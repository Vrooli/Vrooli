#!/usr/bin/env bash
# GridLAB-D Resource - Test Library Functions

set -euo pipefail

# Source configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/config/defaults.sh"

# Test phases
test_smoke() {
    echo "Running smoke tests..."
    local exit_code=0
    
    # Test 1: Health check responds
    echo -n "  Testing health endpoint... "
    if timeout 5 curl -sf "http://localhost:${GRIDLABD_PORT}/health" > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        exit_code=1
    fi
    
    # Test 2: Version endpoint works
    echo -n "  Testing version endpoint... "
    if timeout 5 curl -sf "http://localhost:${GRIDLABD_PORT}/version" > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        exit_code=1
    fi
    
    # Test 3: Examples endpoint works
    echo -n "  Testing examples endpoint... "
    if timeout 5 curl -sf "http://localhost:${GRIDLABD_PORT}/examples" > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        exit_code=1
    fi
    
    # Test 4: Service PID exists
    echo -n "  Testing service PID file... "
    if [ -f "${GRIDLABD_DATA_DIR}/api.pid" ]; then
        echo "✓"
    else
        echo "✗"
        exit_code=1
    fi
    
    if [ $exit_code -eq 0 ]; then
        echo "Smoke tests passed"
    else
        echo "Smoke tests failed"
    fi
    
    return $exit_code
}

test_integration() {
    echo "Running integration tests..."
    local exit_code=0
    
    # Test 1: Can add a model
    echo -n "  Testing model addition... "
    echo "// Test model" > /tmp/test_model.glm
    if "${SCRIPT_DIR}/cli.sh" content add /tmp/test_model.glm > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        exit_code=1
    fi
    rm -f /tmp/test_model.glm
    
    # Test 2: Can list models
    echo -n "  Testing model listing... "
    if "${SCRIPT_DIR}/cli.sh" content list > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        exit_code=1
    fi
    
    # Test 3: Simulate endpoint accepts POST
    echo -n "  Testing simulation endpoint... "
    if timeout 5 curl -sf -X POST "http://localhost:${GRIDLABD_PORT}/simulate" \
        -H "Content-Type: application/json" \
        -d '{"model":"test"}' > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        exit_code=1
    fi
    
    # Test 4: Power flow endpoint works
    echo -n "  Testing power flow endpoint... "
    if timeout 5 curl -sf -X POST "http://localhost:${GRIDLABD_PORT}/powerflow" \
        -H "Content-Type: application/json" \
        -d '{"model":"ieee13"}' > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        exit_code=1
    fi
    
    # Test 5: Restart works
    echo -n "  Testing service restart... "
    # Ensure service is running first
    "${SCRIPT_DIR}/cli.sh" manage start --wait > /dev/null 2>&1 || true
    sleep 1
    
    # Now test restart
    if "${SCRIPT_DIR}/cli.sh" manage stop > /dev/null 2>&1; then
        sleep 2
        if "${SCRIPT_DIR}/cli.sh" manage start --wait > /dev/null 2>&1; then
            if timeout 5 curl -sf "http://localhost:${GRIDLABD_PORT}/health" > /dev/null 2>&1; then
                echo "✓"
            else
                echo "✗"
                exit_code=1
            fi
        else
            echo "✗"
            exit_code=1
        fi
    else
        echo "✗"
        exit_code=1
    fi
    
    if [ $exit_code -eq 0 ]; then
        echo "Integration tests passed"
    else
        echo "Integration tests failed"
    fi
    
    return $exit_code
}

test_unit() {
    echo "Running unit tests..."
    local exit_code=0
    
    # Test 1: Configuration loading
    echo -n "  Testing configuration loading... "
    if [ -n "${GRIDLABD_PORT}" ] && [ -n "${GRIDLABD_DATA_DIR}" ]; then
        echo "✓"
    else
        echo "✗"
        exit_code=1
    fi
    
    # Test 2: Directory creation
    echo -n "  Testing directory creation... "
    if [ -d "${GRIDLABD_DATA_DIR}" ] && [ -d "${GRIDLABD_LOG_DIR}" ]; then
        echo "✓"
    else
        echo "✗"
        exit_code=1
    fi
    
    # Test 3: Runtime JSON validation
    echo -n "  Testing runtime.json structure... "
    if jq -e '.startup_order and .dependencies and .startup_timeout' \
        "${SCRIPT_DIR}/config/runtime.json" > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        exit_code=1
    fi
    
    # Test 4: Schema JSON validation
    echo -n "  Testing schema.json structure... "
    if jq -e '.properties.port and .properties.data_dir' \
        "${SCRIPT_DIR}/config/schema.json" > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        exit_code=1
    fi
    
    # Test 5: CLI help works
    echo -n "  Testing CLI help command... "
    if "${SCRIPT_DIR}/cli.sh" help > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        exit_code=1
    fi
    
    if [ $exit_code -eq 0 ]; then
        echo "Unit tests passed"
    else
        echo "Unit tests failed"
    fi
    
    return $exit_code
}

test_all() {
    echo "Running all test phases..."
    echo ""
    
    local total_exit_code=0
    
    # Run smoke tests
    test_smoke || total_exit_code=$?
    echo ""
    
    # Run integration tests
    test_integration || total_exit_code=$?
    echo ""
    
    # Run unit tests
    test_unit || total_exit_code=$?
    echo ""
    
    if [ $total_exit_code -eq 0 ]; then
        echo "All tests passed successfully"
    else
        echo "Some tests failed"
    fi
    
    return $total_exit_code
}