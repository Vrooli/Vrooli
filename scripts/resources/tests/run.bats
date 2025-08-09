#!/usr/bin/env bats
# Tests for Resource Integration Test Runner

# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091  
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Source the run script
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/run.sh"
    
    # Create test environment
    export TEST_TMPDIR="${BATS_TEST_TMPDIR}/run_test"
    mkdir -p "$TEST_TMPDIR"
    
    # Mock healthy resources list
    export HEALTHY_RESOURCES_STR="test_resource1 test_resource2"
}

teardown() {
    vrooli_cleanup_test
    rm -rf "$TEST_TMPDIR" 2>/dev/null || true
}

@test "run::parse_args: handles --verbose flag" {
    VERBOSE="false"
    
    run::parse_args --verbose
    
    [[ "$VERBOSE" == "true" ]]
}

@test "run::parse_args: handles --help flag" {
    run run::parse_args --help
    
    assert_success
}

@test "run::show_help: displays help information" {
    run run::show_help
    
    assert_success
    assert_output --partial "Resource Integration Test Runner"
    assert_output --partial "--verbose"
    assert_output --partial "--help"
}

@test "run::main: fails without HEALTHY_RESOURCES_STR" {
    unset HEALTHY_RESOURCES_STR
    
    run run::main
    
    assert_failure
    assert_output --partial "No resources provided"
}

@test "run::main: succeeds with empty resources list" {
    export HEALTHY_RESOURCES_STR=""
    
    run run::main
    
    assert_success
    assert_output --partial "No healthy resources to test"
}

@test "run::main: processes resources list" {
    export HEALTHY_RESOURCES_STR="resource1 resource2"
    
    # Mock the resource test function to avoid actual testing
    run::test_resource() {
        echo "Mocked test for $1"
        return 0
    }
    
    run run::main
    
    assert_success
    assert_output --partial "Testing 2 healthy resources"
    assert_output --partial "All resource tests passed"
}

@test "run::test_resource: handles missing integration test" {
    local test_resource="nonexistent_resource"
    
    # Mock resources::get_category to return a test category
    resources::get_category() {
        echo "test_category"
    }
    
    run run::test_resource "$test_resource"
    
    # Should return 2 for skipped
    [[ $status -eq 2 ]]
    assert_output --partial "No integration tests found"
}

@test "run::test_resource: fails with unknown category" {
    local test_resource="unknown_resource"
    
    # Mock resources::get_category to fail
    resources::get_category() {
        return 1
    }
    
    run run::test_resource "$test_resource"
    
    assert_failure
    assert_output --partial "Unable to determine resource category"
}