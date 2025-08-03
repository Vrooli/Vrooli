#!/usr/bin/env bats

@test "minimal test: bats works" {
    run echo "hello"
    [ "$status" -eq 0 ]
    [ "$output" = "hello" ]
}

@test "minimal test: arithmetic works" {
    result=$((2 + 2))
    [ "$result" -eq 4 ]
}