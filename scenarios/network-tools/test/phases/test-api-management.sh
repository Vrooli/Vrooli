#!/usr/bin/env bash
# Test Phase: API Management Feature
# Validates API definition management and discovery functionality

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Get API port from environment
API_PORT=${API_PORT:-17124}
API_BASE="http://localhost:$API_PORT"

echo "================================================"
echo "🔧 API Management Feature Tests"
echo "================================================"

TESTS_PASSED=0
TESTS_FAILED=0
API_ID=""

# Wait for API to be ready
echo "Waiting for API to be ready..."
for i in {1..30}; do
    if curl -s -f "$API_BASE/api/health" > /dev/null 2>&1; then
        echo "✅ API is ready"
        break
    fi
    sleep 1
done

# Helper function - prints status to stderr, returns JSON to stdout
test_api_endpoint() {
    local name="$1"
    local method="$2"
    local path="$3"
    local data="$4"
    local expected_status="${5:-200}"

    echo "  Testing: $name" >&2

    local response
    if [[ "$method" == "GET" ]]; then
        response=$(curl -s -w "\n%{http_code}" --max-time 10 "$API_BASE$path")
    elif [[ "$method" == "DELETE" ]]; then
        response=$(curl -s -w "\n%{http_code}" --max-time 10 -X DELETE "$API_BASE$path")
    else
        response=$(curl -s -w "\n%{http_code}" --max-time 10 \
            -X "$method" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$API_BASE$path")
    fi

    local body=$(echo "$response" | head -n -1)
    local status=$(echo "$response" | tail -n 1)

    if [ "$status" -eq "$expected_status" ]; then
        echo "    ✅ Returns HTTP $status" >&2
        ((TESTS_PASSED++))
        echo "$body"
        return 0
    else
        echo "    ❌ Expected HTTP $expected_status, got $status" >&2
        echo "    Response: $body" >&2
        ((TESTS_FAILED++))
        return 1
    fi
}

# Test 1: List API definitions (initially empty)
echo ""
echo "📋 Test 1: List API Definitions"
if response=$(test_api_endpoint "List API definitions" "GET" "/api/v1/api/definitions" ""); then
    if echo "$response" | jq -e '.data | type == "array"' > /dev/null 2>&1; then
        echo "    ✅ Returns array of definitions"
        ((TESTS_PASSED++))
    else
        echo "    ❌ Response is not an array"
        ((TESTS_FAILED++))
    fi
fi

# Test 2: Create API definition
echo ""
echo "📝 Test 2: Create API Definition"
CREATE_PAYLOAD='{
    "name": "Test API",
    "base_url": "https://httpbingo.org",
    "version": "1.0",
    "specification": "openapi"
}'

if response=$(test_api_endpoint "Create API definition" "POST" "/api/v1/api/definitions" "$CREATE_PAYLOAD"); then
    API_ID=$(echo "$response" | jq -r '.data.id' 2>/dev/null)
    if [[ -n "$API_ID" ]] && [[ "$API_ID" != "null" ]]; then
        echo "    ✅ API definition created with ID: $API_ID"
        ((TESTS_PASSED++))
    else
        echo "    ❌ No ID returned in response"
        ((TESTS_FAILED++))
        API_ID=""
    fi
fi

# Test 3: Get API definition by ID
if [[ -n "$API_ID" ]]; then
    echo ""
    echo "🔍 Test 3: Get API Definition by ID"
    if response=$(test_api_endpoint "Get API definition" "GET" "/api/v1/api/definitions/$API_ID" ""); then
        name=$(echo "$response" | jq -r '.data.name' 2>/dev/null)
        if [[ "$name" == "Test API" ]]; then
            echo "    ✅ Retrieved correct API definition"
            ((TESTS_PASSED++))
        else
            echo "    ❌ Retrieved wrong API definition (name: $name)"
            ((TESTS_FAILED++))
        fi
    fi
fi

