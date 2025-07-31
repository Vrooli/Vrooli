#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Expensive setup operations run once per file
setup_file() {
    # Path to the script under test
    SCRIPT_PATH="$BATS_TEST_DIRNAME/api.sh"
    VAULT_DIR="$BATS_TEST_DIRNAME/.."
    COMMON_PATH="$BATS_TEST_DIRNAME/common.sh"
    
    # Source dependencies once
    RESOURCES_DIR="$VAULT_DIR/../.."
    HELPERS_DIR="$RESOURCES_DIR/../helpers"
    
    # Source utilities once
    source "$HELPERS_DIR/utils/log.sh"
    source "$HELPERS_DIR/utils/system.sh"
    source "$RESOURCES_DIR/common.sh"
    
    # Source the scripts under test once
    source "$COMMON_PATH"
    source "$SCRIPT_PATH"
    
    # Source messages configuration once
    source "$VAULT_DIR/config/messages.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_PATH="$SCRIPT_PATH"
    export SETUP_FILE_VAULT_DIR="$VAULT_DIR"
    export SETUP_FILE_COMMON_PATH="$COMMON_PATH"
}

# Lightweight per-test setup
setup() {
    # Basic mock functions (lightweight)
    # Mock resources functions to avoid hang
    declare -A DEFAULT_PORTS=(
        ["ollama"]="11434"
        ["agent-s2"]="4113"
        ["browserless"]="3000"
        ["unstructured-io"]="8000"
        ["n8n"]="5678"
        ["node-red"]="1880"
        ["huginn"]="3000"
        ["windmill"]="8000"
        ["judge0"]="2358"
        ["searxng"]="8080"
        ["qdrant"]="6333"
        ["questdb"]="9000"
        ["vault"]="8200"
    )
    resources::get_default_port() { echo "${DEFAULT_PORTS[$1]:-8080}"; }
    export -f resources::get_default_port
    
    mock::network::set_online() { return 0; }
    setup_standard_mocks() { 
    
    # Load the functions we are testing (required for bats isolation)
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    source "${SCRIPT_DIR}/api.sh"
        export FORCE="${FORCE:-no}"
        export YES="${YES:-no}"
        export OUTPUT_FORMAT="${OUTPUT_FORMAT:-text}"
        export QUIET="${QUIET:-no}"
        mock::network::set_online
    }
    
    # Setup mocks
    setup_standard_mocks
    
    # Use paths from setup_file
    SCRIPT_PATH="${SETUP_FILE_SCRIPT_PATH}"
    VAULT_DIR="${SETUP_FILE_VAULT_DIR}"
    COMMON_PATH="${SETUP_FILE_COMMON_PATH}"
    
    # Set all required test environment variables (lightweight per-test)
    export VAULT_CONTAINER_NAME='vault-test'
    export VAULT_NAMESPACE_PREFIX='test'
    export VAULT_SECRET_ENGINE='secret'
    export VAULT_SECRET_VERSION='2'
    export VAULT_BASE_URL='http://localhost:8200'
    export VAULT_PORT='8200'
    export VAULT_MODE='dev'
    export VAULT_DEV_ROOT_TOKEN_ID='test-token'
    export VAULT_DATA_DIR='/tmp/vault-test-data'
    export VAULT_CONFIG_DIR='/tmp/vault-test-config'
    export VAULT_LOGS_DIR='/tmp/vault-test-logs'
    export VAULT_TOKEN_FILE='/tmp/vault-test-token'
    export VAULT_UNSEAL_KEYS_FILE='/tmp/vault-test-keys'
    export SCRIPT_DIR="$VAULT_DIR"
    
    # Re-source only essential functions to ensure availability
    source "$COMMON_PATH"
    source "$SCRIPT_PATH"
    vault::messages::init
    
    # Mock vault functions for API tests (lightweight)
    vault::is_healthy() { return 0; }
    vault::is_sealed() { return 1; }
    vault::api_request() {
        case "$1" in
            DELETE|POST|PUT|GET)
                echo "API Request: Method=$1 Endpoint=$2"
                if [[ -n "$3" ]]; then
                    echo "Body: $3"
                fi
                return 0
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    # Override jq for API response testing
    jq() {
        case "$*" in
            *'empty'*) return 0 ;;
            *'.data.data'*) echo '{"value":"secret123"}' ;;
            *'@tsv'*) echo -e 'key1\tvalue1\nkey2\tvalue2' ;;
            *) echo '{"result":"mocked"}';;
        esac
    }
    
    # Export all mock functions
    export -f jq vault::is_healthy vault::is_sealed vault::api_request
}

