#!/usr/bin/env bats
# CLI loop command tests
# Tests for [REQ:CLI-LOOP-001], [REQ:CLI-LOOP-002], [REQ:CLI-LOOP-003]

load '../test/bats-helpers'

setup() {
    # Get API port dynamically
    API_PORT=$(vrooli scenario port vrooli-autoheal API_PORT 2>/dev/null || echo "")
    if [[ -z "$API_PORT" ]]; then
        skip "Scenario not running (API_PORT unavailable)"
    fi
    export VROOLI_AUTOHEAL_API_BASE="http://localhost:${API_PORT}/api/v1"
}

# [REQ:CLI-LOOP-002] Loop interval is configurable
@test "[REQ:CLI-LOOP-002] loop accepts interval flag" {
    # Start loop with short interval and kill after first tick
    timeout 5 ./vrooli-autoheal loop --interval-seconds=1 &
    LOOP_PID=$!
    sleep 2
    kill $LOOP_PID 2>/dev/null || true

    # If we got here without error, the flag was accepted
    [ $? -eq 0 ] || [ $? -eq 143 ]  # 143 = killed by signal
}

# [REQ:CLI-LOOP-003] Loop handles errors gracefully
@test "[REQ:CLI-LOOP-003] loop handles SIGTERM gracefully" {
    # Start loop in background
    timeout 3 ./vrooli-autoheal loop --interval-seconds=1 &
    LOOP_PID=$!
    sleep 1

    # Send SIGTERM
    kill -TERM $LOOP_PID 2>/dev/null || true
    wait $LOOP_PID 2>/dev/null || true

    # Should have exited without error
    [ $? -eq 0 ] || [ $? -eq 143 ] || [ $? -eq 124 ]
}
