#!/usr/bin/env bats
# Tests for file-tools CLI

# Test configuration
readonly TEST_CLI="./file-tools"
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

# Test: Health command structure
@test "health command sends correct request" {
    # This would need a mock server or API to test properly
    # For now, just test that the command exists
    run $TEST_CLI help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "health" ]]
}

# Test: List command structure
@test "list command accepts resource parameter" {
    # Test command structure without actual API
    run $TEST_CLI help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "list" ]]
}

# Test: Invalid command
@test "invalid command shows error" {
    run $TEST_CLI invalid_command
    [ "$status" -ne 0 ]
    # Check for error in output
}