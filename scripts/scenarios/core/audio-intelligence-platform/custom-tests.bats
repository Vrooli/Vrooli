#!/usr/bin/env bats
# Tests for audio-intelligence-platform custom-tests script
# Validates custom test functionality

bats_require_minimum_version 1.5.0

# Load test setup
# shellcheck disable=SC1091
load "../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Load the script under test
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/custom-tests.sh"
}

teardown() {
    vrooli_cleanup_test
}

@test "custom-tests.sh loads without errors" {
    # Test that the script can be sourced without errors
    true
}

@test "custom-tests.sh custom test function is defined" {
    # Test that the main test function is defined
    declare -F custom_tests::test_audio_intelligence_platform_workflow >/dev/null
}

@test "custom-tests.sh custom test function executes successfully" {
    # Test that the custom test function can be called and returns success
    run custom_tests::test_audio_intelligence_platform_workflow
    assert_success
}

@test "custom-tests.sh produces expected output" {
    # Test that the custom test function produces expected log messages
    run custom_tests::test_audio_intelligence_platform_workflow
    assert_output --partial "Testing audio intelligence platform business workflow"
    assert_output --partial "Core services operational"
    assert_output --partial "Business workflow functional"
    assert_output --partial "Integration points validated"
}

@test "custom-tests.sh log functions work" {
    run log::info "test message"
    assert_success
    
    run log::success "test success"
    assert_success
}

# Assertion functions are provided by test fixtures