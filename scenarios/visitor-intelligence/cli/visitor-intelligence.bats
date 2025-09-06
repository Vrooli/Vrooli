#!/usr/bin/env bats

# Visitor Intelligence CLI Test Suite
# Tests CLI functionality and API integration

setup() {
    # Set test environment
    export API_PORT="${API_PORT:-8080}"
    CLI_SCRIPT="$BATS_TEST_DIRNAME/visitor-intelligence"
}

@test "CLI script exists and is executable" {
    [ -f "$CLI_SCRIPT" ]
    [ -x "$CLI_SCRIPT" ]
}

@test "CLI shows help when called without arguments" {
    run "$CLI_SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Visitor Intelligence CLI" ]]
    [[ "$output" =~ "USAGE:" ]]
    [[ "$output" =~ "COMMANDS:" ]]
}

@test "CLI help command works" {
    run "$CLI_SCRIPT" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Visitor Intelligence CLI" ]]
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "track" ]]
    [[ "$output" =~ "profile" ]]
    [[ "$output" =~ "analytics" ]]
}

@test "CLI version command works" {
    run "$CLI_SCRIPT" version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "CLI Version:" ]]
    [[ "$output" =~ "API Version:" ]]
    [[ "$output" =~ "API Port:" ]]
}

@test "CLI version command works with --json flag" {
    run "$CLI_SCRIPT" version --json
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"cli_version"' ]]
    [[ "$output" =~ '"api_version"' ]]
    [[ "$output" =~ '"api_port"' ]]
}

@test "CLI shows command-specific help" {
    run "$CLI_SCRIPT" status --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage: visitor-intelligence status" ]]
    [[ "$output" =~ "--json" ]]
    [[ "$output" =~ "--verbose" ]]
}

@test "CLI track command shows help" {
    run "$CLI_SCRIPT" track --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage: visitor-intelligence track" ]]
    [[ "$output" =~ "EVENT_TYPE" ]]
    [[ "$output" =~ "SCENARIO" ]]
}

@test "CLI profile command shows help" {
    run "$CLI_SCRIPT" profile --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage: visitor-intelligence profile" ]]
    [[ "$output" =~ "VISITOR_ID" ]]
    [[ "$output" =~ "--events" ]]
    [[ "$output" =~ "--sessions" ]]
}

@test "CLI analytics command shows help" {
    run "$CLI_SCRIPT" analytics --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage: visitor-intelligence analytics" ]]
    [[ "$output" =~ "SCENARIO" ]]
    [[ "$output" =~ "--timeframe" ]]
    [[ "$output" =~ "--format" ]]
}

@test "CLI handles unknown command gracefully" {
    run "$CLI_SCRIPT" unknown-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command: unknown-command" ]]
}

@test "CLI track command requires arguments" {
    run "$CLI_SCRIPT" track
    [ "$status" -eq 1 ]
    [[ "$output" =~ "EVENT_TYPE and SCENARIO are required" ]]
}

@test "CLI profile command requires arguments" {
    run "$CLI_SCRIPT" profile
    [ "$status" -eq 1 ]
    [[ "$output" =~ "VISITOR_ID is required" ]]
}

@test "CLI analytics command requires arguments" {
    run "$CLI_SCRIPT" analytics
    [ "$status" -eq 1 ]
    [[ "$output" =~ "SCENARIO is required" ]]
}

# Integration tests (only run if API is available)
@test "CLI status command when API is not running" {
    # Test with a non-existent port
    export API_PORT=99999
    run "$CLI_SCRIPT" status
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not running" ]]
}

@test "CLI handles API connection errors gracefully" {
    # Test with a non-existent port
    export API_PORT=99999
    run "$CLI_SCRIPT" track pageview test-scenario
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not running" ]]
}

# Test installation script
@test "Install script exists and is executable" {
    install_script="$BATS_TEST_DIRNAME/install.sh"
    [ -f "$install_script" ]
    [ -x "$install_script" ]
}

# Performance tests
@test "CLI commands respond quickly" {
    start_time=$(date +%s%3N)
    run "$CLI_SCRIPT" help
    end_time=$(date +%s%3N)
    duration=$((end_time - start_time))
    
    # Should respond in less than 1000ms (1 second)
    [ "$duration" -lt 1000 ]
    [ "$status" -eq 0 ]
}

@test "CLI version command responds quickly" {
    start_time=$(date +%s%3N)
    run "$CLI_SCRIPT" version
    end_time=$(date +%s%3N)
    duration=$((end_time - start_time))
    
    # Should respond in less than 2000ms (2 seconds) to account for API call
    [ "$duration" -lt 2000 ]
    [ "$status" -eq 0 ]
}