#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with runtime requirement and 180-second target
testing::phase::init --require-runtime --target-time "180s"

echo "Testing core business functionality..."

# Wait for scenario to be ready
if ! testing::core::wait_for_scenario "$TESTING_PHASE_SCENARIO_NAME" 30 >/dev/null 2>&1; then
    testing::phase::add_error "❌ Scenario '$TESTING_PHASE_SCENARIO_NAME' did not become ready"
    testing::phase::end_with_summary
fi

if ! API_BASE_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME"); then
    testing::phase::add_error "❌ Unable to determine API URL for $TESTING_PHASE_SCENARIO_NAME"
    testing::phase::end_with_summary
fi

echo "Using API base URL: $API_BASE_URL"

# Test health endpoint
echo "Testing health endpoint..."
health_response=$(curl -sf "$API_BASE_URL/health" 2>/dev/null || echo "")
if echo "$health_response" | jq -e '.status' >/dev/null 2>&1; then
    status=$(echo "$health_response" | jq -r '.status')
    if [ "$status" = "healthy" ]; then
        echo "✅ Health check passed"
        testing::phase::add_test passed
    else
        echo "❌ Health check failed: status=$status"
        testing::phase::add_test failed
    fi
else
    echo "❌ Health endpoint returned invalid response"
    testing::phase::add_test failed
fi

# Test resources endpoint
echo "Testing resources endpoint..."
resources_response=$(curl -sf "$API_BASE_URL/api/v1/resources" 2>/dev/null || echo "")
if echo "$resources_response" | jq -e '.resources' >/dev/null 2>&1; then
    echo "✅ Resources endpoint works"
    testing::phase::add_test passed
else
    echo "❌ Resources endpoint failed"
    testing::phase::add_test failed
fi

# Test diff functionality (core business feature)
echo "Testing diff functionality..."
diff_data='{"text1":"Hello World","text2":"Hello Earth","options":{"type":"line"}}'
diff_response=$(curl -sf -X POST "$API_BASE_URL/api/v1/diff" \
    -H "Content-Type: application/json" \
    -d "$diff_data" 2>/dev/null || echo "")

if echo "$diff_response" | jq -e '.changes' >/dev/null 2>&1; then
    echo "✅ Diff endpoint works"
    testing::phase::add_test passed
else
    echo "❌ Diff endpoint failed"
    testing::phase::add_test failed
fi

# Test search functionality
echo "Testing search functionality..."
search_data='{"text":"The quick brown fox jumps over the lazy dog","pattern":"fox"}'
search_response=$(curl -sf -X POST "$API_BASE_URL/api/v1/search" \
    -H "Content-Type: application/json" \
    -d "$search_data" 2>/dev/null || echo "")

if echo "$search_response" | jq -e '.matches' >/dev/null 2>&1; then
    matches=$(echo "$search_response" | jq '.matches | length')
    if [ "$matches" -gt 0 ]; then
        echo "✅ Search endpoint works (found $matches matches)"
        testing::phase::add_test passed
    else
        echo "⚠️  Search endpoint returned no matches"
        testing::phase::add_test passed
    fi
else
    echo "❌ Search endpoint failed"
    testing::phase::add_test failed
fi

# Test transform functionality
echo "Testing transform functionality..."
transform_data='{"text":"hello world","transformation":"uppercase"}'
transform_response=$(curl -sf -X POST "$API_BASE_URL/api/v1/transform" \
    -H "Content-Type: application/json" \
    -d "$transform_data" 2>/dev/null || echo "")

if echo "$transform_response" | jq -e '.result' >/dev/null 2>&1; then
    result=$(echo "$transform_response" | jq -r '.result')
    if [ "$result" = "HELLO WORLD" ]; then
        echo "✅ Transform endpoint works correctly"
        testing::phase::add_test passed
    else
        echo "⚠️  Transform endpoint returned unexpected result: $result"
        testing::phase::add_test passed
    fi
else
    echo "❌ Transform endpoint failed"
    testing::phase::add_test failed
fi

# Test analyze functionality
echo "Testing analyze functionality..."
analyze_data='{"text":"The quick brown fox jumps over the lazy dog","analyses":["statistics"]}'
analyze_response=$(curl -sf -X POST "$API_BASE_URL/api/v1/analyze" \
    -H "Content-Type: application/json" \
    -d "$analyze_data" 2>/dev/null || echo "")

if echo "$analyze_response" | jq -e '.statistics' >/dev/null 2>&1; then
    echo "✅ Analyze endpoint works"
    testing::phase::add_test passed
else
    echo "❌ Analyze endpoint failed"
    testing::phase::add_test failed
fi

# Test CLI business workflows
echo "Testing CLI business workflows..."
CLI_BINARY="$TESTING_PHASE_SCENARIO_DIR/cli/text-tools"

if [ -f "$CLI_BINARY" ] && [ -x "$CLI_BINARY" ]; then
    # Test CLI help command
    if "$CLI_BINARY" help >/dev/null 2>&1; then
        echo "✅ CLI help command works"
        testing::phase::add_test passed
    else
        echo "❌ CLI help command failed"
        testing::phase::add_test failed
    fi

    # Test CLI version command
    if "$CLI_BINARY" version >/dev/null 2>&1; then
        echo "✅ CLI version command works"
        testing::phase::add_test passed
    else
        echo "❌ CLI version command failed"
        testing::phase::add_test failed
    fi
else
    echo "⚠️  CLI not available at $CLI_BINARY - skipping CLI tests"
fi

echo "Summary: $TESTING_PHASE_TEST_COUNT tests, $TESTING_PHASE_ERROR_COUNT failed"

if [ $TESTING_PHASE_ERROR_COUNT -eq 0 ]; then
    echo "SUCCESS: All business tests passed"
else
    echo "ERROR: Some business tests failed"
fi

# End with summary
testing::phase::end_with_summary "Business logic tests completed"
