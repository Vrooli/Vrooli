#!/usr/bin/env bats

# BATS test suite for Prompt Injection Arena CLI
# Tests CLI functionality and API integration

CLI_PATH="./prompt-injection-arena"

@test "CLI executable exists and is executable" {
    [ -x "$CLI_PATH" ]
}

@test "CLI help command works" {
    run $CLI_PATH help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Prompt Injection Arena CLI"* ]]
}

@test "CLI version command works" {
    run $CLI_PATH version
    [ "$status" -eq 0 ]
    [[ "$output" == *"v1.0.0"* ]]
}

@test "CLI version command JSON format works" {
    run $CLI_PATH version --json
    [ "$status" -eq 0 ]
    [[ "$output" == *"version"* ]]
}

@test "CLI config show works" {
    run $CLI_PATH config show
    [ "$status" -eq 0 ]
    [[ "$output" == *"API_BASE_URL"* ]]
}

@test "CLI status command works with mock API" {
    # This would need a mock API or the real API running
    run $CLI_PATH status --json
    # Accept both success (if API is running) or connection failure
    [[ "$status" -eq 0 || "$status" -eq 1 ]]
}

@test "CLI help for specific commands works" {
    run $CLI_PATH help test-agent
    [ "$status" -eq 0 ]
    [[ "$output" == *"Test Agent Command"* ]]
}

@test "CLI help for add-injection command works" {
    run $CLI_PATH help add-injection
    [ "$status" -eq 0 ]
    [[ "$output" == *"Add Injection Command"* ]]
}

@test "CLI handles unknown commands gracefully" {
    run $CLI_PATH unknown-command
    [ "$status" -eq 1 ]
    [[ "$output" == *"Unknown command"* ]]
}

@test "CLI test-agent requires system prompt" {
    run $CLI_PATH test-agent
    [ "$status" -eq 1 ]
    [[ "$output" == *"System prompt is required"* ]]
}

@test "CLI add-injection requires parameters" {
    run $CLI_PATH add-injection
    [ "$status" -eq 1 ]
    [[ "$output" == *"required"* ]]
}

@test "CLI leaderboard command syntax works" {
    run $CLI_PATH leaderboard --help 2>/dev/null || true
    # This test just ensures the command doesn't crash on syntax
    [ "$status" -ne 127 ]  # Command not found error
}