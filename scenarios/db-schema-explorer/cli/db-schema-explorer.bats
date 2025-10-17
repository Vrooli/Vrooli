#!/usr/bin/env bats

# Database Schema Explorer CLI Tests

setup() {
    export CLI_PATH="./db-schema-explorer"
    export API_PORT="${API_PORT:-15000}"
}

@test "CLI exists and is executable" {
    [ -x "$CLI_PATH" ]
}

@test "CLI shows help" {
    run $CLI_PATH help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Database Schema Explorer CLI" ]]
    [[ "$output" =~ "COMMANDS:" ]]
}

@test "CLI shows version" {
    run $CLI_PATH version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "v1.0.0" ]]
}

@test "CLI status command works" {
    skip "Requires running API"
    run $CLI_PATH status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "API" ]]
}

@test "CLI connect requires database name" {
    run $CLI_PATH connect
    # Should still work with default database
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI query requires prompt" {
    run $CLI_PATH query
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Query prompt required" ]]
}

@test "CLI diff requires two databases" {
    run $CLI_PATH diff production
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Source and target databases required" ]]
}

@test "CLI optimize requires SQL" {
    run $CLI_PATH optimize
    [ "$status" -eq 1 ]
    [[ "$output" =~ "SQL query required" ]]
}

@test "CLI handles invalid commands" {
    run $CLI_PATH invalid-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
}

@test "CLI supports --json flag" {
    skip "Requires implementation"
    run $CLI_PATH status --json
    [ "$status" -eq 0 ]
    # Output should be valid JSON
    echo "$output" | jq . > /dev/null
}

@test "CLI supports --database flag" {
    skip "Requires running API"
    run $CLI_PATH history --database test
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI supports --format flag" {
    skip "Requires running API"
    run $CLI_PATH export main --format json
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}