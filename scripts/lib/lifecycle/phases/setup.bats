#!/usr/bin/env bats
################################################################################
# Tests for setup.sh - Universal setup phase handler
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
load "${BATS_TEST_DIRNAME}/../../../__test/fixtures/mocks/docker.sh"

setup() {
    vrooli_setup_unit_test
    
    # Reset all mocks
    mock::commands::reset
    mock::system::reset
    mock::filesystem::reset
    mock::docker::reset
    
    # Mock all the required functions that setup.sh depends on
    mock::commands::setup_function "phase::init" 0
    mock::commands::setup_function "phase::complete" 0
    mock::commands::setup_function "phase::run_hook" 0
    mock::commands::setup_function "log::info" 0
    mock::commands::setup_function "log::debug" 0
    mock::commands::setup_function "log::header" 0
    mock::commands::setup_function "log::success" 0
    mock::commands::setup_function "log::warning" 0
    mock::commands::setup_function "log::error" 0
    mock::commands::setup_function "flow::is_yes" 0
    mock::commands::setup_function "flow::can_run_sudo" 0
    mock::commands::setup_function "target_matcher::match_target" 0 "native-linux"
    
    # Mock system functions
    mock::commands::setup_function "permissions::make_scripts_executable" 0
    mock::commands::setup_function "clock::fix" 0
    mock::commands::setup_function "network_diagnostics::run" 0
    mock::commands::setup_function "firewall::setup" 0
    mock::commands::setup_function "system::update_and_upgrade" 0
    mock::commands::setup_function "common_deps::check_and_install" 0
    mock::commands::setup_function "config::init" 0
    mock::commands::setup_function "bats::install" 0
    mock::commands::setup_function "shellcheck::install" 0
    mock::commands::setup_function "docker::setup" 0
    mock::commands::setup_function "docker::diagnose" 0
    
    # Set up test environment variables
    export TARGET="native-linux"
    export CLEAN="no"
    export IS_CI="no"
    export SUDO_MODE="ask"
    export ENVIRONMENT="development"
    export RESOURCES="enabled"
    export VROOLI_CONTEXT="monorepo"
    export var_ROOT_DIR="${MOCK_ROOT}"
    export var_SCRIPTS_RESOURCES_DIR="${MOCK_ROOT}/scripts/resources"
    
    # Create mock git directory
    mock::filesystem::create_directory "${MOCK_ROOT}/.git"
    
    # Mock git command
    mock::commands::setup_command "command -v git" "" "0"
    mock::commands::setup_command "git" "config core.filemode false" ""
    
    # Source the phase utilities (mocked functions)
    export -f phase::init phase::complete phase::run_hook
}

teardown() {
    vrooli_cleanup_test
}

################################################################################
# Setup Main Function Tests
################################################################################

@test "setup::universal_main initializes phase correctly" {
    # Skip resource installation for this test
    export RESOURCES="none"
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::called "phase::init" "Setup"
    mock::commands::assert::called "phase::complete"
}

@test "setup::universal_main validates target parameter" {
    export TARGET="invalid-target"
    mock::commands::setup_function "target_matcher::match_target" 1  # Return error
    mock::commands::setup_function "log::error" 0
    
    run setup::universal_main
    
    assert_failure
    mock::commands::assert::called "log::error"
    mock::commands::assert::called_with "target_matcher::match_target" "invalid-target"
}

@test "setup::universal_main sets sudo mode to skip in standalone context" {
    export VROOLI_CONTEXT="standalone"
    unset SUDO_MODE_EXPLICIT
    export RESOURCES="none"  # Skip resource installation
    
    run setup::universal_main
    
    assert_success
    assert_equal "$SUDO_MODE" "skip"
}

@test "setup::universal_main skips system preparation in CI" {
    export IS_CI="yes"
    export RESOURCES="none"  # Skip resource installation
    mock::commands::setup_function "flow::is_yes" 0 "yes"  # Return true for CI check
    
    run setup::universal_main
    
    assert_success
    # These should not be called in CI
    mock::commands::assert::not_called "permissions::make_scripts_executable"
    mock::commands::assert::not_called "clock::fix"
    mock::commands::assert::not_called "network_diagnostics::run"
}

