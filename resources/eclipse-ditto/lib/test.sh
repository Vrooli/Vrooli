#!/bin/bash
# Eclipse Ditto Test Library

set -euo pipefail

# Test command dispatcher
test_main() {
    local test_type="${1:-all}"
    
    case "$test_type" in
        smoke)
            test_smoke
            ;;
        integration)
            test_integration
            ;;
        unit)
            test_unit
            ;;
        all)
            test_all
            ;;
        *)
            echo "Error: Unknown test type: $test_type" >&2
            echo "Valid types: smoke, integration, unit, all" >&2
            exit 1
            ;;
    esac
}

# Run all tests
test_all() {
    local failed=0
    
    echo "Running all Eclipse Ditto tests..."
    echo "================================"
    
    test_smoke || failed=$((failed + 1))
    echo ""
    
    test_integration || failed=$((failed + 1))
    echo ""
    
    test_unit || failed=$((failed + 1))
    
    if [[ $failed -gt 0 ]]; then
        echo ""
        echo "❌ $failed test suite(s) failed"
        exit 1
    else
        echo ""
        echo "✅ All tests passed"
    fi
}

# Smoke test - Quick health check
test_smoke() {
    echo "Running smoke tests..."
    echo "----------------------"
    
    local failed=0
    
    # Test 1: Check if Docker is available
    echo -n "1. Docker availability... "
    if command -v docker &>/dev/null; then
        echo "✓"
    else
        echo "✗ Docker not found"
        failed=$((failed + 1))
    fi
    
    # Test 2: Check if services are running
    echo -n "2. Service status... "
    if docker ps --format "table {{.Names}}" | grep -q "ditto-gateway"; then
        echo "✓"
    else
        echo "✗ Gateway not running"
        failed=$((failed + 1))
    fi
    
    # Test 3: Health endpoint
    echo -n "3. Health endpoint... "
    if timeout 5 curl -sf "http://localhost:${DITTO_GATEWAY_PORT}/health" &>/dev/null; then
        echo "✓"
    else
        echo "✗ Health check failed"
        failed=$((failed + 1))
    fi
    
    # Test 4: API accessibility
    echo -n "4. API endpoint... "
    if timeout 5 curl -sf -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
        "http://localhost:${DITTO_GATEWAY_PORT}/api/2" &>/dev/null; then
        echo "✓"
    else
        echo "✗ API not accessible"
        failed=$((failed + 1))
    fi
    
    # Test 5: WebSocket endpoint
    echo -n "5. WebSocket endpoint... "
    if timeout 5 curl -sf -I "http://localhost:${DITTO_GATEWAY_PORT}/ws/2" | grep -q "Upgrade: websocket"; then
        echo "✓"
    else
        echo "✗ WebSocket not available"
        failed=$((failed + 1))
    fi
    
    if [[ $failed -gt 0 ]]; then
        echo ""
        echo "Smoke tests failed: $failed/5"
        return 1
    else
        echo ""
        echo "All smoke tests passed (5/5)"
    fi
}

# Integration test - Full functionality
test_integration() {
    echo "Running integration tests..."
    echo "---------------------------"
    
    local failed=0
    local test_twin_id="test:twin:$(date +%s)"
    
    # Test 1: Create digital twin
    echo -n "1. Create twin... "
    if curl -X PUT -sf \
        -H "Content-Type: application/json" \
        -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
        -d "{\"thingId\":\"$test_twin_id\",\"attributes\":{\"test\":true}}" \
        "http://localhost:${DITTO_GATEWAY_PORT}/api/2/things/$test_twin_id" &>/dev/null; then
        echo "✓"
    else
        echo "✗ Failed to create twin"
        failed=$((failed + 1))
    fi
    
    # Test 2: Read twin
    echo -n "2. Read twin... "
    if curl -sf -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
        "http://localhost:${DITTO_GATEWAY_PORT}/api/2/things/$test_twin_id" | grep -q "$test_twin_id"; then
        echo "✓"
    else
        echo "✗ Failed to read twin"
        failed=$((failed + 1))
    fi
    
    # Test 3: Update twin attributes
    echo -n "3. Update attributes... "
    if curl -X PUT -sf \
        -H "Content-Type: application/json" \
        -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
        -d "{\"updated\":\"$(date +%s)\"}" \
        "http://localhost:${DITTO_GATEWAY_PORT}/api/2/things/$test_twin_id/attributes/metadata" &>/dev/null; then
        echo "✓"
    else
        echo "✗ Failed to update attributes"
        failed=$((failed + 1))
    fi
    
    # Test 4: Search twins
    echo -n "4. Search twins... "
    if curl -sf -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
        -G --data-urlencode "filter=eq(thingId,\"$test_twin_id\")" \
        "http://localhost:${DITTO_GATEWAY_PORT}/api/2/search/things" | grep -q "$test_twin_id"; then
        echo "✓"
    else
        echo "✗ Search failed"
        failed=$((failed + 1))
    fi
    
    # Test 5: Delete twin
    echo -n "5. Delete twin... "
    if curl -X DELETE -sf \
        -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
        "http://localhost:${DITTO_GATEWAY_PORT}/api/2/things/$test_twin_id" &>/dev/null; then
        echo "✓"
    else
        echo "✗ Failed to delete twin"
        failed=$((failed + 1))
    fi
    
    # Test 6: MongoDB persistence
    echo -n "6. MongoDB connectivity... "
    if docker exec ditto-mongodb mongosh --eval "db.adminCommand('ping')" &>/dev/null; then
        echo "✓"
    else
        echo "✗ MongoDB not accessible"
        failed=$((failed + 1))
    fi
    
    if [[ $failed -gt 0 ]]; then
        echo ""
        echo "Integration tests failed: $failed/6"
        return 1
    else
        echo ""
        echo "All integration tests passed (6/6)"
    fi
}

# Unit test - Library functions
test_unit() {
    echo "Running unit tests..."
    echo "--------------------"
    
    local failed=0
    
    # Test 1: Log functions
    echo -n "1. Logging functions... "
    if type -t log_info &>/dev/null && type -t log_error &>/dev/null; then
        echo "✓"
    else
        echo "✗ Logging functions not defined"
        failed=$((failed + 1))
    fi
    
    # Test 2: Configuration loading
    echo -n "2. Configuration... "
    if [[ -n "${DITTO_GATEWAY_PORT:-}" ]] && [[ -n "${DITTO_USERNAME:-}" ]]; then
        echo "✓"
    else
        echo "✗ Configuration not loaded"
        failed=$((failed + 1))
    fi
    
    # Test 3: Directory structure
    echo -n "3. Directory structure... "
    if [[ -d "${CONFIG_DIR}" ]] && [[ -d "${RESOURCE_DIR}" ]]; then
        echo "✓"
    else
        echo "✗ Directory structure invalid"
        failed=$((failed + 1))
    fi
    
    # Test 4: Twin functions
    echo -n "4. Twin functions... "
    if type -t twin_create &>/dev/null && type -t twin_update &>/dev/null; then
        echo "✓"
    else
        echo "✗ Twin functions not defined"
        failed=$((failed + 1))
    fi
    
    # Test 5: Docker compose file generation
    echo -n "5. Docker compose generation... "
    if type -t create_docker_compose &>/dev/null; then
        echo "✓"
    else
        echo "✗ Docker compose function not defined"
        failed=$((failed + 1))
    fi
    
    if [[ $failed -gt 0 ]]; then
        echo ""
        echo "Unit tests failed: $failed/5"
        return 1
    else
        echo ""
        echo "All unit tests passed (5/5)"
    fi
}