#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Expensive setup operations run once per file
setup_file() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Path to the scripts under test
    SCRIPT_PATH="$BATS_TEST_DIRNAME/install.sh"
    VAULT_DIR="$BATS_TEST_DIRNAME/.."
    COMMON_PATH="$BATS_TEST_DIRNAME/common.sh"
    DOCKER_PATH="$BATS_TEST_DIRNAME/docker.sh"
    API_PATH="$BATS_TEST_DIRNAME/api.sh"
    
    # Source dependencies once per file
    RESOURCES_DIR="$VAULT_DIR/../.."
    HELPERS_DIR="$RESOURCES_DIR/../helpers"
    
    # Source utilities once
    source "$HELPERS_DIR/utils/log.sh"
    source "$HELPERS_DIR/utils/system.sh"
    source "$HELPERS_DIR/utils/flow.sh"
    source "$RESOURCES_DIR/common.sh"
    
    # Source required scripts once
    source "$COMMON_PATH"
    source "$DOCKER_PATH"
    source "$API_PATH"
    
    # Source messages configuration and initialize once
    source "$VAULT_DIR/config/messages.sh"
    vault::messages::init
    
    source "$SCRIPT_PATH"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    setup_standard_mocks
    
    # Load the functions we are testing (required for bats isolation)
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    source "${SCRIPT_DIR}/install.sh"
    
    # Set all required test environment variables (lightweight per-test)
    export VAULT_CONTAINER_NAME="vault-test"
    export VAULT_MODE="dev"
    export VAULT_DEV_ROOT_TOKEN_ID="test-token"
    export VAULT_DATA_DIR="/tmp/vault-test-data"
    export VAULT_CONFIG_DIR="/tmp/vault-test-config"
    export VAULT_LOGS_DIR="/tmp/vault-test-logs"
    export VAULT_TOKEN_FILE="/tmp/vault-test-token"
    export VAULT_UNSEAL_KEYS_FILE="/tmp/vault-test-keys"
    export VAULT_SECRET_ENGINE="secret"
    export VAULT_SECRET_VERSION="2"
    export VAULT_NAMESPACE_PREFIX="test"
    export VAULT_DEFAULT_PATHS="environments/dev resources"
    export VAULT_PORT="8200"
    export VAULT_BASE_URL="http://localhost:8200"
    export SCRIPT_DIR="$VAULT_DIR"
    
    # Mock functions for install tests (lightweight)
    resources::add_rollback_action() { 
        echo "Rollback action added: $1" >&2
        return 0
    }
    vault::export_config() { : ; }
    
    # Export mock functions
    export -f resources::add_rollback_action vault::export_config
}

# ============================================================================
# Install Tests
# ============================================================================

@test "vault::install skips if already installed" {
    vault::is_installed() { return 0; }
    run vault::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Vault is already installed" ]]
}

@test "vault::install checks prerequisites" {
    vault::is_installed() { return 1; }
    vault::check_prerequisites() { return 1; }
    run vault::install
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Prerequisites not met" ]]
}

@test "vault::install creates required directories" {
    vault::is_installed() { return 1; }
    vault::check_prerequisites() { return 0; }
    vault::create_directories() { 
        echo "Creating directories for Vault"
        return 0 
    }
    vault::start_container() { return 0; }
    vault::wait_for_healthy() { return 0; }
    vault::setup_initial_auth() { return 0; }
    vault::create_default_paths() { return 0; }
    
    run vault::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Creating directories" ]]
}

@test "vault::install handles directory creation failure" {
    vault::is_installed() { return 1; }
    vault::check_prerequisites() { return 0; }
    vault::create_directories() { return 1; }
    
    run vault::install
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Failed to create directories" ]]
}

@test "vault::install starts container successfully" {
    vault::is_installed() { return 1; }
    vault::check_prerequisites() { return 0; }
    vault::create_directories() { return 0; }
    vault::start_container() { 
        echo "Starting Vault container"
        return 0 
    }
    vault::wait_for_healthy() { return 0; }
    vault::setup_initial_auth() { return 0; }
    vault::create_default_paths() { return 0; }
    
    run vault::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Starting Vault container" ]]
}

