#!/usr/bin/env bats
################################################################################
# Tests for common.sh - Universal lifecycle phase utilities
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

# Source the file under test
setup() {
    vrooli_setup_unit_test
    
    # Reset all mocks
    mock::commands::reset
    mock::system::reset
    mock::filesystem::reset
    
    # Source common.sh
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/common.sh"
    
    # Set up test environment
    export PHASE_NAME="test-phase"
    export PHASE_START_TIME=1234567890
    export VROOLI_CONTEXT="monorepo"
}

teardown() {
    vrooli_cleanup_test
}

################################################################################
# Phase Initialization Tests
################################################################################

@test "phase::init sets global variables correctly" {
    run phase::init "TestPhase"
    
    assert_success
    assert_equal "$PHASE_NAME" "TestPhase"
    assert [ -n "$PHASE_START_TIME" ]
}

@test "phase::init logs phase start information" {
    # Mock log functions
    mock::commands::setup_function "log::header" 0
    mock::commands::setup_function "log::debug" 0
    
    run phase::init "TestPhase"
    
    assert_success
    mock::commands::assert::called "log::header"
    mock::commands::assert::called_with "log::header" "🚀 Starting TestPhase phase"
}

@test "phase::complete calculates duration correctly" {
    export PHASE_START_TIME=1234567890
    mock::commands::setup_command "date" "+%s" "1234567900"
    mock::commands::setup_function "log::success" 0
    
    run phase::complete
    
    assert_success
    mock::commands::assert::called_with "log::success" "✅ $PHASE_NAME phase completed in 10s"
}

################################################################################
# Context Detection Tests
################################################################################

@test "phase::is_ci detects CI environment correctly" {
    export CI="true"
    run phase::is_ci
    assert_success
    
    unset CI
    export IS_CI="yes"
    run phase::is_ci
    assert_success
    
    export GITHUB_ACTIONS="true"
    run phase::is_ci
    assert_success
    
    export GITLAB_CI="true"
    run phase::is_ci
    assert_success
}

@test "phase::is_ci returns false for non-CI environment" {
    unset CI IS_CI GITHUB_ACTIONS GITLAB_CI
    run phase::is_ci
    assert_failure
}

@test "phase::is_monorepo detects monorepo context" {
    export VROOLI_CONTEXT="monorepo"
    run phase::is_monorepo
    assert_success
    
    export VROOLI_CONTEXT="standalone"
    run phase::is_monorepo
    assert_failure
}

@test "phase::is_standalone detects standalone context" {
    export VROOLI_CONTEXT="standalone"
    run phase::is_standalone
    assert_success
    
    export VROOLI_CONTEXT="monorepo"
    run phase::is_standalone
    assert_failure
}

################################################################################
# Configuration Management Tests
################################################################################

@test "phase::get_config returns empty string for missing path" {
    run phase::get_config ""
    assert_success
    assert_output ""
}

@test "phase::get_config returns empty string for missing file" {
    export var_SERVICE_JSON_FILE="/nonexistent/file.json"
    run phase::get_config ".some.path"
    assert_success
    assert_output ""
}

@test "phase::get_config uses jq when available" {
    # Create mock service.json
    mock::filesystem::create_file "${MOCK_ROOT}/service.json" '{
        "lifecycle": {
            "setup": {
                "description": "Test setup"
            }
        }
    }'
    
    export var_SERVICE_JSON_FILE="${MOCK_ROOT}/service.json"
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-r \".lifecycle.setup.description // empty\" \"${MOCK_ROOT}/service.json\"" "Test setup"
    
    run phase::get_config ".lifecycle.setup.description"
    
    assert_success
    assert_output "Test setup"
}

@test "phase::get_config returns empty when jq unavailable" {
    export var_SERVICE_JSON_FILE="${MOCK_ROOT}/service.json"
    mock::commands::setup_command "command -v jq" "" "1"
    
    run phase::get_config ".some.path"
    
    assert_success
    assert_output ""
}

################################################################################
# Command Execution Tests
################################################################################

@test "phase::execute runs command successfully" {
    mock::commands::setup_function "log::debug" 0
    mock::commands::setup_command "echo" "test" "test"
    
    run phase::execute "echo test"
    
    assert_success
    mock::commands::assert::called "log::debug"
}

@test "phase::execute handles command failure" {
    mock::commands::setup_function "log::debug" 0
    mock::commands::setup_function "log::error" 0
    mock::commands::setup_command "false" "" "1"
    
    run phase::execute "false"
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error"
}

@test "phase::execute_if skips monorepo-only commands in standalone context" {
    export VROOLI_CONTEXT="standalone"
    mock::commands::setup_function "log::debug" 0
    
    run phase::execute_if "monorepo" "echo test"
    
    assert_success
    mock::commands::assert::called_with "log::debug" "Skipping monorepo-only command: echo test"
}

@test "phase::execute_if skips standalone-only commands in monorepo context" {
    export VROOLI_CONTEXT="monorepo"
    mock::commands::setup_function "log::debug" 0
    
    run phase::execute_if "standalone" "echo test"
    
    assert_success
    mock::commands::assert::called_with "log::debug" "Skipping standalone-only command: echo test"
}

