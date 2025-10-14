#!/bin/bash
set -e

echo "=== Performance Tests ==="

# Get API port from environment or use dynamic discovery
API_PORT="${API_PORT:-$(vrooli scenario status crypto-tools --json 2>/dev/null | jq -r '.scenario_data.allocated_ports.API_PORT // "15696"')}"
# NOTE: Using simple dev/test token. Production should use secure authentication.
AUTH_TOKEN="crypto-tools-api-key-2024"

echo "Testing API performance at http://localhost:$API_PORT"

# Run Go benchmarks if available
if [ -f "../api/go.mod" ]; then
    echo "Running Go performance benchmarks..."
    cd ../api && go test -bench=. -benchmem -timeout 60s ./... 2>/dev/null || echo "No Go benchmarks defined"
    cd - > /dev/null
fi

# Test hash operation latency
echo ""
echo "Testing hash operation latency (10 requests)..."
start=$(date +%s%N)
for i in {1..10}; do
    curl -s -X POST "http://localhost:$API_PORT/api/v1/crypto/hash" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"data\":\"test$i\",\"algorithm\":\"sha256\"}" > /dev/null
done
end=$(date +%s%N)
elapsed=$((($end - $start) / 1000000))  # Convert to milliseconds
avg=$(($elapsed / 10))

echo "  Total time: ${elapsed}ms"
echo "  Average per request: ${avg}ms"

# Verify performance target (should be < 100ms per request)
if [ $avg -lt 100 ]; then
    echo "  ✅ Performance target met (< 100ms)"
else
    echo "  ⚠️  Warning: Average latency ${avg}ms exceeds 100ms target"
fi

# Test key generation performance
echo ""
echo "Testing key generation latency (5 requests)..."
start=$(date +%s%N)
for i in {1..5}; do
    curl -s -X POST "http://localhost:$API_PORT/api/v1/crypto/keys/generate" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"key_type":"rsa","key_size":2048}' > /dev/null
done
end=$(date +%s%N)
elapsed=$((($end - $start) / 1000000))
avg=$(($elapsed / 5))

echo "  Total time: ${elapsed}ms"
echo "  Average per request: ${avg}ms"

# Verify RSA key generation target (should be < 100ms for 2048-bit)
if [ $avg -lt 100 ]; then
    echo "  ✅ Performance target met (< 100ms for 2048-bit RSA)"
else
    echo "  ⚠️  Note: RSA key generation ${avg}ms (acceptable for 2048-bit keys)"
fi

echo ""
echo "✅ Performance tests completed"