@test "vault::install handles container start failure" {
    vault::is_installed() { return 1; }
    vault::check_prerequisites() { return 0; }
    vault::create_directories() { return 0; }
    vault::start_container() { return 1; }
    
    run vault::install
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Failed to start container" ]]
}

@test "vault::install waits for health check" {
    vault::is_installed() { return 1; }
    vault::check_prerequisites() { return 0; }
    vault::create_directories() { return 0; }
    vault::start_container() { return 0; }
    vault::wait_for_healthy() { 
        echo "Waiting for Vault to become healthy"
        return 0 
    }
    vault::setup_initial_auth() { return 0; }
    vault::create_default_paths() { return 0; }
    
    run vault::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Waiting for Vault" ]]
}

@test "vault::install handles health check timeout" {
    vault::is_installed() { return 1; }
    vault::check_prerequisites() { return 0; }
    vault::create_directories() { return 0; }
    vault::start_container() { return 0; }
    vault::wait_for_healthy() { return 1; }
    
    run vault::install
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Health check failed" ]]
}

@test "vault::install sets up initial authentication" {
    vault::is_installed() { return 1; }
    vault::check_prerequisites() { return 0; }
    vault::create_directories() { return 0; }
    vault::start_container() { return 0; }
    vault::wait_for_healthy() { return 0; }
    vault::setup_initial_auth() { 
        echo "Setting up initial authentication"
        return 0 
    }
    vault::create_default_paths() { return 0; }
    
    run vault::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Setting up initial authentication" ]]
}

@test "vault::install creates default paths" {
    vault::is_installed() { return 1; }
    vault::check_prerequisites() { return 0; }
    vault::create_directories() { return 0; }
    vault::start_container() { return 0; }
    vault::wait_for_healthy() { return 0; }
    vault::setup_initial_auth() { return 0; }
    vault::create_default_paths() { 
        echo "Creating default secret paths"
        return 0 
    }
    
    run vault::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Creating default secret paths" ]]
}

# ============================================================================
# Uninstall Tests
# ============================================================================

@test "vault::uninstall skips if not installed" {
    vault::is_installed() { return 1; }
    run vault::uninstall
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Vault is not installed" ]]
}

@test "vault::uninstall removes container and data" {
    vault::is_installed() { return 0; }
    vault::stop_container() { 
        echo "Stopping Vault container"
        return 0 
    }
    vault::remove_container() { 
        echo "Removing Vault container"
        return 0 
    }
    vault::cleanup_data() { 
        echo "Cleaning up Vault data"
        return 0 
    }
    
    run vault::uninstall
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Stopping Vault container" ]]
    [[ "$output" =~ "Removing Vault container" ]]
    [[ "$output" =~ "Cleaning up Vault data" ]]
}

@test "vault::uninstall handles confirmation" {
    vault::is_installed() { return 0; }
    vault::confirm_uninstall() { return 1; }
    
    run vault::uninstall
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Uninstall cancelled" ]]
}

# ============================================================================
# Prerequisites Tests
# ============================================================================

@test "vault::check_prerequisites succeeds when all requirements met" {
    system::is_command() {
        case "$1" in
            docker|jq) return 0 ;;
            *) return 1 ;;
        esac
    }
    docker() {
        if [[ "$1" == "version" ]]; then
            echo "Docker version 20.10.0"
            return 0
        fi
        return 1
    }
    export -f docker
    
    run vault::check_prerequisites
    [ "$status" -eq 0 ]
}

@test "vault::check_prerequisites fails when Docker missing" {
    system::is_command() {
        case "$1" in
            docker) return 1 ;;
            jq) return 0 ;;
            *) return 1 ;;
        esac
    }
    
    run vault::check_prerequisites
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Docker" ]]
}

@test "vault::check_prerequisites fails when jq missing" {
    system::is_command() {
        case "$1" in
            docker) return 0 ;;
            jq) return 1 ;;
            *) return 1 ;;
        esac
    }
    
    run vault::check_prerequisites
    [ "$status" -eq 1 ]
    [[ "$output" =~ "jq" ]]
}
