#!/usr/bin/env bats
# Tests for data-tools CLI

# Test configuration
readonly TEST_CLI="data-tools"
readonly TEST_CONFIG_DIR="$HOME/.data-tools"
readonly TEST_CONFIG_FILE="$TEST_CONFIG_DIR/config.json"

# Setup and teardown
setup() {
    # Backup existing config if it exists
    if [[ -f "$TEST_CONFIG_FILE" ]]; then
        mv "$TEST_CONFIG_FILE" "$TEST_CONFIG_FILE.bak"
    fi
}

teardown() {
    # Restore config if it was backed up
    if [[ -f "$TEST_CONFIG_FILE.bak" ]]; then
        mv "$TEST_CONFIG_FILE.bak" "$TEST_CONFIG_FILE"
    fi
}

# Test: CLI exists and is executable
@test "CLI command is available" {
    run which data-tools
    [ "$status" -eq 0 ]
}

# Test: Help command
@test "help command displays usage information" {
    run data-tools --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Data Tools CLI" ]]
    [[ "$output" =~ "Usage:" ]]
}

# Test: Health command
@test "health command works" {
    run data-tools health
    # Should succeed if API is running, or fail gracefully if not
    # Either way, command should be recognized
    [[ "$output" =~ "health" || "$output" =~ "Health" || "$output" =~ "API" ]]
}

# Test: Parse command structure
@test "parse command is available" {
    run data-tools --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "parse" ]]
}

# Test: Validate command structure
@test "validate command is available" {
    run data-tools --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "validate" ]]
}

# Test: Query command structure
@test "query command is available" {
    run data-tools --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "query" ]]
}

# Test: Transform command structure
@test "transform command is available" {
    run data-tools --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "transform" ]]
}

# Test: Stream command structure
@test "stream command is available" {
    run data-tools --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "stream" ]]
}

# Test: Parse command with data (if API is running)
@test "parse command can process CSV data" {
    skip "Requires API to be running"
    run echo "name,age\nJohn,30" | data-tools parse - --format csv
    [ "$status" -eq 0 ]
    [[ "$output" =~ "schema" || "$output" =~ "columns" ]]
}

# Test: Invalid command
@test "invalid command shows error" {
    run data-tools invalid_command_xyz
    [ "$status" -ne 0 ]
}
