#!/usr/bin/env bash
# Vault Resource Mock Implementation
# Provides realistic mock responses for HashiCorp Vault secrets management service

# Prevent duplicate loading
if [[ "${VAULT_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export VAULT_MOCK_LOADED="true"

#######################################
# Setup Vault mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::vault::setup() {
    local state="${1:-healthy}"
    
    # Configure Vault-specific environment
    export VAULT_PORT="${VAULT_PORT:-8200}"
    export VAULT_BASE_URL="http://localhost:${VAULT_PORT}"
    export VAULT_CONTAINER_NAME="${TEST_NAMESPACE}_vault"
    export VAULT_TOKEN="${VAULT_TOKEN:-test-root-token}"
    export VAULT_ADDR="$VAULT_BASE_URL"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "$state"
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::vault::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::vault::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::vault::setup_installing_endpoints
            ;;
        "stopped")
            mock::vault::setup_stopped_endpoints
            ;;
        *)
            echo "[VAULT_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[VAULT_MOCK] Vault mock configured with state: $state"
}

#######################################
# Setup healthy Vault endpoints
#######################################
mock::vault::setup_healthy_endpoints() {
    # Health endpoint
    mock::http::set_endpoint_response "$VAULT_BASE_URL/v1/sys/health" \
        '{
            "initialized": true,
            "sealed": false,
            "standby": false,
            "performance_standby": false,
            "replication_performance_mode": "disabled",
            "replication_dr_mode": "disabled",
            "server_time_utc": 1705318800,
            "version": "1.15.4",
            "cluster_name": "vault-cluster-test",
            "cluster_id": "abc-123-def"
        }'
    
    # Seal status endpoint
    mock::http::set_endpoint_response "$VAULT_BASE_URL/v1/sys/seal-status" \
        '{
            "type": "shamir",
            "initialized": true,
            "sealed": false,
            "t": 1,
            "n": 1,
            "progress": 0,
            "nonce": "",
            "version": "1.15.4",
            "build_date": "2023-12-04T17:45:28Z",
            "migration": false,
            "cluster_name": "vault-cluster-test",
            "cluster_id": "abc-123-def",
            "recovery_seal": false,
            "storage_type": "inmem"
        }'
    
    # Auth methods endpoint
    mock::http::set_endpoint_response "$VAULT_BASE_URL/v1/sys/auth" \
        '{
            "token/": {
                "type": "token",
                "description": "token based credentials",
                "accessor": "auth_token_abc123",
                "config": {
                    "default_lease_ttl": 0,
                    "max_lease_ttl": 0
                },
                "local": false,
                "seal_wrap": false
            },
            "userpass/": {
                "type": "userpass",
                "description": "Username and password auth",
                "accessor": "auth_userpass_def456",
                "config": {
                    "default_lease_ttl": 0,
                    "max_lease_ttl": 0
                }
            }
        }'
    
    # Secret engines endpoint
    mock::http::set_endpoint_response "$VAULT_BASE_URL/v1/sys/mounts" \
        '{
            "secret/": {
                "type": "kv",
                "description": "key/value secret storage",
                "accessor": "kv_abc123",
                "config": {
                    "default_lease_ttl": 0,
                    "max_lease_ttl": 0
                },
                "options": {
                    "version": "2"
                }
            },
            "database/": {
                "type": "database",
                "description": "Database secret engine",
                "accessor": "database_def456"
            }
        }'
    
    # KV v2 secret read
    mock::http::set_endpoint_response "$VAULT_BASE_URL/v1/secret/data/test" \
        '{
            "request_id": "req-123",
            "lease_id": "",
            "renewable": false,
            "lease_duration": 0,
            "data": {
                "data": {
                    "username": "testuser",
                    "password": "testpass123",
                    "api_key": "test-api-key-abc123"
                },
                "metadata": {
                    "created_time": "2024-01-15T10:00:00.000000Z",
                    "custom_metadata": null,
                    "deletion_time": "",
                    "destroyed": false,
                    "version": 1
                }
            }
        }'
    
    # KV v2 secret write response
    mock::http::set_endpoint_response "$VAULT_BASE_URL/v1/secret/data/new" \
        '{
            "request_id": "req-456",
            "data": {
                "created_time": "2024-01-15T12:00:00.000000Z",
                "custom_metadata": null,
                "deletion_time": "",
                "destroyed": false,
                "version": 1
            }
        }' \
        "POST"
    
    # Token lookup self
    mock::http::set_endpoint_response "$VAULT_BASE_URL/v1/auth/token/lookup-self" \
        '{
            "data": {
                "accessor": "token_accessor_123",
                "creation_time": 1705318800,
                "creation_ttl": 0,
                "display_name": "root",
                "entity_id": "",
                "expire_time": null,
                "explicit_max_ttl": 0,
                "id": "test-root-token",
                "meta": null,
                "num_uses": 0,
                "orphan": true,
                "path": "auth/token/root",
                "policies": ["root"],
                "ttl": 0,
                "type": "service"
            }
        }'
}

