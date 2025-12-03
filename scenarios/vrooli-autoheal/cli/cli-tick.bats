#!/usr/bin/env bats
# CLI tick command tests
# Tests for [REQ:CLI-TICK-001] and [REQ:CLI-TICK-002]

load '../test/bats-helpers'

setup() {
    # Get API port dynamically
    API_PORT=$(vrooli scenario port vrooli-autoheal API_PORT 2>/dev/null || echo "")
    if [[ -z "$API_PORT" ]]; then
        skip "Scenario not running (API_PORT unavailable)"
    fi
    export VROOLI_AUTOHEAL_API_BASE="http://localhost:${API_PORT}/api/v1"
}

# [REQ:CLI-TICK-001] Tick command executes single health cycle
@test "[REQ:CLI-TICK-001] tick command executes health cycle" {
    run ./vrooli-autoheal tick
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Running health check" ]] || [[ "$output" =~ "Tick completed" ]]
}

# [REQ:CLI-TICK-001] Tick command with force flag
@test "[REQ:CLI-TICK-001] tick --force runs all checks" {
    run ./vrooli-autoheal tick --force
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Tick completed" ]]
}

# [REQ:CLI-TICK-002] Tick command emits structured output
@test "[REQ:CLI-TICK-002] tick --json outputs valid JSON" {
    run ./vrooli-autoheal tick --json
    [ "$status" -eq 0 ]
    # Skip the first line (status message) and parse JSON
    local json_output=$(echo "$output" | tail -n +2)
    echo "$json_output" | jq -e '.status' >/dev/null
}

# [REQ:CLI-TICK-002] JSON output contains summary
@test "[REQ:CLI-TICK-002] tick --json contains summary fields" {
    run ./vrooli-autoheal tick --json
    [ "$status" -eq 0 ]
    # Skip the first line (status message) and parse JSON
    local json_output=$(echo "$output" | tail -n +2)
    echo "$json_output" | jq -e '.summary.total >= 0' >/dev/null
    echo "$json_output" | jq -e '.summary.ok >= 0' >/dev/null
}
