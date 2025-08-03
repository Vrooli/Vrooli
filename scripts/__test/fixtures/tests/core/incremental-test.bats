#!/usr/bin/env bats

# Set VROOLI_TEST_ROOT
export VROOLI_TEST_ROOT="${VROOLI_TEST_ROOT:-/home/matthalloran8/Vrooli/scripts/__test}"

# Try sourcing just the config
source "$VROOLI_TEST_ROOT/shared/config-simple.bash"

@test "incremental test: with config only" {
    run echo "hello"
    [ "$status" -eq 0 ]
}