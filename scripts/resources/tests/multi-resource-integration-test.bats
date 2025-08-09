#!/usr/bin/env bats
# Tests for Multi-Resource Integration Test

# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091  
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Create test environment
    export TEST_TMPDIR="${BATS_TEST_TMPDIR}/multi_resource_test"
    mkdir -p "$TEST_TMPDIR"
    
    # Mock environment variables
    export VERBOSE="false"
    export TEST_TIMEOUT="30"
}

teardown() {
    vrooli_cleanup_test
    rm -rf "$TEST_TMPDIR" 2>/dev/null || true
}

@test "multi-resource integration test script exists and is executable" {
    [[ -f "${BATS_TEST_DIRNAME}/multi-resource-integration-test.sh" ]]
    [[ -x "${BATS_TEST_DIRNAME}/multi-resource-integration-test.sh" ]]
}

@test "multi-resource integration test script has proper shebang" {
    run head -1 "${BATS_TEST_DIRNAME}/multi-resource-integration-test.sh"
    
    assert_success
    assert_output --partial "#!/usr/bin/env bash"
}

@test "multi-resource integration test script sources required libraries" {
    run grep -c "source.*integration-test-lib.sh" "${BATS_TEST_DIRNAME}/multi-resource-integration-test.sh"
    
    assert_success
    [[ "$output" -gt 0 ]]
}

@test "multi-resource integration test script has configuration section" {
    run grep -c "CONFIGURATION" "${BATS_TEST_DIRNAME}/multi-resource-integration-test.sh"
    
    assert_success
    [[ "$output" -gt 0 ]]
}

@test "multi-resource integration test script validates syntax" {
    run bash -n "${BATS_TEST_DIRNAME}/multi-resource-integration-test.sh"
    
    assert_success
}

@test "multi-resource integration test script can show help" {
    run "${BATS_TEST_DIRNAME}/multi-resource-integration-test.sh" --help
    
    assert_success
    assert_output --partial "Multi-Resource Integration Test"
}

@test "multi-resource integration test fails gracefully without resources" {
    # Run with no healthy resources
    run timeout 10 "${BATS_TEST_DIRNAME}/multi-resource-integration-test.sh" --dry-run
    
    # Should not crash, may succeed or fail gracefully
    [[ $status -eq 0 || $status -eq 1 || $status -eq 2 ]]
}

@test "multi-resource integration test handles verbose flag" {
    run "${BATS_TEST_DIRNAME}/multi-resource-integration-test.sh" --verbose --help
    
    assert_success
    assert_output --partial "Multi-Resource Integration Test"
}