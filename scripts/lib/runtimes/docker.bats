#!/usr/bin/env bats
# Tests for docker.sh - Docker management functions

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-assert/load"

setup() {
    # Setup will load the required mocks via setup.bash
    vrooli_setup_unit_test
    
    # Source the docker utilities
    source "${BATS_TEST_DIRNAME}/docker.sh"
    
    # Reset mocks if functions are available
    if declare -F docker_mock_reset >/dev/null 2>&1; then
        docker_mock_reset
    fi
    if declare -F system_mock_reset >/dev/null 2>&1; then
        system_mock_reset
    fi
}

teardown() {
    vrooli_cleanup_test
}

# Test basic functionality
@test "docker::run executes docker commands" {
    # Tier 2 mocks use different function names
    if declare -F docker_mock_set_container >/dev/null 2>&1; then
        docker_mock_set_container "test-container" "running"
    fi
    
    run docker::run ps --format "{{.Names}}"
    assert_success
    assert_output --partial "test-container"
}

@test "docker::compose detects compose command availability" {
    # Test plugin version detection
    run docker::_get_compose_command
    assert_success
    assert_output "docker compose"
}

@test "docker::install checks for docker installation" {
    # Tier 2 mocks don't have system mock, skip if not available
    skip "System mock not yet available in Tier 2"
    
    run docker::install
    assert_success
}

@test "docker::start handles already running docker" {
    # Mock docker as already running
    mock::docker::set_container_state "test" "running"
    
    run docker::start
    assert_success
    assert_output --partial "already running"
}

@test "docker::diagnose shows docker status" {
    mock::system::set_command_available "docker" true
    mock::docker::set_container_state "test" "running"
    
    run docker::diagnose
    assert_success
    assert_output --partial "Docker Diagnostics"
}

@test "docker::kill_all handles no containers gracefully" {
    mock::system::set_command_available "docker" true
    
    run docker::kill_all
    assert_success
    assert_output --partial "No running containers"
}