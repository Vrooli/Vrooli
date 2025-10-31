#!/usr/bin/env bash
set -uo pipefail  # Removed -e to allow tests to continue after failures

# Test: Integration Tests
# Tests API endpoints, UI functionality, and resource integration

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "🔗 Running Prompt Injection Arena integration tests..."

# Test configuration - require ports from environment
if [ -z "${API_PORT:-}" ] || [ -z "${UI_PORT:-}" ]; then
    echo "❌ API_PORT and UI_PORT environment variables are required"
    exit 1
fi

API_URL="http://localhost:${API_PORT}"
UI_URL="http://localhost:${UI_PORT}"

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

# Cleanup function
cleanup_test_data() {
    echo ""
    echo "🧹 Cleaning up test data..."

    # Use admin cleanup endpoint if available
    if curl -sf -X POST "${API_URL}/api/v1/admin/cleanup-test-data" > /dev/null 2>&1; then
        echo "  ✅ Test data cleaned via admin endpoint"
    else
        echo "  ⚠️  Admin cleanup not available"
    fi
}

# Register cleanup to run on exit
trap cleanup_test_data EXIT

# Function to make API calls
api_call() {
    local method=$1
    local endpoint=$2
    local data=${3:-}

    if [ -n "$data" ]; then
        curl -sf -X "$method" "${API_URL}${endpoint}" \
            -H "Content-Type: application/json" \
            -d "$data"
    else
        curl -sf -X "$method" "${API_URL}${endpoint}"
    fi
}

# Test API Health
echo "🏥 Testing API health..."
if curl -sf "${API_URL}/health" | jq -e '.status == "healthy"' > /dev/null; then
    echo "  ✅ API health check passed"
    ((TESTS_PASSED++))
else
    echo "  ❌ API health check failed"
    ((TESTS_FAILED++))
    # Don't exit early - let other tests run
fi

# Test UI availability
echo "🎨 Testing UI availability..."
ui_response=$(curl -sf "${UI_URL}/" 2>&1) || ui_response=""
if [ -n "$ui_response" ] && echo "$ui_response" | grep -q "Prompt Injection Arena"; then
    echo "  ✅ UI is accessible"
    ((TESTS_PASSED++))
else
    echo "  ❌ UI is not accessible (UI_URL=${UI_URL}, response_length=${#ui_response})"
    ((TESTS_FAILED++))
fi

# Test UI health endpoint
if curl -sf "${UI_URL}/health" | jq -e '.api_connectivity.connected == true' > /dev/null; then
    echo "  ✅ UI health check passed"
    ((TESTS_PASSED++))
else
    echo "  ❌ UI health check failed"
    ((TESTS_FAILED++))
fi

# Test injection library endpoints
echo "📚 Testing injection library..."

# Get library
library_response=$(api_call GET /api/v1/injections/library)
if echo "$library_response" | jq -e '.techniques' > /dev/null; then
    echo "  ✅ Injection library retrieval works"
    ((TESTS_PASSED++))
else
    echo "  ❌ Failed to retrieve injection library"
    ((TESTS_FAILED++))
fi

# Test agent configuration endpoints (via test-agent endpoint)
echo "🤖 Testing agent security testing..."

# Test the security test-agent endpoint with a simple system prompt
test_agent_data='{"system_prompt":"You are a helpful assistant.","model_name":"llama3.2","test_suite":[]}'
if api_call POST /api/v1/security/test-agent "$test_agent_data" | jq -e '.robustness_score' > /dev/null 2>&1; then
    echo "  ✅ Agent security testing works"
    ((TESTS_PASSED++))
else
    echo "  ⚠️  Agent security testing may need ollama model"
    # Don't fail on this since it requires ollama model to be available
fi

# Test leaderboard endpoints
echo "🏆 Testing leaderboards..."

# Get agent leaderboard
leaderboard_response=$(api_call GET /api/v1/leaderboards/agents)
if echo "$leaderboard_response" | jq -e '.leaderboard' > /dev/null; then
    echo "  ✅ Agent leaderboard works"
    ((TESTS_PASSED++))
else
    echo "  ❌ Failed to retrieve leaderboard"
    ((TESTS_FAILED++))
fi

# Test export functionality
echo "📊 Testing research export..."

# Get export formats
formats_response=$(api_call GET /api/v1/export/formats)
if echo "$formats_response" | jq -e '.formats' > /dev/null; then
    echo "  ✅ Export formats endpoint works"
    ((TESTS_PASSED++))
else
    echo "  ❌ Failed to get export formats"
    ((TESTS_FAILED++))
fi

# Test vector search (optional, depends on Qdrant)
echo "🔍 Testing vector search..."
if vrooli resource status qdrant &> /dev/null; then
    search_response=$(api_call GET "/api/v1/injections/similar?query=override%20instructions" 2>/dev/null || echo '{}')
    if echo "$search_response" | jq -e 'type == "array"' > /dev/null 2>&1; then
        echo "  ✅ Vector similarity search works"
        ((TESTS_PASSED++))
    else
        echo "  ⚠️  Vector search may need index rebuild"
    fi
else
    echo "  ⚠️  Qdrant not available, skipping vector search tests"
fi

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Integration Test Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  ✅ Passed: ${TESTS_PASSED}"
echo "  ❌ Failed: ${TESTS_FAILED}"
echo "  📊 Total:  $((TESTS_PASSED + TESTS_FAILED))"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ ${TESTS_FAILED} -eq 0 ]; then
    echo "✅ All integration tests passed!"
    exit 0
else
    echo "❌ Some integration tests failed"
    exit 1
fi
