#!/usr/bin/env bats
# Tests for test.sh

# Source test setup infrastructure
source "${{BATS_TEST_FILENAME}%/*}/../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
}

teardown() {
    vrooli_cleanup_test
}

@test "test script sources var.sh correctly" {
    # Test that var.sh is sourced and variables are available
    source "$BATS_TEST_DIRNAME/test.sh"
    
    # Check that var_ variables are available
    [[ -n "$var_SCRIPTS_SCENARIOS_DIR" ]]
    [[ -n "$var_SCRIPTS_DIR" ]]
}

@test "test script resolves paths using var.sh variables" {
    # Source the script
    source "$BATS_TEST_DIRNAME/test.sh"
    
    # Check that FRAMEWORK_DIR uses var_ variable
    [[ "$FRAMEWORK_DIR" =~ $var_SCRIPTS_SCENARIOS_DIR ]]
}

@test "test script handles missing framework gracefully" {
    # Mock the scenario-test-runner.sh to avoid actual execution
    mkdir -p "$VROOLI_TEST_TMPDIR/framework"
    cat > "$VROOLI_TEST_TMPDIR/framework/scenario-test-runner.sh" <<EOF
#!/bin/bash
echo "Mock scenario test runner"
exit 0
EOF
    chmod +x "$VROOLI_TEST_TMPDIR/framework/scenario-test-runner.sh"
    
    # Temporarily override var_SCRIPTS_SCENARIOS_DIR
    export var_SCRIPTS_SCENARIOS_DIR="$VROOLI_TEST_TMPDIR"
    
    # Run the test script
    run bash "$BATS_TEST_DIRNAME/test.sh"
    
    # Should succeed with mock
    assert_success
    assert_output_contains "Resume Screening Assistant Business Scenario"
}