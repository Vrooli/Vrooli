#!/usr/bin/env bash
# Mesa Smoke Tests
# Quick health validation per v2.0 contract (<30s)

set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
readonly TEST_DIR

# Source the CLI for testing
source "${TEST_DIR}/cli.sh"

echo "=== Mesa Smoke Tests ==="
echo "Testing basic health and responsiveness..."

# Test 1: Service is installed
echo -n "1. Checking Mesa installation... "
if [[ -d "${TEST_DIR}/venv" ]]; then
    echo "✓"
else
    echo "✗ (Not installed)"
    echo "  Run: vrooli resource mesa manage install"
    exit 1
fi

# Test 2: Service can start
echo -n "2. Checking if Mesa can start... "
if mesa::is_running; then
    echo "✓ (Already running)"
elif mesa::start > /dev/null 2>&1; then
    echo "✓ (Started successfully)"
    sleep 5  # Give it time to initialize
else
    echo "✗ (Failed to start)"
    exit 1
fi

# Test 3: Health endpoint responds
echo -n "3. Checking health endpoint... "
if timeout 5 curl -sf "http://localhost:9512/health" > /dev/null; then
    echo "✓"
else
    echo "✗ (Health check failed)"
    exit 1
fi

# Test 4: API is accessible
echo -n "4. Checking API accessibility... "
if timeout 5 curl -sf "http://localhost:9512/models" > /dev/null; then
    echo "✓"
else
    echo "✗ (API not responding)"
    exit 1
fi

# Test 5: Can list models
echo -n "5. Checking model listing... "
if mesa::content_list > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗ (Failed to list models)"
    exit 1
fi

echo ""
echo "=== Smoke Tests Passed ==="
echo "Mesa is healthy and responsive!"
exit 0