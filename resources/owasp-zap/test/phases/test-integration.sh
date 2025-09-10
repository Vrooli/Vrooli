#!/usr/bin/env bash
################################################################################
# OWASP ZAP Resource - Integration Tests
# Tests interaction with external systems and full functionality
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ZAP_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Use the resource CLI
CLI_PATH="${ZAP_DIR}/cli.sh"

echo "Running OWASP ZAP integration tests..."

# Test 1: Docker availability
echo -n "Test 1 - Docker available: "
if command -v docker &>/dev/null; then
    echo "PASS"
else
    echo "SKIP - Docker not available"
    exit 0
fi

# Test 2: Can pull ZAP image (or verify it exists)
echo -n "Test 2 - ZAP image check: "
IMAGE="ghcr.io/zaproxy/zaproxy:stable"
if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "${IMAGE}"; then
    echo "PASS - Image already exists"
elif docker pull "${IMAGE}" &>/dev/null; then
    echo "PASS - Image pulled successfully"
else
    echo "SKIP - Cannot pull image (may be offline)"
    exit 0
fi

# Test 3: Port availability
echo -n "Test 3 - Port 8180 available: "
if ! netstat -tlnp 2>/dev/null | grep -q ":8180 "; then
    echo "PASS"
else
    echo "WARN - Port 8180 in use, tests may fail"
fi

# Test 4: Can create data directory
echo -n "Test 4 - Data directory writable: "
TEST_DIR="/tmp/zap-test-$$"
if mkdir -p "${TEST_DIR}" && touch "${TEST_DIR}/test" && rm -rf "${TEST_DIR}"; then
    echo "PASS"
else
    echo "FAIL - Cannot create data directory"
    exit 1
fi

# Test 5: CLI manage commands exist
echo -n "Test 5 - Manage commands available: "
if "${CLI_PATH}" help 2>&1 | grep -q "manage"; then
    echo "PASS"
else
    echo "FAIL - Manage commands not found"
    exit 1
fi

# Test 6: Test command structure
echo -n "Test 6 - Test commands available: "
if "${CLI_PATH}" help 2>&1 | grep -q "test"; then
    echo "PASS"
else
    echo "FAIL - Test commands not found"
    exit 1
fi

echo "All integration tests passed!"
exit 0