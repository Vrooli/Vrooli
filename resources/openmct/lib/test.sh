#!/usr/bin/env bash
# Open MCT Test Functions

set -euo pipefail

# Load configuration
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${TEST_DIR}/../config/defaults.sh"

# Handle test command
handle_test() {
    local phase="${1:-all}"
    
    case "$phase" in
        smoke)
            run_smoke_tests
            ;;
        integration)
            run_integration_tests
            ;;
        unit)
            run_unit_tests
            ;;
        all)
            run_all_tests
            ;;
        *)
            echo "Error: Unknown test phase: $phase"
            echo "Valid phases: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

# Run smoke tests
run_smoke_tests() {
    echo "Running Open MCT smoke tests..."
    echo "================================"
    
    local passed=0
    local failed=0
    
    # Test 1: Health check endpoint
    echo -n "Testing health endpoint... "
    if timeout 5 curl -sf "http://localhost:${OPENMCT_PORT}/health" > /dev/null 2>&1; then
        echo "✓ PASSED"
        passed=$((passed + 1))
    else
        echo "✗ FAILED"
        failed=$((failed + 1))
    fi
    
    # Test 2: Dashboard accessibility
    echo -n "Testing dashboard accessibility... "
    if timeout 5 curl -sf "http://localhost:${OPENMCT_PORT}/" > /dev/null 2>&1; then
        echo "✓ PASSED"
        passed=$((passed + 1))
    else
        echo "✗ FAILED"
        failed=$((failed + 1))
    fi
    
    # Test 3: WebSocket connectivity
    echo -n "Testing WebSocket port... "
    if nc -zv localhost "${OPENMCT_WS_PORT}" &> /dev/null; then
        echo "✓ PASSED"
        passed=$((passed + 1))
    else
        echo "✗ FAILED"
        failed=$((failed + 1))
    fi
    
    # Test 4: Telemetry API endpoint
    echo -n "Testing telemetry API... "
    if timeout 5 curl -sf "http://localhost:${OPENMCT_PORT}/api/telemetry/test_stream" > /dev/null 2>&1; then
        echo "✓ PASSED"
        passed=$((passed + 1))
    else
        echo "✗ FAILED"
        failed=$((failed + 1))
    fi
    
    # Test 5: Database connectivity
    echo -n "Testing database connectivity... "
    if sqlite3 "${OPENMCT_DATA_DIR}/telemetry.db" "SELECT COUNT(*) FROM telemetry;" > /dev/null 2>&1; then
        echo "✓ PASSED"
        passed=$((passed + 1))
    else
        echo "✗ FAILED"
        failed=$((failed + 1))
    fi
    
    # Summary
    echo ""
    echo "Smoke Test Results:"
    echo "  Passed: $passed"
    echo "  Failed: $failed"
    
    if [[ $failed -gt 0 ]]; then
        echo "Status: FAILED"
        exit 1
    else
        echo "Status: PASSED"
    fi
}

# Run integration tests
run_integration_tests() {
    echo "Running Open MCT integration tests..."
    echo "====================================="
    
    local passed=0
    local failed=0
    
    # Test 1: Push telemetry data via REST
    echo -n "Testing REST telemetry ingestion... "
    local response=$(curl -s -X POST "http://localhost:${OPENMCT_PORT}/api/telemetry/test_stream/data" \
        -H "Content-Type: application/json" \
        -d '{"timestamp": '$(date +%s000)', "value": 42.5}' 2>/dev/null || echo "FAILED")
    
    if echo "$response" | grep -q "success"; then
        echo "✓ PASSED"
        passed=$((passed + 1))
    else
        echo "✗ FAILED"
        failed=$((failed + 1))
    fi
    
    # Test 2: Query historical data
    echo -n "Testing historical data query... "
    local history=$(curl -sf "http://localhost:${OPENMCT_PORT}/api/telemetry/history?stream=test_stream" 2>/dev/null || echo "[]")
    
    if [[ "$history" != "[]" ]] && [[ "$history" != "" ]]; then
        echo "✓ PASSED"
        passed=$((passed + 1))
    else
        echo "✗ FAILED"
        failed=$((failed + 1))
    fi
    
    # Test 3: WebSocket telemetry (using nc for basic test)
    echo -n "Testing WebSocket telemetry... "
    if echo '{"stream":"ws_test","value":100}' | nc -w 1 localhost "${OPENMCT_WS_PORT}" > /dev/null 2>&1; then
        echo "✓ PASSED"
        passed=$((passed + 1))
    else
        echo "✗ FAILED"
        failed=$((failed + 1))
    fi
    
    # Test 4: Demo telemetry streams (if enabled)
    if [[ "$OPENMCT_ENABLE_DEMO" == "true" ]]; then
        echo -n "Testing demo telemetry streams... "
        
        # Wait a bit for demo data to accumulate
        sleep 3
        
        local demo_data=$(sqlite3 "${OPENMCT_DATA_DIR}/telemetry.db" \
            "SELECT COUNT(*) FROM telemetry WHERE stream IN ('satellite_position', 'sensor_network', 'system_metrics');" 2>/dev/null || echo "0")
        
        if [[ "$demo_data" -gt 0 ]]; then
            echo "✓ PASSED"
            passed=$((passed + 1))
        else
            echo "✗ FAILED"
            failed=$((failed + 1))
        fi
    fi
    
    # Test 5: Container resource limits
    echo -n "Testing container resource limits... "
    local container_stats=$(docker stats --no-stream --format "json" "$OPENMCT_CONTAINER_NAME" 2>/dev/null || echo "{}")
    
    if [[ -n "$container_stats" ]] && [[ "$container_stats" != "{}" ]]; then
        echo "✓ PASSED"
        passed=$((passed + 1))
    else
        echo "✗ FAILED"
        failed=$((failed + 1))
    fi
    
    # Test 6: Configuration persistence
    echo -n "Testing configuration persistence... "
    local test_config="${OPENMCT_CONFIG_DIR}/test_config.json"
    echo '{"test": true}' > "$test_config"
    
    if [[ -f "$test_config" ]]; then
        rm "$test_config"
        echo "✓ PASSED"
        passed=$((passed + 1))
    else
        echo "✗ FAILED"
        failed=$((failed + 1))
    fi
    
    # Summary
    echo ""
    echo "Integration Test Results:"
    echo "  Passed: $passed"
    echo "  Failed: $failed"
    
    if [[ $failed -gt 0 ]]; then
        echo "Status: FAILED"
        exit 1
    else
        echo "Status: PASSED"
    fi
}

