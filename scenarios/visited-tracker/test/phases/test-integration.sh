#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with runtime requirement and 120-second target
testing::phase::init --require-runtime --target-time "120s"

echo "Comprehensive integration testing: API, CLI (BATS), and Database"

TEST_CAMPAIGN_ID=""
API_URL=""

cleanup_campaign() {
    if [ -n "$TEST_CAMPAIGN_ID" ]; then
        curl -sf -X DELETE "$API_URL/api/v1/campaigns/$TEST_CAMPAIGN_ID" >/dev/null 2>&1 || true
    fi
}

# Register cleanup function
testing::phase::register_cleanup cleanup_campaign

if ! testing::core::wait_for_scenario "$TESTING_PHASE_SCENARIO_NAME" 30; then
    testing::phase::add_error "❌ Scenario '$TESTING_PHASE_SCENARIO_NAME' did not become ready in time"
    testing::phase::end_with_summary
fi

if ! API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME"); then
    testing::phase::add_error "❌ Could not resolve API URL for $TESTING_PHASE_SCENARIO_NAME"
    testing::phase::end_with_summary
fi

# Ensure downstream phases inherit the resolved ports
API_PORT="${API_URL##*:}"
if UI_URL=$(testing::connectivity::get_ui_url "$TESTING_PHASE_SCENARIO_NAME" 2>/dev/null); then
    UI_PORT="${UI_URL##*:}"
fi

echo ""
echo "🌐 Testing API Integration..."
if curl -sf --max-time 10 "$API_URL/health" >/dev/null 2>&1; then
    echo "✅ API health check passed ($API_URL/health)"
else
    echo "❌ API integration tests failed - service not responding"
    echo "   Expected API at: $API_URL"
    echo "   💡 Tip: Start with 'vrooli scenario run $TESTING_PHASE_SCENARIO_NAME'"
    testing::phase::add_error
    testing::phase::end_with_summary
fi

echo "🔍 Testing API endpoints..."
if curl -sf --max-time 10 "$API_URL/api/v1/campaigns" >/dev/null 2>&1; then
    echo "  ✅ Campaigns endpoint accessible"
else
    echo "  ❌ Campaigns endpoint failed"
    testing::phase::add_error
    testing::phase::end_with_summary
fi

health_response=$(curl -sf --max-time 10 "$API_URL/health" 2>/dev/null || echo "")
if [[ "$health_response" =~ "status" ]]; then
    echo "  ✅ Health endpoint returns valid response"
else
    echo "  ❌ Health endpoint response unexpected"
    testing::phase::add_error
    testing::phase::end_with_summary
fi

if command -v jq >/dev/null 2>&1; then
    campaign_payload=$(jq -n --arg name "integration-suite-$(date +%s)" --arg from_agent "integration-tests" --argjson patterns '["**/*.js","**/*.md"]' '{name:$name, from_agent:$from_agent, patterns:$patterns}')
else
    campaign_payload='{"name":"integration-suite","from_agent":"integration-tests","patterns":["**/*.js"]}'
fi

campaign_response=$(curl -sf --max-time 15 -X POST -H "Content-Type: application/json" -d "$campaign_payload" "$API_URL/api/v1/campaigns" 2>/dev/null || echo "")
if echo "$campaign_response" | jq -e '.id' >/dev/null 2>&1; then
    TEST_CAMPAIGN_ID=$(echo "$campaign_response" | jq -r '.id')
    export VISITED_TRACKER_CAMPAIGN_ID="$TEST_CAMPAIGN_ID"
    echo "✅ Test campaign prepared: $TEST_CAMPAIGN_ID"
else
    echo "❌ Failed to create campaign for integration tests"
    testing::phase::add_error
    testing::phase::end_with_summary
fi

echo ""
echo "🖥️  Testing CLI Integration with BATS..."
CLI_TEST_SCRIPT="$TESTING_PHASE_SCENARIO_DIR/test/cli/run-cli-tests.sh"
if [ -x "$CLI_TEST_SCRIPT" ]; then
    if API_PORT="$API_PORT" UI_PORT="${UI_PORT:-}" "$CLI_TEST_SCRIPT"; then
        echo "✅ CLI BATS integration tests passed"
    else
        echo "❌ CLI BATS integration tests failed"
        testing::phase::add_error
        testing::phase::end_with_summary
    fi
else
    echo "❌ CLI test runner missing or not executable: $CLI_TEST_SCRIPT"
    testing::phase::add_error
    testing::phase::end_with_summary
fi

echo ""
echo "🗄️  Testing resource connectivity (PostgreSQL if configured)..."
if command -v resource-postgres >/dev/null 2>&1; then
    if resource-postgres test smoke >/dev/null 2>&1; then
        echo "✅ Database integration tests passed"
    else
        echo "❌ Database integration tests failed"
        testing::phase::add_error
        testing::phase::end_with_summary
    fi
else
    echo "ℹ️  PostgreSQL resource CLI not available; skipping"
fi

echo ""
echo "🔄 Testing end-to-end workflow..."
sync_payload='{"patterns":["**/*.js"]}'
workflow_response=$(curl -sf --max-time 15 -X POST -H "Content-Type: application/json" -d "$sync_payload" "$API_URL/api/v1/campaigns/${TEST_CAMPAIGN_ID}/structure/sync" 2>/dev/null || echo "")
if echo "$workflow_response" | jq -e '.total' >/dev/null 2>&1; then
    echo "✅ End-to-end workflow test passed"
else
    echo "❌ End-to-end workflow test failed: $workflow_response"
    testing::phase::add_error
    testing::phase::end_with_summary
fi

echo ""
echo "📊 Integration Test Summary:"
echo "════════════════════════════"
echo "   API health: passed"
echo "   CLI integration: passed"
echo "   Resource checks: completed"
echo "   Workflow sync: passed"

echo ""
log::success "✅ SUCCESS: All integration tests passed!"

# End with summary
testing::phase::end_with_summary "Integration tests completed"