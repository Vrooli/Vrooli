#!/usr/bin/env bash
################################################################################
# ESPHome Test Library
################################################################################

set -euo pipefail

# ==============================================================================
# TEST IMPLEMENTATIONS
# ==============================================================================

esphome::test::smoke() {
    log::info "Running ESPHome smoke tests..."
    
    local test_passed=0
    local test_failed=0
    
    # Test 1: Container is running
    echo -n "Testing container status... "
    if docker ps --format "{{.Names}}" | grep -q "^${ESPHOME_CONTAINER_NAME}$"; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Container not running"
        ((test_failed++))
    fi
    
    # Test 2: Health check passes (ESPHome doesn't have /health, check dashboard)
    echo -n "Testing health endpoint... "
    if timeout 5 curl -sf "${ESPHOME_BASE_URL}" > /dev/null 2>&1; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Health check failed"
        ((test_failed++))
    fi
    
    # Test 3: Dashboard is accessible
    echo -n "Testing dashboard accessibility... "
    if timeout 5 curl -sf "${ESPHOME_BASE_URL}" > /dev/null 2>&1; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Dashboard not accessible"
        ((test_failed++))
    fi
    
    # Test 4: Configuration directory exists
    echo -n "Testing configuration directory... "
    if [[ -d "${ESPHOME_CONFIG_DIR}" ]]; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Config directory missing"
        ((test_failed++))
    fi
    
    # Test 5: Example configuration exists
    echo -n "Testing example configuration... "
    if [[ -f "${ESPHOME_CONFIG_DIR}/example.yaml" ]]; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Example config missing"
        ((test_failed++))
    fi
    
    echo ""
    echo "Smoke Test Results:"
    echo "  Passed: $test_passed"
    echo "  Failed: $test_failed"
    
    if [[ $test_failed -eq 0 ]]; then
        log::success "All smoke tests passed!"
        return 0
    else
        log::error "Some smoke tests failed"
        return 1
    fi
}

esphome::test::integration() {
    log::info "Running ESPHome integration tests..."
    
    local test_passed=0
    local test_failed=0
    
    # Test 1: Validate example configuration
    echo -n "Testing configuration validation... "
    if docker exec "${ESPHOME_CONTAINER_NAME}" \
        esphome config "/config/example.yaml" > /dev/null 2>&1; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Config validation failed"
        ((test_failed++))
    fi
    
    # Test 2: Test configuration addition
    echo -n "Testing configuration management... "
    local test_config="/tmp/test_esphome_$$.yaml"
    cat > "$test_config" << 'EOF'
esphome:
  name: test-device
  platform: ESP32
  board: esp32dev

wifi:
  ssid: "TestSSID"
  password: "TestPassword"

logger:
ota:
api:
EOF
    
    if esphome::add_config "$test_config" > /dev/null 2>&1; then
        echo "✓ PASSED"
        ((test_passed++))
        # Clean up
        rm -f "${ESPHOME_CONFIG_DIR}/test_esphome_$$.yaml"
    else
        echo "✗ FAILED - Config addition failed"
        ((test_failed++))
    fi
    rm -f "$test_config"
    
    # Test 3: List configurations
    echo -n "Testing configuration listing... "
    if esphome::list_configs > /dev/null 2>&1; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Config listing failed"
        ((test_failed++))
    fi
    
    # Test 4: Docker network connectivity
    echo -n "Testing Docker network... "
    if docker exec "${ESPHOME_CONTAINER_NAME}" ping -c 1 google.com > /dev/null 2>&1; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Network connectivity issue"
        ((test_failed++))
    fi
    
    # Test 5: PlatformIO availability
    echo -n "Testing PlatformIO installation... "
    if docker exec "${ESPHOME_CONTAINER_NAME}" which platformio > /dev/null 2>&1; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - PlatformIO not available"
        ((test_failed++))
    fi
    
    echo ""
    echo "Integration Test Results:"
    echo "  Passed: $test_passed"
    echo "  Failed: $test_failed"
    
    if [[ $test_failed -eq 0 ]]; then
        log::success "All integration tests passed!"
        return 0
    else
        log::error "Some integration tests failed"
        return 1
    fi
}

