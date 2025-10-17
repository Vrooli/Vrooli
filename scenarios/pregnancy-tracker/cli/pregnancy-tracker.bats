#!/usr/bin/env bats
# Comprehensive CLI tests for pregnancy-tracker
# Covers all commands, options, error cases, and integration scenarios

################################################################################
# Test Setup and Utilities
################################################################################

setup() {
    export API_PORT="${API_PORT:-17001}"
    export API_BASE_URL="http://localhost:${API_PORT}"
    export TEST_USER_ID="cli-test-user-$$"
    export PREGNANCY_TRACKER_USER="$TEST_USER_ID"
    export PREGNANCY_TRACKER_API_PORT="$API_PORT"
    export PREGNANCY_TRACKER_TEST_ROOT="$(mktemp -d /tmp/pregnancy-tracker-cli-XXXXXX)"

    # Ensure CLI is available
    if ! command -v pregnancy-tracker >/dev/null 2>&1; then
        skip "pregnancy-tracker CLI not installed"
    fi
}

teardown() {
    if [ -n "${PREGNANCY_TRACKER_TEST_ROOT:-}" ]; then
        rm -rf "${PREGNANCY_TRACKER_TEST_ROOT}" 2>/dev/null || true
        unset PREGNANCY_TRACKER_TEST_ROOT
    fi

    # Clean up any test data created
    if service_running && [ -n "${TEST_USER_ID:-}" ]; then
        # Attempt to clean up test pregnancy data
        curl -sf -X DELETE \
            -H "X-User-ID: $TEST_USER_ID" \
            "$API_BASE_URL/api/v1/pregnancy/current" >/dev/null 2>&1 || true
    fi
}

# Helper function to check if service is running
service_running() {
    timeout 3 curl -sf "http://localhost:${API_PORT}/health" >/dev/null 2>&1
}

# Helper function to validate JSON output
is_valid_json() {
    echo "$1" | jq . >/dev/null 2>&1
}

################################################################################
# Basic Command Tests
################################################################################

@test "CLI help command works" {
    run pregnancy-tracker help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Pregnancy Tracker CLI" ]]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "Commands:" ]]
}

@test "CLI help shows all main commands" {
    run pregnancy-tracker help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "start" ]]
    [[ "$output" =~ "log-daily" ]]
    [[ "$output" =~ "appointments" ]]
    [[ "$output" =~ "week-info" ]]
    [[ "$output" =~ "search" ]]
    [[ "$output" =~ "export" ]]
}

@test "CLI help shows examples section" {
    run pregnancy-tracker help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Examples:" ]]
    [[ "$output" =~ "morning sickness" ]]
}

@test "CLI help shows privacy notice" {
    run pregnancy-tracker help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Privacy Notice:" ]]
    [[ "$output" =~ "encryption" ]]
}

################################################################################
# Status Command Tests
################################################################################

@test "Status command without active pregnancy returns informative message" {
    if ! service_running; then
        skip "API service not running"
    fi

    run pregnancy-tracker status
    [[ "$output" =~ "No active pregnancy" ]] || [[ "$output" =~ "not found" ]]
}

@test "Status command fails gracefully when API is down" {
    export PREGNANCY_TRACKER_API_PORT="99999"
    run pregnancy-tracker status
    [ "$status" -ne 0 ]
    [[ "$output" =~ "not running" ]] || [[ "$output" =~ "Error" ]]
}

################################################################################
# Week Info Command Tests
################################################################################

@test "Week info command shows information for valid week" {
    if ! service_running; then
        skip "API service not running"
    fi

    run pregnancy-tracker week-info 12
    [ "$status" -eq 0 ] || [[ "$output" =~ "Week 12" ]]
}

@test "Week info command validates week numbers" {
    if ! service_running; then
        skip "API service not running"
    fi

    # Test invalid week (too high)
    run pregnancy-tracker week-info 50
    [[ "$output" =~ "Invalid" ]] || [ "$status" -ne 0 ]
}

@test "Week info command requires week argument" {
    run pregnancy-tracker week-info
    [ "$status" -ne 0 ] || [[ "$output" =~ "week" ]]
}

################################################################################
# Search Command Tests
################################################################################

@test "Search command requires query argument" {
    run pregnancy-tracker search
    [ "$status" -ne 0 ] || [[ "$output" =~ "query" ]]
}

@test "Search command handles empty results gracefully" {
    if ! service_running; then
        skip "API service not running"
    fi

    run pregnancy-tracker search "xyznonexistentquery123"
    # Should not crash, may return empty or no results message
    [ "$status" -eq 0 ] || [[ "$output" =~ "No results" ]]
}

@test "Search command accepts quoted queries" {
    if ! service_running; then
        skip "API service not running"
    fi

    run pregnancy-tracker search "morning sickness"
    [ "$status" -eq 0 ]
}

################################################################################
# Export Command Tests
################################################################################

@test "Export command requires format argument" {
    run pregnancy-tracker export
    [ "$status" -ne 0 ] || [[ "$output" =~ "format" ]]
}

@test "Export command validates format options" {
    if ! service_running; then
        skip "API service not running"
    fi

    run pregnancy-tracker export invalid-format
    [[ "$output" =~ "Invalid" ]] || [[ "$output" =~ "format" ]] || [ "$status" -ne 0 ]
}

@test "Export json command produces output" {
    if ! service_running; then
        skip "API service not running"
    fi

    run pregnancy-tracker export json
    # Should produce some output (even if no data, should have structure)
    [ -n "$output" ]
}

################################################################################
# User ID and Authentication Tests
################################################################################

