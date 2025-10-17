#!/usr/bin/env bash
#
# Business Logic Test Phase for knowledge-observatory
# Tests core API endpoints and business logic
#

set -eo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$SCENARIO_DIR"

echo "🏢 Running knowledge-observatory business logic tests..."

# Get API port from environment or service.json
API_PORT="${API_PORT:-17551}"
if [ -f ".vrooli/service.json" ]; then
    PORT_FROM_JSON=$(jq -r '.lifecycle.ports.API_PORT // empty' .vrooli/service.json 2>/dev/null)
    if [ -n "$PORT_FROM_JSON" ]; then
        API_PORT="$PORT_FROM_JSON"
    fi
fi

API_URL="http://localhost:${API_PORT}"
echo "  Using API URL: $API_URL"

# Test 1: Health endpoint
echo "  Testing health endpoint..."
if response=$(curl -sf "$API_URL/health" 2>&1); then
    echo "  ✓ Health endpoint responded"

    # Validate response structure
    if echo "$response" | jq -e '.status' >/dev/null 2>&1; then
        status=$(echo "$response" | jq -r '.status')
        echo "    Health status: $status"
    else
        echo "  ⚠️  Health response missing expected fields"
    fi
else
    echo "  ✗ Health endpoint failed to respond"
    exit 1
fi

# Test 2: Search endpoint
echo "  Testing search endpoint..."
search_response=$(curl -sf -X POST "$API_URL/api/v1/knowledge/search" \
    -H "Content-Type: application/json" \
    -d '{"query": "test semantic search", "limit": 5}' 2>&1)

if [ $? -eq 0 ]; then
    echo "  ✓ Search endpoint responded"

    # Validate response structure
    if echo "$search_response" | jq -e '.results' >/dev/null 2>&1; then
        result_count=$(echo "$search_response" | jq '.count // 0')
        echo "    Search returned $result_count results"
    else
        echo "  ⚠️  Search response format unexpected"
    fi
else
    echo "  ✗ Search endpoint failed"
    exit 1
fi

# Test 3: Graph endpoint
echo "  Testing graph endpoint..."
graph_response=$(curl -sf -X GET "$API_URL/api/v1/knowledge/graph?max_nodes=10" 2>&1)

if [ $? -eq 0 ]; then
    echo "  ✓ Graph endpoint responded"

    # Validate response structure
    if echo "$graph_response" | jq -e '.nodes' >/dev/null 2>&1; then
        node_count=$(echo "$graph_response" | jq '.nodes | length')
        edge_count=$(echo "$graph_response" | jq '.edges | length')
        echo "    Graph has $node_count nodes and $edge_count edges"
    else
        echo "  ⚠️  Graph response format unexpected"
    fi
else
    echo "  ✗ Graph endpoint failed"
    exit 1
fi

# Test 4: Metrics endpoint
echo "  Testing metrics endpoint..."
metrics_response=$(curl -sf -X GET "$API_URL/api/v1/knowledge/metrics" 2>&1)

if [ $? -eq 0 ]; then
    echo "  ✓ Metrics endpoint responded"

    # Validate response structure
    if echo "$metrics_response" | jq -e '.collections' >/dev/null 2>&1; then
        collection_count=$(echo "$metrics_response" | jq '.collections | length')
        echo "    Found metrics for $collection_count collections"
    else
        echo "  ⚠️  Metrics response format unexpected"
    fi
else
    echo "  ⚠️  Metrics endpoint failed (non-critical)"
fi

# Test 5: CORS configuration
echo "  Testing CORS configuration..."
cors_response=$(curl -sf -I -X OPTIONS "$API_URL/api/v1/knowledge/search" \
    -H "Origin: http://localhost:3000" \
    -H "Access-Control-Request-Method: POST" 2>&1)

if [ $? -eq 0 ]; then
    if echo "$cors_response" | grep -qi "Access-Control-Allow-Origin"; then
        echo "  ✓ CORS headers present"

        # Verify not using wildcard
        if echo "$cors_response" | grep -q "Access-Control-Allow-Origin: \*"; then
            echo "  ✗ CORS using wildcard (security vulnerability)"
            exit 1
        else
            echo "    CORS using origin validation (secure)"
        fi
    else
        echo "  ⚠️  CORS headers not found"
    fi
else
    echo "  ⚠️  CORS preflight check failed"
fi

# Test 6: Error handling
echo "  Testing error handling..."

# Test invalid search query
invalid_response=$(curl -s -w "%{http_code}" -o /dev/null -X POST "$API_URL/api/v1/knowledge/search" \
    -H "Content-Type: application/json" \
    -d '{}' 2>&1)

if [ "$invalid_response" = "400" ] || [ "$invalid_response" = "422" ]; then
    echo "  ✓ API correctly rejects invalid search query"
else
    echo "  ⚠️  API may not be validating search requests properly (got HTTP $invalid_response)"
fi

# Test 7: Response time validation
echo "  Testing response time requirements..."
start_time=$(date +%s%3N)
curl -sf "$API_URL/health" >/dev/null 2>&1
end_time=$(date +%s%3N)
response_time=$((end_time - start_time))

echo "    Health endpoint response time: ${response_time}ms"

if [ "$response_time" -lt 1500 ]; then
    echo "  ✓ Response time meets performance target (<1.5s)"
elif [ "$response_time" -lt 3000 ]; then
    echo "  ⚠️  Response time acceptable but slower than target (${response_time}ms)"
else
    echo "  ✗ Response time exceeds acceptable threshold (${response_time}ms)"
    exit 1
fi

echo "✅ All business logic tests passed"
