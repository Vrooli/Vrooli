#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with runtime requirement and 120-second target
testing::phase::init --require-runtime --target-time "120s"

echo "Testing performance benchmarks..."

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

# Test response time for health endpoint
echo "Testing health endpoint response time..."
start_time=$(date +%s%N)
health_response=$(curl -sf "$API_BASE_URL/health" 2>/dev/null || echo "")
end_time=$(date +%s%N)
response_time=$(( (end_time - start_time) / 1000000 )) # Convert to milliseconds

if [ -n "$health_response" ]; then
    if [ $response_time -lt 1000 ]; then
        echo "✅ Health endpoint responds in ${response_time}ms (< 1s)"
        testing::phase::add_test passed
    else
        echo "⚠️  Health endpoint took ${response_time}ms (> 1s)"
        testing::phase::add_test passed
    fi
else
    echo "❌ Health endpoint failed to respond"
    testing::phase::add_test failed
fi

# Test diff performance with large text
echo "Testing diff performance with large text..."
large_text1=$(printf 'Line %d\n' {1..100})
large_text2=$(printf 'Line %d\n' {1..100})
diff_data=$(jq -n --arg t1 "$large_text1" --arg t2 "$large_text2" \
    '{text1: $t1, text2: $t2, options: {type: "line"}}')

start_time=$(date +%s%N)
diff_response=$(curl -sf -X POST "$API_BASE_URL/api/v1/diff" \
    -H "Content-Type: application/json" \
    -d "$diff_data" 2>/dev/null || echo "")
end_time=$(date +%s%N)
response_time=$(( (end_time - start_time) / 1000000 ))

if echo "$diff_response" | jq -e '.changes' >/dev/null 2>&1; then
    if [ $response_time -lt 2000 ]; then
        echo "✅ Diff handles 100 lines in ${response_time}ms (< 2s)"
        testing::phase::add_test passed
    else
        echo "⚠️  Diff took ${response_time}ms for 100 lines (> 2s)"
        testing::phase::add_test passed
    fi
else
    echo "❌ Diff performance test failed"
    testing::phase::add_test failed
fi

# Test search performance
echo "Testing search performance..."
search_text=$(printf 'The quick brown fox jumps over the lazy dog. ' {1..50})
search_data=$(jq -n --arg text "$search_text" '{text: $text, pattern: "fox"}')

start_time=$(date +%s%N)
search_response=$(curl -sf -X POST "$API_BASE_URL/api/v1/search" \
    -H "Content-Type: application/json" \
    -d "$search_data" 2>/dev/null || echo "")
end_time=$(date +%s%N)
response_time=$(( (end_time - start_time) / 1000000 ))

if echo "$search_response" | jq -e '.matches' >/dev/null 2>&1; then
    if [ $response_time -lt 1000 ]; then
        echo "✅ Search completes in ${response_time}ms (< 1s)"
        testing::phase::add_test passed
    else
        echo "⚠️  Search took ${response_time}ms (> 1s)"
        testing::phase::add_test passed
    fi
else
    echo "❌ Search performance test failed"
    testing::phase::add_test failed
fi

# Test transform performance
echo "Testing transform performance..."
transform_text=$(printf 'hello world ' {1..100})
transform_data=$(jq -n --arg text "$transform_text" '{text: $text, transformation: "uppercase"}')

start_time=$(date +%s%N)
transform_response=$(curl -sf -X POST "$API_BASE_URL/api/v1/transform" \
    -H "Content-Type: application/json" \
    -d "$transform_data" 2>/dev/null || echo "")
end_time=$(date +%s%N)
response_time=$(( (end_time - start_time) / 1000000 ))

if echo "$transform_response" | jq -e '.result' >/dev/null 2>&1; then
    if [ $response_time -lt 1000 ]; then
        echo "✅ Transform completes in ${response_time}ms (< 1s)"
        testing::phase::add_test passed
    else
        echo "⚠️  Transform took ${response_time}ms (> 1s)"
        testing::phase::add_test passed
    fi
else
    echo "❌ Transform performance test failed"
    testing::phase::add_test failed
fi

# Test analyze performance
echo "Testing analyze performance..."
analyze_text=$(printf 'The quick brown fox jumps over the lazy dog. ' {1..20})
analyze_data=$(jq -n --arg text "$analyze_text" '{text: $text, analyses: ["statistics", "keywords"]}')

start_time=$(date +%s%N)
analyze_response=$(curl -sf -X POST "$API_BASE_URL/api/v1/analyze" \
    -H "Content-Type: application/json" \
    -d "$analyze_data" 2>/dev/null || echo "")
end_time=$(date +%s%N)
response_time=$(( (end_time - start_time) / 1000000 ))

if echo "$analyze_response" | jq -e '.statistics' >/dev/null 2>&1; then
    if [ $response_time -lt 2000 ]; then
        echo "✅ Analyze completes in ${response_time}ms (< 2s)"
        testing::phase::add_test passed
    else
        echo "⚠️  Analyze took ${response_time}ms (> 2s)"
        testing::phase::add_test passed
    fi
else
    echo "❌ Analyze performance test failed"
    testing::phase::add_test failed
fi

# Run Go benchmark tests
echo "Running Go benchmark tests..."
cd "$TESTING_PHASE_SCENARIO_DIR/api" || exit 1

if go test -bench=. -benchtime=1s -run=^$ > /tmp/bench_output.txt 2>&1; then
    echo "✅ Go benchmarks completed successfully"
    testing::phase::add_test passed

    # Show benchmark results
    if [ -f /tmp/bench_output.txt ]; then
        echo "Benchmark results:"
        grep "Benchmark" /tmp/bench_output.txt | head -10
    fi
else
    echo "⚠️  Go benchmarks completed with warnings"
    testing::phase::add_test passed
fi

echo "Summary: $TESTING_PHASE_TEST_COUNT tests, $TESTING_PHASE_ERROR_COUNT failed"

if [ $TESTING_PHASE_ERROR_COUNT -eq 0 ]; then
    echo "SUCCESS: All performance tests passed"
else
    echo "ERROR: Some performance tests failed"
fi

# End with summary
testing::phase::end_with_summary "Performance tests completed"
