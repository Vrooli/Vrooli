#!/usr/bin/env bats
# CLI status command tests
# Tests for [REQ:CLI-STATUS-001] and [REQ:CLI-STATUS-002]

load '../test/bats-helpers'

setup() {
    # Get API port dynamically
    API_PORT=$(vrooli scenario port vrooli-autoheal API_PORT 2>/dev/null || echo "")
    if [[ -z "$API_PORT" ]]; then
        skip "Scenario not running (API_PORT unavailable)"
    fi
    export VROOLI_AUTOHEAL_API_BASE="http://localhost:${API_PORT}/api/v1"
}

# [REQ:CLI-STATUS-001] Status command shows health summary
@test "[REQ:CLI-STATUS-001] status shows health summary" {
    run ./vrooli-autoheal status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Vrooli Autoheal Status" ]] || [[ "$output" =~ "Overall" ]]
}

# [REQ:CLI-STATUS-001] Status shows platform info
@test "[REQ:CLI-STATUS-001] status shows platform information" {
    run ./vrooli-autoheal status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Platform" ]] || [[ "$output" =~ "platform" ]]
}

# [REQ:CLI-STATUS-002] Status supports JSON output
@test "[REQ:CLI-STATUS-002] status --json outputs valid JSON" {
    run ./vrooli-autoheal status --json
    [ "$status" -eq 0 ]
    echo "$output" | jq -e '.status' >/dev/null
}

# [REQ:CLI-STATUS-002] JSON output contains summary
@test "[REQ:CLI-STATUS-002] status --json contains summary" {
    run ./vrooli-autoheal status --json
    [ "$status" -eq 0 ]
    echo "$output" | jq -e '.summary' >/dev/null
    echo "$output" | jq -e '.platform' >/dev/null
}
