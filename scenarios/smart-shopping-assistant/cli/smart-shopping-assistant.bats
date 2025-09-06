#!/usr/bin/env bats

# Smart Shopping Assistant CLI Tests
# Version: 1.0.0

setup() {
    export CLI_PATH="./smart-shopping-assistant"
    export API_PORT="${API_PORT:-3300}"
}

@test "CLI script exists and is executable" {
    [ -f "$CLI_PATH" ]
    [ -x "$CLI_PATH" ]
}

@test "CLI shows help with --help flag" {
    run $CLI_PATH --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Smart Shopping Assistant CLI" ]]
    [[ "$output" =~ "research" ]]
    [[ "$output" =~ "track" ]]
}

@test "CLI shows help with help command" {
    run $CLI_PATH help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "USAGE:" ]]
}

@test "CLI shows version" {
    run $CLI_PATH version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "v1.0.0" ]]
}

@test "CLI requires query for research command" {
    run $CLI_PATH research
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Query required" ]]
}

@test "CLI requires URL for track command" {
    run $CLI_PATH track
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Product URL required" ]]
}

@test "CLI handles unknown commands gracefully" {
    run $CLI_PATH unknown-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
}

@test "CLI accepts JSON output flag" {
    run $CLI_PATH --json version
    [ "$status" -eq 0 ]
}

@test "CLI accepts profile flag" {
    run $CLI_PATH --profile test-profile version
    [ "$status" -eq 0 ]
}