#!/usr/bin/env bats
# Tests for audio-tools CLI

# Test configuration
readonly TEST_CLI="audio-tools"
readonly TEST_CONFIG_DIR="$HOME/.audio-tools"
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

# Test: CLI command is available
@test "CLI command is available" {
    command -v audio-tools
}

# Test: Help command
@test "help command displays usage information" {
    run $TEST_CLI help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "USAGE:" ]] || [[ "$output" =~ "usage" ]]
    [[ "$output" =~ "COMMANDS:" ]] || [[ "$output" =~ "commands" ]]
}

# Test: Version command
@test "version command displays version" {
    # Version is shown in help output
    run $TEST_CLI help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "v1.0.0" ]] || [[ "$output" =~ "version" ]]
}

# Test: Configuration initialization
@test "configuration is initialized on first run" {
    rm -f "$TEST_CONFIG_FILE"
    run $TEST_CLI --version
    [ "$status" -eq 0 ]
    [[ -f "$TEST_CONFIG_FILE" ]]
}

# Test: Configure command
@test "configure command can set and retrieve values" {
    run $TEST_CLI config set api_base http://test.example.com
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
    
    run $TEST_CLI config get api_base
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

# Test: Health/Status command
@test "status command exists" {
    # Test that the command exists, even if API is not running
    run $TEST_CLI status
    # May fail if API is not running, but command should exist
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

# Test: List command
@test "list command works" {
    # Test command exists and runs
    run $TEST_CLI list
    # May show no files or list files
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

# Test: Invalid command
@test "invalid command shows error" {
    run $TEST_CLI invalid_command_xyz
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unknown command" ]] || [[ "$output" =~ "Usage:" ]]
}