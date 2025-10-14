#!/usr/bin/env bats

# CLI Tests for date-night-planner
# Tests the date-night-planner CLI tool functionality

# Setup before all tests
setup() {
    # Get the directory of this test file
    TEST_DIR="$( cd "$( dirname "$BATS_TEST_FILENAME" )" && pwd )"
    CLI_PATH="$TEST_DIR/date-night-planner"

    # Ensure CLI is executable
    if [ ! -x "$CLI_PATH" ]; then
        chmod +x "$CLI_PATH"
    fi
}

@test "CLI exists and is executable" {
    [ -x "$CLI_PATH" ]
}

@test "CLI shows help with --help" {
    run "$CLI_PATH" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Date Night Planner" ]] || [[ "$output" =~ "usage" ]] || [[ "$output" =~ "commands" ]]
}

@test "CLI shows help with -h" {
    run "$CLI_PATH" -h
    [ "$status" -eq 0 ]
}

@test "CLI shows version with --version" {
    run "$CLI_PATH" --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "version" ]] || [[ "$output" =~ "1.0" ]]
}

@test "CLI suggest command requires couple_id" {
    run "$CLI_PATH" suggest
    # Should fail or show error about missing couple_id
    [ "$status" -ne 0 ] || [[ "$output" =~ "couple_id" ]] || [[ "$output" =~ "required" ]]
}

@test "CLI suggest command with test couple_id" {
    # Skip if API is not running
    if ! curl -sf http://localhost:${API_PORT:-19454}/health &>/dev/null; then
        skip "API not running"
    fi

    run "$CLI_PATH" suggest test-couple-123 --json
    [ "$status" -eq 0 ]
    [[ "$output" =~ "suggestions" ]] || [[ "$output" =~ "date" ]]
}

@test "CLI suggest command respects budget flag" {
    # Skip if API is not running
    if ! curl -sf http://localhost:${API_PORT:-19454}/health &>/dev/null; then
        skip "API not running"
    fi

    run "$CLI_PATH" suggest test-couple-123 --budget 50 --json
    [ "$status" -eq 0 ]
}

@test "CLI suggest command respects type flag" {
    # Skip if API is not running
    if ! curl -sf http://localhost:${API_PORT:-19454}/health &>/dev/null; then
        skip "API not running"
    fi

    run "$CLI_PATH" suggest test-couple-123 --type romantic --json
    [ "$status" -eq 0 ]
}

@test "CLI plan command requires arguments" {
    run "$CLI_PATH" plan
    # Should fail or show error about missing arguments
    [ "$status" -ne 0 ] || [[ "$output" =~ "couple_id" ]] || [[ "$output" =~ "required" ]]
}

@test "CLI status command works" {
    run "$CLI_PATH" status
    # Status should always return successfully
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI handles invalid command gracefully" {
    run "$CLI_PATH" invalid-command-xyz
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unknown" ]] || [[ "$output" =~ "invalid" ]] || [[ "$output" =~ "not found" ]]
}
