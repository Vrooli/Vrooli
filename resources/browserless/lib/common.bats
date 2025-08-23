#!/usr/bin/env bats
# Tests for Browserless common.sh functions
bats_require_minimum_version 1.5.0

# Setup paths and source var.sh first
SCRIPT_DIR="$(cd "$(dirname "${BATS_TEST_FILENAME}")" && pwd)"
# shellcheck disable=SC1091
source "$(cd "${SCRIPT_DIR}/../../../../../" && pwd)/lib/utils/var.sh"

# Load Vrooli test infrastructure using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "browserless"
    
    # Set up directories and paths once
    export BROWSERLESS_DIR="${SCRIPT_DIR}/.."
    export MOCK_DIR="${var_SCRIPTS_TEST_DIR}/fixtures/mocks"
    
    # Load common resources
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
    
    # Load the system mock once
    if [[ -f "$MOCK_DIR/system.sh" ]]; then
        # shellcheck disable=SC1091
        source "$MOCK_DIR/system.sh"
    fi
    
    # Load the docker mock once
    if [[ -f "$MOCK_DIR/docker.sh" ]]; then
        # shellcheck disable=SC1091
        source "$MOCK_DIR/docker.sh"
    fi
    
    # Load configuration and messages
    # shellcheck disable=SC1091
    source "${BROWSERLESS_DIR}/config/defaults.sh"
    # shellcheck disable=SC1091
    source "${BROWSERLESS_DIR}/config/messages.sh"
    
    # Load common functions
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/common.sh"
    
    # Export functions for subshells
    export -f browserless::check_docker
    export -f browserless::container_exists
    export -f browserless::is_running
    export -f browserless::check_port
    export -f browserless::validate_prerequisites
    export -f browserless::check_existing_installation
    export -f browserless::wait_for_ready
    export -f browserless::show_resource_usage
    export -f browserless::backup_data
}

# Lightweight per-test setup
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Set test environment
    export BROWSERLESS_CUSTOM_PORT="9999"
    export BROWSERLESS_CONTAINER_NAME="browserless-test"
    export BROWSERLESS_PORT="9999"
    export BROWSERLESS_BASE_URL="http://localhost:9999"
    
    # Export configuration
    browserless::export_config
    browserless::export_messages
    
    # Configure mocks for default successful scenario
    if declare -f mock::system::set_command_available >/dev/null 2>&1; then
        mock::system::set_command_available "docker" "true"
    fi
    
    if declare -f mock::docker::set_daemon_running >/dev/null 2>&1; then
        mock::docker::set_daemon_running "true"
    fi
    
    if declare -f mock::system::set_port_in_use >/dev/null 2>&1; then
        mock::system::set_port_in_use "9999" "false"
    fi
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# Test Docker check with Docker available
@test "browserless::check_docker succeeds when docker is available" {
    # Use official mocks - docker is available and working
    if declare -f mock::system::set_command_available >/dev/null 2>&1; then
        mock::system::set_command_available "docker" "true"
    fi
    
    if declare -f mock::docker::set_daemon_running >/dev/null 2>&1; then
        mock::docker::set_daemon_running "true"
    fi
    
    if declare -f mock::docker::set_permissions >/dev/null 2>&1; then
        mock::docker::set_permissions "true"
    fi
    
    run browserless::check_docker
    [ "$status" -eq 0 ]
}

# Test Docker check fails when docker not found
@test "browserless::check_docker fails when docker not found" {
    # Use official mock to simulate docker not available
    if declare -f mock::system::set_command_available >/dev/null 2>&1; then
        mock::system::set_command_available "docker" "false"
    fi
    
    run browserless::check_docker
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Docker not found" ]] || [[ "$output" =~ "docker" ]]
}

# Test port checking
@test "browserless::check_port succeeds when port is available" {
    # Use official mock - port is available
    if declare -f mock::system::set_port_in_use >/dev/null 2>&1; then
        mock::system::set_port_in_use "$BROWSERLESS_PORT" "false"
    fi
    
    run browserless::check_port
    [ "$status" -eq 0 ]
}

# Test port checking fails when port in use
@test "browserless::check_port fails when port is in use" {
    # Use official mock - port is in use
    if declare -f mock::system::set_port_in_use >/dev/null 2>&1; then
        mock::system::set_port_in_use "$BROWSERLESS_PORT" "true"
    fi
    
    run browserless::check_port
    [ "$status" -eq 1 ]
    [[ "$output" =~ "in use" ]] || [[ "$output" =~ "port" ]]
}

# Test container existence check
@test "browserless::container_exists works with docker ps output" {
    # Use official docker mock - container exists
    if declare -f mock::docker::set_container_state >/dev/null 2>&1; then
        mock::docker::set_container_state "$BROWSERLESS_CONTAINER_NAME" "stopped" "ghcr.io/browserless/chrome"
    fi
    
    run browserless::container_exists
    [ "$status" -eq 0 ]
}

# Test running check
@test "browserless::is_running works with docker ps output" {
    # Use official docker mock - container is running
    if declare -f mock::docker::set_container_state >/dev/null 2>&1; then
        mock::docker::set_container_state "$BROWSERLESS_CONTAINER_NAME" "running" "ghcr.io/browserless/chrome"
    fi
    
    run browserless::is_running
    [ "$status" -eq 0 ]
}

# Test prerequisites validation
@test "browserless::validate_prerequisites succeeds with all requirements met" {
    # Set up all prerequisites as available
    if declare -f mock::system::set_command_available >/dev/null 2>&1; then
        mock::system::set_command_available "docker" "true"
    fi
    
    if declare -f mock::docker::set_daemon_running >/dev/null 2>&1; then
        mock::docker::set_daemon_running "true"
        mock::docker::set_permissions "true"
    fi
    
    if declare -f mock::system::set_port_in_use >/dev/null 2>&1; then
        mock::system::set_port_in_use "$BROWSERLESS_PORT" "false"
    fi
    
    # Mock resources::validate_port to succeed
    resources::validate_port() {
        return 0
    }
    export -f resources::validate_port
    
    run browserless::validate_prerequisites
    [ "$status" -eq 0 ]
}

# Test wait for ready timeout
@test "browserless::wait_for_ready times out appropriately" {
    # Set up container not running to trigger timeout
    if declare -f mock::docker::set_container_state >/dev/null 2>&1; then
        mock::docker::set_container_state "$BROWSERLESS_CONTAINER_NAME" "stopped" "ghcr.io/browserless/chrome"
    fi
    
    # Mock ss command to not find listening port
    ss() {
        return 1
    }
    export -f ss
    
    # Use very short timeout for testing
    run browserless::wait_for_ready "test service" 2
    [ "$status" -eq 1 ]
    [[ "$output" =~ "timeout" ]] || [[ "$output" =~ "Timeout" ]]
}