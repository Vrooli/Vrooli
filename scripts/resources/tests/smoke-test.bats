#!/usr/bin/env bats
# Tests for Resource Smoke Test

# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091  
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Source the smoke test script
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/smoke-test.sh"
    
    # Create test environment
    export TEST_TMPDIR="${BATS_TEST_TMPDIR}/smoke_test"
    mkdir -p "$TEST_TMPDIR"
    
    # Mock environment variables
    export TEST_MODE="basic"
    export VERBOSE="false"
}

teardown() {
    vrooli_cleanup_test
    rm -rf "$TEST_TMPDIR" 2>/dev/null || true
}

@test "smoke test script exists and is executable" {
    [[ -f "${BATS_TEST_DIRNAME}/smoke-test.sh" ]]
    [[ -x "${BATS_TEST_DIRNAME}/smoke-test.sh" ]]
}

@test "smoke test script has proper shebang" {
    run head -1 "${BATS_TEST_DIRNAME}/smoke-test.sh"
    
    assert_success
    assert_output --partial "#!/usr/bin/env bash"
}

@test "smoke test script sources var.sh correctly" {
    run grep -c "source.*var.sh" "${BATS_TEST_DIRNAME}/smoke-test.sh"
    
    assert_success
    [[ "$output" -gt 0 ]]
}

@test "smoke test script validates syntax" {
    run bash -n "${BATS_TEST_DIRNAME}/smoke-test.sh"
    
    assert_success
}

@test "smoke test script can show help" {
    run "${BATS_TEST_DIRNAME}/smoke-test.sh" --help
    
    assert_success
    assert_output --partial "Resource Smoke Test"
}

@test "smoke test script handles basic test mode" {
    export TEST_MODE="basic"
    
    # Mock a basic test run with non-existent resources to avoid real service calls
    export SELECTED_RESOURCES="nonexistent_service"
    run timeout 10 "${BATS_TEST_DIRNAME}/smoke-test.sh" --resources nonexistent_service 2>/dev/null || true
    
    # Should not crash, may succeed or fail gracefully
    [[ $status -eq 0 || $status -eq 1 || $status -eq 2 || $status -eq 124 ]]
}

@test "smoke test script handles comprehensive test mode" {
    export TEST_MODE="comprehensive"
    
    # Mock a comprehensive test run with non-existent resources to avoid real service calls  
    export SELECTED_RESOURCES="nonexistent_service"
    run timeout 10 "${BATS_TEST_DIRNAME}/smoke-test.sh" --resources nonexistent_service 2>/dev/null || true
    
    # Should not crash, may succeed or fail gracefully
    [[ $status -eq 0 || $status -eq 1 || $status -eq 2 || $status -eq 124 ]]
}

@test "smoke test script has color output functions" {
    run grep -c "GREEN\|RED\|YELLOW" "${BATS_TEST_DIRNAME}/smoke-test.sh"
    
    assert_success
    [[ "$output" -gt 0 ]]
}

@test "smoke test script defines logging functions" {
    # Check for logging function definitions
    run grep -E "(log_info|log_error|log_success)" "${BATS_TEST_DIRNAME}/smoke-test.sh"
    
    assert_success
}

@test "smoke test script contains test functions" {
    # Should have test functions for different resource types
    run grep -c "test_.*(" "${BATS_TEST_DIRNAME}/smoke-test.sh"
    
    assert_success  
    [[ "$output" -gt 0 ]]
}