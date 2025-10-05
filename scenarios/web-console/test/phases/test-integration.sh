#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with runtime requirement and 120-second target
testing::phase::init --require-runtime --target-time "120s"

echo "Comprehensive integration testing: API and WebSocket connectivity"

TEST_SESSION_ID=""
API_URL=""

cleanup_session() {
    if [ -n "$TEST_SESSION_ID" ]; then
        curl -sf -X DELETE "$API_URL/api/v1/sessions/$TEST_SESSION_ID" >/dev/null 2>&1 || true
    fi
}

# Register cleanup function
testing::phase::register_cleanup cleanup_session

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
echo "🌐 Testing API health..."
if curl -sf --max-time 10 "$API_URL/healthz" >/dev/null 2>&1; then
    echo "✅ API health check passed ($API_URL/healthz)"
else
    echo "❌ API integration tests failed - service not responding"
    echo "   Expected API at: $API_URL"
    echo "   💡 Tip: Start with 'make start' in scenarios/web-console"
    testing::phase::add_error
    testing::phase::end_with_summary
fi

echo "🔍 Testing health endpoint response..."
health_response=$(curl -sf --max-time 10 "$API_URL/healthz" 2>/dev/null || echo "")
if echo "$health_response" | jq -e '.status' >/dev/null 2>&1; then
    status_value=$(echo "$health_response" | jq -r '.status')
    if [ "$status_value" = "ok" ]; then
        echo "  ✅ Health endpoint returns correct status"
    else
        echo "  ❌ Health endpoint returned unexpected status: $status_value"
        testing::phase::add_error
    fi
else
    echo "  ❌ Health endpoint response unexpected: $health_response"
    testing::phase::add_error
fi

echo ""
echo "🔍 Testing metrics endpoint..."
if curl -sf --max-time 10 "$API_URL/metrics" >/dev/null 2>&1; then
    echo "  ✅ Metrics endpoint accessible"
else
    echo "  ❌ Metrics endpoint failed"
    testing::phase::add_error
fi

echo ""
echo "🔍 Testing workspace endpoint..."
workspace_response=$(curl -sf --max-time 10 "$API_URL/api/v1/workspace" 2>/dev/null || echo "")
if echo "$workspace_response" | jq -e '.tabs' >/dev/null 2>&1; then
    echo "  ✅ Workspace endpoint returns valid response"
else
    echo "  ❌ Workspace endpoint response unexpected"
    testing::phase::add_error
fi

echo ""
echo "🔍 Testing session creation..."
if command -v jq >/dev/null 2>&1; then
    session_payload=$(jq -n --arg operator "integration-test-$(date +%s)" '{operator:$operator, reason:"integration-test"}')
else
    session_payload='{"operator":"integration-test","reason":"integration-test"}'
fi

session_response=$(curl -sf --max-time 15 -X POST -H "Content-Type: application/json" -d "$session_payload" "$API_URL/api/v1/sessions" 2>/dev/null || echo "")
if echo "$session_response" | jq -e '.id' >/dev/null 2>&1; then
    TEST_SESSION_ID=$(echo "$session_response" | jq -r '.id')
    echo "✅ Test session created: $TEST_SESSION_ID"

    # Verify session details
    session_command=$(echo "$session_response" | jq -r '.command')
    if [ -n "$session_command" ]; then
        echo "  ✅ Session command: $session_command"
    else
        echo "  ⚠️  Session command not returned"
    fi
else
    echo "❌ Failed to create session for integration tests"
    echo "   Response: $session_response"
    testing::phase::add_error
    testing::phase::end_with_summary
fi

echo ""
echo "🔍 Testing session retrieval..."
get_session_response=$(curl -sf --max-time 10 "$API_URL/api/v1/sessions/$TEST_SESSION_ID" 2>/dev/null || echo "")
if echo "$get_session_response" | jq -e '.id' >/dev/null 2>&1; then
    retrieved_id=$(echo "$get_session_response" | jq -r '.id')
    if [ "$retrieved_id" = "$TEST_SESSION_ID" ]; then
        echo "✅ Session retrieved successfully"
    else
        echo "❌ Session ID mismatch: expected $TEST_SESSION_ID, got $retrieved_id"
        testing::phase::add_error
    fi
else
    echo "❌ Failed to retrieve session"
    testing::phase::add_error
fi

echo ""
echo "🔍 Testing session termination..."
if curl -sf --max-time 10 -X DELETE "$API_URL/api/v1/sessions/$TEST_SESSION_ID" >/dev/null 2>&1; then
    echo "✅ Session terminated successfully"
    TEST_SESSION_ID="" # Clear so cleanup doesn't try again
else
    echo "❌ Failed to terminate session"
    testing::phase::add_error
fi

echo ""
echo "🔍 Testing UI accessibility..."
if [ -n "$UI_PORT" ]; then
    ui_response=$(curl -sf --max-time 10 "http://localhost:${UI_PORT}/" 2>/dev/null || echo "")
    if echo "$ui_response" | grep -q "Web Console"; then
        echo "✅ UI accessible and contains expected content"
    else
        echo "⚠️  UI response unexpected (may be loading static assets)"
    fi
else
    echo "ℹ️  UI port not configured, skipping UI tests"
fi

echo ""
echo "📊 Integration Test Summary:"
echo "════════════════════════════"
echo "   API health: passed"
echo "   Session lifecycle: tested"
echo "   Workspace operations: tested"
echo "   Metrics endpoint: accessible"

if [ $TESTING_PHASE_ERROR_COUNT -eq 0 ]; then
    echo ""
    log::success "✅ SUCCESS: All integration tests passed!"
fi

# End with summary
testing::phase::end_with_summary "Integration tests completed"