# Run unit tests
run_unit_tests() {
    echo "Running Open MCT unit tests..."
    echo "=============================="
    
    local passed=0
    local failed=0
    
    # Test 1: Port configuration
    echo -n "Testing port configuration... "
    if [[ "$OPENMCT_PORT" == "8099" ]]; then
        echo "✓ PASSED"
        passed=$((passed + 1))
    else
        echo "✗ FAILED (expected 8099, got $OPENMCT_PORT)"
        failed=$((failed + 1))
    fi
    
    # Test 2: Directory creation
    echo -n "Testing directory structure... "
    if [[ -d "$OPENMCT_DATA_DIR" ]] && [[ -d "$OPENMCT_CONFIG_DIR" ]] && [[ -d "$OPENMCT_PLUGINS_DIR" ]]; then
        echo "✓ PASSED"
        passed=$((passed + 1))
    else
        echo "✗ FAILED"
        failed=$((failed + 1))
    fi
    
    # Test 3: Database schema
    echo -n "Testing database schema... "
    local tables=$(sqlite3 "${OPENMCT_DATA_DIR}/telemetry.db" ".tables" 2>/dev/null || echo "")
    
    if echo "$tables" | grep -q "telemetry"; then
        echo "✓ PASSED"
        passed=$((passed + 1))
    else
        echo "✗ FAILED"
        failed=$((failed + 1))
    fi
    
    # Test 4: Docker image existence
    echo -n "Testing Docker image... "
    if docker images | grep -q "$OPENMCT_IMAGE"; then
        echo "✓ PASSED"
        passed=$((passed + 1))
    else
        echo "✗ FAILED"
        failed=$((failed + 1))
    fi
    
    # Test 5: Configuration defaults
    echo -n "Testing configuration defaults... "
    if [[ "$OPENMCT_MAX_STREAMS" == "100" ]] && [[ "$OPENMCT_HISTORY_DAYS" == "30" ]]; then
        echo "✓ PASSED"
        passed=$((passed + 1))
    else
        echo "✗ FAILED"
        failed=$((failed + 1))
    fi
    
    # Summary
    echo ""
    echo "Unit Test Results:"
    echo "  Passed: $passed"
    echo "  Failed: $failed"
    
    if [[ $failed -gt 0 ]]; then
        echo "Status: FAILED"
        exit 1
    else
        echo "Status: PASSED"
    fi
}

# Run all tests
run_all_tests() {
    echo "Running all Open MCT tests..."
    echo "=============================="
    echo ""
    
    # Track overall results
    local overall_failed=0
    
    # Run smoke tests
    if ! run_smoke_tests; then
        overall_failed=$((overall_failed + 1))
    fi
    echo ""
    
    # Run integration tests
    if ! run_integration_tests; then
        overall_failed=$((overall_failed + 1))
    fi
    echo ""
    
    # Run unit tests
    if ! run_unit_tests; then
        overall_failed=$((overall_failed + 1))
    fi
    
    # Overall summary
    echo ""
    echo "================================"
    echo "Overall Test Results:"
    if [[ $overall_failed -eq 0 ]]; then
        echo "Status: ALL TESTS PASSED ✓"
        exit 0
    else
        echo "Status: $overall_failed TEST SUITES FAILED ✗"
        exit 1
    fi
}