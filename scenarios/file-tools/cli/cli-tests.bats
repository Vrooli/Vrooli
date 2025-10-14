#!/usr/bin/env bats
# Tests for file-tools CLI

# Test configuration
# Support running from either the CLI directory or project root
if [[ -f "./file-tools" ]]; then
    readonly TEST_CLI="./file-tools"
elif [[ -f "./cli/file-tools" ]]; then
    readonly TEST_CLI="./cli/file-tools"
elif command -v file-tools >/dev/null 2>&1; then
    readonly TEST_CLI="file-tools"
else
    readonly TEST_CLI="$HOME/.local/bin/file-tools"
fi

readonly TEST_CONFIG_DIR="$HOME/.file-tools"
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
@test "CLI script exists and is executable" {
    [[ -f "$TEST_CLI" ]]
    [[ -x "$TEST_CLI" ]]
}

# Test: Help command
@test "help command displays usage information" {
    run $TEST_CLI help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "Commands:" ]]
}

# Test: Version command
@test "version command displays version" {
    run $TEST_CLI version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "v" ]]
}

# Test: Configuration initialization
@test "configuration is initialized on first run" {
    rm -f "$TEST_CONFIG_FILE"
    run $TEST_CLI version
    [ "$status" -eq 0 ]
    [[ -f "$TEST_CONFIG_FILE" ]]
}

# Test: Configure command
@test "configure command can set and retrieve values" {
    run $TEST_CLI config set api_base http://test.example.com
    [ "$status" -eq 0 ]

    run $TEST_CLI config list
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test.example.com" ]]
}

# Test: Status command structure
@test "status command exists in help" {
    # This would need a mock server or API to test properly
    # For now, just test that the command exists
    run $TEST_CLI help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]]
}

# Test: Compress command structure
@test "compress command exists in help" {
    # Test command structure without actual API
    run $TEST_CLI help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "compress" ]]
}

# Test: Invalid command
@test "invalid command shows error" {
    run $TEST_CLI invalid_command
    [ "$status" -ne 0 ]
    # Check for error in output
}