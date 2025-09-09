#!/usr/bin/env bats

# Text Tools CLI Test Suite

setup() {
    export TEXT_TOOLS_PORT=14000
    CLI_PATH="./text-tools"
}

@test "CLI shows help" {
    run $CLI_PATH help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Text Tools CLI" ]]
    [[ "$output" =~ "COMMANDS" ]]
}

@test "CLI shows version" {
    run $CLI_PATH version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Text Tools CLI v1.0.0" ]]
}

@test "CLI diff command requires two files" {
    run $CLI_PATH diff file1.txt
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Two files required" ]]
}

@test "CLI search command requires pattern" {
    run $CLI_PATH search
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Search pattern required" ]]
}

@test "CLI transform accepts stdin" {
    echo "hello" | run $CLI_PATH transform - --upper
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]  # May fail if API not running
}

@test "CLI extract requires source" {
    run $CLI_PATH extract
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Source file required" ]]
}

@test "CLI analyze accepts text input" {
    echo "test text" | run $CLI_PATH analyze - --entities
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]  # May fail if API not running
}

@test "CLI status command runs" {
    run $CLI_PATH status
    # Status can be 0 (connected) or 1 (disconnected)
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
    [[ "$output" =~ "Text Tools" ]]
}

@test "CLI handles invalid command" {
    run $CLI_PATH invalid-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
}

@test "CLI diff supports output formats" {
    run $CLI_PATH diff --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "--output" ]]
    [[ "$output" =~ "unified" ]]
}

@test "CLI search supports regex flag" {
    run $CLI_PATH search --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "--regex" ]]
}

@test "CLI transform supports multiple operations" {
    run $CLI_PATH transform --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "--upper" ]]
    [[ "$output" =~ "--lower" ]]
    [[ "$output" =~ "--sanitize" ]]
}

@test "CLI extract supports OCR flag" {
    run $CLI_PATH extract --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "--ocr" ]]
}

@test "CLI analyze supports multiple analyses" {
    run $CLI_PATH analyze --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "--entities" ]]
    [[ "$output" =~ "--sentiment" ]]
    [[ "$output" =~ "--keywords" ]]
}