#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/common.sh"
VAULT_DIR="$BATS_TEST_DIRNAME/.."

# Source dependencies in correct order (matching manage.sh)
RESOURCES_DIR="$VAULT_DIR/../.."
HELPERS_DIR="$RESOURCES_DIR/../helpers"

# Source utilities first
. "$HELPERS_DIR/utils/log.sh"
. "$HELPERS_DIR/utils/system.sh"
. "$HELPERS_DIR/utils/ports.sh"
. "$HELPERS_DIR/utils/flow.sh"
. "$RESOURCES_DIR/port-registry.sh"

# Standard test setup string for bash -c tests
VAULT_COMMON_TEST_SETUP="
    VAULT_DIR='$VAULT_DIR'
    RESOURCES_DIR='$RESOURCES_DIR'  
    HELPERS_DIR='$HELPERS_DIR'
    SCRIPT_PATH='$SCRIPT_PATH'
    
    source \"\$HELPERS_DIR/utils/log.sh\"
    source \"\$HELPERS_DIR/utils/system.sh\"
    source \"\$HELPERS_DIR/utils/ports.sh\"
    source \"\$HELPERS_DIR/utils/flow.sh\"
    source \"\$RESOURCES_DIR/port-registry.sh\"
    source \"\$RESOURCES_DIR/common.sh\"
    
    # Set all required test environment variables
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
    export SCRIPT_DIR='\$VAULT_DIR'
    
    # Source vault configuration
    source \"\$VAULT_DIR/config/defaults.sh\"
    source \"\$VAULT_DIR/config/messages.sh\"
    vault::messages::init
    
    # Source the script under test
    source \"\$SCRIPT_PATH\"
    
    # Mock log functions
    log::success() { echo \"SUCCESS: \$*\"; }
    log::info() { echo \"INFO: \$*\"; }
    log::error() { echo \"ERROR: \$*\"; }
    log::warn() { echo \"WARN: \$*\"; }
    log::header() { echo \"HEADER: \$*\"; }
    
    # Mock utility functions
    resources::add_rollback_action() { 
        echo \"Rollback action added: \$1\" >&2
        return 0
    }
"

# Standard test setup string for no_defaults tests (avoids readonly variable conflicts)
VAULT_COMMON_TEST_SETUP_NO_DEFAULTS="
    VAULT_DIR='$VAULT_DIR'
    RESOURCES_DIR='$RESOURCES_DIR'  
    HELPERS_DIR='$HELPERS_DIR'
    SCRIPT_PATH='$SCRIPT_PATH'
    
    source \"\$HELPERS_DIR/utils/log.sh\"
    source \"\$HELPERS_DIR/utils/system.sh\"
    source \"\$RESOURCES_DIR/common.sh\"
    
    # Source the script under test only
    source \"\$SCRIPT_PATH\"
    
    # Mock log functions
    log::success() { echo \"SUCCESS: \$*\"; }
    log::info() { echo \"INFO: \$*\"; }
    log::error() { echo \"ERROR: \$*\"; }
    log::warn() { echo \"WARN: \$*\"; }
    log::header() { echo \"HEADER: \$*\"; }
"

# Helper function for proper sourcing in tests
setup_vault_test_env() {
    local script_dir="$VAULT_DIR"
    local resources_dir="$VAULT_DIR/../.."
    local helpers_dir="$resources_dir/../helpers"
    
    # Set all required environment variables before sourcing
    export VAULT_CONTAINER_NAME="vault"
    export VAULT_PORT="8200"
    export VAULT_MODE="dev"
    export VAULT_BASE_URL="http://localhost:8200"
    export VAULT_DATA_DIR="/tmp/vault-test-data"
    export VAULT_CONFIG_DIR="/tmp/vault-test-config"
    export VAULT_LOGS_DIR="/tmp/vault-test-logs"
    export VAULT_DEV_ROOT_TOKEN_ID="myroot"
    export VAULT_SECRET_ENGINE="secret"
    export VAULT_NAMESPACE_PREFIX="vrooli"
    export VAULT_TOKEN_FILE="/tmp/vault-test-token"
    export VAULT_UNSEAL_KEYS_FILE="/tmp/vault-test-keys"
    export SCRIPT_DIR="$script_dir"
    
    # Source utilities first
    source "$helpers_dir/utils/log.sh"
    source "$helpers_dir/utils/system.sh"
    source "$helpers_dir/utils/ports.sh"
    source "$helpers_dir/utils/flow.sh"
    source "$resources_dir/port-registry.sh"
    
    # Source common.sh (provides resources functions)
    source "$resources_dir/common.sh"
    
    # Source config (depends on resources functions)
    source "$script_dir/config/defaults.sh"
    
    # Source messages configuration and initialize
    source "$script_dir/config/messages.sh"
    vault::messages::init
    
    # Source the script under test
    source "$SCRIPT_PATH"
    
    # Mock log functions
    log::success() { echo "SUCCESS: $*"; }
    log::info() { echo "INFO: $*"; }
    log::error() { echo "ERROR: $*"; }
    log::warn() { echo "WARN: $*"; }
    log::header() { echo "HEADER: $*"; }
    
    # Mock functions for tests
    resources::add_rollback_action() { 
        echo "Rollback action added: $1" >&2
        return 0
    }
}

# Helper function for directory tests (avoids readonly variable conflicts)
setup_vault_test_env_no_defaults() {
    local script_dir="$VAULT_DIR"
    local resources_dir="$VAULT_DIR/../.."
    local helpers_dir="$resources_dir/../helpers"
    
    # Source only what's needed without defaults
    source "$helpers_dir/utils/log.sh"
    source "$helpers_dir/utils/system.sh"
    source "$resources_dir/common.sh"
    source "$SCRIPT_PATH"
}

# ============================================================================
# Container State Tests
# ============================================================================

@test "vault::is_installed returns 0 when container exists" {
    setup_vault_test_env
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
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        docker() { 
            if [[ \"\$1 \$2 \$3\" == 'container inspect vault' ]]; then
                return 1
            fi
            return 0
        }
        export -f docker
        vault::is_installed
    "
    [ "$status" -eq 1 ]
}

@test "vault::is_running returns 0 when container is running" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        docker() { 
            case \"\$*\" in
                'container inspect vault')
                    return 0
                    ;;
                \"container inspect --format={{.State.Status}} vault\")
                    echo 'running'
                    return 0
                    ;;
            esac
            return 1
        }
        export -f docker
        vault::is_running
    "
    [ "$status" -eq 0 ]
}

@test "vault::is_running returns 1 when container is stopped" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        docker() { 
            case \"\$*\" in
                'container inspect vault')
                    return 0
                    ;;
                \"container inspect --format={{.State.Status}} vault\")
                    echo 'exited'
                    return 0
                    ;;
            esac
            return 1
        }
        export -f docker
        vault::is_running
    "
    [ "$status" -eq 1 ]
}

@test "vault::is_healthy returns 0 when API responds" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        vault::is_running() { return 0; }
        curl() {
            if [[ \"\$*\" =~ 'sys/health' ]]; then
                return 0
            fi
            return 1
        }
        export -f curl
        vault::is_healthy
    "
    [ "$status" -eq 0 ]
}

@test "vault::is_healthy returns 1 when not running" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        vault::is_running() { return 1; }
        vault::is_healthy
    "
    [ "$status" -eq 1 ]
}

@test "vault::is_initialized returns 0 when initialized" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        vault::is_healthy() { return 0; }
        curl() {
            if [[ \"\$*\" =~ 'sys/init' ]]; then
                echo '{\"initialized\": true}'
                return 0
            fi
            return 1
        }
        export -f curl
        jq() {
            echo 'true'
        }
        export -f jq
        vault::is_initialized
    "
    [ "$status" -eq 0 ]
}

@test "vault::is_sealed returns 0 when sealed" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        vault::is_healthy() { return 0; }
        curl() {
            if [[ \"\$*\" =~ 'seal-status' ]]; then
                echo '{\"sealed\": true}'
                return 0
            fi
            return 1
        }
        export -f curl
        jq() {
            echo 'true'
        }
        export -f jq
        vault::is_sealed
    "
    [ "$status" -eq 0 ]
}

# ============================================================================
# Status Tests
# ============================================================================

@test "vault::get_status returns not_installed when container doesn't exist" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        vault::is_installed() { return 1; }
        vault::get_status
    "
    [ "$status" -eq 0 ]
    [ "$output" = "not_installed" ]
}

@test "vault::get_status returns stopped when container exists but not running" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        vault::is_installed() { return 0; }
        vault::is_running() { return 1; }
        vault::get_status
    "
    [ "$status" -eq 0 ]
    [ "$output" = "stopped" ]
}

@test "vault::get_status returns unhealthy when running but not healthy" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        vault::is_installed() { return 0; }
        vault::is_running() { return 0; }
        vault::is_healthy() { return 1; }
        vault::get_status
    "
    [ "$status" -eq 0 ]
    [ "$output" = "unhealthy" ]
}

@test "vault::get_status returns sealed when healthy but sealed" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        vault::is_installed() { return 0; }
        vault::is_running() { return 0; }
        vault::is_healthy() { return 0; }
        vault::is_sealed() { return 0; }
        vault::get_status
    "
    [ "$status" -eq 0 ]
    [ "$output" = "sealed" ]
}

@test "vault::get_status returns healthy when unsealed" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        vault::is_installed() { return 0; }
        vault::is_running() { return 0; }
        vault::is_healthy() { return 0; }
        vault::is_sealed() { return 1; }
        vault::get_status
    "
    [ "$status" -eq 0 ]
    [ "$output" = "healthy" ]
}

# ============================================================================
# Wait Functions Tests
# ============================================================================

@test "vault::wait_for_health returns 0 when healthy immediately" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        vault::is_healthy() { return 0; }
        vault::wait_for_health 5 1
    "
    [ "$status" -eq 0 ]
}

@test "vault::wait_for_health returns 1 on timeout" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        vault::is_healthy() { return 1; }
        sleep() { : ; }  # Mock sleep to speed up test
        vault::wait_for_health 2 1
    "
    [ "$status" -eq 1 ]
}

# ============================================================================
# Directory Management Tests
# ============================================================================

@test "vault::ensure_directories creates missing directories" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP_NO_DEFAULTS
        export VAULT_DATA_DIR='/tmp/vault-test-data'
        export VAULT_CONFIG_DIR='/tmp/vault-test-config'
        export VAULT_LOGS_DIR='/tmp/vault-test-logs'
        rm -rf /tmp/vault-test-*
        vault::ensure_directories
        [[ -d \$VAULT_DATA_DIR ]] && [[ -d \$VAULT_CONFIG_DIR ]] && [[ -d \$VAULT_LOGS_DIR ]]
    "
    [ "$status" -eq 0 ]
    # Cleanup
    rm -rf /tmp/vault-test-*
}

# ============================================================================
# Configuration Tests
# ============================================================================

@test "vault::create_config creates dev config" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP_NO_DEFAULTS
        export VAULT_CONFIG_DIR='/tmp/vault-test-config'
        export VAULT_MODE='dev'
        export VAULT_PORT='8200'
        export VAULT_TLS_DISABLE='1'
        export VAULT_BASE_URL='http://localhost:8200'
        export VAULT_DEV_ROOT_TOKEN_ID='myroot'
        mkdir -p \$VAULT_CONFIG_DIR
        vault::ensure_directories() { : ; }
        vault::create_config dev
        grep -q 'inmem' \$VAULT_CONFIG_DIR/vault.hcl
    "
    [ "$status" -eq 0 ]
    # Cleanup
    rm -rf /tmp/vault-test-*
}

@test "vault::create_config creates prod config" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP_NO_DEFAULTS
        export VAULT_CONFIG_DIR='/tmp/vault-test-config'
        export VAULT_MODE='prod'
        export VAULT_PORT='8200'
        export VAULT_TLS_DISABLE='0'
        export VAULT_BASE_URL='http://localhost:8200'
        mkdir -p \$VAULT_CONFIG_DIR
        vault::ensure_directories() { : ; }
        vault::create_config prod
        grep -q 'storage \"file\"' \$VAULT_CONFIG_DIR/vault.hcl
    "
    [ "$status" -eq 0 ]
    # Cleanup
    rm -rf /tmp/vault-test-*
}

