#!/usr/bin/env bash
# Test script for the idea-generator API endpoints

set -euo pipefail

API_URL="${1:-http://localhost:8500}"

echo "Testing Idea Generator API at $API_URL"

# Test health endpoint
echo "Testing /health endpoint..."
if curl -sf "$API_URL/health" >/dev/null; then
    echo "✅ Health endpoint responding"
else
    echo "❌ Health endpoint failed"
    exit 1
fi

# Test status endpoint
echo "Testing /status endpoint..."
if curl -sf "$API_URL/status" >/dev/null; then
    echo "✅ Status endpoint responding"
else
    echo "❌ Status endpoint failed"
    exit 1
fi

# Test campaigns endpoint
echo "Testing /campaigns endpoint..."
if curl -sf "$API_URL/campaigns" >/dev/null; then
    echo "✅ Campaigns endpoint responding"
else
    echo "❌ Campaigns endpoint failed"
    exit 1
fi

# Test ideas endpoint
echo "Testing /ideas endpoint..."
if curl -sf "$API_URL/ideas" >/dev/null; then
    echo "✅ Ideas endpoint responding"
else
    echo "❌ Ideas endpoint failed"
    exit 1
fi

# Test workflows endpoint
echo "Testing /workflows endpoint..."
if curl -sf "$API_URL/workflows" >/dev/null; then
    echo "✅ Workflows endpoint responding"
else
    echo "❌ Workflows endpoint failed"
    exit 1
fi

echo "✅ All API endpoints are responding correctly"