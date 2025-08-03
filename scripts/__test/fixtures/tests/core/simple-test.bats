#!/usr/bin/env bats
# Simple test to verify infrastructure

# Load test setup
source "${VROOLI_TEST_ROOT:-/home/matthalloran8/Vrooli/scripts/__test}/fixtures/setup.bash"

@test "simple test: setup loads successfully" {
    # Just verify we can run a test
    run echo "hello"
    [ "$status" -eq 0 ]
    [ "$output" = "hello" ]
}

@test "simple test: config is available" {
    # Check config function exists
    type vrooli_config_get >/dev/null 2>&1
}

@test "simple test: basic assertion works" {
    assert_equals "test" "test"
}