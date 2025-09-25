#!/usr/bin/env bats

@test "algorithm-library help command" {
    run algorithm-library --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Algorithm Library CLI"* ]]
}

@test "algorithm-library search command" {
    run algorithm-library search quicksort
    [ "$status" -eq 0 ]
    [[ "$output" == *"QuickSort"* ]]
}

@test "algorithm-library categories command" {
    run algorithm-library categories
    [ "$status" -eq 0 ]
    [[ "$output" == *"sorting"* ]]
}

@test "algorithm-library stats command" {
    run algorithm-library stats
    [ "$status" -eq 0 ]
    [[ "$output" == *"Total Algorithms"* ]]
}

@test "algorithm-library health command" {
    run algorithm-library health
    [ "$status" -eq 0 ]
    [[ "$output" == *"healthy"* ]]
}