#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/api.sh"
VAULT_DIR="$BATS_TEST_DIRNAME/.."
COMMON_PATH="$BATS_TEST_DIRNAME/common.sh"

# Source dependencies
RESOURCES_DIR="$VAULT_DIR/../.."
HELPERS_DIR="$RESOURCES_DIR/../helpers"

# Standard test setup - used in all tests
VAULT_TEST_SETUP="
        # Source dependencies
        VAULT_DIR='$VAULT_DIR'
        RESOURCES_DIR='$RESOURCES_DIR'  
        HELPERS_DIR='$HELPERS_DIR'
        SCRIPT_PATH='$SCRIPT_PATH'
        COMMON_PATH='$COMMON_PATH'
        
        source \"\$HELPERS_DIR/utils/log.sh\"
        source \"\$HELPERS_DIR/utils/system.sh\"
        source \"\$RESOURCES_DIR/common.sh\"
        
        # Set all required test environment variables
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
        export SCRIPT_DIR='\$VAULT_DIR'
        
        source \"\$COMMON_PATH\"
        source \"\$SCRIPT_PATH\"
        
        # Source messages configuration and initialize
        source \"\$VAULT_DIR/config/messages.sh\"
        vault::messages::init
        
        # Mock log functions
        log::success() { echo \"SUCCESS: \$*\"; }
        log::info() { echo \"INFO: \$*\"; }
        log::error() { echo \"ERROR: \$*\"; }
        log::warn() { echo \"WARN: \$*\"; }
        log::header() { echo \"HEADER: \$*\"; }
        
        # Mock vault functions
        vault::is_healthy() { return 0; }
        vault::is_sealed() { return 1; }
        vault::api_request() {
            case \"\$1\" in
                DELETE|POST|PUT|GET)
                    echo \"API Request: Method=\$1 Endpoint=\$2\"
                    if [[ -n \"\$3\" ]]; then
                        echo \"Body: \$3\"
                    fi
                    return 0
                    ;;
                *)
                    return 1
                    ;;
            esac
        }
        jq() {
            case \"\$*\" in
                *'empty'*) return 0 ;;
                *'.data.data'*) echo '{\"value\":\"secret123\"}' ;;
                *'@tsv'*) echo -e 'key1\tvalue1\nkey2\tvalue2' ;;
                *) echo '{\"result\":\"mocked\"}';;
            esac
        }
        export -f jq
"

# ============================================================================
# Put Secret Tests
# ============================================================================

@test "vault::put_secret stores secret successfully" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::construct_secret_path() { echo 'secret/data/test/environments/dev/api-key'; }
        
        vault::put_secret 'environments/dev/api-key' 'secret123'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Secret stored successfully" ]]
}

@test "vault::put_secret validates required arguments" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::put_secret '' 'value'
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "vault::put_secret fails when Vault is not healthy" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::is_healthy() { return 1; }
        vault::put_secret 'test/key' 'value'
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not ready" ]]
}

@test "vault::put_secret fails when Vault is sealed" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::is_sealed() { return 0; }
        vault::put_secret 'test/key' 'value'
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "sealed" ]]
}

@test "vault::put_secret supports custom key name" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::construct_secret_path() { echo 'secret/data/test/path'; }
        vault::put_secret 'test/path' 'value123' 'custom_key'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Secret stored successfully" ]]
}

# ============================================================================
# Get Secret Tests
# ============================================================================

@test "vault::get_secret retrieves secret successfully" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::construct_secret_path() { echo 'secret/data/test/path'; }
        vault::api_request() {
            echo '{\"data\": {\"data\": {\"value\": \"secret123\"}}}'
            return 0
        }
        jq() {
            if [[ \"\$*\" =~ '.data.data.\"value\"' ]]; then
                echo 'secret123'
            else
                echo '{\"data\": {\"data\": {\"value\": \"secret123\"}}}'
            fi
        }
        export -f jq
        vault::get_secret 'test/path'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "secret123" ]]
}

@test "vault::get_secret validates required path" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::get_secret ''
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "vault::get_secret supports json format" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::construct_secret_path() { echo 'secret/data/test/path'; }
        vault::api_request() {
            echo '{\"data\": {\"data\": {\"key1\": \"value1\", \"key2\": \"value2\"}}}'
            return 0
        }
        jq() {
            if [[ \"\$*\" =~ '.data.data' ]] && [[ ! \"\$*\" =~ '\"' ]]; then
                echo '{\"key1\": \"value1\", \"key2\": \"value2\"}'
            else
                echo '{\"data\": {\"data\": {\"key1\": \"value1\", \"key2\": \"value2\"}}}'
            fi
        }
        export -f jq
        vault::get_secret 'test/path' 'value' 'json'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "key1" ]]
    [[ "$output" =~ "value1" ]]
}

@test "vault::get_secret fails for invalid format" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::get_secret 'test/path' 'value' 'invalid'
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Invalid format: invalid" ]]
}

# ============================================================================
# List Secrets Tests
# ============================================================================

@test "vault::list_secrets lists secrets successfully" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::api_request() {
            echo '{\"data\": {\"keys\": [\"key1\", \"key2\", \"key3\"]}}'
            return 0
        }
        jq() {
            if [[ \"\$*\" =~ '.data.keys[]' ]]; then
                echo -e 'key1\nkey2\nkey3'
            else
                echo '[\"key1\", \"key2\", \"key3\"]'
            fi
        }
        export -f jq
        vault::list_secrets 'environments/'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "key1" ]]
    [[ "$output" =~ "key2" ]]
    [[ "$output" =~ "key3" ]]
}

