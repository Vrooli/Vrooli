#!/bin/bash
# Traccar Smoke Tests - Quick health validation (30s max)

set -euo pipefail

# Test configuration
TRACCAR_HOST="${TRACCAR_HOST:-localhost}"
TRACCAR_PORT="${TRACCAR_PORT:-8082}"
TRACCAR_CONTAINER="${TRACCAR_CONTAINER_NAME:-vrooli-traccar}"
TRACCAR_ADMIN_EMAIL="${TRACCAR_ADMIN_EMAIL:-admin@example.com}"
TRACCAR_ADMIN_PASSWORD="${TRACCAR_ADMIN_PASSWORD:-admin}"

echo "Running Traccar smoke tests..."

# Test 1: Check if container is running
echo -n "1. Checking if Traccar container is running... "
if docker ps --format "table {{.Names}}" | grep -q "^${TRACCAR_CONTAINER}$"; then
    echo "✓"
else
    echo "✗"
    echo "   Error: Traccar container is not running"
    exit 1
fi

# Test 2: Check health endpoint
echo -n "2. Checking Traccar health endpoint... "
if timeout 5 curl -sf "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/server" &>/dev/null; then
    echo "✓"
else
    echo "✗"
    echo "   Error: Traccar health endpoint not responding"
    exit 1
fi

# Test 3: Check API authentication
echo -n "3. Testing API authentication... "
if response=$(timeout 5 curl -sf -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
    "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/session" 2>&1); then
    echo "✓"
else
    echo "✗"
    echo "   Error: Failed to authenticate with Traccar API"
    echo "   Response: $response"
    exit 1
fi

# Test 4: Check devices endpoint
echo -n "4. Checking devices endpoint... "
if timeout 5 curl -sf -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
    "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices" &>/dev/null; then
    echo "✓"
else
    echo "✗"
    echo "   Error: Devices endpoint not accessible"
    exit 1
fi

# Test 5: Check WebSocket endpoint availability
echo -n "5. Checking WebSocket endpoint... "
if timeout 5 curl -sf -I "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/socket" 2>/dev/null | \
    grep -q "HTTP/1.1"; then
    echo "✓"
else
    echo "✗"
    echo "   Warning: WebSocket endpoint may not be available"
    # Non-critical, don't exit
fi

echo ""
echo "Smoke tests completed successfully!"
exit 0