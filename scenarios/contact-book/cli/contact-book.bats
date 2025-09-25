#!/usr/bin/env bats

# Contact Book CLI Tests
# Test the contact-book CLI commands for cross-scenario integration

setup() {
    # Ensure API is running
    contact-book status >/dev/null 2>&1 || skip "Contact Book API not running"
}

@test "CLI: status command returns healthy" {
    run contact-book status --json
    [ "$status" -eq 0 ]
    echo "$output" | jq -r '.status' | grep -q "healthy"
}

@test "CLI: help command displays usage" {
    run contact-book help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "Commands:" ]]
}

@test "CLI: version command shows version info" {
    run contact-book version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Contact Book" ]]
}

@test "CLI: list command returns contacts" {
    run contact-book list --json
    [ "$status" -eq 0 ]
    # Check it returns valid JSON with persons array
    echo "$output" | jq -r '.persons' >/dev/null
}

@test "CLI: list with limit parameter" {
    run contact-book list --limit 2 --json
    [ "$status" -eq 0 ]
    # Check we got at most 2 results
    count=$(echo "$output" | jq '.persons | length')
    [ "$count" -le 2 ]
}

@test "CLI: search command finds contacts" {
    run contact-book search "test" --json
    [ "$status" -eq 0 ]
    echo "$output" | jq -r '.results' >/dev/null
}

@test "CLI: search with no results" {
    run contact-book search "xyz123nonexistent" --json
    [ "$status" -eq 0 ]
    count=$(echo "$output" | jq '.count')
    [ "$count" -eq 0 ]
}

@test "CLI: analytics command returns analytics" {
    run contact-book analytics --json
    [ "$status" -eq 0 ]
    echo "$output" | jq -r '.analytics' >/dev/null
}

@test "CLI: maintenance command shows contacts needing attention" {
    run contact-book maintenance --json
    [ "$status" -eq 0 ]
    echo "$output" | jq -r '.contacts' >/dev/null
}

@test "CLI: connect command with missing arguments fails gracefully" {
    run contact-book connect
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Error" ]] || [[ "$output" =~ "required" ]]
}

@test "CLI: invalid command shows error" {
    run contact-book nonexistentcommand
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
}