# ============================================================================
# Token Management Tests
# ============================================================================

@test "vault::get_root_token returns dev token in dev mode" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        export VAULT_MODE='dev'
        export VAULT_DEV_ROOT_TOKEN_ID='test-token'
        vault::get_root_token
    "
    [ "$status" -eq 0 ]
    [ "$output" = "test-token" ]
}

@test "vault::get_root_token reads from file in prod mode" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP_NO_DEFAULTS
        export VAULT_MODE='prod'
        export VAULT_TOKEN_FILE='/tmp/vault-test-token'
        echo 'prod-token' > \$VAULT_TOKEN_FILE
        vault::get_root_token
    "
    [ "$status" -eq 0 ]
    [ "$output" = "prod-token" ]
    # Cleanup
    rm -f /tmp/vault-test-token
}

# ============================================================================
# Secret Path Validation Tests
# ============================================================================

@test "vault::validate_secret_path rejects empty path" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        vault::validate_secret_path ''
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "empty" ]]
}

@test "vault::validate_secret_path rejects path with spaces" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        vault::validate_secret_path 'path with spaces'
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "spaces" ]]
}

@test "vault::validate_secret_path rejects path starting with /" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        vault::validate_secret_path '/absolute/path'
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "start with" ]]
}

@test "vault::validate_secret_path accepts valid path" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        vault::validate_secret_path 'environments/dev/api-key'
    "
    [ "$status" -eq 0 ]
}

@test "vault::construct_secret_path adds namespace and engine" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        export VAULT_NAMESPACE_PREFIX='vrooli'
        export VAULT_SECRET_ENGINE='secret'
        vault::construct_secret_path 'test/key'
    "
    [ "$status" -eq 0 ]
    [ "$output" = "secret/data/vrooli/test/key" ]
}

# ============================================================================
# API Request Tests
# ============================================================================

@test "vault::api_request includes auth token" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        vault::get_root_token() { echo 'test-token'; }
        curl() {
            echo \"Args: \$@\" | grep -q 'X-Vault-Token: test-token' && echo 'Token included'
        }
        export -f curl
        vault::api_request GET /v1/sys/health
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Token included" ]]
}

# ============================================================================
# Random String Generation Tests
# ============================================================================

@test "vault::generate_random_string generates correct length" {
    run bash -c "
$VAULT_COMMON_TEST_SETUP
        openssl() {
            echo 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/'
        }
        export -f openssl
        result=\$(vault::generate_random_string 16)
        echo \${#result}
    "
    [ "$status" -eq 0 ]
    [ "$output" = "16" ]
}