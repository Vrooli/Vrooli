#!/usr/bin/env bats
# Tests for Browserless docker.sh functions

# Setup for each test
setup() {
    # Set test environment
    export BROWSERLESS_CUSTOM_PORT="9999"
    export BROWSERLESS_CONTAINER_NAME="browserless-test"
    export BROWSERLESS_NETWORK_NAME="browserless-test-network"
    export BROWSERLESS_MAX_BROWSERS="3"
    export BROWSERLESS_TIMEOUT="15000"
    export BROWSERLESS_HEADLESS="yes"
    export BROWSERLESS_DATA_DIR="/tmp/browserless-test"
    export BROWSERLESS_IMAGE="ghcr.io/browserless/chrome:test"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    BROWSERLESS_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and messages
    source "${BROWSERLESS_DIR}/config/defaults.sh"
    source "${BROWSERLESS_DIR}/config/messages.sh"
    browserless::export_config
    browserless::export_messages
    
    # Mock rollback functions
    resources::add_rollback_action() {
        # Mock implementation
        return 0
    }
    
    # Load Docker functions
    source "${SCRIPT_DIR}/docker.sh"
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