#!/usr/bin/env bats
# Tests for app-monitor validate script
# Validates validation script functionality

bats_require_minimum_version 1.5.0

# Load test setup
# shellcheck disable=SC1091
load "../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Load the script under test
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/validate.sh"
}

teardown() {
    vrooli_cleanup_test
}

@test "validate.sh loads without errors" {
    # Test that the script can be sourced without errors
    true
}

@test "validate.sh log functions work" {
    run log::info "test message"
    assert_success
    
    run log::success "test success"
    assert_success
    
    run log::error "test error"
    assert_success
}

@test "validate.sh resource functions are available" {
    # Test that resource functions can be called
    run resources::get_default_port "postgres"
    # Should not crash, may or may not succeed depending on environment
    [[ "$status" -eq 0 ]] || [[ "$status" -eq 1 ]]
    
    run resources::get_default_port "redis"
    [[ "$status" -eq 0 ]] || [[ "$status" -eq 1 ]]
    
    run resources::get_default_port "ui"
    [[ "$status" -eq 0 ]] || [[ "$status" -eq 1 ]]
    
    run resources::get_default_port "api"
    [[ "$status" -eq 0 ]] || [[ "$status" -eq 1 ]]
}

# Assertion functions are provided by test fixtures