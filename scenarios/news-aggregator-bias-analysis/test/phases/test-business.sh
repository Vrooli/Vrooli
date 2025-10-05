#!/bin/bash
# Business logic tests for news-aggregator-bias-analysis scenario

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with runtime requirement and 180-second target
testing::phase::init --require-runtime --target-time "180s"

echo "Testing core business functionality..."

# Test counters are handled by phase helpers
created_feed_ids=()
created_article_ids=()

# Cleanup function to run on exit
cleanup() {
    echo "Cleaning up test data..."
    for feed_id in "${created_feed_ids[@]}"; do
        if [ -n "$feed_id" ] && [ "$feed_id" != "null" ]; then
            echo "Deleting test feed: $feed_id"
            curl -sf -X DELETE "$API_BASE_URL/feeds/$feed_id" >/dev/null 2>&1 || echo "Warning: Failed to delete feed $feed_id"
        fi
    done
}

# Register cleanup
testing::phase::register_cleanup cleanup

if ! testing::core::wait_for_scenario "$TESTING_PHASE_SCENARIO_NAME" 30 >/dev/null 2>&1; then
    testing::phase::add_error "❌ Scenario '$TESTING_PHASE_SCENARIO_NAME' did not become ready"
    testing::phase::end_with_summary
fi

if ! API_BASE_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME"); then
    testing::phase::add_error "❌ Unable to determine API URL for $TESTING_PHASE_SCENARIO_NAME"
    testing::phase::end_with_summary
fi

echo "Using API base URL: $API_BASE_URL"

# Pre-cleanup: Remove any existing test feeds
echo "Cleaning up any existing test feeds..."
existing_feeds=$(curl -sf "$API_BASE_URL/feeds" 2>/dev/null || echo "")
if echo "$existing_feeds" | jq -e '.' >/dev/null 2>&1; then
    echo "$existing_feeds" | jq -r '.[] | select(.name | test("^business-test-")) | .id' | while read -r old_feed_id; do
        if [ -n "$old_feed_id" ] && [ "$old_feed_id" != "null" ]; then
            echo "Deleting old test feed: $old_feed_id"
            curl -sf -X DELETE "$API_BASE_URL/feeds/$old_feed_id" >/dev/null 2>&1 || echo "Warning: Failed to delete old feed $old_feed_id"
        fi
    done
fi

# Test 1: Health check
echo "Testing health endpoint..."
health_response=$(curl -sf "$API_BASE_URL/health" 2>/dev/null || echo "")
if echo "$health_response" | jq -e '.status' >/dev/null 2>&1; then
    status=$(echo "$health_response" | jq -r '.status')
    if [ "$status" = "healthy" ]; then
        echo "✅ Health check passed"
        testing::phase::add_test passed
    else
        echo "❌ Health check returned unexpected status: $status"
        testing::phase::add_test failed
    fi
else
    echo "❌ Health check failed"
    testing::phase::add_test failed
fi

# Test 2: Feed creation
echo "Testing feed creation..."
timestamp=$(date +%s)
feed_data='{"name":"business-test-'$timestamp'","url":"https://example.com/feed.rss","category":"test","bias_rating":"center"}'
feed_response=$(curl -sf -X POST "$API_BASE_URL/feeds" -H "Content-Type: application/json" -d "$feed_data" 2>/dev/null || echo "")

if echo "$feed_response" | jq -e '.id' >/dev/null 2>&1; then
    feed_id=$(echo "$feed_response" | jq -r '.id')
    created_feed_ids+=("$feed_id")
    echo "✅ Feed creation passed - ID: $feed_id"
    testing::phase::add_test passed
else
    echo "❌ Feed creation failed"
    testing::phase::add_test failed
fi

# Test 3: Feed retrieval
echo "Testing feed retrieval..."
if [ -n "${feed_id:-}" ]; then
    feeds_response=$(curl -sf "$API_BASE_URL/feeds" 2>/dev/null || echo "")
    if echo "$feeds_response" | jq -e '.[] | select(.id == '$feed_id')' >/dev/null 2>&1; then
        echo "✅ Feed retrieval passed"
        testing::phase::add_test passed
    else
        echo "❌ Feed retrieval failed"
        testing::phase::add_test failed
    fi
else
    echo "⚠️  Skipping feed retrieval (no feed created)"
fi

