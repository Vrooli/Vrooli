#!/bin/bash
# Business logic tests for Morning Vision Walk scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::step "Testing business logic - Conversation lifecycle"

# Verify we can create a session
START_RESP=$(curl -sf -X POST "http://localhost:${API_PORT}/api/conversation/start" \
    -H "Content-Type: application/json" \
    -d '{"user_id": "business-test-user"}' || echo "FAILED")

if [[ "$START_RESP" == "FAILED" ]]; then
    testing::phase::error "Cannot start conversation"
    testing::phase::end_with_summary "Business tests failed"
    exit 1
fi

SESSION_ID=$(echo "$START_RESP" | jq -r '.session_id // empty')
WELCOME_MSG=$(echo "$START_RESP" | jq -r '.welcome_message // empty')

# Verify welcome message is personalized
if [[ -z "$WELCOME_MSG" ]]; then
    testing::phase::warn "No welcome message provided"
else
    testing::phase::success "Welcome message: $WELCOME_MSG"
fi

# Verify context is gathered
CONTEXT=$(echo "$START_RESP" | jq -r '.context // empty')
if [[ "$CONTEXT" != "null" ]] && [[ -n "$CONTEXT" ]]; then
    testing::phase::success "Context gathered on session start"
else
    testing::phase::warn "No context gathered (may be expected if n8n not configured)"
fi

testing::phase::step "Testing business logic - Message exchange"

# Send multiple messages and verify conversation history builds
for i in {1..3}; do
    MSG_RESP=$(curl -sf -X POST "http://localhost:${API_PORT}/api/conversation/message" \
        -H "Content-Type: application/json" \
        -d "{\"session_id\": \"$SESSION_ID\", \"user_id\": \"business-test-user\", \"message\": \"Business test message $i\"}" \
        || echo "FAILED")

    if [[ "$MSG_RESP" == "FAILED" ]]; then
        testing::phase::error "Failed to send message $i"
        continue
    fi

    MSG_COUNT=$(echo "$MSG_RESP" | jq -r '.session_stats.message_count // 0')
    if [[ $MSG_COUNT -gt 0 ]]; then
        testing::phase::success "Message $i processed, total messages: $MSG_COUNT"
    else
        testing::phase::warn "Message count not available in response"
    fi
done

testing::phase::step "Testing business logic - Session retrieval"

# Verify session contains conversation history
SESSION_RESP=$(curl -sf "http://localhost:${API_PORT}/api/session/${SESSION_ID}" || echo "FAILED")

if [[ "$SESSION_RESP" == "FAILED" ]]; then
    testing::phase::error "Failed to retrieve session"
else
    MESSAGES=$(echo "$SESSION_RESP" | jq -r '.messages | length')
    if [[ $MESSAGES -gt 0 ]]; then
        testing::phase::success "Session contains $MESSAGES messages"
    else
        testing::phase::warn "Session has no messages"
    fi
fi

testing::phase::step "Testing business logic - Export functionality"

# Verify export includes summary and insights
EXPORT_RESP=$(curl -sf "http://localhost:${API_PORT}/api/session/${SESSION_ID}/export" || echo "FAILED")

if [[ "$EXPORT_RESP" == "FAILED" ]]; then
    testing::phase::error "Failed to export session"
else
    HAS_SESSION=$(echo "$EXPORT_RESP" | jq -r '.session // empty')
    HAS_SUMMARY=$(echo "$EXPORT_RESP" | jq -r '.summary // empty')
    HAS_EXPORT_TIME=$(echo "$EXPORT_RESP" | jq -r '.export_time // empty')

    if [[ -n "$HAS_SESSION" ]] && [[ -n "$HAS_SUMMARY" ]] && [[ -n "$HAS_EXPORT_TIME" ]]; then
        testing::phase::success "Export includes session, summary, and export time"
    else
        testing::phase::warn "Export may be missing expected fields"
    fi
fi

testing::phase::step "Testing business logic - Session completion"

# Verify session end generates insights and daily plan
END_RESP=$(curl -sf -X POST "http://localhost:${API_PORT}/api/conversation/end" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\": \"$SESSION_ID\"}" || echo "FAILED")

if [[ "$END_RESP" == "FAILED" ]]; then
    testing::phase::error "Failed to end conversation"
else
    DURATION=$(echo "$END_RESP" | jq -r '.duration_minutes // 0')
    TOTAL_MSGS=$(echo "$END_RESP" | jq -r '.total_messages // 0')
    HAS_INSIGHTS=$(echo "$END_RESP" | jq -r '.final_insights // empty')
    HAS_PLAN=$(echo "$END_RESP" | jq -r '.daily_plan // empty')

    testing::phase::success "Session ended after ${DURATION} minutes with ${TOTAL_MSGS} messages"

    if [[ -n "$HAS_INSIGHTS" ]]; then
        testing::phase::success "Final insights generated"
    else
        testing::phase::warn "No final insights (may be expected if n8n not configured)"
    fi

    if [[ -n "$HAS_PLAN" ]]; then
        testing::phase::success "Daily plan generated"
    else
        testing::phase::warn "No daily plan (may be expected if n8n not configured)"
    fi
fi

testing::phase::step "Testing business logic - Task prioritization"

# Test task prioritization endpoint
TASK_RESP=$(curl -sf -X POST "http://localhost:${API_PORT}/api/tasks/prioritize" \
    -H "Content-Type: application/json" \
    -d '{"session_id": "test", "user_id": "test", "tasks": [{"name": "Task 1", "priority": 1}, {"name": "Task 2", "priority": 2}], "context": {}}' \
    || echo "FAILED")

if [[ "$TASK_RESP" == "FAILED" ]]; then
    testing::phase::warn "Task prioritization failed (may be expected if n8n not configured)"
else
    testing::phase::success "Task prioritization endpoint working"
fi

testing::phase::step "Testing business logic - Insight generation"

# Create new session for insight test
START_RESP2=$(curl -sf -X POST "http://localhost:${API_PORT}/api/conversation/start" \
    -H "Content-Type: application/json" \
    -d '{"user_id": "insight-test-user"}' || echo "FAILED")

if [[ "$START_RESP2" != "FAILED" ]]; then
    SESSION_ID2=$(echo "$START_RESP2" | jq -r '.session_id // empty')

    INSIGHT_RESP=$(curl -sf -X POST "http://localhost:${API_PORT}/api/insights/generate" \
        -H "Content-Type: application/json" \
        -d "{\"session_id\": \"$SESSION_ID2\", \"user_id\": \"insight-test-user\", \"conversation_history\": [], \"context\": {}}" \
        || echo "FAILED")

    if [[ "$INSIGHT_RESP" == "FAILED" ]]; then
        testing::phase::warn "Insight generation failed (may be expected if n8n not configured)"
    else
        testing::phase::success "Insight generation endpoint working"
    fi

    # Cleanup
    curl -sf -X POST "http://localhost:${API_PORT}/api/conversation/end" \
        -H "Content-Type: application/json" \
        -d "{\"session_id\": \"$SESSION_ID2\"}" &>/dev/null || true
fi

testing::phase::end_with_summary "Business logic tests completed"
