#!/usr/bin/env bats
# Basic CLI tests for vrooli-autoheal
# Tests basic CLI functionality (help, version, config)

# Test configuration
readonly TEST_CLI="./vrooli-autoheal"
readonly TEST_CONFIG_DIR="$HOME/.vrooli-autoheal"
readonly TEST_CONFIG_FILE="$TEST_CONFIG_DIR/config.json"

# Setup and teardown
setup() {
    cd "$(dirname "$BATS_TEST_FILENAME")" || exit 1
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
    [[ "$output" =~ "version" ]]
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
    run $TEST_CLI configure api_base http://test.example.com
    [ "$status" -eq 0 ]

    run $TEST_CLI configure
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test.example.com" ]]
}

# Test: Help lists supported commands
@test "help output lists supported commands" {
    run $TEST_CLI help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "configure" ]]
}

# Test: Invalid command
@test "invalid command shows error" {
    run $TEST_CLI invalid_command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
}
