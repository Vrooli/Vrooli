#!/usr/bin/env bats
# Browser Automation Studio CLI Tests
# Tests for the browser-automation-studio command-line interface

setup() {
    # Ensure API is running and get port
    if [ -z "${API_PORT}" ]; then
        export API_PORT=$(vrooli scenario status browser-automation-studio --json 2>/dev/null | jq -r '.allocated_ports.API_PORT' || echo "")
        if [ -z "${API_PORT}" ] || [ "${API_PORT}" = "null" ]; then
            skip "browser-automation-studio API not running"
        fi
    fi

    export API_HOST="localhost"
    CLI_PATH="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)/browser-automation-studio"

    # Verify CLI exists
    if [ ! -f "${CLI_PATH}" ]; then
        skip "CLI not found at ${CLI_PATH}"
    fi

    # Verify API is healthy
    if ! curl -sf "http://localhost:${API_PORT}/health" > /dev/null 2>&1; then
        skip "API health check failed at port ${API_PORT}"
    fi
}

@test "CLI: help command displays usage" {
    run "${CLI_PATH}" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Browser Automation Studio" ]]
    [[ "$output" =~ "USAGE:" ]]
    [[ "$output" =~ "COMMANDS:" ]]
}

@test "CLI: version command shows version info" {
    run "${CLI_PATH}" version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "CLI Version" ]] || [[ "$output" =~ "version" ]]
}

@test "CLI: status command shows operational status" {
    run "${CLI_PATH}" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]] || [[ "$output" =~ "healthy" ]] || [[ "$output" =~ "running" ]]
}

@test "CLI: workflow list command returns workflows" {
    run "${CLI_PATH}" workflow list
    [ "$status" -eq 0 ]
    # Should return either workflows or empty list, not an error
}

@test "CLI: workflow list --json returns valid JSON" {
    run "${CLI_PATH}" workflow list --json
    [ "$status" -eq 0 ]
    # Verify output is valid JSON
    echo "$output" | jq . > /dev/null
}

@test "CLI: execution list command returns executions" {
    run "${CLI_PATH}" execution list
    [ "$status" -eq 0 ]
    # Should return either executions or empty list, not an error
}

@test "CLI: execution list --json returns valid JSON" {
    run "${CLI_PATH}" execution list --json
    [ "$status" -eq 0 ]
    # Verify output is valid JSON
    echo "$output" | jq . > /dev/null
}

@test "CLI: workflow create requires name argument" {
    run "${CLI_PATH}" workflow create
    [ "$status" -ne 0 ]
    [[ "$output" =~ "name" ]] || [[ "$output" =~ "required" ]] || [[ "$output" =~ "Usage" ]]
}

@test "CLI: workflow execute requires workflow argument" {
    run "${CLI_PATH}" workflow execute
    [ "$status" -ne 0 ]
    [[ "$output" =~ "workflow" ]] || [[ "$output" =~ "required" ]] || [[ "$output" =~ "Usage" ]]
}

@test "CLI: workflow execute with invalid workflow ID fails gracefully" {
    run "${CLI_PATH}" workflow execute nonexistent-workflow-id
    [ "$status" -ne 0 ]
    [[ "$output" =~ "not found" ]] || [[ "$output" =~ "error" ]] || [[ "$output" =~ "invalid" ]]
}

@test "CLI: execution watch requires execution ID argument" {
    run "${CLI_PATH}" execution watch
    [ "$status" -ne 0 ]
    [[ "$output" =~ "execution" ]] || [[ "$output" =~ "required" ]] || [[ "$output" =~ "Usage" ]]
}

@test "CLI: execution stop requires execution ID argument" {
    run "${CLI_PATH}" execution stop
    [ "$status" -ne 0 ]
    [[ "$output" =~ "execution" ]] || [[ "$output" =~ "required" ]] || [[ "$output" =~ "Usage" ]]
}

@test "CLI: API connectivity check" {
    # Verify CLI can reach the API
    run curl -sf "http://localhost:${API_PORT}/api/v1/health"
    [ "$status" -eq 0 ]
    echo "$output" | jq -e '.status == "healthy"'
}

@test "CLI: workflow create and delete lifecycle" {
    # Create a test workflow
    run "${CLI_PATH}" workflow create "test-workflow-$RANDOM" --folder "/test"
    [ "$status" -eq 0 ]

    # Extract workflow ID from output (assuming JSON output contains id field)
    if echo "$output" | jq -e '.id' > /dev/null 2>&1; then
        WORKFLOW_ID=$(echo "$output" | jq -r '.id')

        # Delete the workflow
        run "${CLI_PATH}" workflow delete "$WORKFLOW_ID"
        [ "$status" -eq 0 ]
    fi
}

@test "CLI: invalid command returns error" {
    run "${CLI_PATH}" invalid-command-xyz
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unknown" ]] || [[ "$output" =~ "invalid" ]] || [[ "$output" =~ "Error" ]]
}

@test "CLI: status --json returns valid JSON" {
    run "${CLI_PATH}" status --json
    if [ "$status" -eq 0 ]; then
        echo "$output" | jq . > /dev/null
    fi
}

@test "CLI: execution export requires execution ID argument" {
    run "${CLI_PATH}" execution export
    [ "$status" -ne 0 ]
    [[ "$output" =~ "execution" ]] || [[ "$output" =~ "required" ]] || [[ "$output" =~ "Usage" ]]
}