esphome::test::unit() {
    log::info "Running ESPHome unit tests..."
    
    local test_passed=0
    local test_failed=0
    
    # Test 1: Configuration validation function
    echo -n "Testing config validation function... "
    esphome::export_config
    if esphome::validate_config 2>/dev/null; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED"
        ((test_failed++))
    fi
    
    # Test 2: Port number validation
    echo -n "Testing port validation... "
    if [[ "$ESPHOME_PORT" =~ ^[0-9]+$ ]] && \
       [ "$ESPHOME_PORT" -ge 1024 ] && \
       [ "$ESPHOME_PORT" -le 65535 ]; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Invalid port: $ESPHOME_PORT"
        ((test_failed++))
    fi
    
    # Test 3: Directory creation
    echo -n "Testing directory creation... "
    local test_dir="/tmp/esphome_test_$$"
    ESPHOME_DATA_DIR="$test_dir" esphome::validate_config 2>/dev/null
    if [[ -d "$test_dir" ]]; then
        echo "✓ PASSED"
        ((test_passed++))
        rm -rf "$test_dir"
    else
        echo "✗ FAILED"
        ((test_failed++))
    fi
    
    # Test 4: Get config function
    echo -n "Testing get_config function... "
    local port_value
    port_value=$(esphome::get_config "port")
    if [[ "$port_value" == "$ESPHOME_PORT" ]]; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED"
        ((test_failed++))
    fi
    
    # Test 5: Timeout values
    echo -n "Testing timeout values... "
    if [[ "$ESPHOME_COMPILE_TIMEOUT" -gt 0 ]] && \
       [[ "$ESPHOME_UPLOAD_TIMEOUT" -gt 0 ]] && \
       [[ "$ESPHOME_STARTUP_TIMEOUT" -gt 0 ]]; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Invalid timeout values"
        ((test_failed++))
    fi
    
    echo ""
    echo "Unit Test Results:"
    echo "  Passed: $test_passed"
    echo "  Failed: $test_failed"
    
    if [[ $test_failed -eq 0 ]]; then
        log::success "All unit tests passed!"
        return 0
    else
        log::error "Some unit tests failed"
        return 1
    fi
}

esphome::test::all() {
    log::info "Running all ESPHome tests..."
    echo ""
    
    local smoke_result=0
    local integration_result=0
    local unit_result=0
    
    # Run smoke tests
    esphome::test::smoke || smoke_result=$?
    echo ""
    
    # Run integration tests only if container is running
    if docker ps --format "{{.Names}}" | grep -q "^${ESPHOME_CONTAINER_NAME}$"; then
        esphome::test::integration || integration_result=$?
        echo ""
    else
        log::warning "Skipping integration tests - container not running"
    fi
    
    # Run unit tests
    esphome::test::unit || unit_result=$?
    echo ""
    
    # Summary
    echo "================================"
    echo "Test Suite Summary:"
    echo "  Smoke Tests: $([ $smoke_result -eq 0 ] && echo '✓ PASSED' || echo '✗ FAILED')"
    echo "  Integration Tests: $([ $integration_result -eq 0 ] && echo '✓ PASSED' || echo '✗ FAILED')"
    echo "  Unit Tests: $([ $unit_result -eq 0 ] && echo '✓ PASSED' || echo '✗ FAILED')"
    echo "================================"
    
    # Return failure if any test suite failed
    if [[ $smoke_result -ne 0 ]] || [[ $integration_result -ne 0 ]] || [[ $unit_result -ne 0 ]]; then
        return 1
    fi
    
    log::success "All test suites passed!"
    return 0
}