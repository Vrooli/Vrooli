#!/bin/bash

# OctoPrint Test Library
# Test implementations for OctoPrint resource

set -euo pipefail

# Smoke test - Quick health check
test_smoke() {
    echo "Running OctoPrint smoke test..."
    local start_time=$(date +%s)
    
    # Check if service is responsive (web interface)
    echo -n "Testing health endpoint... "
    if timeout 5 curl -sf -o /dev/null "http://localhost:${OCTOPRINT_PORT}/" 2>/dev/null; then
        echo "✓ PASS"
    else
        echo "✗ FAIL - Service not responding"
        return 1
    fi
    
    # Check web interface is accessible
    echo -n "Testing web interface accessibility... "
    local http_code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${OCTOPRINT_PORT}/" 2>/dev/null)
    if [[ "${http_code}" == "200" ]]; then
        echo "✓ PASS"
    else
        echo "✗ FAIL - Web interface not accessible (HTTP ${http_code})"
        return 1
    fi
    
    # Check WebSocket endpoint (if enabled)
    if [[ "${OCTOPRINT_WEBSOCKET_ENABLED}" == "true" ]]; then
        echo -n "Testing WebSocket endpoint... "
        if timeout 5 curl -sf -o /dev/null "http://localhost:${OCTOPRINT_PORT}/sockjs/info" 2>/dev/null; then
            echo "✓ PASS"
        else
            echo "⚠ WARN - WebSocket endpoint not responding"
        fi
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "Smoke test completed in ${duration}s"
    echo "Result: PASS"
    return 0
}

# Integration test - Full functionality
test_integration() {
    echo "Running OctoPrint integration test..."
    local start_time=$(date +%s)
    local test_failed=false
    
    # Test 1: Service lifecycle
    echo ""
    echo "Test 1: Service Lifecycle"
    echo "-------------------------"
    
    # Stop if running
    echo -n "Stopping service... "
    octoprint_stop &>/dev/null
    echo "✓"
    
    # Start service
    echo -n "Starting service... "
    if OCTOPRINT_VIRTUAL_PRINTER=true octoprint_start --wait &>/dev/null; then
        echo "✓ PASS"
    else
        echo "✗ FAIL"
        test_failed=true
    fi
    
    # Test 2: API Operations
    echo ""
    echo "Test 2: API Operations"
    echo "----------------------"
    
    # Get version
    echo -n "Getting version via API... "
    local version=$(timeout 5 curl -sf "http://localhost:${OCTOPRINT_PORT}/api/version" 2>/dev/null || echo "")
    if [[ -n "${version}" ]]; then
        echo "✓ PASS"
    else
        echo "✗ FAIL"
        test_failed=true
    fi
    
    # Test printer state (with virtual printer)
    echo -n "Getting printer state... "
    local api_key="${OCTOPRINT_API_KEY}"
    if [[ -f "${OCTOPRINT_CONFIG_DIR}/api_key" ]]; then
        api_key=$(cat "${OCTOPRINT_CONFIG_DIR}/api_key")
    fi
    
    if [[ -n "${api_key}" ]] && [[ "${api_key}" != "auto" ]]; then
        local printer_state=$(timeout 5 curl -sf -H "X-Api-Key: ${api_key}" \
            "http://localhost:${OCTOPRINT_PORT}/api/printer" 2>/dev/null || echo "")
        if [[ -n "${printer_state}" ]]; then
            echo "✓ PASS"
        else
            echo "⚠ WARN - Printer state not available"
        fi
    else
        echo "⚠ SKIP - API key not configured"
    fi
    
    # Test 3: File Management
    echo ""
    echo "Test 3: File Management"
    echo "-----------------------"
    
    # Create test G-code file
    local test_file="/tmp/test_octoprint.gcode"
    cat > "${test_file}" << 'EOF'
; Test G-code file
G28 ; Home all axes
G1 X10 Y10 Z0.3 F5000.0 ; Move to start position
G1 X10 Y20 Z0.3 F1500.0 E10 ; Draw line
G1 X10 Y30 Z0.3 F1500.0 E20 ; Draw line
M84 ; Disable motors
EOF
    
    echo -n "Uploading test G-code... "
    if content_add "${test_file}" &>/dev/null; then
        echo "✓ PASS"
    else
        echo "✗ FAIL"
        test_failed=true
    fi
    
    echo -n "Listing uploaded files... "
    local files=$(content_list 2>/dev/null | grep -c "test_octoprint.gcode" || echo "0")
    if [[ "${files}" -gt 0 ]]; then
        echo "✓ PASS"
    else
        echo "✗ FAIL"
        test_failed=true
    fi
    
    echo -n "Removing test file... "
    if content_remove "test_octoprint.gcode" &>/dev/null; then
        echo "✓ PASS"
    else
        echo "⚠ WARN - Could not remove test file"
    fi
    
    # Clean up
    rm -f "${test_file}"
    
    # Test 4: Performance
    echo ""
    echo "Test 4: Performance"
    echo "-------------------"
    
    echo -n "API response time... "
    local response_start=$(date +%s%N)
    timeout 5 curl -sf "http://localhost:${OCTOPRINT_PORT}/api/version" &>/dev/null
    local response_end=$(date +%s%N)
    local response_time=$(( (response_end - response_start) / 1000000 ))
    
    if [[ ${response_time} -lt 200 ]]; then
        echo "✓ PASS (${response_time}ms)"
    else
        echo "⚠ WARN (${response_time}ms > 200ms)"
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "Integration test completed in ${duration}s"
    
    if [[ "${test_failed}" == true ]]; then
        echo "Result: FAIL"
        return 1
    else
        echo "Result: PASS"
        return 0
    fi
}

