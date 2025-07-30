#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Setup for each test  
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Path to the script under test
    SCRIPT_PATH="$BATS_TEST_DIRNAME/common.sh"
    VAULT_DIR="$BATS_TEST_DIRNAME/.."
    
    # Source dependencies
    RESOURCES_DIR="$VAULT_DIR/../.."
    HELPERS_DIR="$RESOURCES_DIR/../helpers"
    
    # Source utilities first
    . "$HELPERS_DIR/utils/log.sh"
    . "$HELPERS_DIR/utils/system.sh"
    . "$HELPERS_DIR/utils/ports.sh" 
    . "$HELPERS_DIR/utils/flow.sh"
    . "$RESOURCES_DIR/port-registry.sh"
    
    # Set test environment variables
    export VAULT_CONTAINER_NAME='vault'
    export VAULT_PORT='8200'
    export VAULT_MODE='dev'
    export VAULT_BASE_URL='http://localhost:8200'
    export VAULT_DATA_DIR='/tmp/vault-test-data'
    export VAULT_CONFIG_DIR='/tmp/vault-test-config'
    export VAULT_LOGS_DIR='/tmp/vault-test-logs'
    export VAULT_DEV_ROOT_TOKEN_ID='myroot'
    export VAULT_SECRET_ENGINE='secret'
    export VAULT_NAMESPACE_PREFIX='vrooli'
    export VAULT_TOKEN_FILE='/tmp/vault-test-token'
    export VAULT_UNSEAL_KEYS_FILE='/tmp/vault-test-keys'
    export SCRIPT_DIR="$VAULT_DIR"
    
    # Source vault configuration
    source "$VAULT_DIR/config/defaults.sh"
    source "$VAULT_DIR/config/messages.sh"
    vault::messages::init
    
    # Source common.sh and the script under test
    source "$RESOURCES_DIR/common.sh"
    source "$SCRIPT_PATH"
    
    # Vault-specific mock functions
    resources::add_rollback_action() { 
        echo "Rollback action added: $1" >&2
        return 0
    }
    
    # Setup standard mocks AFTER sourcing real utilities (to override them)
    setup_standard_mocks
}

# ============================================================================
# Container State Tests - Simplified versions using shared mocks
# ============================================================================

@test "vault::is_installed returns 0 when container exists" {
    docker() { 
        if [[ "$1 $2 $3" == 'container inspect vault' ]]; then
            echo 'container exists'
            return 0
        fi
        return 1
    }
    export -f docker
    run vault::is_installed
    [ "$status" -eq 0 ]
}

@test "vault::is_installed returns 1 when container doesn't exist" {
    docker() { 
        if [[ "$1 $2 $3" == 'container inspect vault' ]]; then
            return 1
        fi
        return 0
    }
    export -f docker
    run vault::is_installed
    [ "$status" -eq 1 ]
}

@test "vault functions can be loaded successfully" {
    # Test that vault functions are available after setup
    declare -f vault::is_installed
    declare -f vault::get_status  
    declare -f vault::is_running
}

@test "shared mocks are available" {
    # Test that shared mocks are working
    run log::info "test message"
    [[ "$output" =~ "[INFO]    test message" ]]
    
    run system::is_command "docker"
    [ "$status" -eq 0 ]
}

@test "vault configuration is loaded" {
    # Test that vault configuration variables are set
    [ -n "$VAULT_CONTAINER_NAME" ]
    [ -n "$VAULT_PORT" ]
    [ "$VAULT_PORT" = "8200" ]
}