#!/usr/bin/env bats
# Tests for app-issue-tracker startup script
# Validates deployment script functionality

bats_require_minimum_version 1.5.0

# Load test setup
# shellcheck disable=SC1091
load "../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Create temporary scenario directory structure for testing
    export TEST_SCENARIO_DIR="${BATS_TEST_TMPDIR}/test_scenario"
    mkdir -p "$TEST_SCENARIO_DIR/initialization/storage/postgres"
    mkdir -p "$TEST_SCENARIO_DIR/initialization/automation/n8n"
    mkdir -p "$TEST_SCENARIO_DIR/initialization/automation/windmill/apps"
    
    # Create mock schema and seed files
    echo "-- Mock schema" > "$TEST_SCENARIO_DIR/initialization/storage/postgres/schema.sql"
    echo "-- Mock seed data" > "$TEST_SCENARIO_DIR/initialization/storage/postgres/seed.sql"
    
    # Create mock workflow files
    echo '{"name": "test-workflow"}' > "$TEST_SCENARIO_DIR/initialization/automation/n8n/test-workflow.json"
    echo '{"name": "test-app"}' > "$TEST_SCENARIO_DIR/initialization/automation/windmill/apps/test-app.json"
    
    # Override SCENARIO_DIR for testing
    export SCENARIO_DIR="$TEST_SCENARIO_DIR"
    
    # Mock environment variables
    export POSTGRES_PASSWORD="postgres"
    export POSTGRES_USER="postgres"
    export POSTGRES_DB="postgres"
    
    # Load the script under test
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/startup.sh"
}

teardown() {
    vrooli_cleanup_test
}

# Test that the script loads without errors
@test "startup.sh loads without errors" {
    # Test that functions are defined
    declare -F startup::check_service >/dev/null
}

# Test service check function
@test "startup::check_service function is defined" {
    declare -F startup::check_service >/dev/null
}

# Test service check with mock (if available)
@test "startup::check_service works with mocked services" {
    if command -v nc &>/dev/null || command -v mock::system::set_port_response &>/dev/null; then
        # If nc is available or mocks are set up, test should work
        if command -v mock::system::set_port_response &>/dev/null; then
            # Set up mock for successful connection
            mock::system::set_port_response "localhost" "5432" "success"
        fi
        
        # This may succeed or fail depending on mock setup, but should not crash
        run startup::check_service "PostgreSQL" "5432"
        # Exit code should be 0 or 1, not something like 127 (command not found)
        [[ "$status" -eq 0 ]] || [[ "$status" -eq 1 ]]
    else
        skip "No network tools or mocks available for testing"
    fi
}

# Test help functionality
@test "startup script shows help with help command" {
    run "$BATS_TEST_DIRNAME/startup.sh" help
    assert_success
    assert_output --partial "Usage:"
    assert_output --partial "Commands:"
    assert_output --partial "deploy"
}

# Test unknown command handling
@test "startup script handles unknown commands" {
    run "$BATS_TEST_DIRNAME/startup.sh" unknown_command
    assert_failure
    assert_output --partial "Unknown command: unknown_command"
}

# Test validation command
@test "startup script validate command exists" {
    run "$BATS_TEST_DIRNAME/startup.sh" validate
    # May succeed or fail depending on available resources, but should not crash
    [[ "$status" -eq 0 ]] || [[ "$status" -eq 1 ]]
    # Should not exit with command not found or similar errors
    [[ "$status" -ne 127 ]]
}

# Test logs command handling
@test "startup script logs command handles missing log file" {
    # Remove any existing log file
    export LOG_FILE="${BATS_TEST_TMPDIR}/nonexistent.log"
    run "$BATS_TEST_DIRNAME/startup.sh" logs
    assert_failure
    assert_output --partial "No log file found"
}

# Test resource port function requirements
@test "startup script requires resource port functions" {
    if ! command -v resources::get_default_port &>/dev/null; then
        skip "Resource port functions not available - this is expected in unit tests"
    else
        # If available, test basic functionality
        run resources::get_default_port "postgres"
        assert_success
    fi
}

# Test database creation logic (with mocks if available)
@test "database creation uses correct PostgreSQL syntax" {
    if command -v mock::postgres::set_database_list &>/dev/null; then
        # Set up mock to simulate database doesn't exist
        mock::postgres::set_database_list "postgres template0 template1"
        
        # The fixed code should not use "CREATE DATABASE IF NOT EXISTS"
        # Instead it should check if database exists first
        run grep -n "CREATE DATABASE IF NOT EXISTS" "$BATS_TEST_DIRNAME/startup.sh" 
        assert_failure  # Should not find this invalid syntax
        
        # Should find the correct pattern
        run grep -n "createdb" "$BATS_TEST_DIRNAME/startup.sh"
        assert_success
    else
        skip "PostgreSQL mocking not available"
    fi
}

# Assertion functions are provided by test fixtures