#######################################
# Setup unhealthy Vault endpoints
#######################################
mock::vault::setup_unhealthy_endpoints() {
    # Health endpoint returns sealed status
    mock::http::set_endpoint_response "$VAULT_BASE_URL/v1/sys/health" \
        '{
            "initialized": true,
            "sealed": true,
            "standby": false,
            "version": "1.15.4"
        }' \
        "GET" \
        "503"
    
    # Other endpoints return errors
    mock::http::set_endpoint_response "$VAULT_BASE_URL/v1/secret/data/test" \
        '{"errors":["Vault is sealed"]}' \
        "GET" \
        "503"
}

#######################################
# Setup installing Vault endpoints
#######################################
mock::vault::setup_installing_endpoints() {
    # Health endpoint returns uninitialized
    mock::http::set_endpoint_response "$VAULT_BASE_URL/v1/sys/health" \
        '{
            "initialized": false,
            "sealed": true,
            "standby": false,
            "version": "1.15.4"
        }' \
        "GET" \
        "501"
    
    # Init endpoint available
    mock::http::set_endpoint_response "$VAULT_BASE_URL/v1/sys/init" \
        '{"initialized":false}'
}

#######################################
# Setup stopped Vault endpoints
#######################################
mock::vault::setup_stopped_endpoints() {
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$VAULT_BASE_URL"
}

#######################################
# Mock vault CLI command
#######################################
vault() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "vault $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # Handle vault commands
    case "$1" in
        "status")
            echo "Key             Value"
            echo "---             -----"
            echo "Seal Type       shamir"
            echo "Initialized     true"
            echo "Sealed          false"
            echo "Total Shares    1"
            echo "Threshold       1"
            echo "Version         1.15.4"
            echo "Storage Type    inmem"
            echo "Cluster Name    vault-cluster-test"
            echo "HA Enabled      false"
            ;;
        "kv")
            case "$2" in
                "get")
                    local path="$3"
                    echo "====== Metadata ======"
                    echo "Key                Value"
                    echo "---                -----"
                    echo "created_time       2024-01-15T10:00:00.000000Z"
                    echo "version            1"
                    echo ""
                    echo "====== Data ======"
                    echo "Key         Value"
                    echo "---         -----"
                    echo "api_key     test-api-key-abc123"
                    echo "password    testpass123"
                    echo "username    testuser"
                    ;;
                "put")
                    echo "Success! Data written to: $3"
                    ;;
                "list")
                    echo "Keys"
                    echo "----"
                    echo "test"
                    echo "config/"
                    echo "credentials/"
                    ;;
            esac
            ;;
        "auth")
            case "$2" in
                "list")
                    echo "Path         Type        Accessor                Description"
                    echo "----         ----        --------                -----------"
                    echo "token/       token       auth_token_abc123       token based credentials"
                    echo "userpass/    userpass    auth_userpass_def456    Username and password auth"
                    ;;
            esac
            ;;
        *)
            echo "Usage: vault <command> [args]"
            ;;
    esac
    
    return 0
}

#######################################
# Mock Vault-specific operations
#######################################

# Mock secret creation
mock::vault::create_secret() {
    local path="$1"
    local data="$2"
    
    echo '{
        "request_id": "'$(uuidgen || echo "req-$(date +%s)")'",
        "data": {
            "created_time": "'$(date -u +%Y-%m-%dT%H:%M:%S.000000Z)'",
            "version": 1
        }
    }'
}

# Mock policy creation
mock::vault::create_policy() {
    local policy_name="$1"
    
    echo '{
        "request_id": "'$(uuidgen || echo "req-$(date +%s)")'",
        "data": {
            "name": "'$policy_name'",
            "rules": "path \"secret/*\" {\n  capabilities = [\"read\", \"list\"]\n}"
        }
    }'
}

# Mock token creation
mock::vault::create_token() {
    local ttl="${1:-24h}"
    
    echo '{
        "auth": {
            "client_token": "s.'$(echo -n "token-$(date +%s)" | md5sum | cut -c1-24)'",
            "accessor": "'$(uuidgen || echo "accessor-$(date +%s)")'",
            "policies": ["default"],
            "token_policies": ["default"],
            "lease_duration": 86400,
            "renewable": true
        }
    }'
}

# Mock database credential generation
mock::vault::generate_db_credentials() {
    local role="$1"
    
    echo '{
        "data": {
            "username": "v-'$role'-'$(date +%s)'",
            "password": "'$(echo -n "$role-$(date +%s)" | md5sum | cut -c1-32)'"
        },
        "lease_duration": 3600,
        "renewable": true
    }'
}

#######################################
# Export mock functions
#######################################
export -f vault
export -f mock::vault::setup
export -f mock::vault::setup_healthy_endpoints
export -f mock::vault::setup_unhealthy_endpoints
export -f mock::vault::setup_installing_endpoints
export -f mock::vault::setup_stopped_endpoints
export -f mock::vault::create_secret
export -f mock::vault::create_policy
export -f mock::vault::create_token
export -f mock::vault::generate_db_credentials

echo "[VAULT_MOCK] Vault mock implementation loaded"