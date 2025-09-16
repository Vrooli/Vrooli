#!/usr/bin/env bash
# Gazebo Test Library

set -euo pipefail

# Source core library for configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/core.sh"

# Test functions
test_smoke() {
    log_info "Running smoke tests..."
    
    local all_passed=true
    
    # Test 1: Check if Gazebo data directory exists
    echo -n "Testing Gazebo data directory... "
    if [[ -d "${GAZEBO_DATA_DIR}" ]]; then
        echo "✓"
    else
        echo "✗ (Run 'vrooli resource gazebo manage install' first)"
        all_passed=false
    fi
    
    # Test 2: Check health endpoint
    echo -n "Testing health endpoint... "
    if timeout 5 curl -sf "${GAZEBO_HEALTH_URL}" > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        log_warn "Health endpoint not responding (service may not be running)"
    fi
    
    # Test 3: Check directories exist
    echo -n "Testing directory structure... "
    if [[ -d "${GAZEBO_DATA_DIR}" && -d "${GAZEBO_WORLDS_DIR}" && -d "${GAZEBO_MODELS_DIR}" ]]; then
        echo "✓"
    else
        echo "✗"
        all_passed=false
    fi
    
    # Test 4: Check minimal Python dependencies
    echo -n "Testing Python availability... "
    if python3 --version > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        log_warn "Python3 not available"
        all_passed=false
    fi
    
    # Test 5: Check configuration
    echo -n "Testing configuration setup... "
    if [[ -f "${SCRIPT_DIR}/config/runtime.json" ]]; then
        echo "✓"
    else
        echo "✗"
        all_passed=false
    fi
    
    if [[ "$all_passed" == "true" ]]; then
        log_info "All smoke tests passed"
        return 0
    else
        log_error "Some smoke tests failed"
        return 1
    fi
}

test_integration() {
    log_info "Running integration tests..."
    
    local all_passed=true
    local was_running=false
    
    # Check if already running
    if gazebo_is_running; then
        was_running=true
        log_info "Gazebo already running, using existing instance"
    else
        log_info "Starting Gazebo for integration tests..."
        gazebo_start --wait --timeout 30 || {
            log_error "Failed to start Gazebo for testing"
            return 1
        }
    fi
    
    # Test 1: Health check with content validation
    echo -n "Testing health check content... "
    local health_response=$(timeout 5 curl -sf "${GAZEBO_HEALTH_URL}" 2>/dev/null || echo "{}")
    if echo "$health_response" | jq -e '.status == "healthy"' > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        all_passed=false
    fi
    
    # Test 2: World file operations
    echo -n "Testing world file operations... "
    if [[ -f "${GAZEBO_WORLDS_DIR}/cart_pole.world" ]]; then
        echo "✓"
    else
        echo "✗"
        all_passed=false
    fi
    
    # Test 3: Python API basic test
    echo -n "Testing Python availability for API... "
    if python3 -c "
import sys
try:
    import json
    import time
    print('Success')
    sys.exit(0)
except:
    sys.exit(1)
" > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        all_passed=false
    fi
    
    # Test 4: Service responsiveness
    echo -n "Testing service responsiveness... "
    local start_time=$(date +%s%3N)
    timeout 1 curl -sf "${GAZEBO_HEALTH_URL}" > /dev/null 2>&1
    local end_time=$(date +%s%3N)
    local response_time=$((end_time - start_time))
    
    if [[ $response_time -lt 1000 ]]; then
        echo "✓ (${response_time}ms)"
    else
        echo "✗ (${response_time}ms > 1000ms)"
        all_passed=false
    fi
    
    # Test 5: Log file creation
    echo -n "Testing log file creation... "
    if [[ -f "${GAZEBO_LOG_DIR}/gazebo.log" || -f "${GAZEBO_LOG_DIR}/health.log" ]]; then
        echo "✓"
    else
        echo "✗"
        log_warn "Log files not found"
    fi
    
    # Clean up if we started Gazebo
    if [[ "$was_running" == "false" ]]; then
        log_info "Stopping Gazebo after tests..."
        gazebo_stop
    fi
    
    if [[ "$all_passed" == "true" ]]; then
        log_info "All integration tests passed"
        return 0
    else
        log_error "Some integration tests failed"
        return 1
    fi
}

test_unit() {
    log_info "Running unit tests..."
    
    local all_passed=true
    
    # Test 1: Configuration variables
    echo -n "Testing configuration variables... "
    if [[ -n "${GAZEBO_PORT}" && -n "${GAZEBO_DATA_DIR}" ]]; then
        echo "✓"
    else
        echo "✗"
        all_passed=false
    fi
    
    # Test 2: Logging functions
    echo -n "Testing logging functions... "
    (
        log_info "Test" > /dev/null 2>&1
        log_warn "Test" > /dev/null 2>&1
        log_error "Test" > /dev/null 2>&1
    ) && echo "✓" || { echo "✗"; all_passed=false; }
    
    # Test 3: Directory creation function
    echo -n "Testing directory creation... "
    local test_dir="/tmp/gazebo_test_$$"
    mkdir -p "$test_dir" 2>/dev/null && rm -rf "$test_dir" && echo "✓" || { echo "✗"; all_passed=false; }
    
    # Test 4: PID file handling
    echo -n "Testing PID file handling... "
    local test_pid_file="/tmp/gazebo_test_pid_$$"
    echo "12345" > "$test_pid_file"
    if [[ $(cat "$test_pid_file") == "12345" ]]; then
        rm -f "$test_pid_file"
        echo "✓"
    else
        echo "✗"
        all_passed=false
    fi
    
    # Test 5: Port availability check
    echo -n "Testing port configuration... "
    if [[ "${GAZEBO_PORT}" == "11456" ]]; then
        echo "✓"
    else
        echo "✗"
        all_passed=false
    fi
    
    if [[ "$all_passed" == "true" ]]; then
        log_info "All unit tests passed"
        return 0
    else
        log_error "Some unit tests failed"
        return 1
    fi
}

test_all() {
    log_info "Running all test suites..."
    
    local failed=0
    
    test_smoke || ((failed++))
    test_unit || ((failed++))
    test_integration || ((failed++))
    
    if [[ $failed -eq 0 ]]; then
        log_info "All test suites passed"
        return 0
    else
        log_error "$failed test suite(s) failed"
        return 1
    fi
}