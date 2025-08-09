#!/usr/bin/env bats
# Tests for permissions.sh - System permissions utilities

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
    
    # Source the permissions utilities
    source "${BATS_TEST_DIRNAME}/permissions.sh"
    
    # Reset mocks
    mock::system::reset
    
    # Mock environment variables
    export EUID=1000  # Non-root user
}

teardown() {
    vrooli_cleanup_test
    unset EUID
}

@test "permissions::check_sudo returns success for root user" {
    export EUID=0  # Root user
    
    run permissions::check_sudo
    assert_success
}

@test "permissions::check_sudo checks sudo availability for non-root" {
    export EUID=1000  # Non-root user
    mock::system::set_command_available "sudo" true
    mock::system::set_sudo_success true
    
    run permissions::check_sudo
    assert_success
}

@test "permissions::check_sudo fails when sudo not available" {
    export EUID=1000  # Non-root user
    mock::system::set_command_available "sudo" true
    mock::system::set_sudo_success false
    
    run permissions::check_sudo
    assert_failure
}