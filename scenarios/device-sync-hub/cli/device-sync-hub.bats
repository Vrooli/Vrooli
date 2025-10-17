#!/usr/bin/env bats
# Device Sync Hub CLI Tests
# Tests for the device-sync-hub CLI tool

setup() {
    # Get the directory containing the CLI script
    CLI_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
    CLI_SCRIPT="$CLI_DIR/device-sync-hub"

    # Ensure CLI script exists and is executable
    if [[ ! -f "$CLI_SCRIPT" ]]; then
        echo "ERROR: CLI script not found at $CLI_SCRIPT" >&2
        return 1
    fi
    chmod +x "$CLI_SCRIPT"

    # Setup test environment with mock URLs to avoid validation errors
    export API_URL="http://localhost:17402"
    export AUTH_URL="http://localhost:15785"
}

teardown() {
    # Clean up test environment
    unset API_URL
    unset AUTH_URL
}

@test "CLI script exists and is executable" {
    [[ -f "$CLI_SCRIPT" ]]
    [[ -x "$CLI_SCRIPT" ]]
}

@test "CLI version command shows version information" {
    run "$CLI_SCRIPT" version
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "Device Sync Hub CLI" ]] || [[ "$output" =~ "1.0.0" ]]
}

@test "CLI help command shows usage information" {
    run "$CLI_SCRIPT" help
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "USAGE:" ]] || [[ "$output" =~ "COMMANDS:" ]]
}

@test "CLI shows help when no command provided" {
    run "$CLI_SCRIPT"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "USAGE:" ]] || [[ "$output" =~ "COMMANDS:" ]]
}

@test "CLI fails gracefully with invalid command" {
    run "$CLI_SCRIPT" invalid-command-xyz
    [[ "$status" -ne 0 ]]
    [[ "$output" =~ "Unknown command" ]] || [[ "$output" =~ "Invalid" ]] || [[ "$output" =~ "ERROR" ]]
}

@test "CLI requires API_URL configuration" {
    unset API_URL
    run "$CLI_SCRIPT" status
    # Should fail with config error or show help
    [[ "$status" -ne 0 ]]
}

@test "CLI list command exists" {
    # Check that help text mentions list command
    run "$CLI_SCRIPT" help
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "list" ]]
}

@test "CLI upload command exists" {
    # Check that help text mentions upload command
    run "$CLI_SCRIPT" help
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "upload" ]]
}

@test "CLI status command exists" {
    # Check that help text mentions status command
    run "$CLI_SCRIPT" help
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "status" ]]
}