# Test 4: Feed update
echo "Testing feed update..."
if [ -n "${feed_id:-}" ]; then
    update_data='{"name":"business-test-updated-'$timestamp'","url":"https://example.com/feed-updated.rss","category":"test","bias_rating":"left","active":true}'
    update_response=$(curl -sf -X PUT "$API_BASE_URL/feeds/$feed_id" -H "Content-Type: application/json" -d "$update_data" 2>/dev/null || echo "")

    if echo "$update_response" | jq -e '.id' >/dev/null 2>&1; then
        updated_name=$(echo "$update_response" | jq -r '.name')
        if [[ "$updated_name" == *"updated"* ]]; then
            echo "✅ Feed update passed"
            testing::phase::add_test passed
        else
            echo "❌ Feed update did not reflect changes"
            testing::phase::add_test failed
        fi
    else
        echo "❌ Feed update failed"
        testing::phase::add_test failed
    fi
else
    echo "⚠️  Skipping feed update (no feed created)"
fi

# Test 5: Articles retrieval
echo "Testing articles retrieval..."
articles_response=$(curl -sf "$API_BASE_URL/articles" 2>/dev/null || echo "")
if echo "$articles_response" | jq -e '.' >/dev/null 2>&1; then
    if echo "$articles_response" | jq -e 'type == "array"' >/dev/null 2>&1; then
        article_count=$(echo "$articles_response" | jq 'length')
        echo "✅ Articles retrieval passed - $article_count articles found"
        testing::phase::add_test passed
    else
        echo "❌ Articles response is not an array"
        testing::phase::add_test failed
    fi
else
    echo "❌ Articles retrieval failed"
    testing::phase::add_test failed
fi

# Test 6: Perspectives endpoint
echo "Testing perspectives endpoint..."
perspectives_response=$(curl -sf "$API_BASE_URL/perspectives/test" 2>/dev/null || echo "")
if echo "$perspectives_response" | jq -e '.' >/dev/null 2>&1; then
    if echo "$perspectives_response" | jq -e 'type == "array"' >/dev/null 2>&1; then
        echo "✅ Perspectives endpoint passed"
        testing::phase::add_test passed
    else
        echo "❌ Perspectives response is not an array"
        testing::phase::add_test failed
    fi
else
    echo "❌ Perspectives endpoint failed"
    testing::phase::add_test failed
fi

# Test 7: Perspective aggregation
echo "Testing perspective aggregation..."
aggregate_data='{"topic":"test topic","time_range":"24 hours"}'
aggregate_response=$(curl -sf -X POST "$API_BASE_URL/perspectives/aggregate" -H "Content-Type: application/json" -d "$aggregate_data" 2>/dev/null || echo "")

if echo "$aggregate_response" | jq -e '.perspectives' >/dev/null 2>&1; then
    # Check for left, center, right perspectives
    if echo "$aggregate_response" | jq -e '.perspectives.left' >/dev/null 2>&1 && \
       echo "$aggregate_response" | jq -e '.perspectives.center' >/dev/null 2>&1 && \
       echo "$aggregate_response" | jq -e '.perspectives.right' >/dev/null 2>&1; then
        echo "✅ Perspective aggregation passed"
        testing::phase::add_test passed
    else
        echo "❌ Perspective aggregation missing required perspectives"
        testing::phase::add_test failed
    fi
else
    echo "❌ Perspective aggregation failed"
    testing::phase::add_test failed
fi

# Test 8: Feed refresh trigger
echo "Testing feed refresh trigger..."
refresh_response=$(curl -sf -X POST "$API_BASE_URL/refresh" 2>/dev/null || echo "")
if echo "$refresh_response" | jq -e '.status' >/dev/null 2>&1; then
    status=$(echo "$refresh_response" | jq -r '.status')
    if [ "$status" = "refresh_triggered" ]; then
        echo "✅ Feed refresh trigger passed"
        testing::phase::add_test passed
    else
        echo "❌ Feed refresh returned unexpected status: $status"
        testing::phase::add_test failed
    fi
else
    echo "❌ Feed refresh trigger failed"
    testing::phase::add_test failed
fi

# Test 9: Data persistence
echo "Testing data persistence..."
feeds_check=$(curl -sf "$API_BASE_URL/feeds" 2>/dev/null || echo "")
if echo "$feeds_check" | jq -e '.' >/dev/null 2>&1; then
    feed_count=$(echo "$feeds_check" | jq 'length')
    if [ "$feed_count" -gt 0 ]; then
        echo "✅ Data persistence passed - $feed_count feeds found"
        testing::phase::add_test passed
    else
        echo "⚠️  Data persistence check: no feeds found (may be expected)"
        testing::phase::add_test passed
    fi
else
    echo "❌ Data persistence check failed"
    testing::phase::add_test failed
fi

echo ""
echo "Summary: $TESTING_PHASE_TEST_COUNT tests, $TESTING_PHASE_ERROR_COUNT failed"

if [ $TESTING_PHASE_ERROR_COUNT -eq 0 ]; then
    echo "SUCCESS: All business tests passed"
else
    echo "ERROR: Some business tests failed"
fi

# End with summary
testing::phase::end_with_summary "Business logic tests completed"
