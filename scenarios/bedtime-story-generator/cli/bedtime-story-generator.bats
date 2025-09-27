#!/usr/bin/env bats
# CLI Test Automation for Bedtime Story Generator

setup() {
    # Get the directory containing this test file
    TEST_DIR="$( cd "$( dirname "$BATS_TEST_FILENAME" )" && pwd )"
    CLI_BIN="${TEST_DIR}/bedtime-story"
    
    # Ensure CLI is executable
    if [[ ! -x "$CLI_BIN" ]]; then
        skip "CLI binary not found or not executable: $CLI_BIN"
    fi
}

@test "CLI: help command shows usage" {
    run "$CLI_BIN" help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Bedtime Story Generator CLI"* ]]
    [[ "$output" == *"generate"* ]]
    [[ "$output" == *"list"* ]]
    [[ "$output" == *"read"* ]]
}

@test "CLI: status command shows service health" {
    run "$CLI_BIN" status
    [ "$status" -eq 0 ]
    [[ "$output" == *"API Server"* ]]
}

@test "CLI: themes command lists available themes" {
    run "$CLI_BIN" themes
    [ "$status" -eq 0 ]
    [[ "$output" == *"Adventure"* ]]
    [[ "$output" == *"Animals"* ]]
    [[ "$output" == *"Fantasy"* ]]
}

@test "CLI: list command shows stories" {
    run "$CLI_BIN" list
    [ "$status" -eq 0 ]
    # Output should be either empty list or contain story info
    [[ "$output" == *"Stories"* ]] || [[ "$output" == *"No stories found"* ]]
}

@test "CLI: list with --json flag outputs JSON" {
    run "$CLI_BIN" list --json
    [ "$status" -eq 0 ]
    # Should be valid JSON
    echo "$output" | jq . > /dev/null
}

@test "CLI: generate command requires age-group" {
    # This should fail without age-group
    run "$CLI_BIN" generate
    [ "$status" -ne 0 ] || [[ "$output" == *"age"* ]]
}

@test "CLI: health command returns healthy status" {
    run "$CLI_BIN" health
    [ "$status" -eq 0 ]
    [[ "$output" == *"healthy"* ]]
}

@test "CLI: invalid command shows error" {
    run "$CLI_BIN" invalid-command
    [ "$status" -ne 0 ] || [[ "$output" == *"Unknown"* ]] || [[ "$output" == *"Invalid"* ]]
}

@test "CLI: version info available" {
    run "$CLI_BIN" --version
    # Either shows version or help (both acceptable)
    [ "$status" -eq 0 ] || [[ "$output" == *"help"* ]]
}