# Unit test - Library functions
test_unit() {
    echo "Running OctoPrint unit tests..."
    local start_time=$(date +%s)
    local test_failed=false
    
    # Test 1: Configuration loading
    echo ""
    echo "Test 1: Configuration"
    echo "--------------------"
    
    echo -n "Checking port configuration... "
    if [[ "${OCTOPRINT_PORT}" == "8197" ]]; then
        echo "✓ PASS"
    else
        echo "✗ FAIL - Expected 8197, got ${OCTOPRINT_PORT}"
        test_failed=true
    fi
    
    echo -n "Checking data directory... "
    if [[ -n "${OCTOPRINT_DATA_DIR}" ]]; then
        echo "✓ PASS"
    else
        echo "✗ FAIL - Data directory not set"
        test_failed=true
    fi
    
    # Test 2: Helper functions
    echo ""
    echo "Test 2: Helper Functions"
    echo "------------------------"
    
    echo -n "Testing directory creation... "
    local test_dir="/tmp/octoprint_test_$$"
    mkdir -p "${test_dir}"
    if [[ -d "${test_dir}" ]]; then
        echo "✓ PASS"
        rm -rf "${test_dir}"
    else
        echo "✗ FAIL"
        test_failed=true
    fi
    
    echo -n "Testing API key generation... "
    local test_key=$(openssl rand -hex 32 2>/dev/null || date +%s | sha256sum | cut -d' ' -f1)
    if [[ ${#test_key} -eq 64 ]]; then
        echo "✓ PASS"
    else
        echo "✗ FAIL - Invalid key length"
        test_failed=true
    fi
    
    # Test 3: Docker detection
    echo ""
    echo "Test 3: Environment Detection"
    echo "-----------------------------"
    
    echo -n "Checking Docker availability... "
    if command -v docker &> /dev/null; then
        echo "✓ Docker available"
    else
        echo "⚠ Docker not available (native mode)"
    fi
    
    echo -n "Checking Python availability... "
    if command -v python3 &> /dev/null; then
        local python_version=$(python3 --version 2>&1 | cut -d' ' -f2)
        echo "✓ Python ${python_version}"
    else
        echo "⚠ Python not available"
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "Unit test completed in ${duration}s"
    
    if [[ "${test_failed}" == true ]]; then
        echo "Result: FAIL"
        return 1
    else
        echo "Result: PASS"
        return 0
    fi
}

# Run all tests
test_all() {
    echo "Running all OctoPrint tests..."
    echo "=============================="
    local all_passed=true
    
    # Run smoke test
    echo ""
    if test_smoke; then
        echo "✓ Smoke test passed"
    else
        echo "✗ Smoke test failed"
        all_passed=false
    fi
    
    # Run unit test
    echo ""
    if test_unit; then
        echo "✓ Unit test passed"
    else
        echo "✗ Unit test failed"
        all_passed=false
    fi
    
    # Run integration test
    echo ""
    if test_integration; then
        echo "✓ Integration test passed"
    else
        echo "✗ Integration test failed"
        all_passed=false
    fi
    
    echo ""
    echo "=============================="
    if [[ "${all_passed}" == true ]]; then
        echo "All tests PASSED"
        return 0
    else
        echo "Some tests FAILED"
        return 1
    fi
}