#!/usr/bin/env bash
# Smoke tests for Fall Foliage Explorer
# Quick validation that core services are running

set -e

# Use registered ports for fall-foliage-explorer
FOLIAGE_API_PORT=17175
FOLIAGE_UI_PORT=36003

echo "🔥 Running smoke tests..."

# Test 1: API health check
echo "  [1/4] Testing API health..."
HEALTH_RESPONSE=$(curl -sf http://localhost:${FOLIAGE_API_PORT}/health || echo "FAILED")
if [[ "$HEALTH_RESPONSE" == "FAILED" ]]; then
    echo "    ❌ API health check failed"
    exit 1
fi
echo "    ✅ API is healthy"

# Test 2: Regions endpoint
echo "  [2/4] Testing regions endpoint..."
if curl -sf http://localhost:${FOLIAGE_API_PORT}/api/regions > /dev/null; then
    echo "    ✅ Regions endpoint working"
else
    echo "    ❌ Regions endpoint failed"
    exit 1
fi

# Test 3: UI accessibility
echo "  [3/4] Testing UI accessibility..."
if curl -sf http://localhost:${FOLIAGE_UI_PORT}/ > /dev/null; then
    echo "    ✅ UI is accessible"
else
    echo "    ❌ UI not accessible"
    exit 1
fi

# Test 4: Database connectivity (via API)
echo "  [4/4] Testing database connectivity..."
if echo "$HEALTH_RESPONSE" | grep -q '"database":"healthy"'; then
    echo "    ✅ Database is connected"
else
    echo "    ❌ Database connection issue"
    exit 1
fi

echo "✅ All smoke tests passed!"
