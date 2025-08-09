#!/usr/bin/env bats
# Tests for config.sh - Configuration initialization utilities

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-assert/load"

setup() {
    vrooli_setup_unit_test
    
    # Source the config utilities
    source "${BATS_TEST_DIRNAME}/config.sh"
    
    # Create mock project structure
    export MOCK_PROJECT_ROOT="${MOCK_TMP_DIR}/project"
    mkdir -p "${MOCK_PROJECT_ROOT}"
    
    # Override var variables for testing
    export var_VROOLI_CONFIG_DIR="${MOCK_PROJECT_ROOT}/.vrooli"
    export var_EXAMPLES_DIR="${MOCK_PROJECT_ROOT}/.vrooli/examples"
}

teardown() {
    vrooli_cleanup_test
}

@test "config::is_development detects development environment" {
    export ENVIRONMENT="development"
    
    run config::is_development
    assert_success
    
    export ENVIRONMENT="dev"
    run config::is_development
    assert_success
    
    export ENVIRONMENT="Development"
    run config::is_development
    assert_success
}

@test "config::is_development detects non-development environment" {
    export ENVIRONMENT="production"
    
    run config::is_development
    assert_failure
    
    export ENVIRONMENT="staging"
    run config::is_development
    assert_failure
}

@test "config::is_development uses parameter override" {
    run config::is_development "development"
    assert_success
    
    run config::is_development "production"
    assert_failure
}

@test "config::init creates .vrooli directory" {
    # Ensure directory doesn't exist initially
    rm -rf "${MOCK_PROJECT_ROOT}/.vrooli"
    
    run config::init
    assert_success
    
    # Check directory was created
    assert [ -d "${MOCK_PROJECT_ROOT}/.vrooli" ]
    assert_output --partial "Creating .vrooli directory"
}

@test "config::init handles existing .vrooli directory" {
    # Create directory first
    mkdir -p "${MOCK_PROJECT_ROOT}/.vrooli"
    
    run config::init
    assert_success
    
    # Directory should still exist
    assert [ -d "${MOCK_PROJECT_ROOT}/.vrooli" ]
}