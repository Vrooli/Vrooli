#!/usr/bin/env bats

# Use full path to CLI binary
CLI_PATH="${BATS_TEST_DIRNAME}/algorithm-library"

@test "algorithm-library help command" {
    run "$CLI_PATH" --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Algorithm Library CLI"* ]]
}

@test "algorithm-library search command" {
    run "$CLI_PATH" search quicksort
    [ "$status" -eq 0 ]
    [[ "$output" == *"QuickSort"* ]]
}

@test "algorithm-library categories command" {
    run "$CLI_PATH" categories
    [ "$status" -eq 0 ]
    [[ "$output" == *"sorting"* ]]
}

@test "algorithm-library stats command" {
    run "$CLI_PATH" stats
    [ "$status" -eq 0 ]
    [[ "$output" == *"Total Algorithms"* ]]
}

@test "algorithm-library health command" {
    run "$CLI_PATH" health
    [ "$status" -eq 0 ]
    [[ "$output" == *"healthy"* ]]
}