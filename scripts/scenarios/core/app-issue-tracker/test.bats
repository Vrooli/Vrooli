#!/usr/bin/env bats
# Tests for app-issue-tracker integration test script
# Validates the test script functionality and helper functions

bats_require_minimum_version 1.5.0

# Load test setup
# shellcheck disable=SC1091
load "../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Create temporary scenario directory structure for testing
    export TEST_SCENARIO_DIR="${BATS_TEST_TMPDIR}/test_scenario"
    mkdir -p "$TEST_SCENARIO_DIR/cli"
    mkdir -p "$TEST_SCENARIO_DIR/deployment"
    
    # Create a mock CLI script for testing
    cat > "$TEST_SCENARIO_DIR/cli/vrooli-tracker.sh" << 'EOF'
#!/usr/bin/env bash
case "$1" in
    --help|help)
        echo "Vrooli Issue Tracker CLI"
        echo "Usage: ..."
        ;;
    *)
        echo "Mock CLI response"
        ;;
esac
EOF
    chmod +x "$TEST_SCENARIO_DIR/cli/vrooli-tracker.sh"
    
    # Override SCENARIO_DIR for testing
    export SCENARIO_DIR="$TEST_SCENARIO_DIR"
    
    # Load the script under test
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/test.sh"
}

teardown() {
    vrooli_cleanup_test
}

# Test that the script loads without errors
@test "test.sh loads without errors" {
    # Test that the test helper function is defined
    declare -F test::run_test >/dev/null
}

# Test the test helper function
@test "test::run_test reports success for passing commands" {
    run test::run_test "Always pass test" "true"
    assert_success
    assert_output --partial "✓ Always pass test"
}

@test "test::run_test reports failure for failing commands" {
    run test::run_test "Always fail test" "false"
    assert_failure
    assert_output --partial "✗ Always fail test"
}

@test "test::run_test updates test counters" {
    # Reset counters
    TESTS_PASSED=0
    TESTS_FAILED=0
    
    # Run a passing test
    test::run_test "Pass test" "true"
    [[ "$TESTS_PASSED" -eq 1 ]]
    [[ "$TESTS_FAILED" -eq 0 ]]
    
    # Run a failing test
    test::run_test "Fail test" "false"
    [[ "$TESTS_PASSED" -eq 1 ]]
    [[ "$TESTS_FAILED" -eq 1 ]]
}

# Test CLI validation works
@test "CLI script validation works" {
    # This should pass since we created a mock CLI script in setup
    run test::run_test "CLI script exists" "[ -f $TEST_SCENARIO_DIR/cli/vrooli-tracker.sh ]"
    assert_success
    
    run test::run_test "CLI is executable" "[ -x $TEST_SCENARIO_DIR/cli/vrooli-tracker.sh ]"
    assert_success
    
    run test::run_test "CLI help command" "$TEST_SCENARIO_DIR/cli/vrooli-tracker.sh --help | grep -q 'Vrooli Issue Tracker CLI'"
    assert_success
}

# Test basic command validation
@test "test helper validates simple commands correctly" {
    run test::run_test "Echo test" "echo 'hello world' | grep -q 'hello'"
    assert_success
    
    run test::run_test "Grep test" "echo 'test string' | grep -q 'nonexistent'"
    assert_failure
}

# Test resource port function availability (mocked)
@test "resource port functions are available if mocked" {
    if command -v resources::get_default_port &>/dev/null; then
        # If mocked, test should work
        run resources::get_default_port "postgres"
        assert_success
    else
        # If not mocked, skip this test
        skip "Resource port functions not mocked"
    fi
}

# Assertion functions are provided by test fixtures