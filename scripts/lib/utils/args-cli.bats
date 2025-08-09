#!/usr/bin/env bats
# Tests for args-cli.sh - Command line argument utilities

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-assert/load"

setup() {
    vrooli_setup_unit_test
    
    # Source the args-cli utilities
    source "${BATS_TEST_DIRNAME}/args-cli.sh"
}

teardown() {
    vrooli_cleanup_test
}

@test "args::reset function exists and can be called" {
    run args::reset
    assert_success
}

@test "args::register validates required name parameter" {
    run args::register --flag "t" --desc "Test argument"
    assert_failure
    assert_output --partial "Missing required parameter --name"
}

@test "args::register works without desc parameter" {
    run args::register --name "test" --flag "t"
    assert_success
}

@test "args::register accepts valid minimal arguments" {
    run args::register --name "test" --flag "t" --desc "Test argument"
    assert_success
}

@test "args::register accepts all optional parameters" {
    run args::register \
        --name "target" \
        --flag "t" \
        --desc "Target environment" \
        --type "value" \
        --default "native-linux" \
        --required "yes"
    
    assert_success
}

@test "args::register validates type parameter" {
    run args::register --name "test" --flag "t" --desc "Test" --type "invalid"
    assert_failure
    assert_output --partial "Type must be 'value' or 'array'"
}

@test "args::register handles unknown parameters" {
    run args::register --name "test" --flag "t" --desc "Test" --unknown "value"
    assert_failure
    assert_output --partial "Unknown parameter"
}