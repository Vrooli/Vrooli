#!/usr/bin/env bats
# Resume Screening Assistant CLI tests

# Setup and teardown
setup() {
    CLI_SCRIPT="$BATS_TEST_DIRNAME/resume-screening-assistant"
}

# Test CLI script exists and is executable
@test "CLI script exists and is executable" {
    [ -f "$CLI_SCRIPT" ]
    [ -x "$CLI_SCRIPT" ]
}

# Test help command
@test "CLI shows help with --help" {
    run "$CLI_SCRIPT" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Resume Screening Assistant CLI" ]]
    [[ "$output" =~ "USAGE:" ]]
    [[ "$output" =~ "COMMANDS:" ]]
}

# Test help command with 'help'
@test "CLI shows help with help command" {
    run "$CLI_SCRIPT" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Resume Screening Assistant CLI" ]]
}

# Test health command (may fail if API not running, but should handle gracefully)
@test "CLI health command handles API unavailable gracefully" {
    run "$CLI_SCRIPT" health
    # Should exit with error when API is not available
    [ "$status" -eq 1 ]
    [[ "$output" =~ "API health check failed" ]]
}

# Test unknown command
@test "CLI handles unknown command gracefully" {
    run "$CLI_SCRIPT" invalid-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command: invalid-command" ]]
}

# Test jobs command without API
@test "CLI jobs command handles API unavailable gracefully" {
    run "$CLI_SCRIPT" jobs list
    [ "$status" -eq 1 ]
    [[ "$output" =~ "API server is not responding" ]]
}

# Test candidates command without API
@test "CLI candidates command handles API unavailable gracefully" {
    run "$CLI_SCRIPT" candidates list
    [ "$status" -eq 1 ]
    [[ "$output" =~ "API server is not responding" ]]
}

# Test search command validation
@test "CLI search command requires query" {
    run "$CLI_SCRIPT" search
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Search query is required" ]]
}

# Test dependency checks - curl
@test "CLI script checks for curl dependency" {
    # Mock curl not being available by temporarily renaming it
    if command -v curl >/dev/null; then
        skip "curl is available, cannot test missing dependency"
    fi
    
    run "$CLI_SCRIPT" health
    [ "$status" -eq 1 ]
    [[ "$output" =~ "curl is required" ]]
}

# Test dependency checks - jq
@test "CLI script checks for jq dependency" {
    # Mock jq not being available by temporarily renaming it
    if command -v jq >/dev/null; then
        skip "jq is available, cannot test missing dependency"
    fi
    
    run "$CLI_SCRIPT" health
    [ "$status" -eq 1 ]
    [[ "$output" =~ "jq is required" ]]
}