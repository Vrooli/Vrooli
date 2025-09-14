#!/usr/bin/env bash
################################################################################
# Zigbee2MQTT Test Library
# 
# Testing functions for Zigbee2MQTT resource
################################################################################

set -euo pipefail

# Source core library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/core.sh"

################################################################################
# Test Functions
################################################################################

# Run smoke tests
zigbee2mqtt::test::smoke() {
    log::info "Running Zigbee2MQTT smoke tests..."
    
    local passed=0
    local failed=0
    
    # Test 1: Container is running
    echo -n "Testing container status... "
    if docker ps --format "{{.Names}}" | grep -q "^${ZIGBEE2MQTT_CONTAINER}$"; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (Container not running)"
        ((failed++))
    fi
    
    # Test 2: Health endpoint responds
    echo -n "Testing health endpoint... "
    if timeout 5 curl -sf "http://localhost:${ZIGBEE2MQTT_PORT}/api/health" &>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (Health check failed)"
        ((failed++))
    fi
    
    # Test 3: MQTT connection
    echo -n "Testing MQTT connectivity... "
    if docker logs "${ZIGBEE2MQTT_CONTAINER}" 2>&1 | grep -q "MQTT publish"; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (MQTT not connected)"
        ((failed++))
    fi
    
    # Test 4: Configuration exists
    echo -n "Testing configuration... "
    if [[ -f "${ZIGBEE2MQTT_DATA_DIR}/configuration.yaml" ]]; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (Configuration missing)"
        ((failed++))
    fi
    
    # Test 5: Web UI accessible
    echo -n "Testing web UI... "
    if timeout 5 curl -sf "http://localhost:${ZIGBEE2MQTT_PORT}/" &>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (Web UI not accessible)"
        ((failed++))
    fi
    
    # Summary
    echo ""
    echo "Smoke Tests: $passed passed, $failed failed"
    
    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    
    log::success "All smoke tests passed"
    return 0
}

# Run integration tests
zigbee2mqtt::test::integration() {
    log::info "Running Zigbee2MQTT integration tests..."
    
    local passed=0
    local failed=0
    
    # Test 1: Device list API
    echo -n "Testing device list API... "
    if curl -sf "http://localhost:${ZIGBEE2MQTT_PORT}/api/devices" | jq -e '.' &>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (API error)"
        ((failed++))
    fi
    
    # Test 2: Bridge state API
    echo -n "Testing bridge state API... "
    if curl -sf "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/state" | jq -e '.state' &>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (Bridge state error)"
        ((failed++))
    fi
    
    # Test 3: Network map API
    echo -n "Testing network map API... "
    if curl -sf "http://localhost:${ZIGBEE2MQTT_PORT}/api/networkmap" | jq -e '.' &>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (Network map error)"
        ((failed++))
    fi
    
    # Test 4: Configuration API
    echo -n "Testing configuration API... "
    if curl -sf "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/config" | jq -e '.version' &>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (Config API error)"
        ((failed++))
    fi
    
    # Test 5: Extensions API
    echo -n "Testing extensions API... "
    if curl -sf "http://localhost:${ZIGBEE2MQTT_PORT}/api/extensions" | jq -e '.' &>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (Extensions API error)"
        ((failed++))
    fi
    
    # Test 6: Permit join functionality
    echo -n "Testing permit join toggle... "
    if curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/request/permit_join" \
        -H "Content-Type: application/json" \
        -d '{"value": false}' 2>/dev/null | jq -e '.' &>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (Permit join error)"
        ((failed++))
    fi
    
    # Test 7: Logs API
    echo -n "Testing logs API... "
    if curl -sf "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/logs" | jq -e '.' &>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (Logs API error)"
        ((failed++))
    fi
    
    # Summary
    echo ""
    echo "Integration Tests: $passed passed, $failed failed"
    
    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    
    log::success "All integration tests passed"
    return 0
}

# Run unit tests
zigbee2mqtt::test::unit() {
    log::info "Running Zigbee2MQTT unit tests..."
    
    local passed=0
    local failed=0
    
    # Test 1: Configuration validation
    echo -n "Testing configuration validation... "
    if [[ -f "${ZIGBEE2MQTT_DATA_DIR}/configuration.yaml" ]]; then
        if grep -q "mqtt:" "${ZIGBEE2MQTT_DATA_DIR}/configuration.yaml" && \
           grep -q "serial:" "${ZIGBEE2MQTT_DATA_DIR}/configuration.yaml"; then
            echo "✓"
            ((passed++))
        else
            echo "✗ (Invalid configuration)"
            ((failed++))
        fi
    else
        echo "✗ (No configuration)"
        ((failed++))
    fi
    
    # Test 2: Port allocation
    echo -n "Testing port allocation... "
    if [[ "${ZIGBEE2MQTT_PORT}" =~ ^[0-9]+$ ]] && [[ "${ZIGBEE2MQTT_PORT}" -gt 1024 ]]; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (Invalid port)"
        ((failed++))
    fi
    
    # Test 3: Data directory
    echo -n "Testing data directory... "
    if [[ -d "${ZIGBEE2MQTT_DATA_DIR}" ]] && [[ -w "${ZIGBEE2MQTT_DATA_DIR}" ]]; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (Data directory issue)"
        ((failed++))
    fi
    
    # Test 4: Docker image
    echo -n "Testing Docker image... "
    if docker images | grep -q "koenkk/zigbee2mqtt"; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (Image not found)"
        ((failed++))
    fi
    
    # Test 5: Environment variables
    echo -n "Testing environment variables... "
    if [[ -n "${MQTT_HOST}" ]] && [[ -n "${MQTT_PORT}" ]]; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (Missing env vars)"
        ((failed++))
    fi
    
    # Summary
    echo ""
    echo "Unit Tests: $passed passed, $failed failed"
    
    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    
    log::success "All unit tests passed"
    return 0
}

# Run all tests
zigbee2mqtt::test::all() {
    log::info "Running all Zigbee2MQTT tests..."
    
    local failed=0
    
    # Run test suites
    zigbee2mqtt::test::smoke || ((failed++))
    echo ""
    zigbee2mqtt::test::unit || ((failed++))
    echo ""
    zigbee2mqtt::test::integration || ((failed++))
    
    if [[ $failed -gt 0 ]]; then
        log::error "$failed test suite(s) failed"
        return 1
    fi
    
    log::success "All test suites passed"
    return 0
}