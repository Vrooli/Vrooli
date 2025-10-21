#!/bin/bash
set -e

echo "=== Testing Integration ==="

# Get the API port from environment or use default
API_PORT="${API_PORT:-18538}"
API_URL="http://localhost:${API_PORT}"

# Function to check if API is ready
wait_for_api() {
    local max_attempts=10
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -sf "${API_URL}/health" > /dev/null 2>&1; then
            echo "API is ready on port ${API_PORT}"
            return 0
        fi
        echo "Waiting for API... (attempt ${attempt}/${max_attempts})"
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo "❌ API failed to start on port ${API_PORT}"
    return 1
}

# Wait for API to be ready
wait_for_api || exit 1

# Test health endpoint
echo "Testing health endpoint..."
if ! curl -sf "${API_URL}/health" | grep -q "healthy"; then
    echo "❌ Health check failed"
    exit 1
fi
echo "✅ Health endpoint working"

# Test categories endpoint
echo "Testing categories endpoint..."
CATEGORIES=$(curl -sf "${API_URL}/api/categories")
if [ -z "$CATEGORIES" ]; then
    echo "❌ Categories endpoint failed"
    exit 1
fi
echo "✅ Categories endpoint working"

# Test search endpoint
echo "Testing search endpoint..."
SEARCH_RESULT=$(curl -sf -X POST "${API_URL}/api/search" \
    -H "Content-Type: application/json" \
    -d '{"query":"test","lat":40.7128,"lon":-74.0060,"radius":5}')

if [ -z "$SEARCH_RESULT" ]; then
    echo "❌ Search endpoint failed"
    exit 1
fi

# Verify response structure matches PRD spec (places and sources fields)
if ! echo "$SEARCH_RESULT" | grep -q '"places"'; then
    echo "❌ Search response missing 'places' field (PRD contract violation)"
    exit 1
fi
if ! echo "$SEARCH_RESULT" | grep -q '"sources"'; then
    echo "❌ Search response missing 'sources' field (PRD contract violation)"
    exit 1
fi
echo "✅ Search endpoint working"

# Test place details endpoint
echo "Testing place details endpoint..."
PLACE_DETAILS=$(curl -sf "${API_URL}/api/places/123")
if [ -z "$PLACE_DETAILS" ]; then
    echo "❌ Place details endpoint failed"
    exit 1
fi
echo "✅ Place details endpoint working"

# Test CORS headers (with valid origin)
echo "Testing CORS headers..."
CORS_HEADER=$(curl -sI -H "Origin: http://localhost:3000" "${API_URL}/health" | grep -i "access-control-allow-origin" || true)
if [ -z "$CORS_HEADER" ]; then
    echo "❌ CORS headers missing with valid origin"
    exit 1
fi
# Verify CORS is restricted (wildcard should not be present)
if echo "$CORS_HEADER" | grep -q "Access-Control-Allow-Origin: \*"; then
    echo "❌ CORS wildcard detected (security issue)"
    exit 1
fi
echo "✅ CORS headers present and properly restricted"

# Test natural language search
echo "Testing natural language search..."
NL_RESULT=$(curl -sf -X POST "${API_URL}/api/search" \
    -H "Content-Type: application/json" \
    -d '{"query":"vegan restaurants within 2 miles","lat":40.7128,"lon":-74.0060}')

if [ -z "$NL_RESULT" ]; then
    echo "❌ Natural language search failed"
    exit 1
fi
echo "✅ Natural language search working"

# Test discover endpoint
echo "Testing discover endpoint..."
DISCOVER_RESULT=$(curl -sf -X POST "${API_URL}/api/discover" \
    -H "Content-Type: application/json" \
    -d '{"lat":40.7128,"lon":-74.0060}')

if [ -z "$DISCOVER_RESULT" ]; then
    echo "❌ Discover endpoint failed"
    exit 1
fi
echo "✅ Discover endpoint working"

echo "✅ Integration tests passed"