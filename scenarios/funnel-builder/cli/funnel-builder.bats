#!/usr/bin/env bats
# BATS tests for funnel-builder CLI

setup() {
    # Ensure CLI is in PATH
    export PATH="$HOME/.local/bin:$PATH"

    # Check if CLI is available
    if ! command -v funnel-builder &> /dev/null; then
        skip "funnel-builder CLI not installed"
    fi
}

@test "CLI: help command works" {
    run funnel-builder help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "funnel-builder" ]]
}

@test "CLI: version command works" {
    run funnel-builder version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "version" ]] || [[ "$output" =~ "1.0" ]]
}

@test "CLI: status command works" {
    run funnel-builder status
    [ "$status" -eq 0 ]
    # Should show either healthy or provide status info
}

@test "CLI: status --json returns valid JSON" {
    run funnel-builder status --json
    [ "$status" -eq 0 ]
    # Basic JSON validation - should contain braces
    [[ "$output" =~ "{" ]]
    [[ "$output" =~ "}" ]]
}

@test "CLI: list command works" {
    run funnel-builder list
    [ "$status" -eq 0 ]
    # Should not error even if empty
}

@test "CLI: invalid command shows error" {
    run funnel-builder invalid-command-xyz
    [ "$status" -ne 0 ]
}

@test "CLI: create requires name argument" {
    run funnel-builder create
    [ "$status" -ne 0 ]
    [[ "$output" =~ "name" ]] || [[ "$output" =~ "required" ]] || [[ "$output" =~ "usage" ]]
}

@test "CLI: analytics requires funnel_id argument" {
    run funnel-builder analytics
    [ "$status" -ne 0 ]
}

@test "CLI: export-leads requires funnel_id argument" {
    run funnel-builder export-leads
    [ "$status" -ne 0 ]
}

@test "CLI: help shows all available commands" {
    run funnel-builder help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "help" ]]
    [[ "$output" =~ "version" ]]
}
