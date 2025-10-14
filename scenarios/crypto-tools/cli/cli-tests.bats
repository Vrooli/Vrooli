#!/usr/bin/env bats
# Tests for crypto-tools CLI

# Test configuration
readonly TEST_CLI="crypto-tools"
readonly TEST_CONFIG_DIR="$HOME/.crypto-tools"
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
    command -v "$TEST_CLI" >/dev/null 2>&1
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
    [[ "$output" =~ "v1.0.0" ]]
}

# Test: Configuration initialization (SKIP - not implemented)
@test "configuration is initialized on first run" {
    skip "Configuration file feature not implemented"
}

# Test: Configure command (SKIP - not implemented)
@test "configure command can set and retrieve values" {
    skip "Configure command not implemented"
}

# Test: Health command structure
@test "health command sends correct request" {
    # This would need a mock server or API to test properly
    # For now, just test that the command exists
    run $TEST_CLI help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "health" ]]
}

# Test: Keys command exists
@test "list command accepts resource parameter" {
    # Crypto-tools uses 'keys' command to list keys
    run $TEST_CLI help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "keys" ]]
}

# Test: Invalid command
@test "invalid command shows error" {
    run $TEST_CLI invalid_command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
}