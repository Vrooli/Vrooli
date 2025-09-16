#!/bin/bash
# Eclipse Ditto Integration Tests - Full functionality validation

set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Source libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/twins.sh"

echo "Eclipse Ditto Integration Tests"
echo "================================"

# Ensure service is running for integration tests
if ! docker ps --format "{{.Names}}" | grep -q "ditto-gateway"; then
    echo "⚠ Eclipse Ditto not running. Starting services..."
    manage_start --wait || {
        echo "✗ Failed to start Eclipse Ditto"
        exit 1
    }
    CLEANUP_REQUIRED=true
fi

# Track failures
failed=0

# Test 1: API Version endpoint
echo -n "1. API version endpoint... "
if curl -sf -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
    "http://localhost:${DITTO_GATEWAY_PORT}/api/2" | grep -q "\"version\":2"; then
    echo "✓"
else
    echo "✗ API version check failed"
    failed=$((failed + 1))
fi

# Test 2: Create and retrieve twin
echo -n "2. Create and retrieve twin... "
TEST_TWIN="test:integration:$(date +%s)"
if curl -X PUT -sf \
    -H "Content-Type: application/json" \
    -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
    -d "{\"thingId\":\"$TEST_TWIN\",\"attributes\":{\"test\":\"integration\"}}" \
    "http://localhost:${DITTO_GATEWAY_PORT}/api/2/things/$TEST_TWIN" &>/dev/null && \
   curl -sf -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
    "http://localhost:${DITTO_GATEWAY_PORT}/api/2/things/$TEST_TWIN" | grep -q "$TEST_TWIN"; then
    echo "✓"
else
    echo "✗ Twin CRUD operations failed"
    failed=$((failed + 1))
fi

# Test 3: Update twin features
echo -n "3. Update twin features... "
if curl -X PUT -sf \
    -H "Content-Type: application/json" \
    -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
    -d "{\"properties\":{\"temperature\":25.5,\"unit\":\"celsius\"}}" \
    "http://localhost:${DITTO_GATEWAY_PORT}/api/2/things/$TEST_TWIN/features/sensor" &>/dev/null; then
    echo "✓"
else
    echo "✗ Feature update failed"
    failed=$((failed + 1))
fi

# Test 4: Search functionality
echo -n "4. Search API... "
if curl -sf -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
    "http://localhost:${DITTO_GATEWAY_PORT}/api/2/search/things" &>/dev/null; then
    echo "✓"
else
    echo "✗ Search API not accessible"
    failed=$((failed + 1))
fi

# Test 5: WebSocket connectivity check
echo -n "5. WebSocket upgrade header... "
if curl -sf -I "http://localhost:${DITTO_GATEWAY_PORT}/ws/2" | grep -qi "upgrade"; then
    echo "✓"
else
    echo "✗ WebSocket headers missing"
    failed=$((failed + 1))
fi

# Test 6: Policy management
echo -n "6. Policy creation... "
if curl -X PUT -sf \
    -H "Content-Type: application/json" \
    -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
    -d "{\"policyId\":\"$TEST_TWIN\",\"entries\":{\"DEFAULT\":{\"subjects\":{\"nginx:ditto\":{\"type\":\"user\"}},\"resources\":{\"thing:/\":{\"grant\":[\"READ\",\"WRITE\"],\"revoke\":[]}}}}}" \
    "http://localhost:${DITTO_GATEWAY_PORT}/api/2/policies/$TEST_TWIN" &>/dev/null; then
    echo "✓"
else
    echo "✗ Policy creation failed"
    failed=$((failed + 1))
fi

# Test 7: MongoDB connectivity
echo -n "7. MongoDB persistence... "
if docker exec ditto-mongodb mongosh --eval "db.adminCommand('ping')" &>/dev/null; then
    echo "✓"
else
    echo "✗ MongoDB not responding"
    failed=$((failed + 1))
fi

# Test 8: Delete test twin
echo -n "8. Cleanup test twin... "
if curl -X DELETE -sf \
    -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
    "http://localhost:${DITTO_GATEWAY_PORT}/api/2/things/$TEST_TWIN" &>/dev/null; then
    echo "✓"
else
    echo "✗ Cleanup failed"
    failed=$((failed + 1))
fi

# Cleanup if we started the service
if [[ "${CLEANUP_REQUIRED:-false}" == "true" ]]; then
    echo ""
    echo "Stopping Eclipse Ditto (was not running before tests)..."
    manage_stop
fi

# Report results
echo ""
if [[ $failed -gt 0 ]]; then
    echo "❌ Integration tests failed: $failed test(s) failed"
    exit 1
else
    echo "✅ All integration tests passed"
fi