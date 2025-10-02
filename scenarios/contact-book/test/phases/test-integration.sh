#!/bin/bash

set -e

echo "=== Integration Tests ==="

# Get API port dynamically
API_PORT=$(lsof -ti:19313 2>/dev/null | head -1 | xargs -I {} lsof -nP -p {} 2>/dev/null | grep LISTEN | awk '{print $9}' | cut -d: -f2 | head -1)
if [ -z "$API_PORT" ]; then
    # Fallback to checking service.json
    API_PORT=$(jq -r '.services.api.port // empty' .vrooli/service.json 2>/dev/null)
fi
if [ -z "$API_PORT" ]; then
    API_PORT="19313"  # Final fallback
fi

API_URL="http://localhost:${API_PORT}"

echo "Testing against API at: $API_URL"

# Wait for API to be ready
echo "Waiting for API to be ready..."
for i in {1..30}; do
    if curl -sf "${API_URL}/health" > /dev/null 2>&1; then
        echo "✅ API is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ API failed to start within 30 seconds"
        exit 1
    fi
    sleep 1
done

# Test health endpoint
echo "Testing health endpoint..."
response=$(curl -sf "${API_URL}/health")
if echo "$response" | jq -e '.status == "healthy"' > /dev/null 2>&1; then
    echo "✅ Health endpoint working"
else
    echo "❌ Health endpoint returned unexpected response"
    exit 1
fi

# Test contacts endpoint
echo "Testing contacts endpoint..."
response=$(curl -sf "${API_URL}/api/v1/contacts")
if echo "$response" | jq -e '.persons' > /dev/null 2>&1; then
    echo "✅ Contacts endpoint working"
else
    echo "❌ Contacts endpoint failed"
    exit 1
fi

# Test search endpoint
echo "Testing search endpoint..."
response=$(curl -sf -X POST "${API_URL}/api/v1/search" \
    -H "Content-Type: application/json" \
    -d '{"query":"test","limit":5}')
if echo "$response" | jq -e '.results' > /dev/null 2>&1; then
    echo "✅ Search endpoint working"
else
    echo "❌ Search endpoint failed"
    exit 1
fi

# Test relationships endpoint
echo "Testing relationships endpoint..."
response=$(curl -sf "${API_URL}/api/v1/relationships")
if echo "$response" | jq -e '.relationships' > /dev/null 2>&1; then
    echo "✅ Relationships endpoint working"
else
    echo "❌ Relationships endpoint failed"
    exit 1
fi

# Test analytics endpoint
echo "Testing analytics endpoint..."
response=$(curl -sf "${API_URL}/api/v1/analytics")
if echo "$response" | jq -e 'has("analytics")' > /dev/null 2>&1; then
    echo "✅ Analytics endpoint working"
else
    echo "❌ Analytics endpoint failed"
    exit 1
fi

# Test CLI commands
echo "Testing CLI commands..."
if command -v contact-book > /dev/null 2>&1; then
    # Test help
    if contact-book help > /dev/null 2>&1; then
        echo "✅ CLI help command working"
    else
        echo "❌ CLI help command failed"
        exit 1
    fi

    # Test status
    if contact-book status --json > /dev/null 2>&1; then
        echo "✅ CLI status command working"
    else
        echo "❌ CLI status command failed"
        exit 1
    fi

    # Test list
    if contact-book list --json --limit 1 > /dev/null 2>&1; then
        echo "✅ CLI list command working"
    else
        echo "❌ CLI list command failed"
        exit 1
    fi
else
    echo "⚠️  CLI not in PATH, skipping CLI tests"
fi

echo "✅ All integration tests passed"
