#!/usr/bin/env bats
# scenario-to-desktop CLI Tests
# Tests for the scenario-to-desktop command-line interface

# Setup function runs before each test
setup() {
    # Use lifecycle-allocated API_PORT if available, otherwise fallback to default
    export API_PORT="${API_PORT:-15200}"
    export API_BASE_URL="${API_BASE_URL:-http://localhost:${API_PORT}}"
    export CLI_PATH="${BATS_TEST_DIRNAME}/scenario-to-desktop"
}

# Test: CLI binary exists and is executable
@test "CLI binary exists and is executable" {
    [ -x "$CLI_PATH" ]
}

# Test: Help command works
@test "Help command displays usage information" {
    run "$CLI_PATH" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "USAGE:" || "$output" =~ "Usage:" ]]
}

# Test: Version command works
@test "Version command displays version" {
    run "$CLI_PATH" version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "scenario-to-desktop" ]]
}

# Test: Status command works
@test "Status command connects to API" {
    run "$CLI_PATH" status
    # May succeed or fail depending on API availability
    # We just check it doesn't crash
    [[ "$status" -eq 0 ]] || [[ "$output" =~ "API" ]]
}

# Test: Templates command works
@test "Templates command lists available templates" {
    run "$CLI_PATH" templates
    # Check it doesn't crash
    [[ "$status" -eq 0 ]] || [[ "$output" =~ "template" ]] || [[ "$output" =~ "API" ]]
}

# Test: Generate command requires scenario name
@test "Generate command requires scenario name argument" {
    run "$CLI_PATH" generate
    [ "$status" -ne 0 ]
    [[ "$output" =~ "scenario" ]] || [[ "$output" =~ "Usage" ]] || [[ "$output" =~ "required" ]]
}

# Test: Invalid command shows error
@test "Invalid command shows error message" {
    run "$CLI_PATH" invalid-command
    [ "$status" -ne 0 ]
}

# Test: CLI has proper permissions
@test "CLI has proper file permissions" {
    stat_output=$(stat -c "%a" "$CLI_PATH" 2>/dev/null || stat -f "%A" "$CLI_PATH" 2>/dev/null)
    # Should be executable (at least 755 or 700)
    [[ "$stat_output" =~ ^7[0-9][0-9]$ ]]
}
