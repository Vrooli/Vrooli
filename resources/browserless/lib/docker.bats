#!/usr/bin/env bats
# Tests for Browserless docker.sh functions
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
    
    # Load mocks
    if [[ -f "$MOCK_DIR/docker.sh" ]]; then
        # shellcheck disable=SC1091
        source "$MOCK_DIR/docker.sh"
    fi
    
    # Load configuration and messages
    # shellcheck disable=SC1091
    source "${BROWSERLESS_DIR}/config/defaults.sh"
    # shellcheck disable=SC1091
    source "${BROWSERLESS_DIR}/config/messages.sh"
    
    # Load docker functions
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/docker.sh"
    
    # Export functions for subshells
    export -f browserless::create_network
    export -f browserless::build_docker_command
    export -f browserless::docker_start
    export -f browserless::docker_stop
    export -f browserless::docker_restart
    export -f browserless::docker_remove
    export -f browserless::docker_logs
    export -f browserless::docker_cleanup
}

# Lightweight per-test setup
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Set test environment
    export BROWSERLESS_CUSTOM_PORT="9999"
    export BROWSERLESS_CONTAINER_NAME="browserless-test"
    export BROWSERLESS_NETWORK_NAME="browserless-test-network"
    export BROWSERLESS_MAX_BROWSERS="3"
    export BROWSERLESS_TIMEOUT="15000"
    export BROWSERLESS_HEADLESS="yes"
    export BROWSERLESS_DATA_DIR="/tmp/browserless-test"
    export BROWSERLESS_IMAGE="ghcr.io/browserless/chrome:test"
    export BROWSERLESS_PORT="9999"
    export BROWSERLESS_BASE_URL="http://localhost:9999"
    
    # Export configuration
    browserless::export_config
    browserless::export_messages
    
    # Mock rollback functions
    resources::add_rollback_action() {
        # Mock implementation
        return 0
    }
    export -f resources::add_rollback_action
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# Test Docker command building
@test "browserless::build_docker_command creates correct command" {
    run browserless::build_docker_command "5" "30000" "yes"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"docker run -d"* ]]
    [[ "$output" == *"--name $BROWSERLESS_CONTAINER_NAME"* ]]
    [[ "$output" == *"-p ${BROWSERLESS_PORT}:3000"* ]]
    [[ "$output" == *"-e CONCURRENT=5"* ]]
    [[ "$output" == *"-e TIMEOUT=30000"* ]]
    [[ "$output" == *"$BROWSERLESS_IMAGE"* ]]
}

# Test Docker command with default parameters
@test "browserless::build_docker_command uses defaults when no params provided" {
    run browserless::build_docker_command
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"-e CONCURRENT=${BROWSERLESS_MAX_BROWSERS}"* ]]
    [[ "$output" == *"-e TIMEOUT=${BROWSERLESS_TIMEOUT}"* ]]
}

# Test network creation
@test "browserless::create_network handles network creation" {
    # Mock docker network commands
    docker() {
        case "$1 $2" in
            "network ls") echo "other-network" ;;  # Network doesn't exist
            "network create") return 0 ;;
            *) return 0 ;;
        esac
    }
    
    log::info() { echo "INFO: $*"; }
    log::success() { echo "SUCCESS: $*"; }
    
    run browserless::create_network
    [ "$status" -eq 0 ]
}

# Test network creation when network already exists
@test "browserless::create_network skips when network exists" {
    # Mock docker network commands
    docker() {
        case "$1 $2" in
            "network ls") echo "$BROWSERLESS_NETWORK_NAME" ;;  # Network exists
            *) return 0 ;;
        esac
    }
    
    run browserless::create_network
    [ "$status" -eq 0 ]
}

# Test container removal
@test "browserless::docker_remove handles cleanup" {
    # Mock docker commands
    docker() {
        case "$1" in
            "stop"|"rm") return 0 ;;
            "network") return 0 ;;
            *) return 0 ;;
        esac
    }
    
    # Mock container existence check
    browserless::container_exists() {
        return 0
    }
    
    log::info() { echo "INFO: $*"; }
    log::success() { echo "SUCCESS: $*"; }
    
    run browserless::docker_remove "no"
    [ "$status" -eq 0 ]
}