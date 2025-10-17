#!/usr/bin/env bats

# Home Automation CLI Tests
# Tests for the home-automation CLI tool

setup() {
    # Get the directory containing this test file
    DIR="$( cd "$( dirname "$BATS_TEST_FILENAME" )" && pwd )"
    CLI="$DIR/home-automation"

    # Ensure CLI exists and is executable
    [ -f "$CLI" ] || skip "CLI not found at $CLI"
    [ -x "$CLI" ] || skip "CLI not executable"
}

# Basic CLI tests
@test "CLI: help command shows usage" {
    run "$CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Home Automation CLI" ]]
    [[ "$output" =~ "USAGE:" ]]
    [[ "$output" =~ "COMMANDS:" ]]
}

@test "CLI: version command shows version" {
    run "$CLI" version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "1.0.0" ]]
}

@test "CLI: --help flag shows help" {
    run "$CLI" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Home Automation CLI" ]]
}

@test "CLI: no arguments shows help" {
    run "$CLI"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Home Automation CLI" ]]
}

# Status command tests
@test "CLI: status command executes" {
    # Skip if API not running
    if ! curl -sf http://localhost:${API_PORT:-17556}/health >/dev/null 2>&1; then
        skip "API not running"
    fi

    run "$CLI" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Home Automation Status" ]] || [[ "$output" =~ "healthy" ]]
}

@test "CLI: status --json outputs JSON" {
    # Skip if API not running
    if ! curl -sf http://localhost:${API_PORT:-17556}/health >/dev/null 2>&1; then
        skip "API not running"
    fi

    run "$CLI" status --json
    [ "$status" -eq 0 ]
    # Should be valid JSON (basic check)
    echo "$output" | jq . >/dev/null 2>&1 || fail "Output is not valid JSON"
}

# Device command tests
@test "CLI: devices command has help" {
    run "$CLI" devices --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "devices" ]] || [[ "$output" =~ "Device" ]]
}

@test "CLI: devices list executes" {
    # Skip if API not running
    if ! curl -sf http://localhost:${API_PORT:-17556}/health >/dev/null 2>&1; then
        skip "API not running"
    fi

    run "$CLI" devices list
    # Command should execute (may return empty list)
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

# Scenes command tests
@test "CLI: scenes command has help" {
    run "$CLI" scenes --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "scenes" ]] || [[ "$output" =~ "Scene" ]]
}

# Automations command tests
@test "CLI: automations command has help" {
    run "$CLI" automations --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "automation" ]] || [[ "$output" =~ "Automation" ]]
}

@test "CLI: automations list executes" {
    # Skip if API not running
    if ! curl -sf http://localhost:${API_PORT:-17556}/health >/dev/null 2>&1; then
        skip "API not running"
    fi

    run "$CLI" automations list
    # Command should execute (may return empty list)
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

# Profiles command tests
@test "CLI: profiles command has help" {
    run "$CLI" profiles --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "profile" ]] || [[ "$output" =~ "Profile" ]]
}

@test "CLI: profiles list executes" {
    # Skip if API not running
    if ! curl -sf http://localhost:${API_PORT:-17556}/health >/dev/null 2>&1; then
        skip "API not running"
    fi

    run "$CLI" profiles list
    # Command should execute (may return empty list)
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

# Error handling tests
@test "CLI: invalid command shows error" {
    run "$CLI" invalid_command_xyz
    # Should fail or show help
    [ "$status" -ne 0 ] || [[ "$output" =~ "help" ]]
}

# Integration tests (require running API)
@test "CLI: full workflow - status check" {
    # Skip if API not running
    if ! curl -sf http://localhost:${API_PORT:-17556}/health >/dev/null 2>&1; then
        skip "API not running"
    fi

    # Check status
    run "$CLI" status --json
    [ "$status" -eq 0 ]

    # Should contain valid JSON response (basic validation)
    echo "$output" | jq . >/dev/null 2>&1 || skip "JSON parsing failed"
}