# ============================================================================
# Put Secret Tests
# ============================================================================

@test "vault::put_secret stores secret successfully" {
    vault::construct_secret_path() { echo 'secret/data/test/environments/dev/api-key'; }
    
    run vault::put_secret 'environments/dev/api-key' 'secret123'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Secret stored successfully" ]]
}

@test "vault::put_secret validates required arguments" {
    run vault::put_secret '' 'value'
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "vault::put_secret fails when Vault is not healthy" {
    vault::is_healthy() { return 1; }
    run vault::put_secret 'test/key' 'value'
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not ready" ]]
}

@test "vault::put_secret fails when Vault is sealed" {
    vault::is_sealed() { return 0; }
    run vault::put_secret 'test/key' 'value'
    [ "$status" -eq 1 ]
    [[ "$output" =~ "sealed" ]]
}

@test "vault::put_secret supports custom key name" {
    vault::construct_secret_path() { echo 'secret/data/test/path'; }
    run vault::put_secret 'test/path' 'value123' 'custom_key'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Secret stored successfully" ]]
}

# ============================================================================
# Get Secret Tests
# ============================================================================

@test "vault::get_secret retrieves secret successfully" {
    vault::construct_secret_path() { echo 'secret/data/test/path'; }
    vault::api_request() {
        echo '{"data": {"data": {"value": "secret123"}}}'
        return 0
    }
    
    run vault::get_secret 'test/path'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "secret123" ]]
}

@test "vault::get_secret validates required arguments" {
    run vault::get_secret ''
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "vault::get_secret fails when Vault is not healthy" {
    vault::is_healthy() { return 1; }
    run vault::get_secret 'test/key'
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not ready" ]]
}

@test "vault::get_secret fails when Vault is sealed" {
    vault::is_sealed() { return 0; }
    run vault::get_secret 'test/key'
    [ "$status" -eq 1 ]
    [[ "$output" =~ "sealed" ]]
}

@test "vault::get_secret handles empty secret" {
    vault::construct_secret_path() { echo 'secret/data/test/path'; }
    vault::api_request() {
        echo '{"data": {"data": null}}'
        return 0
    }
    
    run vault::get_secret 'test/path'
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not found" ]]
}

# ============================================================================
# List Secrets Tests  
# ============================================================================

@test "vault::list_secrets shows available secrets" {
    vault::construct_secret_path() { echo 'secret/metadata/test/'; }
    vault::api_request() {
        echo '{"data": {"keys": ["key1", "key2", "subdir/"]}}'
        return 0
    }
    
    run vault::list_secrets 'test/'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "key1" ]]
    [[ "$output" =~ "key2" ]]
}

@test "vault::list_secrets handles empty directory" {
    vault::construct_secret_path() { echo 'secret/metadata/empty/'; }
    vault::api_request() {
        echo '{"data": {"keys": []}}'
        return 0
    }
    
    run vault::list_secrets 'empty/'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No secrets found" ]]
}

# ============================================================================
# Delete Secret Tests
# ============================================================================

@test "vault::delete_secret removes secret successfully" {
    vault::construct_secret_path() { echo 'secret/data/test/path'; }
    
    run vault::delete_secret 'test/path'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Secret deleted successfully" ]]
}

@test "vault::delete_secret validates required arguments" {
    run vault::delete_secret ''
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "vault::delete_secret fails when Vault is not healthy" {
    vault::is_healthy() { return 1; }
    run vault::delete_secret 'test/key'
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not ready" ]]
}

@test "vault::delete_secret fails when Vault is sealed" {
    vault::is_sealed() { return 0; }
    run vault::delete_secret 'test/key'
    [ "$status" -eq 1 ]
    [[ "$output" =~ "sealed" ]]
}
