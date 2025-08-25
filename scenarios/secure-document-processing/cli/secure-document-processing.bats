#!/usr/bin/env bats

# Test suite for Secure Document Processing CLI

setup() {
    CLI_PATH="${BATS_TEST_FILENAME%/*}/secure-document-processing"
    export SERVICE_PORT="${SERVICE_PORT:-8090}"
}

@test "CLI shows help when run without arguments" {
    run "$CLI_PATH"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Secure Document Processing CLI" ]]
}

@test "CLI shows help with --help flag" {
    run "$CLI_PATH" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "USAGE:" ]]
    [[ "$output" =~ "COMMANDS:" ]]
}

@test "CLI shows help with help command" {
    run "$CLI_PATH" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Secure Document Processing CLI" ]]
}

@test "CLI handles unknown commands gracefully" {
    run "$CLI_PATH" unknown-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command: unknown-command" ]]
}

@test "CLI status command works when API is running" {
    # This test assumes the API is running during tests
    # In real scenarios, we'd mock or skip if API is not available
    if curl -sf "http://localhost:$SERVICE_PORT/health" >/dev/null 2>&1; then
        run "$CLI_PATH" status
        [ "$status" -eq 0 ]
        [[ "$output" =~ "API is running" ]]
    else
        skip "API not running"
    fi
}