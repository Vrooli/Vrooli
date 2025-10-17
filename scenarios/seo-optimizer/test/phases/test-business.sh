#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with runtime requirement and 180-second target
testing::phase::init --require-runtime --target-time "180s"

echo "Testing core business functionality..."

# Cleanup function to run on exit
cleanup() {
    echo "Cleaning up test data..."
    # No persistent data to clean up for SEO optimizer
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

# Test SEO Audit Business Logic
echo "Testing SEO audit business logic..."
audit_data='{"url":"https://example.com","depth":2}'
audit_response=$(curl -sf -X POST "$API_BASE_URL/api/seo-audit" \
    -H "Content-Type: application/json" \
    -d "$audit_data" 2>/dev/null || echo "")

if echo "$audit_response" | jq -e '.technical_audit' >/dev/null 2>&1; then
    # Verify required audit components
    has_technical=$(echo "$audit_response" | jq -e '.technical_audit' >/dev/null 2>&1 && echo "true" || echo "false")
    has_content=$(echo "$audit_response" | jq -e '.content_audit' >/dev/null 2>&1 && echo "true" || echo "false")
    has_performance=$(echo "$audit_response" | jq -e '.performance_audit' >/dev/null 2>&1 && echo "true" || echo "false")
    has_score=$(echo "$audit_response" | jq -e '.overall_score' >/dev/null 2>&1 && echo "true" || echo "false")

    if [ "$has_technical" = "true" ] && [ "$has_content" = "true" ] && [ "$has_performance" = "true" ] && [ "$has_score" = "true" ]; then
        echo "SEO audit business logic tests passed - all components present"
        testing::phase::add_test passed
    else
        echo "SEO audit business logic tests failed - missing components"
        testing::phase::add_test failed
    fi
else
    echo "SEO audit business logic tests failed - invalid response"
    testing::phase::add_test failed
fi

# Test Content Optimization Business Logic
echo "Testing content optimization business logic..."
content_data='{"content":"This is a test blog post about SEO optimization. SEO is important for websites. We need to optimize our content for better rankings.","target_keywords":"SEO,optimization,rankings","content_type":"blog"}'
content_response=$(curl -sf -X POST "$API_BASE_URL/api/content-optimize" \
    -H "Content-Type: application/json" \
    -d "$content_data" 2>/dev/null || echo "")

if echo "$content_response" | jq -e '.content_analysis' >/dev/null 2>&1; then
    # Verify optimization components
    has_analysis=$(echo "$content_response" | jq -e '.content_analysis' >/dev/null 2>&1 && echo "true" || echo "false")
    has_issues=$(echo "$content_response" | jq -e '.issues' >/dev/null 2>&1 && echo "true" || echo "false")
    has_recommendations=$(echo "$content_response" | jq -e '.recommendations' >/dev/null 2>&1 && echo "true" || echo "false")
    has_score=$(echo "$content_response" | jq -e '.content_score' >/dev/null 2>&1 && echo "true" || echo "false")

    if [ "$has_analysis" = "true" ] && [ "$has_issues" = "true" ] && [ "$has_recommendations" = "true" ] && [ "$has_score" = "true" ]; then
        echo "Content optimization business logic tests passed"
        testing::phase::add_test passed
    else
        echo "Content optimization business logic tests failed - missing components"
        testing::phase::add_test failed
    fi
else
    echo "Content optimization business logic tests failed - invalid response"
    testing::phase::add_test failed
fi

# Test Keyword Research Business Logic
echo "Testing keyword research business logic..."
keyword_data='{"seed_keyword":"SEO optimization","target_location":"United States","language":"en"}'
keyword_response=$(curl -sf -X POST "$API_BASE_URL/api/keyword-research" \
    -H "Content-Type: application/json" \
    -d "$keyword_data" 2>/dev/null || echo "")

if echo "$keyword_response" | jq -e '.keywords' >/dev/null 2>&1; then
    # Verify keyword research components
    has_keywords=$(echo "$keyword_response" | jq -e '.keywords' >/dev/null 2>&1 && echo "true" || echo "false")
    has_related=$(echo "$keyword_response" | jq -e '.related_terms' >/dev/null 2>&1 && echo "true" || echo "false")
    has_questions=$(echo "$keyword_response" | jq -e '.questions' >/dev/null 2>&1 && echo "true" || echo "false")

    if [ "$has_keywords" = "true" ] && [ "$has_related" = "true" ] && [ "$has_questions" = "true" ]; then
        keyword_count=$(echo "$keyword_response" | jq '.keywords | length')
        echo "Keyword research business logic tests passed - $keyword_count keywords found"
        testing::phase::add_test passed
    else
        echo "Keyword research business logic tests failed - missing components"
        testing::phase::add_test failed
    fi
else
    echo "Keyword research business logic tests failed - invalid response"
    testing::phase::add_test failed
fi

# Test Competitor Analysis Business Logic
echo "Testing competitor analysis business logic..."
competitor_data='{"your_url":"https://example.com","competitor_url":"https://competitor.com","analysis_type":"comprehensive"}'
competitor_response=$(curl -sf -X POST "$API_BASE_URL/api/competitor-analysis" \
    -H "Content-Type: application/json" \
    -d "$competitor_data" 2>/dev/null || echo "")

if echo "$competitor_response" | jq -e '.comparison' >/dev/null 2>&1; then
    # Verify competitor analysis components
    has_comparison=$(echo "$competitor_response" | jq -e '.comparison' >/dev/null 2>&1 && echo "true" || echo "false")
    has_opportunities=$(echo "$competitor_response" | jq -e '.opportunities' >/dev/null 2>&1 && echo "true" || echo "false")
    has_threats=$(echo "$competitor_response" | jq -e '.threats' >/dev/null 2>&1 && echo "true" || echo "false")
    has_recommendations=$(echo "$competitor_response" | jq -e '.recommendations' >/dev/null 2>&1 && echo "true" || echo "false")

    if [ "$has_comparison" = "true" ] && [ "$has_opportunities" = "true" ] && [ "$has_threats" = "true" ] && [ "$has_recommendations" = "true" ]; then
        echo "Competitor analysis business logic tests passed"
        testing::phase::add_test passed
    else
        echo "Competitor analysis business logic tests failed - missing components"
        testing::phase::add_test failed
    fi
else
    echo "Competitor analysis business logic tests failed - invalid response"
    testing::phase::add_test failed
fi

# Test CLI business workflows
echo "Testing CLI business workflows..."
CLI_BINARY="$TESTING_PHASE_SCENARIO_DIR/cli/seo-optimizer"

if [ -f "$CLI_BINARY" ] && [ -x "$CLI_BINARY" ]; then
    if "$CLI_BINARY" version >/dev/null 2>&1; then
        echo "CLI business workflow tests passed"
        testing::phase::add_test passed
    else
        echo "CLI business workflow tests failed"
        testing::phase::add_test failed
    fi
else
    echo "CLI not available at $CLI_BINARY - skipping"
fi

# Test error handling business logic
echo "Testing error handling business logic..."
invalid_audit_data='{"url":"","depth":2}'
error_response=$(curl -sf -X POST "$API_BASE_URL/api/seo-audit" \
    -H "Content-Type: application/json" \
    -d "$invalid_audit_data" 2>&1 || echo "error")

if echo "$error_response" | grep -q "error\|required\|invalid" 2>/dev/null; then
    echo "Error handling business logic tests passed"
    testing::phase::add_test passed
else
    echo "Error handling business logic tests failed - should reject empty URL"
    testing::phase::add_test failed
fi

echo "Summary: $TESTING_PHASE_TEST_COUNT tests, $TESTING_PHASE_ERROR_COUNT failed"

if [ $TESTING_PHASE_ERROR_COUNT -eq 0 ]; then
    echo "SUCCESS: All business tests passed"
else
    echo "ERROR: Some business tests failed"
fi

# End with summary
testing::phase::end_with_summary "Business logic tests completed"