@test "phase::execute_if runs commands for 'all' context" {
    mock::commands::setup_function "log::debug" 0
    mock::commands::setup_command "echo" "test" "test"
    
    run phase::execute_if "all" "echo test"
    
    assert_success
}

@test "phase::execute_if fails for unknown context" {
    mock::commands::setup_function "log::warning" 0
    
    run phase::execute_if "invalid" "echo test"
    
    assert_failure
    assert_equal "$status" 1
}

################################################################################
# Script Sourcing Tests
################################################################################

@test "phase::source_if_exists sources existing script" {
    mock::filesystem::create_file "${MOCK_ROOT}/test_script.sh" "#!/bin/bash\nexport TEST_SOURCED=true"
    chmod +x "${MOCK_ROOT}/test_script.sh"
    mock::commands::setup_function "log::debug" 0
    
    run phase::source_if_exists "${MOCK_ROOT}/test_script.sh"
    
    assert_success
    mock::commands::assert::called "log::debug"
}

@test "phase::source_if_exists skips missing script" {
    mock::commands::setup_function "log::debug" 0
    
    run phase::source_if_exists "${MOCK_ROOT}/missing_script.sh"
    
    assert_success
    mock::commands::assert::called_with "log::debug" "Script not found (skipping): ${MOCK_ROOT}/missing_script.sh"
}

@test "phase::source_if_exists fails on script error" {
    mock::filesystem::create_file "${MOCK_ROOT}/bad_script.sh" "#!/bin/bash\nexit 1"
    chmod +x "${MOCK_ROOT}/bad_script.sh"
    mock::commands::setup_function "log::debug" 0
    mock::commands::setup_function "log::error" 0
    
    run phase::source_if_exists "${MOCK_ROOT}/bad_script.sh"
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error"
}

################################################################################
# Hook Execution Tests
################################################################################

@test "phase::run_hook executes hook from lifecycle.hooks" {
    # Create mock service.json with hook
    mock::filesystem::create_file "${MOCK_ROOT}/service.json" '{
        "lifecycle": {
            "hooks": {
                "postSetup": "echo post-setup hook executed"
            }
        }
    }'
    
    export var_SERVICE_JSON_FILE="${MOCK_ROOT}/service.json"
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-r \".lifecycle.hooks.postSetup // empty\" \"${MOCK_ROOT}/service.json\"" "echo post-setup hook executed"
    mock::commands::setup_function "log::info" 0
    mock::commands::setup_function "log::debug" 0
    mock::commands::setup_function "log::success" 0
    mock::commands::setup_command "echo" "post-setup hook executed" "post-setup hook executed"
    
    run phase::run_hook "postSetup"
    
    assert_success
    mock::commands::assert::called "log::info"
    mock::commands::assert::called "log::success"
}

@test "phase::run_hook falls back to legacy repository.hooks location" {
    # Create mock service.json with legacy hook location
    mock::filesystem::create_file "${MOCK_ROOT}/service.json" '{
        "repository": {
            "hooks": {
                "postSetup": "echo legacy hook executed"
            }
        }
    }'
    
    export var_SERVICE_JSON_FILE="${MOCK_ROOT}/service.json"
    mock::commands::setup_command "command -v jq" "" "0"
    # First call returns empty (no lifecycle.hooks.postSetup)
    mock::commands::setup_command "jq" "-r \".lifecycle.hooks.postSetup // empty\" \"${MOCK_ROOT}/service.json\"" ""
    # Second call returns legacy hook
    mock::commands::setup_command "jq" "-r \".repository.hooks.postSetup // empty\" \"${MOCK_ROOT}/service.json\"" "echo legacy hook executed"
    mock::commands::setup_function "log::info" 0
    mock::commands::setup_function "log::debug" 0
    mock::commands::setup_function "log::success" 0
    mock::commands::setup_command "echo" "legacy hook executed" "legacy hook executed"
    
    run phase::run_hook "postSetup"
    
    assert_success
}

@test "phase::run_hook logs debug message when no hook defined" {
    export var_SERVICE_JSON_FILE="${MOCK_ROOT}/service.json"
    mock::filesystem::create_file "${MOCK_ROOT}/service.json" '{}'
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-r \".lifecycle.hooks.nonexistent // empty\" \"${MOCK_ROOT}/service.json\"" ""
    mock::commands::setup_command "jq" "-r \".repository.hooks.nonexistent // empty\" \"${MOCK_ROOT}/service.json\"" ""
    mock::commands::setup_function "log::debug" 0
    
    run phase::run_hook "nonexistent"
    
    assert_success
    mock::commands::assert::called_with "log::debug" "No nonexistent hook defined"
}

@test "phase::run_hook handles hook execution failure" {
    mock::filesystem::create_file "${MOCK_ROOT}/service.json" '{
        "lifecycle": {
            "hooks": {
                "testHook": "false"
            }
        }
    }'
    
    export var_SERVICE_JSON_FILE="${MOCK_ROOT}/service.json"
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-r \".lifecycle.hooks.testHook // empty\" \"${MOCK_ROOT}/service.json\"" "false"
    mock::commands::setup_function "log::info" 0
    mock::commands::setup_function "log::debug" 0
    mock::commands::setup_function "log::warning" 0
    mock::commands::setup_command "false" "" "1"
    
    run phase::run_hook "testHook"
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::warning"
}