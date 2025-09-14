#!/usr/bin/env bats

# Basic CLI tests for visited-tracker

setup() {
    # Ensure CLI is available
    if ! command -v visited-tracker >/dev/null 2>&1; then
        skip "visited-tracker CLI not installed"
    fi
}

@test "CLI help command works" {
    run visited-tracker help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Visited Tracker CLI" ]]
}

@test "CLI version command works" {
    run visited-tracker version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "visited-tracker CLI" ]]
}

@test "CLI status command works" {
    run visited-tracker status
    # Status might fail if service not running, but should not crash
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI help shows main commands" {
    run visited-tracker help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "visit" ]]
    [[ "$output" =~ "sync" ]]
    [[ "$output" =~ "least-visited" ]]
    [[ "$output" =~ "most-stale" ]]
    [[ "$output" =~ "coverage" ]]
}

@test "CLI version shows correct format" {
    run visited-tracker version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "v[0-9]+\.[0-9]+\.[0-9]+" ]]
}

@test "CLI handles unknown command gracefully" {
    run visited-tracker nonexistent-command
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unknown command" ]] || [[ "$output" =~ "not found" ]] || [[ "$output" =~ "invalid" ]]
}

@test "CLI visit command shows usage when no args" {
    run visited-tracker visit
    # Should show usage or error message
    [ "$status" -ne 0 ] || [[ "$output" =~ "Usage" ]] || [[ "$output" =~ "usage" ]]
}

@test "CLI sync command shows usage when no args" {
    run visited-tracker sync
    # Should show usage or error message  
    [ "$status" -ne 0 ] || [[ "$output" =~ "Usage" ]] || [[ "$output" =~ "usage" ]]
}

# Integration tests (might fail if service not running)
@test "CLI least-visited works with service running" {
    # Skip if service is not running
    if ! curl -sf "http://localhost:${API_PORT:-20252}/health" >/dev/null 2>&1; then
        skip "visited-tracker service not running"
    fi
    
    run visited-tracker least-visited --limit 1 --json
    [ "$status" -eq 0 ]
    [[ "$output" =~ "files" ]]
}

@test "CLI coverage works with service running" {
    # Skip if service is not running
    if ! curl -sf "http://localhost:${API_PORT:-20252}/health" >/dev/null 2>&1; then
        skip "visited-tracker service not running"
    fi
    
    run visited-tracker coverage --json
    [ "$status" -eq 0 ]
    [[ "$output" =~ "coverage_percentage" ]]
}
