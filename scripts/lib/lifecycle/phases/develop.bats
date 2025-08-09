#!/usr/bin/env bats
################################################################################
# Tests for develop.sh - Universal develop phase handler
################################################################################

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../../../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../../__test/helpers/bats-assert/load"

# Load required mocks
load "${BATS_TEST_DIRNAME}/../../../__test/fixtures/mocks/commands.bash"
load "${BATS_TEST_DIRNAME}/../../../__test/fixtures/mocks/system.sh"
load "${BATS_TEST_DIRNAME}/../../../__test/fixtures/mocks/filesystem.sh"

setup() {
    vrooli_setup_unit_test
    
    # Reset all mocks
    mock::commands::reset
    mock::system::reset
    mock::filesystem::reset
    
    # Mock phase utilities and dependencies
    mock::commands::setup_function "phase::init" 0
    mock::commands::setup_function "phase::complete" 0
    mock::commands::setup_function "phase::run_hook" 0
    mock::commands::setup_function "log::info" 0
    mock::commands::setup_function "log::debug" 0
    mock::commands::setup_function "log::header" 0
    mock::commands::setup_function "log::success" 0
    mock::commands::setup_function "log::warning" 0
    mock::commands::setup_function "log::error" 0
    mock::commands::setup_function "flow::is_yes" 1  # Default to "no"
    mock::commands::setup_function "target_matcher::match_target" 0 "native-linux"
    mock::commands::setup_function "ports::is_port_in_use" 1  # Default to not in use
    
    # Set up test environment
    export TARGET="native-linux"
    export DETACHED="no"
    export ENVIRONMENT="development"
    export SKIP_INSTANCE_CHECK="no"
    export CLEAN_INSTANCES="no"
    export var_ROOT_DIR="${MOCK_ROOT}"
    export var_LIB_SERVICE_DIR="${MOCK_ROOT}/lib/service"
    export var_ENV_DEV_FILE="${MOCK_ROOT}/.env-dev"
    export var_SCRIPTS_SCENARIOS_DIR="${MOCK_ROOT}/scripts/scenarios"
    
    # Source develop functions for testing
    source "${BATS_TEST_DIRNAME}/develop.sh"
}

teardown() {
    vrooli_cleanup_test
}

################################################################################
# Port Management Tests
################################################################################

@test "develop::check_port returns success for free port" {
    mock::commands::setup_function "ports::is_port_in_use" 1  # Port not in use
    
    run develop::check_port "TestService" "8080"
    
    assert_success
    mock::commands::assert::called "ports::is_port_in_use" "8080"
}

@test "develop::check_port returns failure for occupied port" {
    mock::commands::setup_function "ports::is_port_in_use" 0  # Port in use
    
    run develop::check_port "TestService" "8080"
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::warning" "Port 8080 for TestService is already in use"
}

@test "develop::resolve_port_conflicts checks default ports" {
    export PORT_UI="3000"
    export PORT_SERVER="5329"
    export PORT_DB="5432"
    export PORT_REDIS="6379"
    export PORT_JOBS="4001"
    
    # All ports free
    mock::commands::setup_function "ports::is_port_in_use" 1
    
    run develop::resolve_port_conflicts
    
    assert_success
    mock::commands::assert::called "ports::is_port_in_use" "3000"
    mock::commands::assert::called "ports::is_port_in_use" "5329"
    mock::commands::assert::called "ports::is_port_in_use" "5432"
    mock::commands::assert::called "ports::is_port_in_use" "6379"
    mock::commands::assert::called "ports::is_port_in_use" "4001"
    mock::commands::assert::called "log::success" "✅ All ports are available"
}

@test "develop::resolve_port_conflicts handles conflicts" {
    # Port 3000 is in use
    ports::is_port_in_use() {
        if [[ "$1" == "3000" ]]; then
            return 0  # In use
        else
            return 1  # Free
        fi
    }
    export -f ports::is_port_in_use
    
    run develop::resolve_port_conflicts
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::warning" "Port conflicts detected. Services may fail to start."
}

@test "develop::resolve_port_conflicts can skip port check" {
    export SKIP_PORT_CHECK="yes"
    # Port 3000 is in use but we skip the check
    ports::is_port_in_use() {
        if [[ "$1" == "3000" ]]; then
            return 0  # In use
        else
            return 1  # Free
        fi
    }
    export -f ports::is_port_in_use
    
    run develop::resolve_port_conflicts
    
    assert_success  # Should succeed despite conflicts
}

################################################################################
# Instance Management Tests
################################################################################

@test "develop::check_instances loads instance manager when available" {
    local instance_script="${var_LIB_SERVICE_DIR}/instance_manager.sh"
    mock::filesystem::create_file "$instance_script" '#!/bin/bash\ninstance::detect_all() { export INSTANCE_STATE="none"; }'
    
    function_exists() { [[ "$1" == "instance::detect_all" ]]; }
    export -f function_exists
    
    run develop::check_instances
    
    assert_success
}

@test "develop::check_instances detects conflicts" {
    local instance_script="${var_LIB_SERVICE_DIR}/instance_manager.sh"
    mock::filesystem::create_file "$instance_script" '#!/bin/bash\ninstance::detect_all() { export INSTANCE_STATE="running"; }'
    
    function_exists() { [[ "$1" == "instance::detect_all" ]]; }
    export -f function_exists
    export INSTANCE_STATE="running"
    
    run develop::check_instances
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::warning" "Detected running instances"
}

