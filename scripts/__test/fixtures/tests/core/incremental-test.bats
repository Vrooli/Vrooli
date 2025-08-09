#!/usr/bin/env bats

# Load config only  
source "${BATS_TEST_DIRNAME}/../../../shared/config-simple.bash"

@test "incremental test: with config only" {
    run echo "hello"
    [ "$status" -eq 0 ]
}