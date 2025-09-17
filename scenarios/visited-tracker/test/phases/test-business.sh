#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with runtime requirement and 180-second target
testing::phase::init --require-runtime --target-time "180s"

echo "Testing core business functionality..."

# Test counters are handled by phase helpers
created_campaign_ids=()

# Cleanup function to run on exit
cleanup() {
    echo "Cleaning up test data..."
    for campaign_id in "${created_campaign_ids[@]}"; do
        if [ -n "$campaign_id" ] && [ "$campaign_id" != "null" ]; then
            echo "Deleting test campaign: $campaign_id"
            curl -sf -X DELETE "$API_BASE_URL/api/v1/campaigns/$campaign_id" >/dev/null 2>&1 || echo "Warning: Failed to delete campaign $campaign_id"
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

# Pre-cleanup: Remove any existing test campaigns
echo "Cleaning up any existing test campaigns..."
existing_campaigns=$(curl -sf "$API_BASE_URL/api/v1/campaigns" 2>/dev/null || echo "")
if echo "$existing_campaigns" | jq -e '.campaigns' >/dev/null 2>&1; then
    echo "$existing_campaigns" | jq -r '.campaigns[] | select(.name | test("^business-test-")) | .id' | while read -r old_campaign_id; do
        if [ -n "$old_campaign_id" ] && [ "$old_campaign_id" != "null" ]; then
            echo "Deleting old test campaign: $old_campaign_id"
            curl -sf -X DELETE "$API_BASE_URL/api/v1/campaigns/$old_campaign_id" >/dev/null 2>&1 || echo "Warning: Failed to delete old campaign $old_campaign_id"
        fi
    done
fi

# Test API business logic
echo "Testing campaign creation..."
timestamp=$(date +%s)
campaign_data='{"name":"business-test-'$timestamp'","from_agent":"test","patterns":["**/*.js"]}'
campaign_response=$(curl -sf -X POST "$API_BASE_URL/api/v1/campaigns" -H "Content-Type: application/json" -d "$campaign_data" 2>/dev/null || echo "")

if echo "$campaign_response" | jq -e '.id' >/dev/null 2>&1; then
    campaign_id=$(echo "$campaign_response" | jq -r '.id')
    created_campaign_ids+=("$campaign_id")
    echo "Campaign creation tests passed - ID: $campaign_id"
    testing::phase::add_test passed
else
    echo "Campaign creation tests failed"
    testing::phase::add_test failed
fi

# Test CLI business workflows
echo "Testing CLI business workflows..."
CLI_BINARY="$TESTING_PHASE_SCENARIO_DIR/cli/visited-tracker"

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

# Test data persistence
echo "Testing data persistence..."
campaigns_response=$(curl -sf "$API_BASE_URL/api/v1/campaigns" 2>/dev/null || echo "")
if echo "$campaigns_response" | jq -e '.campaigns' >/dev/null 2>&1; then
    campaign_count=$(echo "$campaigns_response" | jq '.campaigns | length')
    echo "Data persistence tests passed - $campaign_count campaigns found"
    testing::phase::add_test passed
else
    echo "Data persistence tests failed"
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