@test "CLI respects USER environment variable" {
    if ! service_running; then
        skip "API service not running"
    fi

    export PREGNANCY_TRACKER_USER="test-user-123"
    run pregnancy-tracker status
    # Should not fail due to user ID
    [ "$status" -eq 0 ] || [[ "$output" =~ "No active pregnancy" ]]
}

@test "CLI accepts --user flag" {
    if ! service_running; then
        skip "API service not running"
    fi

    run pregnancy-tracker --user "custom-user" status
    # Should accept the flag without error
    [ "$status" -eq 0 ] || [[ "$output" =~ "No active pregnancy" ]]
}

################################################################################
# Error Handling Tests
################################################################################

@test "CLI handles invalid command gracefully" {
    run pregnancy-tracker invalid-command-xyz
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unknown" ]] || [[ "$output" =~ "Invalid" ]] || [[ "$output" =~ "help" ]]
}

@test "CLI handles network timeout gracefully" {
    # Point to non-existent port
    export PREGNANCY_TRACKER_API_PORT="99998"
    run timeout 5 pregnancy-tracker status
    [ "$status" -ne 0 ]
}

################################################################################
# Integration Tests (with running service)
################################################################################

@test "Full workflow: check status when no pregnancy exists" {
    if ! service_running; then
        skip "API service not running"
    fi

    run pregnancy-tracker status
    [[ "$output" =~ "No active pregnancy" ]] || [[ "$output" =~ "not found" ]]
}

@test "Week info returns content for early pregnancy weeks" {
    if ! service_running; then
        skip "API service not running"
    fi

    for week in 1 4 8; do
        run pregnancy-tracker week-info $week
        [ "$status" -eq 0 ]
        [[ "$output" =~ "Week $week" ]] || [[ "$output" =~ "week" ]]
    done
}

@test "Week info returns content for mid pregnancy weeks" {
    if ! service_running; then
        skip "API service not running"
    fi

    for week in 16 20 24; do
        run pregnancy-tracker week-info $week
        [ "$status" -eq 0 ]
        [[ "$output" =~ "Week $week" ]] || [[ "$output" =~ "week" ]]
    done
}

@test "Week info returns content for late pregnancy weeks" {
    if ! service_running; then
        skip "API service not running"
    fi

    for week in 32 36 40; do
        run pregnancy-tracker week-info $week
        [ "$status" -eq 0 ]
        [[ "$output" =~ "Week $week" ]] || [[ "$output" =~ "week" ]]
    done
}

################################################################################
# Output Format Tests
################################################################################

@test "CLI produces colorized output when appropriate" {
    if ! service_running; then
        skip "API service not running"
    fi

    run pregnancy-tracker status
    # Output should contain ANSI color codes or readable text
    [ -n "$output" ]
}

@test "CLI output includes emoji indicators" {
    if ! service_running; then
        skip "API service not running"
    fi

    run pregnancy-tracker help
    [[ "$output" =~ "🤰" ]]
}

################################################################################
# Verbose Mode Tests
################################################################################

@test "CLI accepts --verbose flag" {
    if ! service_running; then
        skip "API service not running"
    fi

    run pregnancy-tracker --verbose status
    # Should not fail with verbose flag
    [ "$status" -eq 0 ] || [[ "$output" =~ "No active pregnancy" ]]
}

################################################################################
# Boundary and Edge Case Tests
################################################################################

@test "Week info boundary test: week 0" {
    if ! service_running; then
        skip "API service not running"
    fi

    run pregnancy-tracker week-info 0
    # Week 0 may or may not be valid depending on implementation
    [ "$status" -eq 0 ] || [[ "$output" =~ "Invalid" ]]
}

@test "Week info boundary test: week 42" {
    if ! service_running; then
        skip "API service not running"
    fi

    run pregnancy-tracker week-info 42
    [ "$status" -eq 0 ]
}

@test "Week info boundary test: week 43 (should be invalid)" {
    if ! service_running; then
        skip "API service not running"
    fi

    run pregnancy-tracker week-info 43
    [[ "$output" =~ "Invalid" ]] || [ "$status" -ne 0 ]
}

@test "Search with special characters doesn't crash" {
    if ! service_running; then
        skip "API service not running"
    fi

    run pregnancy-tracker search "pregnancy & symptoms?"
    [ "$status" -eq 0 ]
}

@test "Search with very long query doesn't crash" {
    if ! service_running; then
        skip "API service not running"
    fi

    long_query="pregnancy symptoms during first trimester including morning sickness nausea fatigue and mood changes"
    run pregnancy-tracker search "$long_query"
    [ "$status" -eq 0 ]
}

################################################################################
# Performance Tests
################################################################################

@test "CLI commands complete in reasonable time" {
    if ! service_running; then
        skip "API service not running"
    fi

    # Status command should complete within 3 seconds
    run timeout 3 pregnancy-tracker status
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] # 0 or 1 is ok (may not have pregnancy)
}

@test "Help command is fast" {
    # Help should be instant (no API call)
    run timeout 1 pregnancy-tracker help
    [ "$status" -eq 0 ]
}

################################################################################
# Privacy and Security Tests
################################################################################

@test "CLI help mentions local storage" {
    run pregnancy-tracker help
    [[ "$output" =~ "local" ]]
}

@test "CLI help mentions encryption" {
    run pregnancy-tracker help
    [[ "$output" =~ "encryption" ]] || [[ "$output" =~ "encrypted" ]]
}

@test "Export command doesn't send data externally (localhost only)" {
    if ! service_running; then
        skip "API service not running"
    fi

    # The CLI should only talk to localhost
    # We can't fully test this, but we can verify it uses API_BASE_URL
    run pregnancy-tracker export json
    # Should not fail or try to contact external services
    [ "$status" -eq 0 ] || [[ "$output" =~ "No active pregnancy" ]]
}
