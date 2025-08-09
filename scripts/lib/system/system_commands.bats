#!/usr/bin/env bats
# Tests for system_commands.sh - System command utilities

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-assert/load"

# Load mocks
load "${BATS_TEST_DIRNAME}/../../__test/fixtures/mocks/system"

setup() {
    vrooli_setup_unit_test
    
    # Source the system command utilities
    source "${BATS_TEST_DIRNAME}/system_commands.sh"
    
    # Reset mocks
    mock::system::reset
}

teardown() {
    vrooli_cleanup_test
}

@test "system::is_command detects available commands" {
    mock::system::set_command_available "git" true
    
    run system::is_command "git"
    assert_success
}

@test "system::is_command detects unavailable commands" {
    mock::system::set_command_available "nonexistent" false
    
    run system::is_command "nonexistent"
    assert_failure
}

@test "system::assert_command succeeds for available commands" {
    mock::system::set_command_available "git" true
    
    run system::assert_command "git"
    assert_success
}

@test "system::assert_command fails for unavailable commands" {
    mock::system::set_command_available "nonexistent" false
    
    run system::assert_command "nonexistent"
    assert_failure
    assert_output --partial "Command nonexistent not found"
}

@test "system::assert_command uses custom error message" {
    mock::system::set_command_available "nonexistent" false
    
    run system::assert_command "nonexistent" "Custom error message"
    assert_failure
    assert_output --partial "Custom error message"
}