@test "setup::universal_main runs system preparation when not in CI" {
    export IS_CI="no"
    export RESOURCES="none"  # Skip resource installation
    mock::commands::setup_function "flow::is_yes" 1 "no"  # Return false for CI check
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::called "permissions::make_scripts_executable"
    mock::commands::assert::called "clock::fix"
    mock::commands::assert::called "network_diagnostics::run"
}

@test "setup::universal_main handles network diagnostics failure gracefully" {
    export IS_CI="no"
    export RESOURCES="none"
    mock::commands::setup_function "flow::is_yes" 1 "no"
    mock::commands::setup_function "network_diagnostics::run" 1  # Simulate failure
    
    run setup::universal_main
    
    assert_success  # Should continue despite network test failures
    mock::commands::assert::called "log::info"
}

@test "setup::universal_main skips firewall setup without sudo" {
    export IS_CI="no"
    export RESOURCES="none"
    mock::commands::setup_function "flow::is_yes" 1 "no"
    mock::commands::setup_function "flow::can_run_sudo" 1  # No sudo access
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::called "log::warning"
    mock::commands::assert::not_called "firewall::setup"
}

@test "setup::universal_main runs firewall setup with sudo" {
    export IS_CI="no"
    export ENVIRONMENT="production"
    export RESOURCES="none"
    mock::commands::setup_function "flow::is_yes" 1 "no"
    mock::commands::setup_function "flow::can_run_sudo" 0  # Has sudo access
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::called "firewall::setup" "production"
}

@test "setup::universal_main updates system packages in monorepo" {
    export VROOLI_CONTEXT="monorepo"
    export IS_CI="no"
    export RESOURCES="none"
    mock::commands::setup_function "flow::is_yes" 1 "no"
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::called "system::update_and_upgrade"
}

@test "setup::universal_main skips system updates in standalone" {
    export VROOLI_CONTEXT="standalone"
    export IS_CI="no"
    export RESOURCES="none"
    mock::commands::setup_function "flow::is_yes" 1 "no"
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::not_called "system::update_and_upgrade"
}

################################################################################
# Dependency Installation Tests
################################################################################

@test "setup::universal_main installs common dependencies" {
    export RESOURCES="none"
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::called "common_deps::check_and_install"
}

@test "setup::universal_main initializes configuration" {
    export RESOURCES="none"
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::called "config::init"
}

@test "setup::universal_main configures git when available" {
    export RESOURCES="none"
    mock::filesystem::create_directory "${MOCK_ROOT}/.git"
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::called "git" "config core.filemode false"
}

@test "setup::universal_main skips git config when git unavailable" {
    export RESOURCES="none"
    mock::commands::setup_command "command -v git" "" "1"  # git not found
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::not_called "git"
}

@test "setup::universal_main skips git config without git directory" {
    export RESOURCES="none"
    # Don't create .git directory
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::not_called "git"
}

################################################################################
# Test Tools Installation Tests
################################################################################

@test "setup::universal_main installs test tools in CI environment" {
    export IS_CI="yes"
    export ENVIRONMENT="development"
    export RESOURCES="none"
    mock::commands::setup_function "flow::is_yes" 0 "yes"  # Return true for CI
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::called "bats::install"
    mock::commands::assert::called "shellcheck::install"
}

@test "setup::universal_main installs test tools in development environment" {
    export IS_CI="no"
    export ENVIRONMENT="development"
    export RESOURCES="none"
    mock::commands::setup_function "flow::is_yes" 1 "no"  # Return false for CI
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::called "bats::install"
    mock::commands::assert::called "shellcheck::install"
}

@test "setup::universal_main skips test tools in production" {
    export IS_CI="no"
    export ENVIRONMENT="production"
    export RESOURCES="none"
    mock::commands::setup_function "flow::is_yes" 1 "no"
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::not_called "bats::install"
    mock::commands::assert::not_called "shellcheck::install"
}

################################################################################
# Docker Setup Tests
################################################################################

@test "setup::universal_main sets up Docker successfully" {
    export RESOURCES="none"
    mock::commands::setup_function "docker::setup" 0  # Success
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::called "docker::setup"
    mock::commands::assert::not_called "docker::diagnose"
}

