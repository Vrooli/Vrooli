#!/usr/bin/env bash
# farmOS Smoke Tests - Quick health validation

set -euo pipefail

# Get resource directory
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

echo "Running farmOS smoke tests..."

# Test 1: Check if service is responding
echo -n "Testing service health... "
# farmOS doesn't have /health endpoint, check homepage (403 is OK, means service is running)
HTTP_STATUS=$(timeout 5 curl -s -o /dev/null -w "%{http_code}" "${FARMOS_BASE_URL}/" 2>/dev/null || echo "000")
if [[ "$HTTP_STATUS" == "200" ]] || [[ "$HTTP_STATUS" == "403" ]] || [[ "$HTTP_STATUS" == "302" ]]; then
    echo "✓ PASS"
else
    # Check if farmOS is even running
    if docker ps | grep -q farmos; then
        echo "✗ FAIL - Service running but not responding (HTTP $HTTP_STATUS)"
    else
        echo "✗ FAIL - Service not running"
    fi
    exit 1
fi

# Test 2: Check if API endpoint is accessible
echo -n "Testing API endpoint... "
if timeout 5 curl -sf "${FARMOS_API_BASE}" > /dev/null 2>&1; then
    echo "✓ PASS"
else
    echo "✗ FAIL - API not accessible"
    exit 1
fi

# Test 3: Check Docker containers
echo -n "Testing Docker containers... "
RUNNING_CONTAINERS=$(docker ps --format "{{.Names}}" | grep -E "farmos|farmos-db" | wc -l)
if [[ $RUNNING_CONTAINERS -ge 2 ]]; then
    echo "✓ PASS - All containers running"
else
    echo "✗ FAIL - Expected 2 containers, found $RUNNING_CONTAINERS"
    exit 1
fi

# Test 4: Check database connectivity
echo -n "Testing database connectivity... "
if docker exec farmos-db pg_isready -U farmos > /dev/null 2>&1; then
    echo "✓ PASS"
else
    echo "✗ FAIL - Database not ready"
    exit 1
fi

# Test 5: Check web interface
echo -n "Testing web interface... "
if timeout 5 curl -sf "${FARMOS_BASE_URL}" | grep -q "farmOS" > /dev/null 2>&1; then
    echo "✓ PASS"
else
    echo "⚠ WARNING - Web interface may not be fully loaded"
    # Don't fail on this as it may take time to fully initialize
fi

echo ""
echo "Smoke tests completed successfully!"
exit 0