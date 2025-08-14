#!/usr/bin/env bats
# App Issue Tracker CLI Tests

load 'test_helper/bats-support/load'
load 'test_helper/bats-assert/load'

CLI_SCRIPT="${BATS_TEST_DIRNAME}/app-issue-tracker.sh"
API_URL="http://localhost:8090/api"

setup() {
    # Skip tests if API is not available
    if ! curl -sf "$API_URL/../health" > /dev/null 2>&1; then
        skip "API server not running at $API_URL"
    fi
}

@test "CLI shows help when no arguments provided" {
    run bash "$CLI_SCRIPT"
    assert_success
    assert_output --partial "App Issue Tracker CLI"
    assert_output --partial "Usage:"
}

@test "CLI shows help with --help flag" {
    run bash "$CLI_SCRIPT" --help
    assert_success
    assert_output --partial "App Issue Tracker CLI"
    assert_output --partial "Commands:"
}

@test "CLI can create an issue" {
    run bash "$CLI_SCRIPT" create --title "Test Issue" --type bug --priority low
    assert_success
    assert_output --partial "Issue created successfully"
    assert_output --partial "Issue ID:"
}

@test "CLI requires title for issue creation" {
    run bash "$CLI_SCRIPT" create --type bug
    assert_failure
    assert_output --partial "Title is required"
}

@test "CLI can list issues" {
    run bash "$CLI_SCRIPT" list --limit 5
    assert_success
    assert_output --partial "Found"
    assert_output --partial "issues"
}

@test "CLI can search issues" {
    run bash "$CLI_SCRIPT" search "test"
    assert_success
    assert_output --partial "Found"
    assert_output --partial "matching issues"
}

@test "CLI can show statistics" {
    run bash "$CLI_SCRIPT" stats
    assert_success
    assert_output --partial "Issue Tracker Statistics"
    assert_output --partial "Total Issues:"
}

@test "CLI handles unknown commands gracefully" {
    run bash "$CLI_SCRIPT" unknown-command
    assert_failure
    assert_output --partial "Unknown command: unknown-command"
}