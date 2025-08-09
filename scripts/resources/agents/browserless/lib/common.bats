#!/usr/bin/env bats
# Tests for Browserless common.sh functions

# Setup for each test
setup() {
    # Set test environment
    export BROWSERLESS_CUSTOM_PORT="9999"
    export BROWSERLESS_CONTAINER_NAME="browserless-test"
    
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    BROWSERLESS_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Mock system functions for testing
    system::is_command() {
        case "$1" in
            "docker") return 0 ;;
            *) return 1 ;;
        esac
    }
    
    system::is_port_in_use() {
        # Mock port as available for testing
        return 1
    }
    
    # Load configuration and messages
    source "${BROWSERLESS_DIR}/config/defaults.sh"
    source "${BROWSERLESS_DIR}/config/messages.sh"
    browserless::export_config
    browserless::export_messages
    
    # Load common functions
    source "${SCRIPT_DIR}/common.sh"
}

# Test Docker check with Docker available
@test "browserless::check_docker succeeds when docker is available" {
    # Mock successful docker commands
    docker() {
        case "$1" in
            "info") return 0 ;;
            "ps") return 0 ;;
            *) return 0 ;;
        esac
    }
    
    run browserless::check_docker
    [ "$status" -eq 0 ]
}

# Test Docker check fails when docker not found
@test "browserless::check_docker fails when docker not found" {
    # Override system::is_command to return false for docker
    system::is_command() {
        return 1
    }
    
    run browserless::check_docker
    [ "$status" -eq 1 ]
}

# Test port checking
@test "browserless::check_port succeeds when port is available" {
    run browserless::check_port
    [ "$status" -eq 0 ]
}

# Test port checking fails when port in use
@test "browserless::check_port fails when port is in use" {
    # Override to return port in use
    system::is_port_in_use() {
        return 0
    }
    
    run browserless::check_port
    [ "$status" -eq 1 ]
}

# Test container existence check
@test "browserless::container_exists works with docker ps output" {
    # Mock docker ps to return container name
    docker() {
        if [[ "$1" == "ps" ]]; then
            echo "$BROWSERLESS_CONTAINER_NAME"
        fi
    }
    
    run browserless::container_exists
    [ "$status" -eq 0 ]
}

# Test running check
@test "browserless::is_running works with docker ps output" {
    # Mock docker ps to return container name
    docker() {
        if [[ "$1" == "ps" ]]; then
            echo "$BROWSERLESS_CONTAINER_NAME"
        fi
    }
    
    run browserless::is_running
    [ "$status" -eq 0 ]
}