#!/usr/bin/env bats
# Validation Test Suite for New Testing Infrastructure
# Migrated from old validation/test-infrastructure.bats

bats_require_minimum_version 1.5.0

# Load new unified test infrastructure
load "${BATS_TEST_DIRNAME}/../../setup"

setup() {
    # Use new setup function
    vrooli_setup_unit_test
}

teardown() {
    # Use new cleanup function
    vrooli_cleanup_test
}

@test "infrastructure loads without errors" {
    # Verify setup completed successfully
    assert_env_set "VROOLI_TEST_SETUP_LOADED"
    assert_equals "$VROOLI_TEST_SETUP_LOADED" "true"
}

@test "configuration system is available" {
    # Test new config system
    run vrooli_config_get "environment_namespace_prefix"
    assert_success
    assert_output_contains "vrooli_test"
}

@test "assertion library is available" {
    # Test assertions are loaded
    run bash -c "declare -f assert_output_contains >/dev/null && echo 'available'"
    assert_success
    assert_output_contains "available"
}

@test "mock system works" {
    # Test that mocks are loaded
    run docker --version
    assert_success
    assert_output_contains "Docker version"
}

@test "test environment is properly configured" {
    # Verify environment setup
    assert_env_set "VROOLI_TEST_TMPDIR"
    assert_env_set "TEST_NAMESPACE"
    
    # Verify temp directory exists
    assert_dir_exists "$VROOLI_TEST_TMPDIR"
}

@test "cleanup functions are available" {
    # Test cleanup system
    run bash -c "declare -f vrooli_cleanup_test >/dev/null && echo 'available'"
    assert_success
    assert_output_contains "available"
}

@test "service setup functions work" {
    # Test service-specific setup
    run bash -c "cd \"${BATS_TEST_DIRNAME}/../../../..\" && source fixtures/setup.bash && vrooli_setup_service_test 'ollama' && echo 'success'"
    assert_success
    assert_output_contains "success"
}

# Migrated: 65 lines → 65 lines (validation test)