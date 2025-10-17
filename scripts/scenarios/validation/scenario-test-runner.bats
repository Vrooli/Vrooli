#!/usr/bin/env bats
# Tests for scenario-test-runner.sh

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Source test setup infrastructure
source "$(dirname "${BATS_TEST_FILENAME}")/../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Create test scenario structure
    export TEST_SCENARIO_DIR="$VROOLI_TEST_TMPDIR/test-scenario"
    mkdir -p "$TEST_SCENARIO_DIR"
    
    # Create minimal test configuration
    cat > "$TEST_SCENARIO_DIR/scenario-test.yaml" <<EOF
version: 1.0
scenario: test-scenario

structure:
  required_files:
    - README.md
    - test.sh

resources:
  required: []
  optional: []

tests:
  - name: "Basic Structure Check"
    type: structure
    validate:
      - README.md
      - test.sh

validation:
  success_rate: 100
EOF
    
    # Create required files for tests
    echo "# Test Scenario" > "$TEST_SCENARIO_DIR/README.md"
    echo "#!/bin/bash" > "$TEST_SCENARIO_DIR/test.sh"
    chmod +x "$TEST_SCENARIO_DIR/test.sh"
}

teardown() {
    vrooli_cleanup_test
}

@test "scenario-test-runner sources var.sh correctly" {
    # Test that var.sh is sourced and variables are available
    source "$BATS_TEST_DIRNAME/scenario-test-runner.sh" --help >/dev/null 2>&1 || true
    
    # Check that var_ variables are available in the script
    run grep -q "var_SCRIPTS_SCENARIOS_DIR" "$BATS_TEST_DIRNAME/scenario-test-runner.sh"
    assert_success
}

@test "scenario-test-runner shows help message" {
    run bash "$BATS_TEST_DIRNAME/scenario-test-runner.sh" --help
    assert_success
    assert_output_contains "Usage:"
    assert_output_contains "scenario-test-runner"
    assert_output_contains "--scenario"
}

@test "scenario-test-runner requires scenario directory" {
    run bash "$BATS_TEST_DIRNAME/scenario-test-runner.sh"
    assert_failure
    assert_output_contains "Scenario directory is required"
}

@test "scenario-test-runner validates scenario directory exists" {
    run bash "$BATS_TEST_DIRNAME/scenario-test-runner.sh" --scenario "/nonexistent"
    assert_failure
    assert_output_contains "Scenario directory does not exist"
}

@test "scenario-test-runner creates minimal config when missing" {
    # Remove the config file
    trash::safe_remove "$TEST_SCENARIO_DIR/scenario-test.yaml" --test-cleanup
    
    # Mock mktemp to avoid temp files in tests
    mktemp() { echo "$VROOLI_TEST_TMPDIR/test-config-$$"; }
    export -f mktemp
    
    run bash "$BATS_TEST_DIRNAME/scenario-test-runner.sh" --scenario "$TEST_SCENARIO_DIR" --dry-run
    assert_success
    assert_output_contains "Configuration file not found"
    assert_output_contains "Creating minimal test configuration"
}

@test "scenario-test-runner handles dry run mode" {
    run bash "$BATS_TEST_DIRNAME/scenario-test-runner.sh" --scenario "$TEST_SCENARIO_DIR" --dry-run
    assert_success
    assert_output_contains "DRY RUN MODE"
    assert_output_contains "Would execute"
}

@test "scenario-test-runner handles verbose mode" {
    run bash "$BATS_TEST_DIRNAME/scenario-test-runner.sh" --scenario "$TEST_SCENARIO_DIR" --verbose --dry-run
    assert_success
    assert_output_contains "Testing scenario:"
}

@test "scenario-test-runner loads configuration" {
    # Create custom config
    cat > "$TEST_SCENARIO_DIR/custom-test.yaml" <<EOF
version: 1.0
scenario: custom-test
tests: []
EOF
    
    run bash "$BATS_TEST_DIRNAME/scenario-test-runner.sh" --scenario "$TEST_SCENARIO_DIR" --config "custom-test.yaml" --dry-run
    assert_success
    assert_output_contains "Loading configuration from: custom-test.yaml"
}

@test "scenario-test-runner handles unknown arguments" {
    run bash "$BATS_TEST_DIRNAME/scenario-test-runner.sh" --unknown-option
    assert_failure
    assert_output_contains "Unknown option: --unknown-option"
}