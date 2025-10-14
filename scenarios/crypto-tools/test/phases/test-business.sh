#!/bin/bash

set -e

echo "=== Business Logic Tests ==="

# Get API port from environment or default
API_PORT="${API_PORT:-15696}"
API_BASE="http://localhost:${API_PORT}"
# NOTE: Using simple dev/test token. Production should use secure authentication.
AUTH_TOKEN="crypto-tools-api-key-2024"

# Test P0 Requirement: Hashing operations
echo "Testing P0: Multiple hash algorithms..."

for algo in sha256 sha512 md5; do
    HASH_RESPONSE=$(curl -sf -X POST "${API_BASE}/api/v1/crypto/hash" \
        -H "Authorization: Bearer ${AUTH_TOKEN}" \
        -H "Content-Type: application/json" \
        -d "{\"data\":\"test\",\"algorithm\":\"${algo}\"}" || echo "")

    if echo "$HASH_RESPONSE" | grep -q "hash"; then
        echo "✅ Hash algorithm ${algo} working"
    else
        echo "❌ Hash algorithm ${algo} failed"
        exit 1
    fi
done

# Test P0 Requirement: Key generation
echo "Testing P0: Key generation..."

KEYGEN_RESPONSE=$(curl -sf -X POST "${API_BASE}/api/v1/crypto/keys/generate" \
    -H "Authorization: Bearer ${AUTH_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{"key_type":"rsa","key_size":2048}' || echo "")

if echo "$KEYGEN_RESPONSE" | grep -q "key_id"; then
    echo "✅ Key generation produces valid key IDs"
else
    echo "❌ Key generation did not return key_id"
    exit 1
fi

# Test P0 Requirement: CLI interface
echo "Testing P0: CLI components..."

if [[ -f "cli/crypto-tools" ]] || [[ -f "cli/install.sh" ]]; then
    echo "✅ CLI components present"
else
    echo "❌ CLI components missing"
    exit 1
fi

# Test P0 Requirement: API documentation
echo "Testing P0: API documentation..."

if grep -q "/api/v1/crypto/hash" README.md 2>/dev/null; then
    echo "✅ API endpoints documented in README"
else
    echo "⚠️  API endpoints may not be fully documented"
fi

# Test business value: Response time < 100ms for hashing
echo "Testing business metric: Hash response time..."

START_TIME=$(date +%s%N)
curl -sf -X POST "${API_BASE}/api/v1/crypto/hash" \
    -H "Authorization: Bearer ${AUTH_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{"data":"test","algorithm":"sha256"}' > /dev/null 2>&1
END_TIME=$(date +%s%N)
ELAPSED_MS=$(( (END_TIME - START_TIME) / 1000000 ))

if [[ $ELAPSED_MS -lt 100 ]]; then
    echo "✅ Hash operation completes in ${ELAPSED_MS}ms (target: <100ms)"
else
    echo "⚠️  Hash operation took ${ELAPSED_MS}ms (target: <100ms)"
fi

# Test PRD completeness
echo "Testing PRD completeness..."

if grep -q "## P0 Requirements" PRD.md 2>/dev/null; then
    echo "✅ PRD has P0 requirements section"
else
    echo "⚠️  PRD may be incomplete"
fi

echo "=== Business Logic Tests Complete ==="
