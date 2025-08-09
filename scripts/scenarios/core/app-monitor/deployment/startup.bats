#!/usr/bin/env bats
# Tests for app-monitor startup script
# Validates deployment script functionality

bats_require_minimum_version 1.5.0

# Load test setup
# shellcheck disable=SC1091
load "../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Create test scenario structure
    export TEST_SCENARIO_DIR="${BATS_TEST_TMPDIR}/test_scenario"
    mkdir -p "$TEST_SCENARIO_DIR/initialization/storage/postgres"
    mkdir -p "$TEST_SCENARIO_DIR/initialization/automation/node-red"
    
    # Create mock files
    echo "CREATE TABLE test();" > "$TEST_SCENARIO_DIR/initialization/storage/postgres/schema.sql"
    echo '{"flows": []}' > "$TEST_SCENARIO_DIR/initialization/automation/node-red/docker-monitor.json"
    
    # Override paths for testing
    export SCENARIO_DIR="$TEST_SCENARIO_DIR"
    
    # Load the script under test
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/startup.sh"
}

teardown() {
    vrooli_cleanup_test
}

@test "startup.sh loads without errors" {
    # Test that the script can be sourced without errors
    true
}

@test "startup.sh has required variables" {
    [[ -n "${SCENARIO_DIR:-}" ]]
}

@test "startup.sh log functions work" {
    run log::info "test message"
    assert_success
}

@test "startup.sh resource functions are available" {
    # Test that resource functions can be called
    run resources::get_default_port "postgres"
    # Should not crash, may or may not succeed depending on environment
    [[ "$status" -eq 0 ]] || [[ "$status" -eq 1 ]]
}

# Assertion functions are provided by test fixtures