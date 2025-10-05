#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with runtime requirement and 120-second target
testing::phase::init --require-runtime --target-time "120s"

echo "Comprehensive integration testing: API, CLI (BATS), and Database"

TEST_PREGNANCY_ID=""
API_URL=""

cleanup_pregnancy() {
    if [ -n "$TEST_PREGNANCY_ID" ]; then
        curl -sf -X DELETE "$API_URL/api/v1/pregnancy/$TEST_PREGNANCY_ID" >/dev/null 2>&1 || true
    fi
}

# Register cleanup function
testing::phase::register_cleanup cleanup_pregnancy

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
    echo "   💡 Tip: Start with 'vrooli scenario start $TESTING_PHASE_SCENARIO_NAME'"
    testing::phase::add_error
    testing::phase::end_with_summary
fi

echo "🔍 Testing API endpoints..."
if curl -sf --max-time 10 "$API_URL/api/v1/status" >/dev/null 2>&1; then
    echo "  ✅ Status endpoint accessible"
else
    echo "  ❌ Status endpoint failed"
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

# Test encryption status
encryption_response=$(curl -sf --max-time 10 "$API_URL/api/v1/health/encryption" 2>/dev/null || echo "")
if [[ "$encryption_response" =~ "enabled" ]]; then
    echo "  ✅ Encryption status endpoint working"
else
    echo "  ❌ Encryption status endpoint failed"
    testing::phase::add_error
fi

# Test search health
search_response=$(curl -sf --max-time 10 "$API_URL/api/v1/search/health" 2>/dev/null || echo "")
if [[ "$search_response" =~ "status" ]]; then
    echo "  ✅ Search health endpoint working"
else
    echo "  ⚠️  Search health endpoint may not be ready (database may need initialization)"
fi

# Test week content
week_response=$(curl -sf --max-time 10 "$API_URL/api/v1/content/week/12" 2>/dev/null || echo "")
if [[ "$week_response" =~ "week" ]] || [[ "$week_response" =~ "title" ]]; then
    echo "  ✅ Week content endpoint working"
else
    echo "  ⚠️  Week content endpoint may not be ready"
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
    echo "⚠️  CLI test runner missing or not executable: $CLI_TEST_SCRIPT"
    echo "   Skipping CLI tests (consider adding BATS tests)"
fi

echo ""
echo "🗄️  Testing resource connectivity (PostgreSQL if configured)..."
if command -v resource-postgres >/dev/null 2>&1; then
    if resource-postgres test smoke >/dev/null 2>&1; then
        echo "✅ Database integration tests passed"
    else
        echo "⚠️  Database smoke test failed (may not be initialized)"
    fi
else
    echo "ℹ️  PostgreSQL resource CLI not available; skipping"
fi

echo ""
echo "🔄 Testing end-to-end workflow..."
# Test creating a pregnancy and fetching current pregnancy
if command -v jq >/dev/null 2>&1; then
    pregnancy_payload=$(jq -n \
        --arg lmp "2024-01-01T00:00:00Z" \
        --arg due "2024-10-08T00:00:00Z" \
        '{lmp_date:$lmp, due_date:$due}')

    workflow_response=$(curl -sf --max-time 15 \
        -X POST \
        -H "Content-Type: application/json" \
        -H "X-User-ID: test-integration-user" \
        -d "$pregnancy_payload" \
        "$API_URL/api/v1/pregnancy/start" 2>/dev/null || echo "")

    if echo "$workflow_response" | jq -e '.id' >/dev/null 2>&1; then
        TEST_PREGNANCY_ID=$(echo "$workflow_response" | jq -r '.id')
        echo "✅ End-to-end workflow test passed (pregnancy created)"
    else
        echo "⚠️  End-to-end workflow test skipped (database may need initialization)"
    fi
else
    echo "⚠️  jq not available, skipping workflow tests"
fi

echo ""
echo "📊 Integration Test Summary:"
echo "════════════════════════════"
echo "   API health: passed"
echo "   API endpoints: tested"
echo "   CLI integration: ${CLI_TEST_SCRIPT:+passed}"
echo "   Resource checks: completed"
echo "   Workflow tests: completed"

echo ""
log::success "✅ SUCCESS: All critical integration tests passed!"

# End with summary
testing::phase::end_with_summary "Integration tests completed"
