#!/bin/bash
# Eclipse Ditto Smoke Tests - Quick health validation

set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

echo "Eclipse Ditto Smoke Tests"
echo "========================="

# Track failures
failed=0

# Test 1: Docker availability
echo -n "1. Docker daemon... "
if docker info &>/dev/null; then
    echo "✓"
else
    echo "✗ Docker not available"
    failed=$((failed + 1))
fi

# Test 2: Docker Compose availability
echo -n "2. Docker Compose... "
if command -v docker-compose &>/dev/null || docker compose version &>/dev/null; then
    echo "✓"
else
    echo "✗ Docker Compose not available"
    failed=$((failed + 1))
fi

# Test 3: Port availability
echo -n "3. Port ${DITTO_GATEWAY_PORT} configuration... "
if [[ -n "${DITTO_GATEWAY_PORT}" ]] && [[ "${DITTO_GATEWAY_PORT}" -gt 1024 ]]; then
    echo "✓"
else
    echo "✗ Invalid port configuration"
    failed=$((failed + 1))
fi

# Test 4: Health check (if running)
echo -n "4. Health endpoint (if running)... "
if docker ps --format "{{.Names}}" | grep -q "ditto-gateway"; then
    if timeout 5 curl -sf "http://localhost:${DITTO_GATEWAY_PORT}/health" &>/dev/null; then
        echo "✓"
    else
        echo "✗ Health check failed"
        failed=$((failed + 1))
    fi
else
    echo "⚠ Not running (skipped)"
fi

# Test 5: Configuration files
echo -n "5. Configuration files... "
if [[ -f "${RESOURCE_DIR}/config/defaults.sh" ]] && \
   [[ -f "${RESOURCE_DIR}/config/schema.json" ]] && \
   [[ -f "${RESOURCE_DIR}/config/runtime.json" ]]; then
    echo "✓"
else
    echo "✗ Missing configuration files"
    failed=$((failed + 1))
fi

# Report results
echo ""
if [[ $failed -gt 0 ]]; then
    echo "❌ Smoke tests failed: $failed test(s) failed"
    exit 1
else
    echo "✅ All smoke tests passed"
fi