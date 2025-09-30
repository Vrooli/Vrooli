#!/usr/bin/env bats

# Document Manager CLI Tests

setup() {
    # Ensure the CLI is in PATH
    export PATH="/home/matthalloran8/.local/bin:$PATH"
    export CLI_CMD="document-manager"
}

@test "CLI help command" {
    run $CLI_CMD --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Document Manager CLI"* ]]
}

@test "CLI version command" {
    run $CLI_CMD --version
    [ "$status" -eq 0 ]
    [[ "$output" == *"2.0.0"* ]]
}

@test "CLI status command" {
    run $CLI_CMD status
    [ "$status" -eq 0 ]
    [[ "$output" == *"healthy"* ]]
}

@test "CLI list applications" {
    run $CLI_CMD applications list
    [ "$status" -eq 0 ]
}

@test "CLI list agents" {
    run $CLI_CMD agents list
    [ "$status" -eq 0 ]
}

@test "CLI list improvements" {
    run $CLI_CMD queue list
    # Accept status 0 or 1 (command works but may have no items)
    [[ "$status" -eq 0 || "$status" -eq 1 ]]
    [[ "$output" == *"Improvement Queue"* ]]
}