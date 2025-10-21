#!/bin/bash
# API tests for Web Scraper Manager

set -e

echo "Running API tests for Web Scraper Manager..."

# Get the API port from vrooli status (don't trust environment variable from other scenarios)
API_PORT=$(vrooli scenario status web-scraper-manager 2>/dev/null | grep "API_PORT:" | awk '{print $2}' || echo '')
if [ -z "$API_PORT" ]; then
    echo "❌ API_PORT not found. Is the scenario running?"
    echo "   Try: vrooli scenario start web-scraper-manager"
    exit 1
fi
API_URL="http://localhost:${API_PORT}"

# Wait for API to be ready
echo "Waiting for API at ${API_URL}..."
for i in {1..30}; do
    if curl -sf "${API_URL}/health" > /dev/null 2>&1; then
        echo "✓ API is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ API failed to start"
        exit 1
    fi
    sleep 1
done

# Test health endpoint
echo "✓ Testing health endpoint..."
HEALTH=$(curl -sf "${API_URL}/health") || { echo "❌ Health check failed"; exit 1; }
if ! echo "$HEALTH" | jq -e '.status == "healthy"' > /dev/null 2>&1; then
    echo "❌ Health status not healthy. Response: $HEALTH"
    exit 1
fi
if ! echo "$HEALTH" | jq -e '.dependencies.database.connected == true' > /dev/null 2>&1; then
    echo "❌ Database not connected. Response: $HEALTH"
    exit 1
fi

# Test agents endpoint
echo "✓ Testing agents list endpoint..."
curl -sf "${API_URL}/api/agents" | jq -e '.success == true' > /dev/null || { echo "❌ Agents list failed"; exit 1; }

# Test platforms endpoint
echo "✓ Testing platforms endpoint..."
PLATFORMS=$(curl -sf "${API_URL}/api/platforms") || { echo "❌ Platforms endpoint failed"; exit 1; }
echo "$PLATFORMS" | jq -e '.success == true' > /dev/null || { echo "❌ Platforms response not successful"; exit 1; }
echo "$PLATFORMS" | jq -e '.data | length >= 3' > /dev/null || { echo "❌ Expected at least 3 platforms"; exit 1; }

# Test status endpoint
echo "✓ Testing status endpoint..."
curl -sf "${API_URL}/api/status" | jq -e '.success == true' > /dev/null || { echo "❌ Status endpoint failed"; exit 1; }

# Test metrics endpoint
echo "✓ Testing metrics endpoint..."
curl -sf "${API_URL}/api/metrics" | jq -e '.success == true' > /dev/null || { echo "❌ Metrics endpoint failed"; exit 1; }

echo "✅ API tests completed"
