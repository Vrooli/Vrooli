#!/usr/bin/env bash
# QwenCoder Smoke Tests - Quick validation (<30s)

set -euo pipefail

# Setup
readonly PHASES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly TEST_DIR="$(dirname "${PHASES_DIR}")"
readonly RESOURCE_DIR="$(dirname "${TEST_DIR}")"

# Source libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"

echo "Running QwenCoder smoke tests..."

# Test counter
tests_passed=0
tests_failed=0

# Test 1: Service is running
echo -n "1. Checking if service is running... "
if qwencoder_is_running; then
    echo "✓"
    ((tests_passed++))
else
    echo "✗ (starting service)"
    if qwencoder_start; then
        echo "   Service started successfully"
        ((tests_passed++))
    else
        echo "   Failed to start service"
        ((tests_failed++))
    fi
fi

# Test 2: Health endpoint responds
echo -n "2. Testing health endpoint... "
if timeout 5 curl -sf "http://localhost:${QWENCODER_PORT}/health" > /dev/null 2>&1; then
    echo "✓"
    ((tests_passed++))
else
    echo "✗"
    ((tests_failed++))
fi

# Test 3: Health returns valid JSON
echo -n "3. Validating health response... "
health_response=$(timeout 5 curl -sf "http://localhost:${QWENCODER_PORT}/health" 2>/dev/null || echo "{}")
if echo "${health_response}" | jq -e '.status' > /dev/null 2>&1; then
    echo "✓"
    ((tests_passed++))
else
    echo "✗"
    ((tests_failed++))
fi

# Test 4: Models endpoint available
echo -n "4. Testing models endpoint... "
if timeout 5 curl -sf "http://localhost:${QWENCODER_PORT}/models" > /dev/null 2>&1; then
    echo "✓"
    ((tests_passed++))
else
    echo "✗"
    ((tests_failed++))
fi

# Test 5: Basic completion works
echo -n "5. Testing basic completion... "
completion_response=$(timeout 10 curl -sf -X POST "http://localhost:${QWENCODER_PORT}/v1/completions" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "qwencoder-1.5b",
        "prompt": "print(",
        "max_tokens": 10
    }' 2>/dev/null || echo "{}")

if echo "${completion_response}" | jq -e '.choices' > /dev/null 2>&1; then
    echo "✓"
    ((tests_passed++))
else
    echo "✗"
    ((tests_failed++))
fi

# Summary
echo ""
echo "Smoke Test Results:"
echo "==================="
echo "Passed: ${tests_passed}"
echo "Failed: ${tests_failed}"

if [[ ${tests_failed} -eq 0 ]]; then
    echo "Status: ✓ All smoke tests passed!"
    exit 0
else
    echo "Status: ✗ Some tests failed"
    exit 1
fi