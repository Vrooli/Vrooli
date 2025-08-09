#!/usr/bin/env bats
# Tests for custom-tests.sh

# Source test setup infrastructure
source "$(dirname "${BATS_TEST_FILENAME}")/../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
}

teardown() {
    vrooli_cleanup_test
}

@test "custom-tests::test_research_assistant_workflow executes successfully" {
    # Source the script to test
    source "$BATS_TEST_DIRNAME/custom-tests.sh"
    
    # Mock print functions since framework may not be available
    print_custom_info() { echo "INFO: $*"; }
    print_custom_success() { echo "SUCCESS: $*"; }
    export -f print_custom_info print_custom_success
    
    # Run the function
    run custom-tests::test_research_assistant_workflow
    
    # Verify execution
    assert_success
    assert_output_contains "Testing research assistant business workflow"
    assert_output_contains "Core services operational"
}

@test "custom-tests::run_custom_tests executes successfully" {
    # Source the script to test
    source "$BATS_TEST_DIRNAME/custom-tests.sh"
    
    # Mock print functions since framework may not be available
    print_custom_info() { echo "INFO: $*"; }
    print_custom_success() { echo "SUCCESS: $*"; }
    export -f print_custom_info print_custom_success
    
    # Run the function
    run custom-tests::run_custom_tests
    
    # Verify execution
    assert_success
    assert_output_contains "Running custom business logic tests"
    assert_output_contains "Testing research assistant business workflow"
}

@test "script sources var.sh correctly" {
    # Test that var.sh is sourced and variables are available
    source "$BATS_TEST_DIRNAME/custom-tests.sh"
    
    # Check that var_ variables are available (these come from var.sh)
    [[ -n "$var_SCRIPTS_SCENARIOS_DIR" ]]
    [[ -n "$var_SCRIPTS_DIR" ]]
}

@test "script handles missing framework gracefully" {
    # Source the script to test - it should not fail if framework is missing
    run bash -c "source '$BATS_TEST_DIRNAME/custom-tests.sh'"
    
    # Should not fail even if framework files don't exist
    assert_success
}