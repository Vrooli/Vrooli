#!/usr/bin/env bats
# Argument Parser Module Tests
# Tests for command-line argument parsing and validation

bats_require_minimum_version 1.5.0

# Load test infrastructure (single entry point)
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test  # Basic mocks and environment
    
    # Source the script under test
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/parser.sh"
}

teardown() {
    vrooli_cleanup_test     # Clean up resources
}

@test "parser::parse_lifecycle_args - parses basic arguments" {
    parser::parse_lifecycle_args --target docker --phase setup
    
    # Check that variables are set
    [[ "$TARGET" == "docker" ]]
    [[ "$LIFECYCLE_PHASE" == "setup" ]]
}

@test "parser::parse_lifecycle_args - handles dry run flag" {
    parser::parse_lifecycle_args --dry-run --phase setup
    
    [[ "$DRY_RUN" == "true" ]]
}

@test "parser::parse_lifecycle_args - handles verbose flag" {
    parser::parse_lifecycle_args --verbose --phase setup
    
    [[ "$VERBOSE" == "true" ]]
}

@test "parser::parse_lifecycle_args - handles environment variables" {
    parser::parse_lifecycle_args --env "TEST_VAR=test_value" --phase setup
    
    # Check that environment variable was exported
    [[ "$TEST_VAR" == "test_value" ]]
}

@test "parser::parse_lifecycle_args - handles invalid environment variable format" {
    run parser::parse_lifecycle_args --env "INVALID_FORMAT" --phase setup
    assert_failure
}

@test "parser::parse_lifecycle_args - handles skip option" {
    parser::parse_lifecycle_args --skip "step1" --phase setup
    
    # Check that skip variable was set
    [[ "$SKIP_STEP1" == "true" ]]
}

@test "parser::parse_lifecycle_args - handles only option" {
    parser::parse_lifecycle_args --only "target_step" --phase setup
    
    [[ "$ONLY_STEP" == "target_step" ]]
}

@test "parser::parse_lifecycle_args - rejects multiple only options" {
    run parser::parse_lifecycle_args --only "step1" --only "step2" --phase setup
    assert_failure
}

@test "parser::parse_lifecycle_args - handles timeout option" {
    parser::parse_lifecycle_args --timeout "600" --phase setup
    
    [[ "$LIFECYCLE_TIMEOUT" == "600" ]]
}

@test "parser::parse_lifecycle_args - handles service.json path" {
    local test_file="${BATS_TEST_TMPDIR}/test-service.json"
    touch "$test_file"
    
    parser::parse_lifecycle_args --service-json "$test_file" --phase setup
    
    [[ "$SERVICE_JSON_PATH" == "$test_file" ]]
}

@test "parser::parse_lifecycle_args - requires phase argument" {
    run parser::parse_lifecycle_args --target docker
    assert_failure
}

@test "parser::parse_lifecycle_args - handles help flag" {
    run parser::parse_lifecycle_args --help
    assert_success
    assert_output_contains "Usage:"
}

@test "parser::parse_lifecycle_args - rejects unknown arguments" {
    run parser::parse_lifecycle_args --unknown-flag --phase setup
    assert_failure
}

@test "parser::validate_args - validates required arguments" {
    LIFECYCLE_PHASE="setup"
    
    run parser::validate_args
    assert_success
}

@test "parser::validate_args - fails without phase" {
    LIFECYCLE_PHASE=""
    
    run parser::validate_args
    assert_failure
}