@test "vault::list_secrets supports json format" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::api_request() {
            echo '{\"data\": {\"keys\": [\"key1\", \"key2\"]}}'
            return 0
        }
        jq() {
            echo '[\"key1\", \"key2\"]'
        }
        export -f jq
        vault::list_secrets 'test/' 'json'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "[" ]]
    [[ "$output" =~ "key1" ]]
}

# ============================================================================
# Delete Secret Tests
# ============================================================================

@test "vault::delete_secret deletes secret successfully" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::construct_secret_path() { echo 'secret/data/test/path'; }
        vault::delete_secret 'test/path'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Secret deleted successfully" ]]
}

@test "vault::delete_secret validates required path" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::delete_secret ''
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

# ============================================================================
# Secret Exists Tests
# ============================================================================

@test "vault::secret_exists returns 0 when secret exists" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::get_secret() { echo 'value'; return 0; }
        vault::secret_exists 'test/path'
    "
    [ "$status" -eq 0 ]
}

@test "vault::secret_exists returns 1 when secret doesn't exist" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::get_secret() { return 1; }
        vault::secret_exists 'test/path'
    "
    [ "$status" -eq 1 ]
}

# ============================================================================
# Metadata Tests
# ============================================================================

@test "vault::get_secret_metadata retrieves metadata" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::api_request() {
            echo '{\"data\": {\"created_time\": \"2024-01-01\", \"version\": 1}}'
            return 0
        }
        jq() {
            echo '{\"created_time\": \"2024-01-01\", \"version\": 1}'
        }
        export -f jq
        vault::get_secret_metadata 'test/path'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "created_time" ]]
    [[ "$output" =~ "version" ]]
}

# ============================================================================
# Bulk Operations Tests
# ============================================================================

@test "vault::bulk_put_secrets stores multiple secrets" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::put_secret() {
            echo \"Storing: \$1 = \$2\"
            return 0
        }
        jq() {
            if [[ \"\$*\" =~ 'to_entries' ]]; then
                echo -e 'key1\tvalue1\nkey2\tvalue2'
                return 0
            elif [[ \"\$*\" =~ 'empty' ]]; then
                return 0
            else
                echo '{\"result\":\"mocked\"}'
                return 0
            fi
        }
        export -f jq
        vault::bulk_put_secrets '{\"key1\": \"value1\", \"key2\": \"value2\"}' 'base'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Storing: base/key1 = value1" ]]
    [[ "$output" =~ "Storing: base/key2 = value2" ]]
}

@test "vault::bulk_put_secrets reads from file" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::put_secret() { return 0; }
        jq() {
            if [[ \"\$*\" =~ 'empty' ]]; then
                return 0
            elif [[ \"\$*\" =~ 'to_entries' ]]; then
                echo -e 'key1\tvalue1'
                return 0
            else
                echo '{\"result\":\"mocked\"}'
                return 0
            fi
        }
        export -f jq
        echo '{\"key1\": \"value1\"}' > /tmp/test-secrets.json
        vault::bulk_put_secrets /tmp/test-secrets.json
        rm -f /tmp/test-secrets.json
    "
    [ "$status" -eq 0 ]
}

# ============================================================================
# Export Secrets Tests
# ============================================================================

@test "vault::export_secrets exports secrets to json" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::list_secrets() { echo -e 'secret1\nsecret2'; }
        vault::get_secret() {
            case \"\$1\" in
                'base/secret1')
                    echo '{\"value\": \"data1\"}'
                    ;;
                'base/secret2')
                    echo '{\"value\": \"data2\"}'
                    ;;
            esac
            return 0
        }
        jq() {
            echo '{\"secret1\": {\"value\": \"data1\"}, \"secret2\": {\"value\": \"data2\"}}'
        }
        export -f jq
        vault::export_secrets 'base'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "secret1" ]]
    [[ "$output" =~ "data1" ]]
}

# ============================================================================
# API Key Generation Tests
# ============================================================================

@test "vault::generate_api_key generates and stores key" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::generate_random_string() { echo 'random-api-key-123'; }
        vault::put_secret() {
            echo \"Storing API key at: \$1\"
            return 0
        }
        vault::generate_api_key 'api/keys/service1'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Storing API key at: api/keys/service1" ]]
}

# ============================================================================
# Secret Rotation Tests
# ============================================================================

@test "vault::rotate_secret rotates with random type" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::generate_random_string() { echo 'new-random-value'; }
        vault::put_secret() {
            echo \"Rotated to: \$2\"
            return 0
        }
        vault::rotate_secret 'test/key' 'random'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Rotated to: new-random-value" ]]
}

@test "vault::rotate_secret rotates with api-key type" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::generate_random_string() {
            case \"\$1\" in
                8) echo 'PREFIX12' ;;
                24) echo 'SUFFIX123456789012345678' ;;
            esac
        }
        vault::put_secret() {
            echo \"API Key: \$2\"
            return 0
        }
        vault::rotate_secret 'test/key' 'api-key'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "API Key: PREFIX12_SUFFIX123456789012345678" ]]
}

@test "vault::rotate_secret validates rotation type" {
    run bash -c "
$VAULT_TEST_SETUP
        vault::rotate_secret 'test/key' 'invalid-type'
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown rotation type" ]]
}