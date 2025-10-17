#!/usr/bin/env bash
################################################################################
# AudioCraft Smoke Tests
# Quick validation of basic functionality
################################################################################
set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "$SCRIPT_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Load libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

echo "🔥 AudioCraft Smoke Tests"
echo "========================"

# Track failures
FAILED=0

# Test 1: Health check
echo -n "Testing health endpoint... "
if timeout 5 curl -sf "http://localhost:${AUDIOCRAFT_PORT}/health" > /dev/null 2>&1; then
    echo "✅ PASSED"
else
    echo "❌ FAILED"
    FAILED=$((FAILED + 1))
fi

# Test 2: Container running
echo -n "Testing container status... "
if docker ps | grep -q "${AUDIOCRAFT_CONTAINER_NAME}"; then
    echo "✅ PASSED"
else
    echo "❌ FAILED"
    FAILED=$((FAILED + 1))
fi

# Test 3: Port accessible
echo -n "Testing port accessibility... "
if nc -z localhost "${AUDIOCRAFT_PORT}" 2>/dev/null; then
    echo "✅ PASSED"
else
    echo "❌ FAILED"
    FAILED=$((FAILED + 1))
fi

# Test 4: Models endpoint
echo -n "Testing models endpoint... "
RESPONSE=$(curl -sf "http://localhost:${AUDIOCRAFT_PORT}/api/models" 2>/dev/null || echo "FAILED")
if [[ "$RESPONSE" != "FAILED" ]] && echo "$RESPONSE" | grep -q "musicgen"; then
    echo "✅ PASSED"
else
    echo "❌ FAILED"
    FAILED=$((FAILED + 1))
fi

# Test 5: Response time
echo -n "Testing response time (<1s)... "
START_TIME=$(date +%s%N)
curl -sf "http://localhost:${AUDIOCRAFT_PORT}/health" > /dev/null 2>&1
END_TIME=$(date +%s%N)
ELAPSED=$((($END_TIME - $START_TIME) / 1000000))  # Convert to milliseconds

if [[ $ELAPSED -lt 1000 ]]; then
    echo "✅ PASSED (${ELAPSED}ms)"
else
    echo "❌ FAILED (${ELAPSED}ms)"
    FAILED=$((FAILED + 1))
fi

# Summary
echo "========================"
if [[ $FAILED -eq 0 ]]; then
    echo "✅ All smoke tests passed"
    exit 0
else
    echo "❌ $FAILED smoke test(s) failed"
    exit 1
fi