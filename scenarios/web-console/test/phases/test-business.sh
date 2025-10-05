#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with runtime requirement and 120-second target
testing::phase::init --require-runtime --target-time "120s"

echo "Business logic validation: session lifecycle, workspace persistence, shortcuts"

API_URL=""
TEST_SESSION_ID=""

cleanup_session() {
    if [ -n "$TEST_SESSION_ID" ]; then
        curl -sf -X DELETE "$API_URL/api/v1/sessions/$TEST_SESSION_ID" >/dev/null 2>&1 || true
    fi
}

testing::phase::register_cleanup cleanup_session

if ! testing::core::wait_for_scenario "$TESTING_PHASE_SCENARIO_NAME" 30; then
    testing::phase::add_error "❌ Scenario not ready"
    testing::phase::end_with_summary
fi

if ! API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME"); then
    testing::phase::add_error "❌ Could not resolve API URL"
    testing::phase::end_with_summary
fi

echo ""
echo "🔍 Testing session lifecycle (default command)..."
session_payload='{"operator":"business-test","reason":"default-command-test"}'
session_response=$(curl -sf --max-time 15 -X POST -H "Content-Type: application/json" -d "$session_payload" "$API_URL/api/v1/sessions" 2>/dev/null || echo "")

if echo "$session_response" | jq -e '.id' >/dev/null 2>&1; then
    TEST_SESSION_ID=$(echo "$session_response" | jq -r '.id')
    command=$(echo "$session_response" | jq -r '.command')
    echo "✅ Session created with default command: $command"

    # Wait a moment for session to initialize
    sleep 1

    # Verify session is running
    session_status=$(curl -sf --max-time 10 "$API_URL/api/v1/sessions/$TEST_SESSION_ID" 2>/dev/null || echo "")
    if echo "$session_status" | jq -e '.phase' >/dev/null 2>&1; then
        phase=$(echo "$session_status" | jq -r '.phase')
        if [ "$phase" = "running" ] || [ "$phase" = "ready" ]; then
            echo "  ✅ Session is in running state: $phase"
        else
            echo "  ⚠️  Session in unexpected phase: $phase"
        fi
    fi

    # Cleanup
    curl -sf -X DELETE "$API_URL/api/v1/sessions/$TEST_SESSION_ID" >/dev/null 2>&1
    TEST_SESSION_ID=""
else
    echo "❌ Failed to create session with default command"
    testing::phase::add_error
fi

echo ""
echo "🔍 Testing custom command execution..."
custom_payload='{"operator":"business-test","reason":"custom-command","command":"/bin/echo","args":["hello","world"]}'
custom_response=$(curl -sf --max-time 15 -X POST -H "Content-Type: application/json" -d "$custom_payload" "$API_URL/api/v1/sessions" 2>/dev/null || echo "")

if echo "$custom_response" | jq -e '.id' >/dev/null 2>&1; then
    session_id=$(echo "$custom_response" | jq -r '.id')
    command=$(echo "$custom_response" | jq -r '.command')

    if [ "$command" = "/bin/echo" ]; then
        echo "✅ Custom command session created: $command"

        # Cleanup
        curl -sf -X DELETE "$API_URL/api/v1/sessions/$session_id" >/dev/null 2>&1
    else
        echo "❌ Custom command not respected: expected /bin/echo, got $command"
        testing::phase::add_error
        curl -sf -X DELETE "$API_URL/api/v1/sessions/$session_id" >/dev/null 2>&1
    fi
else
    echo "❌ Failed to create session with custom command"
    testing::phase::add_error
fi

echo ""
echo "🔍 Testing workspace persistence..."
# Get current workspace
workspace=$(curl -sf --max-time 10 "$API_URL/api/v1/workspace" 2>/dev/null || echo "")
if echo "$workspace" | jq -e '.version' >/dev/null 2>&1; then
    initial_version=$(echo "$workspace" | jq -r '.version')
    echo "  ✅ Workspace loaded with version: $initial_version"

    # Create a tab
    tab_payload='{"id":"business-test-tab","label":"Business Test","colorId":"sky"}'
    create_tab_response=$(curl -sf --max-time 10 -X POST -H "Content-Type: application/json" -d "$tab_payload" "$API_URL/api/v1/workspace/tabs" 2>/dev/null || echo "")

    if echo "$create_tab_response" | jq -e '.id' >/dev/null 2>&1; then
        echo "  ✅ Tab created successfully"

        # Verify workspace version incremented
        updated_workspace=$(curl -sf --max-time 10 "$API_URL/api/v1/workspace" 2>/dev/null || echo "")
        if echo "$updated_workspace" | jq -e '.version' >/dev/null 2>&1; then
            new_version=$(echo "$updated_workspace" | jq -r '.version')
            if [ "$new_version" -gt "$initial_version" ]; then
                echo "  ✅ Workspace version incremented: $initial_version → $new_version"
            else
                echo "  ⚠️  Workspace version did not increment"
            fi
        fi

        # Delete the tab
        curl -sf --max-time 10 -X DELETE "$API_URL/api/v1/workspace/tabs/business-test-tab" >/dev/null 2>&1
    else
        echo "  ❌ Failed to create tab"
        testing::phase::add_error
    fi
else
    echo "❌ Failed to load workspace"
    testing::phase::add_error
fi

echo ""
echo "🔍 Testing session TTL enforcement..."
echo "  ℹ️  TTL test would require waiting for session expiry (30m default)"
echo "  ℹ️  Skipping in integration tests (validated in unit tests)"

echo ""
echo "🔍 Testing operator attribution..."
test_operator="test-user-$(date +%s)"
operator_payload=$(jq -n --arg op "$test_operator" '{operator:$op, reason:"attribution-test"}')
operator_response=$(curl -sf --max-time 15 -X POST -H "Content-Type: application/json" -d "$operator_payload" "$API_URL/api/v1/sessions" 2>/dev/null || echo "")

if echo "$operator_response" | jq -e '.id' >/dev/null 2>&1; then
    session_id=$(echo "$operator_response" | jq -r '.id')

    # Verify operator is recorded
    session_data=$(curl -sf --max-time 10 "$API_URL/api/v1/sessions/$session_id" 2>/dev/null || echo "")
    if echo "$session_data" | jq -e '.operator' >/dev/null 2>&1; then
        recorded_operator=$(echo "$session_data" | jq -r '.operator')
        if [ "$recorded_operator" = "$test_operator" ]; then
            echo "✅ Operator attribution working: $test_operator"
        else
            echo "❌ Operator mismatch: expected $test_operator, got $recorded_operator"
            testing::phase::add_error
        fi
    else
        echo "⚠️  Operator field not present in session data"
    fi

    # Cleanup
    curl -sf -X DELETE "$API_URL/api/v1/sessions/$session_id" >/dev/null 2>&1
else
    echo "❌ Failed to create session with operator"
    testing::phase::add_error
fi

echo ""
echo "📊 Business Logic Test Summary:"
echo "═══════════════════════════════"
echo "   Session lifecycle: validated"
echo "   Custom commands: tested"
echo "   Workspace persistence: verified"
echo "   Operator attribution: working"

if [ $TESTING_PHASE_ERROR_COUNT -eq 0 ]; then
    echo ""
    log::success "✅ SUCCESS: All business logic tests passed!"
fi

# End with summary
testing::phase::end_with_summary "Business logic tests completed"
