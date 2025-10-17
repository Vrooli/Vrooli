#!/bin/bash
# SU2 Smoke Tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Load configuration
source "${RESOURCE_DIR}/config/defaults.sh"

echo "=== SU2 Smoke Tests ==="
echo "Testing basic health and connectivity..."

# Test 1: Service health check
echo -n "1. Health endpoint... "
if timeout 5 curl -sf "http://localhost:${SU2_PORT}/health" > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "   Health check failed - is service running?"
    exit 1
fi

# Test 2: API status
echo -n "2. API status... "
response=$(timeout 5 curl -sf "http://localhost:${SU2_PORT}/api/status" 2>/dev/null || echo "failed")
if [[ "$response" != "failed" ]] && echo "$response" | grep -q "capabilities"; then
    echo "✓"
else
    echo "✗"
    echo "   API status check failed"
    exit 1
fi

# Test 3: Container running
echo -n "3. Container status... "
if docker ps --format '{{.Names}}' | grep -q "^${SU2_CONTAINER_NAME}$"; then
    echo "✓"
else
    echo "✗"
    echo "   Container not running"
    exit 1
fi

echo -e "\n✓ Smoke tests passed"
exit 0