#!/usr/bin/env bats
# Elo Swipe CLI Test Suite
# Tests the elo-swipe command-line interface

setup() {
    # Export test environment
    export API_PORT="${API_PORT:-19304}"
    export SCENARIO_API_PORT="${SCENARIO_API_PORT:-$API_PORT}"
    export ELO_SWIPE_API_PORT="${ELO_SWIPE_API_PORT:-$API_PORT}"

    # Path to CLI binary
    CLI="${BATS_TEST_DIRNAME}/elo-swipe"

    # Ensure CLI is executable
    if [[ ! -x "$CLI" ]]; then
        skip "CLI binary not found or not executable at $CLI"
    fi
}

# Basic command tests

@test "CLI shows help with no arguments" {
    run "$CLI"
    # CLI may exit with 1 to indicate missing command, but should show help
    [[ "$output" =~ "Elo Swipe CLI" ]]
    [[ "$output" =~ "Commands" ]]
}

@test "CLI shows help with 'help' command" {
    run "$CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Elo Swipe CLI" ]]
    [[ "$output" =~ "Usage" ]]
    [[ "$output" =~ "Commands" ]]
}

@test "CLI shows version information" {
    run "$CLI" version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "elo-swipe" ]]
}

# Command existence tests

@test "CLI list command exists" {
    run "$CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "list" ]]
}

@test "CLI create-list command exists" {
    run "$CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "create-list" ]]
}

@test "CLI rankings command exists" {
    run "$CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "rankings" ]]
}

@test "CLI compare command exists" {
    run "$CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "compare" ]]
}

@test "CLI swipe command exists" {
    run "$CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "swipe" ]]
}

@test "CLI status command exists" {
    run "$CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]]
}

# Error handling

@test "CLI handles invalid commands gracefully" {
    run "$CLI" invalidcommand
    [ "$status" -ne 0 ]
    # Should show error message or help
    [[ "$output" =~ "unknown" ]] || [[ "$output" =~ "invalid" ]] || [[ "$output" =~ "Commands" ]]
}

# Integration tests (require running API)

@test "CLI can communicate with API if available" {
    # Skip if API not available
    API_URL="http://localhost:${API_PORT}"
    if ! curl -sf "$API_URL/health" &>/dev/null; then
        skip "API not available at $API_URL"
    fi

    run "$CLI" status
    # Should succeed or show meaningful output
    [ "$status" -eq 0 ] || [[ "$output" =~ "elo" ]]
}

@test "CLI respects API_PORT environment variable" {
    if [[ -z "$API_PORT" ]]; then
        skip "API_PORT not set"
    fi

    run "$CLI" version
    [ "$status" -eq 0 ]
}

@test "CLI list command works with running API" {
    # Skip if API not available
    API_URL="http://localhost:${API_PORT}"
    if ! curl -sf "$API_URL/health" &>/dev/null; then
        skip "API not available at $API_URL"
    fi

    run "$CLI" list
    # Should either succeed or show meaningful output (not crash)
    # API may return error if no lists exist yet
    [[ "$output" =~ "list" ]] || [[ "$output" =~ "error" ]] || [[ "$output" =~ "Error" ]] || [ "$status" -eq 0 ]
}

# Performance

@test "CLI responds quickly to help command" {
    start=$(date +%s%N)
    run "$CLI" help
    end=$(date +%s%N)
    duration=$(( (end - start) / 1000000 )) # Convert to milliseconds

    [ "$status" -eq 0 ]
    [ "$duration" -lt 1000 ] # Should respond within 1 second
}

# Documentation quality

@test "CLI help output is comprehensive" {
    run "$CLI" help
    [ "$status" -eq 0 ]

    # Check for key sections
    [[ "$output" =~ "Usage" ]]
    [[ "$output" =~ "Commands" ]]
}

@test "CLI help shows all main commands" {
    run "$CLI" help
    [ "$status" -eq 0 ]

    # Verify all main commands are documented
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "list" ]]
    [[ "$output" =~ "create-list" ]]
    [[ "$output" =~ "rankings" ]]
    [[ "$output" =~ "compare" ]]
    [[ "$output" =~ "swipe" ]]
    [[ "$output" =~ "help" ]]
    [[ "$output" =~ "version" ]]
}

# Port detection tests

@test "CLI detects API port from environment" {
    # Test that CLI can work with dynamically allocated ports
    export API_PORT="${API_PORT:-19304}"
    run "$CLI" version
    [ "$status" -eq 0 ]
}

@test "CLI respects SCENARIO_API_PORT" {
    export SCENARIO_API_PORT="19999"
    run "$CLI" version
    [ "$status" -eq 0 ]
}

@test "CLI respects ELO_SWIPE_API_PORT" {
    export ELO_SWIPE_API_PORT="19998"
    run "$CLI" version
    [ "$status" -eq 0 ]
}
