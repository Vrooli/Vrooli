#!/bin/bash
# Integration testing phase for symbol-search scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running integration tests for symbol-search..."

# Test database connectivity
echo "Checking database connectivity..."
if [ -z "$POSTGRES_URL" ]; then
    echo "❌ POSTGRES_URL not set - integration tests require database"
    exit 1
fi

# Test API endpoints
echo "Testing API endpoints..."
API_PORT="${API_PORT:-8080}"
API_URL="http://localhost:${API_PORT}"

# Wait for API to be ready
max_wait=30
waited=0
while [ $waited -lt $max_wait ]; do
    if curl -s "${API_URL}/health" > /dev/null 2>&1; then
        echo "✅ API is ready"
        break
    fi
    sleep 1
    waited=$((waited + 1))
done

if [ $waited -eq $max_wait ]; then
    echo "❌ API did not become ready within ${max_wait}s"
    exit 1
fi

# Test health endpoint
echo "Testing /health endpoint..."
response=$(curl -s "${API_URL}/health")
if echo "$response" | grep -q '"status":"healthy"'; then
    echo "✅ Health check passed"
else
    echo "❌ Health check failed: $response"
    exit 1
fi

# Test search endpoint
echo "Testing /api/search endpoint..."
response=$(curl -s "${API_URL}/api/search?q=LATIN&limit=10")
if echo "$response" | grep -q '"characters"'; then
    echo "✅ Search endpoint passed"
else
    echo "❌ Search endpoint failed: $response"
    exit 1
fi

# Test categories endpoint
echo "Testing /api/categories endpoint..."
response=$(curl -s "${API_URL}/api/categories")
if echo "$response" | grep -q '"categories"'; then
    echo "✅ Categories endpoint passed"
else
    echo "❌ Categories endpoint failed: $response"
    exit 1
fi

# Test blocks endpoint
echo "Testing /api/blocks endpoint..."
response=$(curl -s "${API_URL}/api/blocks")
if echo "$response" | grep -q '"blocks"'; then
    echo "✅ Blocks endpoint passed"
else
    echo "❌ Blocks endpoint failed: $response"
    exit 1
fi

# Test character detail endpoint
echo "Testing /api/character/:codepoint endpoint..."
response=$(curl -s "${API_URL}/api/character/U+0041")
if echo "$response" | grep -q '"character"'; then
    echo "✅ Character detail endpoint passed"
else
    echo "❌ Character detail endpoint failed: $response"
    exit 1
fi

# Test bulk range endpoint
echo "Testing /api/bulk/range endpoint..."
response=$(curl -s -X POST "${API_URL}/api/bulk/range" \
    -H "Content-Type: application/json" \
    -d '{"ranges":[{"start":"U+0041","end":"U+005A"}]}')
if echo "$response" | grep -q '"characters"'; then
    echo "✅ Bulk range endpoint passed"
else
    echo "❌ Bulk range endpoint failed: $response"
    exit 1
fi

testing::phase::end_with_summary "Integration tests completed"