################################################################################
# Scenario Auto-Conversion Tests
################################################################################

@test "develop::auto_convert_scenarios skips when no scenarios directory" {
    # No scenarios directory
    
    run develop::auto_convert_scenarios
    
    assert_success
    mock::commands::assert::called "log::debug" "No scenarios directory found, skipping auto-conversion"
}

@test "develop::auto_convert_scenarios skips when no auto-converter" {
    mock::filesystem::create_directory "${var_SCRIPTS_SCENARIOS_DIR}"
    # No auto-converter.sh
    
    run develop::auto_convert_scenarios
    
    assert_success
    mock::commands::assert::called "log::debug"
}

@test "develop::auto_convert_scenarios runs converter successfully" {
    local converter="${var_SCRIPTS_SCENARIOS_DIR}/auto-converter.sh"
    mock::filesystem::create_file "$converter" '#!/bin/bash\nexit 0'
    chmod +x "$converter"
    mock::commands::setup_command "$converter" "" ""
    
    run develop::auto_convert_scenarios
    
    assert_success
    mock::commands::assert::called "log::success" "✅ Scenario auto-conversion completed"
}

@test "develop::auto_convert_scenarios handles converter failure gracefully" {
    local converter="${var_SCRIPTS_SCENARIOS_DIR}/auto-converter.sh"
    mock::filesystem::create_file "$converter" '#!/bin/bash\nexit 1'
    chmod +x "$converter"
    mock::commands::setup_command "$converter" "" "" "1"
    
    run develop::auto_convert_scenarios
    
    assert_success  # Should not fail develop phase
    mock::commands::assert::called "log::warning" "⚠️  Some scenarios failed to convert (check logs above)"
}

################################################################################
# Main Develop Function Tests
################################################################################

@test "develop::universal::main initializes phase correctly" {
    export RESOURCES="none"  # Skip resource installation
    mock::commands::setup_function "flow::is_yes" 1  # Default to no for all checks
    
    run develop::universal::main
    
    assert_success
    mock::commands::assert::called "phase::init" "Develop"
    mock::commands::assert::called "phase::complete"
}

@test "develop::universal::main validates target parameter" {
    export TARGET="invalid-target"
    mock::commands::setup_function "target_matcher::match_target" 1  # Return error
    
    run develop::universal::main
    
    assert_failure
    mock::commands::assert::called "log::error" "Invalid target: invalid-target"
}

@test "develop::universal::main handles clean instances request" {
    export CLEAN_INSTANCES="yes"
    mock::commands::setup_function "flow::is_yes" 0  # Return true for clean check
    local instance_script="${var_LIB_SERVICE_DIR}/instance_manager.sh"
    mock::filesystem::create_file "$instance_script" '#!/bin/bash'
    
    function_exists() { [[ "$1" == "instance::shutdown_all" ]]; }
    instance::shutdown_all() { return 0; }
    export -f function_exists instance::shutdown_all
    
    run develop::universal::main
    
    assert_success
    mock::commands::assert::called "log::success" "✅ All instances stopped"
}

@test "develop::universal::main skips instance check when requested" {
    export SKIP_INSTANCE_CHECK="yes"
    mock::commands::setup_function "flow::is_yes" 0  # Return true for skip check
    
    run develop::universal::main
    
    assert_success
    # Instance checking should be skipped
}

@test "develop::universal::main loads environment variables" {
    mock::filesystem::create_file "${var_ENV_DEV_FILE}" 'export TEST_VAR="loaded"'
    
    develop::universal::main
    
    assert_equal "$TEST_VAR" "loaded"
}

@test "develop::universal::main exports parameters for service.json" {
    export TARGET="docker"
    export DETACHED="yes"
    export ENVIRONMENT="staging"
    
    develop::universal::main
    
    assert_equal "$TARGET" "docker"
    assert_equal "$DETACHED" "yes" 
    assert_equal "$ENVIRONMENT" "staging"
}

@test "develop::universal::main runs hooks correctly" {
    run develop::universal::main
    
    assert_success
    mock::commands::assert::called "phase::run_hook" "preDevelop"
    mock::commands::assert::called "phase::run_hook" "postDevelop"
}

################################################################################
# Cleanup Tests
################################################################################

@test "develop::cleanup runs cleanup hook and instance manager" {
    local instance_script="${var_LIB_SERVICE_DIR}/instance_manager.sh"
    mock::filesystem::create_file "$instance_script" '#!/bin/bash'
    
    function_exists() { [[ "$1" == "instance::shutdown_all" ]]; }
    instance::shutdown_all() { return 0; }
    export -f function_exists instance::shutdown_all
    
    run develop::cleanup
    
    assert_success
    mock::commands::assert::called "phase::run_hook" "cleanupDevelop"
    mock::commands::assert::called "log::success" "✅ Cleanup complete"
}

################################################################################
# Entry Point Tests
################################################################################

@test "develop script requires lifecycle engine invocation" {
    unset LIFECYCLE_PHASE
    mock::commands::setup_function "log::error" 0
    mock::commands::setup_function "log::info" 0
    
    run bash -c 'BASH_SOURCE=("test_script"); 0="test_script"; source "${BATS_TEST_DIRNAME}/develop.sh"'
    
    assert_failure
    assert_equal "$status" 1
}