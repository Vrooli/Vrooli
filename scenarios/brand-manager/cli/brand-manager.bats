#!/usr/bin/env bats
# Brand Manager CLI Tests

setup() {
    # Get the directory of this test file
    CLI_DIR="$(cd "${BATS_TEST_FILENAME%/*}" && pwd)"
    CLI_SCRIPT="$CLI_DIR/brand-manager-cli.sh"
    
    # Ensure CLI script exists and is executable
    [[ -f "$CLI_SCRIPT" ]]
    chmod +x "$CLI_SCRIPT"
    
    # Set up test environment
    export BRAND_MANAGER_API_BASE="http://localhost:${API_PORT:-8090}"
    export BRAND_MANAGER_TOKEN="test_token"
}

@test "CLI script exists and is executable" {
    [[ -x "$CLI_SCRIPT" ]]
}

@test "CLI shows version" {
    run "$CLI_SCRIPT" version
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "brand-manager CLI version" ]]
}

@test "CLI shows help" {
    run "$CLI_SCRIPT" help
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "Brand Manager CLI" ]]
    [[ "$output" =~ "Commands:" ]]
}

@test "CLI handles unknown command" {
    run "$CLI_SCRIPT" unknown_command
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Unknown command" ]]
}

@test "CLI shows API base URL" {
    run "$CLI_SCRIPT" api
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "http://localhost:" ]]
}

@test "CLI validates create command arguments" {
    run "$CLI_SCRIPT" create
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI validates get command arguments" {
    run "$CLI_SCRIPT" get
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI validates integrate command arguments" {
    run "$CLI_SCRIPT" integrate
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Usage:" ]]
}

# Note: These tests require the API to be running
# They are designed to pass even if the API is not available
# by checking for appropriate error handling

@test "CLI handles API connection gracefully when service unavailable" {
    # Temporarily override API base to non-existent service
    export BRAND_MANAGER_API_BASE="http://localhost:9999"
    
    run "$CLI_SCRIPT" health
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Failed to connect to API" ]]
}