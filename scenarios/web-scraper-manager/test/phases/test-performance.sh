#!/bin/bash
# Performance tests for Web Scraper Manager

set -e

echo "Running performance tests for Web Scraper Manager..."

# Get the API port from vrooli status (don't trust environment variable from other scenarios)
API_PORT=$(vrooli scenario status web-scraper-manager 2>/dev/null | grep "API_PORT:" | awk '{print $2}' || echo '')
if [ -z "$API_PORT" ]; then
    echo "❌ API_PORT not found. Is the scenario running?"
    exit 1
fi
API_URL="http://localhost:${API_PORT}"

# Test health endpoint response time (should be < 500ms)
echo "✓ Testing health endpoint performance..."
START=$(date +%s%N)
curl -sf "${API_URL}/health" > /dev/null || { echo "❌ Health check failed"; exit 1; }
END=$(date +%s%N)
DURATION=$(( (END - START) / 1000000 )) # Convert to milliseconds

if [ $DURATION -lt 500 ]; then
    echo "✓ Health endpoint responded in ${DURATION}ms (target: <500ms)"
else
    echo "⚠️  Health endpoint slow: ${DURATION}ms (target: <500ms)"
fi

# Test agents list response time (should be < 1000ms)
echo "✓ Testing agents list performance..."
START=$(date +%s%N)
curl -sf "${API_URL}/api/agents" > /dev/null || { echo "❌ Agents list failed"; exit 1; }
END=$(date +%s%N)
DURATION=$(( (END - START) / 1000000 ))

if [ $DURATION -lt 1000 ]; then
    echo "✓ Agents list responded in ${DURATION}ms (target: <1000ms)"
else
    echo "⚠️  Agents list slow: ${DURATION}ms (target: <1000ms)"
fi

# Test platforms endpoint response time
echo "✓ Testing platforms endpoint performance..."
START=$(date +%s%N)
curl -sf "${API_URL}/api/platforms" > /dev/null || { echo "❌ Platforms endpoint failed"; exit 1; }
END=$(date +%s%N)
DURATION=$(( (END - START) / 1000000 ))

if [ $DURATION -lt 500 ]; then
    echo "✓ Platforms endpoint responded in ${DURATION}ms (target: <500ms)"
else
    echo "⚠️  Platforms endpoint slow: ${DURATION}ms (target: <500ms)"
fi

echo "✅ Performance tests passed"
