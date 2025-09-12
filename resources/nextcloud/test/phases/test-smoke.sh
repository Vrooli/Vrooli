#!/usr/bin/env bash
# Nextcloud Smoke Tests - Quick health validation

set -euo pipefail

# Get resource directory
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${RESOURCE_DIR}/config/defaults.sh"

echo "Running Nextcloud smoke tests..."

# Test 1: Check if container is running
echo -n "  Checking container status... "
if docker ps --format "{{.Names}}" | grep -q "^${NEXTCLOUD_CONTAINER_NAME}$"; then
    echo "✓"
else
    echo "✗"
    echo "Error: Nextcloud container is not running"
    exit 1
fi

# Test 2: Check health endpoint
echo -n "  Checking health endpoint... "
if timeout 5 curl -sf "http://localhost:${NEXTCLOUD_PORT}/status.php" > /dev/null; then
    echo "✓"
else
    echo "✗"
    echo "Error: Health endpoint not responding"
    exit 1
fi

# Test 3: Check database connectivity
echo -n "  Checking database connection... "
if docker exec nextcloud_postgres pg_isready -U "${NEXTCLOUD_DB_USER}" > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "Error: Database not ready"
    exit 1
fi

# Test 4: Check Redis connectivity
echo -n "  Checking Redis connection... "
if docker exec nextcloud_redis redis-cli ping | grep -q PONG; then
    echo "✓"
else
    echo "✗"
    echo "Error: Redis not responding"
    exit 1
fi

# Test 5: Check WebDAV endpoint
echo -n "  Checking WebDAV endpoint... "
if curl -sf -u "${NEXTCLOUD_ADMIN_USER}:${NEXTCLOUD_ADMIN_PASSWORD}" \
        -X PROPFIND \
        "http://localhost:${NEXTCLOUD_PORT}/remote.php/dav/" > /dev/null; then
    echo "✓"
else
    echo "✗"
    echo "Error: WebDAV endpoint not accessible"
    exit 1
fi

echo "All smoke tests passed!"
exit 0