@test "setup::universal_main handles Docker setup failure" {
    export RESOURCES="none"
    mock::commands::setup_function "docker::setup" 1  # Failure
    
    run setup::universal_main
    
    assert_failure
    mock::commands::assert::called "docker::setup"
    mock::commands::assert::called "docker::diagnose"
    mock::commands::assert::called "log::error"
}

@test "setup::universal_main provides WSL-specific error message" {
    export RESOURCES="none"
    mock::commands::setup_function "docker::setup" 1
    # Mock WSL detection
    mock::filesystem::create_file "/proc/version" "Linux version 4.4.0-43-Microsoft"
    mock::commands::setup_command "grep" "-qi microsoft /proc/version" ""
    
    run setup::universal_main
    
    assert_failure
    mock::commands::assert::called "log::error"
}

################################################################################
# Resource Installation Tests
################################################################################

@test "setup::universal_main skips resource installation when RESOURCES=none" {
    export RESOURCES="none"
    
    run setup::universal_main
    
    assert_success
    # Should not try to run resource script
    assert [ ! -f "${MOCK_ROOT}/scripts/resources/index.sh" ]
}

@test "setup::universal_main installs resources when script exists" {
    export RESOURCES="enabled"
    
    # Create mock resource script
    mock::filesystem::create_file "${var_SCRIPTS_RESOURCES_DIR}/index.sh" '#!/bin/bash\nexit 0'
    chmod +x "${var_SCRIPTS_RESOURCES_DIR}/index.sh"
    mock::commands::setup_command "bash" "${var_SCRIPTS_RESOURCES_DIR}/index.sh install" ""
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::called "bash" "${var_SCRIPTS_RESOURCES_DIR}/index.sh install"
}

@test "setup::universal_main handles resource installation failure gracefully" {
    export RESOURCES="enabled"
    
    # Create mock resource script that fails
    mock::filesystem::create_file "${var_SCRIPTS_RESOURCES_DIR}/index.sh" '#!/bin/bash\nexit 1'
    chmod +x "${var_SCRIPTS_RESOURCES_DIR}/index.sh"
    mock::commands::setup_command "bash" "${var_SCRIPTS_RESOURCES_DIR}/index.sh install" "" "1"
    
    run setup::universal_main
    
    assert_success  # Should not fail the entire setup
    mock::commands::assert::called "log::warning"
}

@test "setup::universal_main logs when resource management unavailable" {
    export RESOURCES="enabled"
    # Don't create resource script
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::called "log::debug"
}

################################################################################
# Hook Execution Tests
################################################################################

@test "setup::universal_main runs postSetup hook" {
    export RESOURCES="none"
    
    run setup::universal_main
    
    assert_success
    mock::commands::assert::called "phase::run_hook" "postSetup"
}

################################################################################
# Variable Export Tests
################################################################################

@test "setup::universal_main exports setup completion variables" {
    export RESOURCES="none"
    export TARGET="native-linux"
    export ENVIRONMENT="development"
    
    setup::universal_main
    
    assert_equal "$SETUP_COMPLETE" "true"
    assert_equal "$SETUP_TARGET" "native-linux"
    assert_equal "$SETUP_ENVIRONMENT" "development"
}

################################################################################
# Entry Point Tests
################################################################################

@test "setup script requires lifecycle engine invocation" {
    unset LIFECYCLE_PHASE
    mock::commands::setup_function "log::error" 0
    mock::commands::setup_function "log::info" 0
    
    # Test direct execution without LIFECYCLE_PHASE
    BASH_SOURCE=("${BATS_TEST_DIRNAME}/setup.sh")
    run bash -c 'source "${BATS_TEST_DIRNAME}/setup.sh"'
    
    assert_failure
    assert_equal "$status" 1
}

@test "setup script runs when called by lifecycle engine" {
    export LIFECYCLE_PHASE="setup"
    export RESOURCES="none"
    
    # Mock the main function
    setup::universal_main() { echo "setup executed"; }
    export -f setup::universal_main
    
    run bash -c '
        export LIFECYCLE_PHASE="setup"
        export RESOURCES="none" 
        BASH_SOURCE=("test_script")
        0="test_script"
        source "${BATS_TEST_DIRNAME}/setup.sh"
    '
    
    assert_success
}