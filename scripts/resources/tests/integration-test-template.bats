#!/usr/bin/env bats
# Tests for Integration Test Template

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091  
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Create test environment
    export TEST_TMPDIR="${BATS_TEST_TMPDIR}/template_test"
    mkdir -p "$TEST_TMPDIR"
}

teardown() {
    vrooli_cleanup_test
    trash::safe_remove "$TEST_TMPDIR" --test-cleanup
}

@test "integration test template exists and is executable" {
    [[ -f "${BATS_TEST_DIRNAME}/integration-test-template.sh" ]]
    [[ -x "${BATS_TEST_DIRNAME}/integration-test-template.sh" ]]
}

@test "integration test template has proper shebang" {
    run head -1 "${BATS_TEST_DIRNAME}/integration-test-template.sh"
    
    assert_success
    assert_output --partial "#!/usr/bin/env bash"
}

@test "integration test template contains RESOURCE_NAME placeholders" {
    run grep -c "RESOURCE_NAME" "${BATS_TEST_DIRNAME}/integration-test-template.sh"
    
    assert_success
    [[ "$output" -gt 0 ]]
}

@test "integration test template sources var.sh correctly" {
    run grep -c "source.*var.sh" "${BATS_TEST_DIRNAME}/integration-test-template.sh"
    
    assert_success
    [[ "$output" -gt 0 ]]
}

@test "integration test template sources integration-test-lib.sh" {
    run grep -c "integration-test-lib.sh" "${BATS_TEST_DIRNAME}/integration-test-template.sh"
    
    assert_success
    [[ "$output" -gt 0 ]]
}

@test "integration test template has service-specific configuration section" {
    run grep -c "SERVICE-SPECIFIC CONFIGURATION" "${BATS_TEST_DIRNAME}/integration-test-template.sh"
    
    assert_success
    [[ "$output" -gt 0 ]]
}

@test "integration test template contains template instructions" {
    run grep -c "This template should be customized" "${BATS_TEST_DIRNAME}/integration-test-template.sh"
    
    assert_success
    [[ "$output" -gt 0 ]]
}

@test "integration test template validates syntax" {
    run bash -n "${BATS_TEST_DIRNAME}/integration-test-template.sh"
    
    assert_success
}

@test "integration test template mentions register_tests function" {
    run grep -c "register_tests" "${BATS_TEST_DIRNAME}/integration-test-template.sh"
    
    assert_success
    [[ "$output" -gt 0 ]]
}

@test "integration test template mentions integration_test_main" {
    run grep -c "integration_test_main" "${BATS_TEST_DIRNAME}/integration-test-template.sh"
    
    assert_success
    [[ "$output" -gt 0 ]]
}