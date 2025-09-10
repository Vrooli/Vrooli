#!/usr/bin/env bash
################################################################################
# SageMath Smoke Tests - v2.0 Universal Contract Compliant
#
# Quick health validation (must complete in <30s)
################################################################################

set -euo pipefail

# Setup paths
PHASES_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
TEST_DIR="$(builtin cd "${PHASES_DIR}/.." && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${TEST_DIR}/.." && builtin pwd)"
APP_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"

# Source config
# shellcheck disable=SC1091
source "${RESOURCE_DIR}/config/defaults.sh"

echo "Running SageMath smoke tests..."

# Test 1: Container exists and is running
echo -n "Checking container status... "
if docker ps --format '{{.Names}}' | grep -q "^${SAGEMATH_CONTAINER_NAME}$"; then
    echo "✓"
else
    echo "✗"
    echo "Error: SageMath container is not running"
    exit 1
fi

# Test 2: Jupyter health check (with timeout)
echo -n "Checking Jupyter health... "
if timeout 5 curl -sf "http://localhost:${SAGEMATH_PORT_JUPYTER}/api" > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "Error: Jupyter interface not responding"
    exit 1
fi

# Test 3: Basic calculation via docker exec
echo -n "Testing basic calculation... "
result=$(docker exec "${SAGEMATH_CONTAINER_NAME}" sage -c "print(2+2)" 2>/dev/null || echo "error")
if [[ "$result" == "4" ]]; then
    echo "✓"
else
    echo "✗"
    echo "Error: Basic calculation failed (result: $result)"
    exit 1
fi

# Test 4: Check mounted directories
echo -n "Checking mounted directories... "
if docker exec "${SAGEMATH_CONTAINER_NAME}" test -d /home/sage/scripts && \
   docker exec "${SAGEMATH_CONTAINER_NAME}" test -d /home/sage/notebooks && \
   docker exec "${SAGEMATH_CONTAINER_NAME}" test -d /home/sage/outputs; then
    echo "✓"
else
    echo "✗"
    echo "Error: Mounted directories not accessible"
    exit 1
fi

# Test 5: Verify SageMath version
echo -n "Checking SageMath version... "
version=$(docker exec "${SAGEMATH_CONTAINER_NAME}" sage --version 2>/dev/null | head -1 || echo "error")
if [[ "$version" == *"SageMath"* ]]; then
    echo "✓ ($version)"
else
    echo "✗"
    echo "Error: Could not determine SageMath version"
    exit 1
fi

echo ""
echo "All smoke tests passed!"
exit 0