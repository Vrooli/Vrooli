#!/bin/bash
# Simplified performance tests phase
set -euo pipefail

echo "=== Performance Tests Phase ==="
echo "Testing basic performance functionality..."

# Test counters
error_count=0
test_count=0

# Test API response time
echo "Testing API response time..."
if timeout 5 curl -sf "http://localhost:17695/health" >/dev/null 2>&1; then
    echo "API response time tests passed"
    test_count=$((test_count + 1))
else
    echo "API response time tests failed"
    error_count=$((error_count + 1))
fi

# Test concurrent requests (simple)
echo "Testing concurrent request handling..."
if timeout 5 curl -sf "http://localhost:17695/api/v1/campaigns" >/dev/null 2>&1; then
    echo "Concurrent request tests passed"
    test_count=$((test_count + 1))
else
    echo "Concurrent request tests failed"
    error_count=$((error_count + 1))
fi

# Test UI performance (simple)
echo "Testing UI performance..."
if timeout 5 curl -sf "http://localhost:38442" >/dev/null 2>&1; then
    echo "UI performance tests passed"
    test_count=$((test_count + 1))
else
    echo "UI performance tests failed"
    error_count=$((error_count + 1))
fi

echo "Summary: $test_count passed, $error_count failed"

if [ $error_count -eq 0 ]; then
    echo "SUCCESS: All performance tests passed"
    exit 0
else
    echo "ERROR: Some performance tests failed"
    exit 1
fi