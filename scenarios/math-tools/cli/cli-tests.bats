#!/usr/bin/env bats
# Tests for math-tools CLI

# Test configuration
readonly TEST_CLI="math-tools"
readonly TEST_CONFIG_DIR="$HOME/.math-tools"
readonly TEST_CONFIG_FILE="$TEST_CONFIG_DIR/config.json"

# Setup and teardown
setup() {
    # Backup existing config if it exists
    if [[ -f "$TEST_CONFIG_FILE" ]]; then
        cp "$TEST_CONFIG_FILE" "$TEST_CONFIG_FILE.bak"
    fi
}

teardown() {
    # Restore config if it was backed up
    if [[ -f "$TEST_CONFIG_FILE.bak" ]]; then
        mv "$TEST_CONFIG_FILE.bak" "$TEST_CONFIG_FILE"
    fi
}

# Test: CLI is installed and executable
@test "CLI binary is accessible" {
    command -v "$TEST_CLI"
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
    [[ "$output" =~ "1.0.0" ]]
}

# Test: Status command works
@test "status command checks API health" {
    run $TEST_CLI status
    [ "$status" -eq 0 ]
    # Should contain JSON response with status
    [[ "$output" =~ "status" || "$output" =~ "healthy" ]]
}

# Test: Calc command performs basic operations
@test "calc add performs addition" {
    run $TEST_CLI calc add 2 3
    [ "$status" -eq 0 ]
    [[ "$output" =~ "5" ]]
}

@test "calc mean calculates average" {
    run $TEST_CLI calc mean 10 20 30
    [ "$status" -eq 0 ]
    [[ "$output" =~ "20" ]]
}

# Test: Stats command
@test "stats command analyzes data" {
    # Create a temporary data file
    echo -e "1\n2\n3\n4\n5" > /tmp/test_data.txt
    run $TEST_CLI stats /tmp/test_data.txt
    [ "$status" -eq 0 ]
    [[ "$output" =~ "mean" || "$output" =~ "median" ]]
    rm /tmp/test_data.txt
}

# Test: Invalid command shows error
@test "invalid command shows error or help" {
    run $TEST_CLI invalid_command_xyz
    # Should either show error (exit 1) or help (exit 0)
    [[ "$status" -eq 1 || "$output" =~ "Usage:" ]]
}
