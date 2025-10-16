#!/usr/bin/env bats
# PRD Control Tower CLI Tests
#
# These tests verify the CLI commands work correctly against a running
# prd-control-tower scenario. The scenario must be running for these tests to pass.

setup() {
    # Load testing helpers if available
    if [[ -f "test/test_helper.bash" ]]; then
        load test/test_helper
    fi

    # Detect API port using same logic as CLI
    if [[ -n "${PRD_CONTROL_TOWER_API_PORT}" ]]; then
        API_PORT="${PRD_CONTROL_TOWER_API_PORT}"
    elif lsof -ti:18600 -sTCP:LISTEN &>/dev/null; then
        API_PORT="18600"
    elif [[ -n "${API_PORT}" ]] && [[ "${API_PORT}" -ge 15000 ]] && [[ "${API_PORT}" -le 19999 ]]; then
        API_PORT="${API_PORT}"
    else
        API_PORT="18600"
    fi

    API_URL="http://localhost:${API_PORT}/api/v1"
    CLI="./cli/prd-control-tower"
}

# Health Check Tests

@test "CLI: help command shows usage" {
    run "$CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "PRD Control Tower CLI" ]]
    [[ "$output" =~ "USAGE:" ]]
    [[ "$output" =~ "COMMANDS:" ]]
}

@test "CLI: --help flag shows usage" {
    run "$CLI" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "PRD Control Tower CLI" ]]
}

@test "CLI: no arguments shows help" {
    run "$CLI"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "PRD Control Tower CLI" ]]
}

@test "CLI: invalid command shows error" {
    run "$CLI" invalid-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
}

# Status Tests

@test "CLI: status command checks API health" {
    # Check if service is running
    if ! curl -sf "${API_URL}/health" &>/dev/null; then
        skip "Service not running on port ${API_PORT}"
    fi

    run "$CLI" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "PRD Control Tower Status" ]]
    [[ "$output" =~ "API:" ]]
}

@test "CLI: status command detects running service" {
    if ! curl -sf "${API_URL}/health" &>/dev/null; then
        skip "Service not running on port ${API_PORT}"
    fi

    run "$CLI" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "healthy" ]] || [[ "$output" =~ "degraded" ]]
}

@test "CLI: status command shows draft count" {
    if ! curl -sf "${API_URL}/health" &>/dev/null; then
        skip "Service not running on port ${API_PORT}"
    fi

    run "$CLI" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Drafts:" ]]
}

# List Drafts Tests

@test "CLI: list-drafts command runs successfully" {
    if ! curl -sf "${API_URL}/health" &>/dev/null; then
        skip "Service not running on port ${API_PORT}"
    fi

    run "$CLI" list-drafts
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Open Drafts" ]]
}

@test "CLI: list-drafts handles no drafts case" {
    if ! curl -sf "${API_URL}/health" &>/dev/null; then
        skip "Service not running on port ${API_PORT}"
    fi

    run "$CLI" list-drafts
    [ "$status" -eq 0 ]
    # Should either show "No drafts found" or list drafts
    [[ "$output" =~ "No drafts found" ]] || [[ "$output" =~ "/" ]]
}

# Validate Tests

@test "CLI: validate command requires entity name" {
    run "$CLI" validate
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Entity name required" ]]
}

@test "CLI: validate command with scenario name" {
    if ! curl -sf "${API_URL}/health" &>/dev/null; then
        skip "Service not running on port ${API_PORT}"
    fi

    # Use prd-control-tower itself as test subject
    run "$CLI" validate prd-control-tower
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Validating PRD: prd-control-tower" ]]
}

# Port Detection Tests

@test "CLI: respects PRD_CONTROL_TOWER_API_PORT environment variable" {
    PRD_CONTROL_TOWER_API_PORT="19999" run "$CLI" status
    # Should attempt connection to port 19999
    # Will fail if service not running there, but that's expected
    [[ "$output" =~ "19999" ]] || [ "$status" -eq 1 ]
}

@test "CLI: detects service on default port" {
    if ! lsof -ti:18600 -sTCP:LISTEN &>/dev/null; then
        skip "Service not running on default port 18600"
    fi

    run "$CLI" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "18600" ]]
}

# Integration Tests

@test "CLI: end-to-end status check workflow" {
    if ! curl -sf "${API_URL}/health" &>/dev/null; then
        skip "Service not running on port ${API_PORT}"
    fi

    # Run status
    run "$CLI" status
    [ "$status" -eq 0 ]

    # Run list-drafts
    run "$CLI" list-drafts
    [ "$status" -eq 0 ]
}

# Error Handling Tests

@test "CLI: graceful error when API not reachable" {
    # Use invalid port to force connection failure
    PRD_CONTROL_TOWER_API_PORT="65535" run "$CLI" status
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not running" ]] || [[ "$output" =~ "unreachable" ]]
}

@test "CLI: validate gracefully handles API errors" {
    if ! curl -sf "${API_URL}/health" &>/dev/null; then
        skip "Service not running on port ${API_PORT}"
    fi

    # Try to validate non-existent entity
    run "$CLI" validate nonexistent-test-entity-12345
    # Should complete without crashing
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

# Output Format Tests

@test "CLI: status output uses color codes" {
    if ! curl -sf "${API_URL}/health" &>/dev/null; then
        skip "Service not running on port ${API_PORT}"
    fi

    run "$CLI" status
    [ "$status" -eq 0 ]
    # Check for ANSI color codes
    [[ "$output" =~ $'\033' ]] || [[ "$output" =~ $'\x1b' ]]
}

@test "CLI: help output is well-formatted" {
    run "$CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "USAGE:" ]]
    [[ "$output" =~ "COMMANDS:" ]]
    [[ "$output" =~ "EXAMPLES:" ]]
    [[ "$output" =~ "ENVIRONMENT:" ]]
}