# Test 4: Update API definition
if [[ -n "$API_ID" ]]; then
    echo ""
    echo "✏️  Test 4: Update API Definition"
    UPDATE_PAYLOAD='{
        "name": "Updated Test API",
        "version": "1.1"
    }'

    test_api_endpoint "Update API definition" "PUT" "/api/v1/api/definitions/$API_ID" "$UPDATE_PAYLOAD"

    # Verify update
    if response=$(test_api_endpoint "Verify update" "GET" "/api/v1/api/definitions/$API_ID" ""); then
        name=$(echo "$response" | jq -r '.data.name' 2>/dev/null)
        version=$(echo "$response" | jq -r '.data.version' 2>/dev/null)
        if [[ "$name" == "Updated Test API" ]] && [[ "$version" == "1.1" ]]; then
            echo "    ✅ API definition updated correctly"
            ((TESTS_PASSED++))
        else
            echo "    ❌ Update verification failed (name: $name, version: $version)"
            ((TESTS_FAILED++))
        fi
    fi
fi

# Test 5: Discover API endpoints from OpenAPI spec
echo ""
echo "🔎 Test 5: Discover API Endpoints"
DISCOVER_PAYLOAD='{
    "spec_url": "https://petstore3.swagger.io/api/v3/openapi.json"
}'

if response=$(test_api_endpoint "Discover endpoints" "POST" "/api/v1/api/discover" "$DISCOVER_PAYLOAD"); then
    endpoints_count=$(echo "$response" | jq -r '.data.info.endpoints_count' 2>/dev/null)
    if [[ -n "$endpoints_count" ]] && [[ "$endpoints_count" -gt 0 ]]; then
        echo "    ✅ Discovered $endpoints_count endpoints"
        ((TESTS_PASSED++))
    else
        echo "    ❌ No endpoints discovered"
        ((TESTS_FAILED++))
    fi
fi

# Test 6: List API definitions (should have our created one)
echo ""
echo "📋 Test 6: List API Definitions (After Creation)"
if response=$(test_api_endpoint "List API definitions" "GET" "/api/v1/api/definitions" ""); then
    count=$(echo "$response" | jq -r '.data | length' 2>/dev/null)
    if [[ "$count" -ge 1 ]]; then
        echo "    ✅ Found $count API definition(s)"
        ((TESTS_PASSED++))
    else
        echo "    ❌ No API definitions found"
        ((TESTS_FAILED++))
    fi
fi

# Test 7: Delete API definition
if [[ -n "$API_ID" ]]; then
    echo ""
    echo "🗑️  Test 7: Delete API Definition"
    test_api_endpoint "Delete API definition" "DELETE" "/api/v1/api/definitions/$API_ID" ""

    # Verify deletion
    if response=$(curl -s -w "\n%{http_code}" --max-time 10 "$API_BASE/api/v1/api/definitions/$API_ID"); then
        status=$(echo "$response" | tail -n 1)
        if [ "$status" -eq 404 ]; then
            echo "    ✅ API definition deleted successfully"
            ((TESTS_PASSED++))
        else
            echo "    ❌ API definition still exists (status: $status)"
            ((TESTS_FAILED++))
        fi
    fi
fi

# Test 8: Error handling - Invalid create
echo ""
echo "⚠️  Test 8: Error Handling - Invalid Create"
INVALID_PAYLOAD='{
    "name": "Missing Base URL"
}'
test_api_endpoint "Create with missing base_url" "POST" "/api/v1/api/definitions" "$INVALID_PAYLOAD" 400

# Test 9: Error handling - Get non-existent ID
echo ""
echo "⚠️  Test 9: Error Handling - Non-existent ID"
test_api_endpoint "Get non-existent API" "GET" "/api/v1/api/definitions/00000000-0000-0000-0000-000000000000" "" 404

# Summary
echo ""
echo "================================================"
echo "📊 API Management Test Summary"
echo "================================================"
echo "  Passed: $TESTS_PASSED"
echo "  Failed: $TESTS_FAILED"

if [ $TESTS_FAILED -eq 0 ]; then
    echo ""
    echo "✅ All API management tests passed!"
    echo "🎯 Features Validated:"
    echo "   • List API definitions"
    echo "   • Create API definition"
    echo "   • Get API definition by ID"
    echo "   • Update API definition"
    echo "   • Delete API definition"
    echo "   • Discover endpoints from OpenAPI spec"
    echo "   • Error handling"
    exit 0
else
    echo ""
    echo "❌ Some API management tests failed"
    exit 1
fi
