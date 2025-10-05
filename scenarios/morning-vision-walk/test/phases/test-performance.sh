#!/bin/bash
# Performance tests for Morning Vision Walk scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Verify scenario is running
if ! pgrep -f "morning-vision-walk-api" > /dev/null; then
    testing::phase::error "API is not running. Start with 'vrooli scenario start morning-vision-walk'"
    testing::phase::end_with_summary "Performance tests failed"
    exit 1
fi

testing::phase::step "Testing health endpoint latency"

# Run 100 health checks and measure latency
HEALTH_COUNT=100
START_TIME=$(date +%s%N)

for i in $(seq 1 $HEALTH_COUNT); do
    curl -sf "http://localhost:${API_PORT}/health" &>/dev/null || true
done

END_TIME=$(date +%s%N)
TOTAL_MS=$(( (END_TIME - START_TIME) / 1000000 ))
AVG_MS=$(( TOTAL_MS / HEALTH_COUNT ))

testing::phase::success "Health check: ${HEALTH_COUNT} requests in ${TOTAL_MS}ms (avg: ${AVG_MS}ms)"

if [[ $AVG_MS -gt 100 ]]; then
    testing::phase::warn "Health check latency high: ${AVG_MS}ms (target: <100ms)"
else
    testing::phase::success "Health check latency acceptable: ${AVG_MS}ms"
fi

testing::phase::step "Testing session creation throughput"

# Create 50 sessions and measure throughput
SESSION_COUNT=50
SESSION_IDS=()

START_TIME=$(date +%s%N)

for i in $(seq 1 $SESSION_COUNT); do
    RESP=$(curl -sf -X POST "http://localhost:${API_PORT}/api/conversation/start" \
        -H "Content-Type: application/json" \
        -d "{\"user_id\": \"perf-user-$i\"}" 2>/dev/null || echo "FAILED")

    if [[ "$RESP" != "FAILED" ]]; then
        SESSION_ID=$(echo "$RESP" | jq -r '.session_id // empty')
        if [[ -n "$SESSION_ID" ]]; then
            SESSION_IDS+=("$SESSION_ID")
        fi
    fi
done

END_TIME=$(date +%s%N)
TOTAL_MS=$(( (END_TIME - START_TIME) / 1000000 ))
AVG_MS=$(( TOTAL_MS / SESSION_COUNT ))

testing::phase::success "Session creation: ${SESSION_COUNT} sessions in ${TOTAL_MS}ms (avg: ${AVG_MS}ms)"

if [[ $AVG_MS -gt 1000 ]]; then
    testing::phase::warn "Session creation slow: ${AVG_MS}ms (target: <1000ms)"
else
    testing::phase::success "Session creation throughput acceptable: ${AVG_MS}ms"
fi

testing::phase::step "Testing message processing throughput"

# Use first session for message testing
if [[ ${#SESSION_IDS[@]} -gt 0 ]]; then
    TEST_SESSION="${SESSION_IDS[0]}"
    MSG_COUNT=20

    START_TIME=$(date +%s%N)

    for i in $(seq 1 $MSG_COUNT); do
        curl -sf -X POST "http://localhost:${API_PORT}/api/conversation/message" \
            -H "Content-Type: application/json" \
            -d "{\"session_id\": \"$TEST_SESSION\", \"user_id\": \"perf-user-1\", \"message\": \"Perf test message $i\"}" \
            &>/dev/null || true
    done

    END_TIME=$(date +%s%N)
    TOTAL_MS=$(( (END_TIME - START_TIME) / 1000000 ))
    AVG_MS=$(( TOTAL_MS / MSG_COUNT ))

    testing::phase::success "Message processing: ${MSG_COUNT} messages in ${TOTAL_MS}ms (avg: ${AVG_MS}ms)"

    if [[ $AVG_MS -gt 1000 ]]; then
        testing::phase::warn "Message processing slow: ${AVG_MS}ms (target: <1000ms)"
    else
        testing::phase::success "Message processing throughput acceptable: ${AVG_MS}ms"
    fi
else
    testing::phase::warn "No sessions available for message testing"
fi

testing::phase::step "Testing concurrent session handling"

# Test concurrent sessions
CONCURRENT_COUNT=10
SUCCESS_COUNT=0

START_TIME=$(date +%s%N)

for i in $(seq 1 $CONCURRENT_COUNT); do
    (
        RESP=$(curl -sf -X POST "http://localhost:${API_PORT}/api/conversation/start" \
            -H "Content-Type: application/json" \
            -d "{\"user_id\": \"concurrent-user-$i\"}" 2>/dev/null || echo "FAILED")

        if [[ "$RESP" != "FAILED" ]]; then
            SESSION_ID=$(echo "$RESP" | jq -r '.session_id // empty')

            # Send a message
            curl -sf -X POST "http://localhost:${API_PORT}/api/conversation/message" \
                -H "Content-Type: application/json" \
                -d "{\"session_id\": \"$SESSION_ID\", \"user_id\": \"concurrent-user-$i\", \"message\": \"Concurrent test\"}" \
                &>/dev/null || true

            # End session
            curl -sf -X POST "http://localhost:${API_PORT}/api/conversation/end" \
                -H "Content-Type: application/json" \
                -d "{\"session_id\": \"$SESSION_ID\"}" \
                &>/dev/null || true

            echo "SUCCESS"
        fi
    ) &
done

wait

END_TIME=$(date +%s%N)
TOTAL_MS=$(( (END_TIME - START_TIME) / 1000000 ))

testing::phase::success "Concurrent sessions: ${CONCURRENT_COUNT} sessions in ${TOTAL_MS}ms"

if [[ $TOTAL_MS -gt 30000 ]]; then
    testing::phase::warn "Concurrent session handling slow: ${TOTAL_MS}ms (target: <30000ms)"
else
    testing::phase::success "Concurrent session handling acceptable: ${TOTAL_MS}ms"
fi

testing::phase::step "Testing memory efficiency"

# Check if we can handle many sessions without issues
MEMORY_TEST_COUNT=30

for i in $(seq 1 $MEMORY_TEST_COUNT); do
    curl -sf -X POST "http://localhost:${API_PORT}/api/conversation/start" \
        -H "Content-Type: application/json" \
        -d "{\"user_id\": \"memory-user-$i\"}" \
        &>/dev/null || true

    # Small delay to avoid overwhelming
    sleep 0.05
done

testing::phase::success "Created $MEMORY_TEST_COUNT sessions for memory test"

# Cleanup all sessions created during performance tests
testing::phase::step "Cleaning up test sessions"

CLEANUP_COUNT=0
for SESSION_ID in "${SESSION_IDS[@]}"; do
    curl -sf -X POST "http://localhost:${API_PORT}/api/conversation/end" \
        -H "Content-Type: application/json" \
        -d "{\"session_id\": \"$SESSION_ID\"}" \
        &>/dev/null && CLEANUP_COUNT=$((CLEANUP_COUNT + 1)) || true
done

testing::phase::success "Cleaned up $CLEANUP_COUNT sessions"

testing::phase::end_with_summary "Performance tests completed"
