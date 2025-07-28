#!/usr/bin/env bats

@test "simple test" {
    [ 1 -eq 1 ]
}

@test "echo test" {
    run echo "hello"
    [ "$status" -eq 0 ]
    [ "$output" = "hello" ]
}