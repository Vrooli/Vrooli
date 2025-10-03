#!/usr/bin/env bats

# Agent Dashboard CLI Tests

setup() {
    export CLI_PATH="./cli/agent-dashboard"
    export API_PORT="${API_PORT:-19087}"
}

@test "CLI help command shows usage" {
    run $CLI_PATH --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Agent Dashboard CLI"* ]]
    [[ "$output" == *"Commands:"* ]]
}

@test "CLI version command shows version" {
    run $CLI_PATH version
    [ "$status" -eq 0 ]
    [[ "$output" == *"Version:"* ]]
}

@test "CLI list command works" {
    run $CLI_PATH list
    [ "$status" -eq 0 ]
    [[ "$output" == *"Active Agents"* ]] || [[ "$output" == *"No agents"* ]]
}

@test "CLI status command shows dashboard status" {
    run $CLI_PATH status
    [ "$status" -eq 0 ]
    [[ "$output" == *"Dashboard Status"* ]] || [[ "$output" == *"API"* ]]
}

@test "CLI scan command triggers discovery" {
    run $CLI_PATH scan
    [ "$status" -eq 0 ]
    [[ "$output" == *"scan"* ]] || [[ "$output" == *"discovery"* ]]
}

@test "CLI with invalid command shows error" {
    run $CLI_PATH invalid-command
    [ "$status" -ne 0 ]
    [[ "$output" == *"Unknown command"* ]] || [[ "$output" == *"Usage:"* ]]
}

@test "CLI logs command with --help shows usage" {
    run $CLI_PATH logs --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"logs"* ]] || [[ "$output" == *"Usage:"* ]]
}