#!/usr/bin/env bats
# CLI tests for idea-generator

setup() {
    # Set test environment
    export IDEA_GENERATOR_API_URL="http://localhost:8500"
    export WINDMILL_URL="http://localhost:5681"
    export N8N_URL="http://localhost:5678"
    
    # Path to CLI script
    CLI_PATH="${BATS_TEST_FILENAME%/*}/idea-generator"
}

@test "CLI shows help when no arguments provided" {
    run $CLI_PATH
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No command specified" ]]
    [[ "$output" =~ "USAGE:" ]]
}

@test "CLI shows help with --help flag" {
    run $CLI_PATH --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Idea Generator CLI" ]]
    [[ "$output" =~ "COMMANDS:" ]]
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "campaigns" ]]
    [[ "$output" =~ "ideas" ]]
}

@test "CLI shows help with help command" {
    run $CLI_PATH help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Idea Generator CLI" ]]
    [[ "$output" =~ "COMMANDS:" ]]
}

@test "CLI handles unknown command gracefully" {
    run $CLI_PATH nonexistent-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command: nonexistent-command" ]]
}

@test "CLI status command exists" {
    # This test just verifies the command is recognized
    # Actual functionality depends on API being available
    run timeout 5 $CLI_PATH status
    # Command should either succeed or fail due to API unavailability
    # but should not fail due to unknown command
    [[ ! "$output" =~ "Unknown command" ]]
}

@test "CLI campaigns command exists" {
    run timeout 5 $CLI_PATH campaigns
    [[ ! "$output" =~ "Unknown command" ]]
}

@test "CLI ideas command exists" {
    run timeout 5 $CLI_PATH ideas
    [[ ! "$output" =~ "Unknown command" ]]
}

@test "CLI workflows command exists" {
    run timeout 5 $CLI_PATH workflows
    [[ ! "$output" =~ "Unknown command" ]]
}

@test "CLI health command exists" {
    run timeout 5 $CLI_PATH health
    [[ ! "$output" =~ "Unknown command" ]]
}