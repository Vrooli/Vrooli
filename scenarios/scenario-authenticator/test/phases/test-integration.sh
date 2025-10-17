#!/bin/bash
set -euo pipefail

echo "=== Running test-integration.sh ==="

# Get API port using dynamic discovery
SCENARIO_NAME="scenario-authenticator"
if command -v vrooli >/dev/null 2>&1; then
  API_PORT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || echo "15785")
else
  API_PORT=${API_PORT:-15785}
fi

API_BASE="http://localhost:${API_PORT}"

# Check if API is running
if ! curl -sf "${API_BASE}/health" >/dev/null 2>&1; then
  echo "⚠ API not running on port ${API_PORT}, skipping integration tests"
  echo "  Start with: make run"
  exit 0
fi

echo "Testing API integration at ${API_BASE}..."

# Test 1: Health endpoint
echo ""
echo "Test 1: Health check endpoint"
HEALTH=$(curl -sf "${API_BASE}/health")
if echo "$HEALTH" | jq -e '.status == "healthy" and .database == true and .redis == true' >/dev/null; then
  echo "✓ Health check passed"
else
  echo "✗ Health check failed"
  echo "Response: $HEALTH"
  exit 1
fi

# Test 2: User registration
echo ""
echo "Test 2: User registration"
RANDOM_EMAIL="test-$(date +%s)@example.com"
REG_RESPONSE=$(curl -sf -X POST "${API_BASE}/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${RANDOM_EMAIL}\",\"password\":\"TestPass123!\"}")

if echo "$REG_RESPONSE" | jq -e '.success == true' >/dev/null; then
  echo "✓ User registration passed"
  USER_TOKEN=$(echo "$REG_RESPONSE" | jq -r '.token')
else
  echo "✗ User registration failed"
  echo "Response: $REG_RESPONSE"
  exit 1
fi

# Test 3: Token validation
echo ""
echo "Test 3: Token validation"
VALIDATE_RESPONSE=$(curl -sf -H "Authorization: Bearer ${USER_TOKEN}" \
  "${API_BASE}/api/v1/auth/validate")

if echo "$VALIDATE_RESPONSE" | jq -e '.valid == true' >/dev/null; then
  echo "✓ Token validation passed"
else
  echo "✗ Token validation failed"
  echo "Response: $VALIDATE_RESPONSE"
  exit 1
fi

# Test 4: User logout
echo ""
echo "Test 4: User logout"
LOGOUT_RESPONSE=$(curl -sf -X POST -H "Authorization: Bearer ${USER_TOKEN}" \
  "${API_BASE}/api/v1/auth/logout")

if echo "$LOGOUT_RESPONSE" | jq -e '.success == true' >/dev/null; then
  echo "✓ Logout passed"
else
  echo "✗ Logout failed"
  echo "Response: $LOGOUT_RESPONSE"
  exit 1
fi

# Test 5: Token invalidation after logout
echo ""
echo "Test 5: Token invalidation after logout"
INVALIDATED_RESPONSE=$(curl -sf -H "Authorization: Bearer ${USER_TOKEN}" \
  "${API_BASE}/api/v1/auth/validate")

if echo "$INVALIDATED_RESPONSE" | jq -e '.valid == false' >/dev/null; then
  echo "✓ Token correctly invalidated after logout"
else
  echo "✗ Token still valid after logout"
  echo "Response: $INVALIDATED_RESPONSE"
  exit 1
fi

# Test 6: CORS headers
echo ""
echo "Test 6: CORS headers"
CORS_TEST=$(curl -sf -I -X OPTIONS \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: GET" \
  "${API_BASE}/health" 2>&1 | grep -i "access-control-allow-origin" || true)

if [ -n "$CORS_TEST" ]; then
  echo "✓ CORS headers configured"
else
  echo "⚠ CORS headers not found (may be expected for some configurations)"
fi

echo ""
echo "✅ test-integration.sh completed successfully"
