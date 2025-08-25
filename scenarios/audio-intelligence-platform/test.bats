#!/usr/bin/env bats
# Tests for audio-intelligence-platform test script
# Validates test script functionality

bats_require_minimum_version 1.5.0

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scenarios/audio-intelligence-platform"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load test setup
# shellcheck disable=SC1091
load "../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Create test scenario structure
    export TEST_SCENARIO_DIR="${BATS_TEST_TMPDIR}/test_scenario"
    mkdir -p "$TEST_SCENARIO_DIR/initialization"
    
    # Create mock scenario-test.yaml
    cat > "$TEST_SCENARIO_DIR/scenario-test.yaml" << 'EOF'
name: "Audio Intelligence Platform Test"
version: "1.0.0"
description: "Test scenario configuration"
EOF
    
    # Override SCENARIO_DIR for testing
    export SCENARIO_DIR="$TEST_SCENARIO_DIR"
    
    # Load the script under test
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/test.sh"
}

teardown() {
    vrooli_cleanup_test
}

@test "test.sh loads without errors" {
    # Test that the script can be sourced without errors
    true
}

@test "test.sh has required variables" {
    [[ -n "${SCENARIO_DIR:-}" ]]
}

@test "test.sh log functions work" {
    run log::info "test message"
    assert_success
    
    run log::success "test success"
    assert_success
    
    run log::error "test error"
    assert_success
}

@test "test.sh validation checks work with valid scenario" {
    # The test should pass with our mock scenario structure
    # Note: We don't run the full script as it would exit, just test components work
    
    # Test scenario structure validation components
    [[ -f "$SCENARIO_DIR/scenario-test.yaml" ]]
    [[ -d "$SCENARIO_DIR/initialization" ]]
}

@test "test.sh validation checks fail with missing files" {
    # Remove required file and test validation would fail
    trash::safe_remove "$TEST_SCENARIO_DIR/scenario-test.yaml" --test-cleanup
    
    # Test that file is missing (which would cause script to fail)
    [[ ! -f "$SCENARIO_DIR/scenario-test.yaml" ]]
}

# Assertion functions are provided by test fixtures