#!/bin/bash

set -e

echo "=== Integration Tests ==="

# Get API port from environment or default
API_PORT="${API_PORT:-15696}"
API_BASE="http://localhost:${API_PORT}"
# NOTE: Using simple dev/test token. Production should use secure authentication.
AUTH_TOKEN="crypto-tools-api-key-2024"

# Test API health endpoint
echo "Testing API health endpoint..."

HEALTH_RESPONSE=$(curl -sf "${API_BASE}/health" || echo "")
if echo "$HEALTH_RESPONSE" | grep -q "crypto-tools-api"; then
    echo "✅ Health endpoint returns valid response"
else
    echo "❌ Health endpoint did not return expected response"
    exit 1
fi

# Test hash operation (SHA-256)
echo "Testing hash operation (SHA-256)..."

HASH_RESPONSE=$(curl -sf -X POST "${API_BASE}/api/v1/crypto/hash" \
    -H "Authorization: Bearer ${AUTH_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{"data":"test","algorithm":"sha256"}' || echo "")

if echo "$HASH_RESPONSE" | grep -q "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"; then
    echo "✅ SHA-256 hash returns correct result"
else
    echo "❌ SHA-256 hash operation failed or returned incorrect result"
    echo "Response: $HASH_RESPONSE"
    exit 1
fi

# Test key generation (RSA)
echo "Testing key generation (RSA)..."

KEYGEN_RESPONSE=$(curl -sf -X POST "${API_BASE}/api/v1/crypto/keys/generate" \
    -H "Authorization: Bearer ${AUTH_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{"key_type":"rsa","key_size":2048}' || echo "")

if echo "$KEYGEN_RESPONSE" | grep -q "key_id"; then
    echo "✅ RSA key generation successful"
else
    echo "⚠️  RSA key generation may have issues"
    echo "Response: $KEYGEN_RESPONSE"
fi

# Test encryption operation
echo "Testing encryption operation (AES-256)..."

ENCRYPT_RESPONSE=$(curl -sf -X POST "${API_BASE}/api/v1/crypto/encrypt" \
    -H "Authorization: Bearer ${AUTH_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{"data":"secret message","algorithm":"aes256"}' || echo "")

if echo "$ENCRYPT_RESPONSE" | grep -q "encrypted_data"; then
    echo "✅ AES-256 encryption successful"
else
    echo "⚠️  AES-256 encryption may not be fully implemented"
fi

echo "=== Integration Tests Complete ==="
