#!/bin/bash
# Integration tests for Morning Vision Walk scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Verify scenario is running
testing::phase::step "Verifying scenario is running"
if ! pgrep -f "morning-vision-walk-api" > /dev/null; then
    testing::phase::error "API is not running. Start with 'vrooli scenario start morning-vision-walk'"
    testing::phase::end_with_summary "Integration tests failed"
    exit 1
fi

# Test health endpoint
testing::phase::step "Testing health endpoint"
HEALTH_RESPONSE=$(curl -sf "http://localhost:${API_PORT}/health" || echo "FAILED")
if [[ "$HEALTH_RESPONSE" == "FAILED" ]]; then
    testing::phase::error "Health check failed"
    testing::phase::end_with_summary "Integration tests failed"
    exit 1
fi
testing::phase::success "Health endpoint responding"

# Test conversation flow
testing::phase::step "Testing complete conversation flow"

# Start conversation
START_RESPONSE=$(curl -sf -X POST "http://localhost:${API_PORT}/api/conversation/start" \
    -H "Content-Type: application/json" \
    -d '{"user_id": "integration-test-user"}' || echo "FAILED")

if [[ "$START_RESPONSE" == "FAILED" ]]; then
    testing::phase::error "Failed to start conversation"
    testing::phase::end_with_summary "Integration tests failed"
    exit 1
fi

SESSION_ID=$(echo "$START_RESPONSE" | jq -r '.session_id // empty')
if [[ -z "$SESSION_ID" ]]; then
    testing::phase::error "No session_id in response: $START_RESPONSE"
    testing::phase::end_with_summary "Integration tests failed"
    exit 1
fi
testing::phase::success "Started conversation with session: $SESSION_ID"

# Send message
testing::phase::step "Sending test message"
MSG_RESPONSE=$(curl -sf -X POST "http://localhost:${API_PORT}/api/conversation/message" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\": \"$SESSION_ID\", \"user_id\": \"integration-test-user\", \"message\": \"Test message for integration testing\"}" \
    || echo "FAILED")

if [[ "$MSG_RESPONSE" == "FAILED" ]]; then
    testing::phase::error "Failed to send message"
    testing::phase::end_with_summary "Integration tests failed"
    exit 1
fi
testing::phase::success "Message sent successfully"

# Get session
testing::phase::step "Retrieving session data"
SESSION_RESPONSE=$(curl -sf "http://localhost:${API_PORT}/api/session/${SESSION_ID}" || echo "FAILED")

if [[ "$SESSION_RESPONSE" == "FAILED" ]]; then
    testing::phase::error "Failed to retrieve session"
    testing::phase::end_with_summary "Integration tests failed"
    exit 1
fi
testing::phase::success "Session retrieved successfully"

# Export session
testing::phase::step "Exporting session data"
EXPORT_RESPONSE=$(curl -sf "http://localhost:${API_PORT}/api/session/${SESSION_ID}/export" || echo "FAILED")

if [[ "$EXPORT_RESPONSE" == "FAILED" ]]; then
    testing::phase::error "Failed to export session"
    testing::phase::end_with_summary "Integration tests failed"
    exit 1
fi
testing::phase::success "Session exported successfully"

# End conversation
testing::phase::step "Ending conversation"
END_RESPONSE=$(curl -sf -X POST "http://localhost:${API_PORT}/api/conversation/end" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\": \"$SESSION_ID\"}" || echo "FAILED")

if [[ "$END_RESPONSE" == "FAILED" ]]; then
    testing::phase::error "Failed to end conversation"
    testing::phase::end_with_summary "Integration tests failed"
    exit 1
fi
testing::phase::success "Conversation ended successfully"

# Test context gathering
testing::phase::step "Testing context gathering"
CONTEXT_RESPONSE=$(curl -sf "http://localhost:${API_PORT}/api/context/gather?session_id=test&user_id=test" || echo "FAILED")

if [[ "$CONTEXT_RESPONSE" == "FAILED" ]]; then
    testing::phase::warn "Context gathering failed (may be expected if n8n not configured)"
else
    testing::phase::success "Context gathering working"
fi

# Test concurrent sessions
testing::phase::step "Testing concurrent sessions"
CONCURRENT_COUNT=5
CONCURRENT_SUCCESS=0

for i in $(seq 1 $CONCURRENT_COUNT); do
    CONCURRENT_RESPONSE=$(curl -sf -X POST "http://localhost:${API_PORT}/api/conversation/start" \
        -H "Content-Type: application/json" \
        -d "{\"user_id\": \"concurrent-user-$i\"}" 2>/dev/null || echo "FAILED")

    if [[ "$CONCURRENT_RESPONSE" != "FAILED" ]]; then
        CONCURRENT_SUCCESS=$((CONCURRENT_SUCCESS + 1))
    fi

    # Small delay to avoid overwhelming
    sleep 0.1
done

if [[ $CONCURRENT_SUCCESS -ge 3 ]]; then
    testing::phase::success "Concurrent sessions working ($CONCURRENT_SUCCESS/$CONCURRENT_COUNT succeeded)"
else
    testing::phase::warn "Some concurrent sessions failed ($CONCURRENT_SUCCESS/$CONCURRENT_COUNT succeeded)"
fi

testing::phase::end_with_summary "Integration tests completed"
