#!/usr/bin/env bats
# Notification Hub CLI Tests

CLI_SCRIPT="${BATS_TEST_DIRNAME}/notification-hub"
# Use the actual allocated API port from the running scenario
API_PORT="${API_PORT:-15309}"
API_URL="http://localhost:${API_PORT}"

setup() {
    # Skip tests if API is not available
    if ! curl -sf "$API_URL/health" > /dev/null 2>&1; then
        skip "API server not running at $API_URL"
    fi
}

@test "CLI shows help when no arguments provided" {
    run "$CLI_SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Notification Hub CLI" ]]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI shows help with --help flag" {
    run "$CLI_SCRIPT" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Notification Hub CLI" ]]
    [[ "$output" =~ "Commands:" ]]
}

@test "CLI lists available commands" {
    run "$CLI_SCRIPT" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "profiles" ]]
    [[ "$output" =~ "contacts" ]]
    [[ "$output" =~ "templates" ]]
    [[ "$output" =~ "notifications" ]]
    [[ "$output" =~ "status" ]]
}

@test "CLI status fails with wrong API URL" {
    run "$CLI_SCRIPT" --api-url="http://localhost:99999" status
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Failed" ]] || [[ "$output" =~ "ERROR" ]]
}

@test "CLI requires API key for profile operations" {
    run "$CLI_SCRIPT" --api-url="$API_URL" profiles list
    [ "$status" -ne 0 ]
    [[ "$output" =~ "API key" ]] || [[ "$output" =~ "api-key" ]]
}

@test "CLI requires API key for notification operations" {
    run "$CLI_SCRIPT" --api-url="$API_URL" notifications send --email test@example.com
    [ "$status" -ne 0 ]
    [[ "$output" =~ "API key" ]] || [[ "$output" =~ "api-key" ]]
}

@test "CLI handles unknown commands gracefully" {
    run "$CLI_SCRIPT" unknown-command
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unknown command" ]] || [[ "$output" =~ "unknown" ]]
}

@test "CLI validates required parameters for notifications" {
    # Should fail without template or recipient
    run "$CLI_SCRIPT" --api-key=test --profile-id=test notifications send
    [ "$status" -ne 0 ]
}

@test "CLI shows usage examples in help" {
    run "$CLI_SCRIPT" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Examples:" ]]
}

@test "CLI supports verbose flag" {
    run "$CLI_SCRIPT" --verbose --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Notification Hub CLI" ]]
}

@test "CLI supports short verbose flag" {
    run "$CLI_SCRIPT" -v --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Notification Hub CLI" ]]
}

@test "CLI supports short help flag" {
    run "$CLI_SCRIPT" -h
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Notification Hub CLI" ]]
}
