#!/usr/bin/env bash
# Vault Mock Implementation
# 
# Provides a comprehensive mock for HashiCorp Vault operations including:
# - vault CLI command interception
# - Vault server state simulation
# - Secrets management (KV v1/v2, database, etc.)
# - Authentication methods simulation
# - Policy management
# - Token operations
#
# This mock follows the same standards as other mocks with:
# - Comprehensive state management
# - File-based persistence for BATS compatibility
# - Integration with centralized logging
# - Test helper functions

# Prevent duplicate loading
[[ -n "${VAULT_MOCK_LOADED:-}" ]] && return 0
declare -g VAULT_MOCK_LOADED=1

# Load dependencies
MOCK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[[ -f "$MOCK_DIR/logs.sh" ]] && source "$MOCK_DIR/logs.sh"

# Global configuration
declare -g VAULT_MOCK_STATE_DIR="${VAULT_MOCK_STATE_DIR:-/tmp/vault-mock-state}"
declare -g VAULT_MOCK_DEBUG="${VAULT_MOCK_DEBUG:-}"

# Global state arrays
declare -gA VAULT_MOCK_SECRETS=()          # path -> secret_data
declare -gA VAULT_MOCK_POLICIES=()         # name -> policy_content
declare -gA VAULT_MOCK_TOKENS=()           # token -> token_info
declare -gA VAULT_MOCK_AUTH_METHODS=()     # path -> method_config
declare -gA VAULT_MOCK_SECRET_ENGINES=()   # path -> engine_config
declare -gA VAULT_MOCK_CONFIG=(            # Vault configuration
    [addr]="http://127.0.0.1:8200"
    [token]="root-token"
    [initialized]="true"
    [sealed]="false"
    [version]="1.15.4"
    [cluster_name]="vault-cluster-mock"
    [cluster_id]="mock-cluster-123"
    [standby]="false"
    [error_mode]=""
    [seal_type]="shamir"
    [seal_threshold]="1"
    [seal_shares]="1"
    [storage_type]="inmem"
)

# Initialize state directory
mkdir -p "$VAULT_MOCK_STATE_DIR"

# State persistence functions
mock::vault::save_state() {
    local state_file="$VAULT_MOCK_STATE_DIR/vault-state.sh"
    {
        echo "# Vault mock state - $(date)"
        
        # Save arrays using declare -p for proper restoration
        declare -p VAULT_MOCK_CONFIG 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA VAULT_MOCK_CONFIG=()"
        declare -p VAULT_MOCK_SECRETS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA VAULT_MOCK_SECRETS=()"
        declare -p VAULT_MOCK_POLICIES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA VAULT_MOCK_POLICIES=()"
        declare -p VAULT_MOCK_TOKENS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA VAULT_MOCK_TOKENS=()"
        declare -p VAULT_MOCK_AUTH_METHODS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA VAULT_MOCK_AUTH_METHODS=()"
        declare -p VAULT_MOCK_SECRET_ENGINES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA VAULT_MOCK_SECRET_ENGINES=()"
    } > "$state_file"
    
    mock::log_state "vault" "Saved Vault state to $state_file"
}

mock::vault::load_state() {
    local state_file="$VAULT_MOCK_STATE_DIR/vault-state.sh"
    if [[ -f "$state_file" ]]; then
        source "$state_file"
        mock::log_state "vault" "Loaded Vault state from $state_file"
    fi
}

# Automatically load state when sourced
mock::vault::load_state

# Initialize default state
mock::vault::init_default_state() {
    # Default auth methods
    VAULT_MOCK_AUTH_METHODS["token/"]='{"type":"token","description":"token based credentials","accessor":"auth_token_abc123"}'
    VAULT_MOCK_AUTH_METHODS["userpass/"]='{"type":"userpass","description":"Username and password auth","accessor":"auth_userpass_def456"}'
    
    # Default secret engines
    VAULT_MOCK_SECRET_ENGINES["secret/"]='{"type":"kv","description":"key/value secret storage","accessor":"kv_abc123","options":{"version":"2"}}'
    VAULT_MOCK_SECRET_ENGINES["database/"]='{"type":"database","description":"Database secret engine","accessor":"database_def456"}'
    
    # Default policies
    VAULT_MOCK_POLICIES["root"]='path "*" { capabilities = ["create", "read", "update", "delete", "list", "sudo"] }'
    VAULT_MOCK_POLICIES["default"]='path "auth/token/lookup-self" { capabilities = ["read"] } path "auth/token/renew-self" { capabilities = ["update"] } path "auth/token/revoke-self" { capabilities = ["update"] }'
    
    # Default root token
    VAULT_MOCK_TOKENS["${VAULT_MOCK_CONFIG[token]}"]='{"accessor":"token_accessor_root","policies":["root"],"ttl":0,"type":"service","orphan":true}'
    
    # Default test secrets
    VAULT_MOCK_SECRETS["secret/data/test"]='{"data":{"username":"testuser","password":"testpass123","api_key":"test-api-key-abc123"},"metadata":{"created_time":"2024-01-15T10:00:00.000000Z","version":1}}'
    
    mock::vault::save_state
}

# Initialize on first load only
if [[ ${#VAULT_MOCK_SECRETS[@]} -eq 0 ]]; then
    mock::vault::init_default_state
fi

# Main vault command interceptor
vault() {
    mock::log_and_verify "vault" "$@"
    
    # Always reload state at the beginning to handle BATS subshells
    mock::vault::load_state
    
    # Check for error injection first (but not for operator commands)
    if [[ -n "${VAULT_MOCK_CONFIG[error_mode]}" ]] && [[ "${1:-}" != "operator" ]]; then
        case "${VAULT_MOCK_CONFIG[error_mode]}" in
            "connection_failed")
                echo "Error: connection failed: connection refused" >&2
                return 1
                ;;
            "token_invalid")
                echo "Error: permission denied" >&2
                return 2
                ;;
            "server_error")
                echo "Error: internal server error" >&2
                return 2
                ;;
        esac
    fi
    
    # Check if Vault is sealed (but allow status and operator commands)
    if [[ "${VAULT_MOCK_CONFIG[sealed]}" == "true" ]] && [[ "${1:-}" != "status" ]] && [[ "${1:-}" != "operator" ]]; then
        echo "Error: Vault is sealed" >&2
        return 2
    fi
    
    # Parse command
    case "${1:-}" in
        "status")
            mock::vault::cmd_status "${@:2}"
            ;;
        "kv")
            mock::vault::cmd_kv "${@:2}"
            ;;
        "read")
            mock::vault::cmd_read "${@:2}"
            ;;
        "write")
            mock::vault::cmd_write "${@:2}"
            ;;
        "list")
            mock::vault::cmd_list "${@:2}"
            ;;
        "delete")
            mock::vault::cmd_delete "${@:2}"
            ;;
        "auth")
            mock::vault::cmd_auth "${@:2}"
            ;;
        "policy")
            mock::vault::cmd_policy "${@:2}"
            ;;
        "token")
            mock::vault::cmd_token "${@:2}"
            ;;
        "secrets")
            mock::vault::cmd_secrets "${@:2}"
            ;;
        "operator")
            mock::vault::cmd_operator "${@:2}"
            ;;
        "server")
            mock::vault::cmd_server "${@:2}"
            ;;
        "-version"|"--version"|"version")
            echo "Vault v${VAULT_MOCK_CONFIG[version]} (mock implementation)"
            ;;
        "-help"|"--help"|"help"|"")
            mock::vault::cmd_help
            ;;
        *)
            echo "Error: unknown command: $1" >&2
            echo "Run 'vault -help' for usage." >&2
            return 1
            ;;
    esac
    
    local result=$?
    
    # Save state after each command
    mock::vault::save_state
    
    return $result
}

# Command implementations
mock::vault::cmd_status() {
    # Check initialization status first
    if [[ "${VAULT_MOCK_CONFIG[initialized]}" == "false" ]]; then
        echo "Vault is not yet initialized"
        return 2
    fi
    
    echo "Key             Value"
    echo "---             -----"
    echo "Seal Type       ${VAULT_MOCK_CONFIG[seal_type]}"
    echo "Initialized     ${VAULT_MOCK_CONFIG[initialized]}"
    echo "Sealed          ${VAULT_MOCK_CONFIG[sealed]}"
    echo "Total Shares    ${VAULT_MOCK_CONFIG[seal_shares]}"
    echo "Threshold       ${VAULT_MOCK_CONFIG[seal_threshold]}"
    echo "Version         ${VAULT_MOCK_CONFIG[version]}"
    echo "Storage Type    ${VAULT_MOCK_CONFIG[storage_type]}"
    echo "Cluster Name    ${VAULT_MOCK_CONFIG[cluster_name]}"
    echo "HA Enabled      false"
    
    # Return error code if sealed, but still show the status output above
    if [[ "${VAULT_MOCK_CONFIG[sealed]}" == "true" ]]; then
        return 2
    fi
}

mock::vault::cmd_kv() {
    local subcommand="${1:-}"
    case "$subcommand" in
        "get")
            shift
            mock::vault::kv_get "$@"
            ;;
        "put")
            shift
            mock::vault::kv_put "$@"
            ;;
        "list")
            shift
            mock::vault::kv_list "$@"
            ;;
        "delete")
            shift
            mock::vault::kv_delete "$@"
            ;;
        "metadata")
            shift
            mock::vault::kv_metadata "$@"
            ;;
        *)
            echo "Usage: vault kv <subcommand> [options] [args]" >&2
            echo "" >&2
            echo "Subcommands:" >&2
            echo "    delete    Deletes versions and metadata of a secret" >&2
            echo "    get       Retrieves data from KV store" >&2
            echo "    list      Lists data or keys" >&2
            echo "    metadata  Interact with Vault's Key-Value storage metadata" >&2
            echo "    put       Sets or updates data in KV store" >&2
            return 1
            ;;
    esac
}

mock::vault::kv_get() {
    local path="${1:-}"
    if [[ -z "$path" ]]; then
        echo "Error: path is required" >&2
        return 1
    fi
    
    # Convert path to storage key
    local storage_key="secret/data/$path"
    if [[ -n "${VAULT_MOCK_SECRETS[$storage_key]}" ]]; then
        local secret_json="${VAULT_MOCK_SECRETS[$storage_key]}"
        
        echo "====== Metadata ======"
        echo "Key                Value"
        echo "---                -----"
        echo "created_time       $(echo "$secret_json" | jq -r '.metadata.created_time // "2024-01-15T10:00:00.000000Z"')"
        echo "custom_metadata    $(echo "$secret_json" | jq -r '.metadata.custom_metadata // "<nil>"')"
        echo "deletion_time      $(echo "$secret_json" | jq -r '.metadata.deletion_time // ""')"
        echo "destroyed          $(echo "$secret_json" | jq -r '.metadata.destroyed // false')"
        echo "version            $(echo "$secret_json" | jq -r '.metadata.version // 1')"
        echo ""
        echo "====== Data ======"
        echo "Key         Value"
        echo "---         -----"
        
        # Extract and display data fields
        echo "$secret_json" | jq -r '.data | to_entries[] | "\(.key)     \(.value)"' 2>/dev/null || {
            # Fallback if jq not available
            echo "api_key     test-api-key-abc123"
            echo "password    testpass123"
            echo "username    testuser"
        }
    else
        echo "No value found at $path" >&2
        return 1
    fi
}

mock::vault::kv_put() {
    local path="${1:-}"
    shift
    if [[ -z "$path" ]]; then
        echo "Error: path is required" >&2
        return 1
    fi
    
    # Parse key=value pairs
    local data_pairs=()
    for arg in "$@"; do
        if [[ "$arg" =~ ^([^=]+)=(.*)$ ]]; then
            data_pairs+=("\"${BASH_REMATCH[1]}\":\"${BASH_REMATCH[2]}\"")
        fi
    done
    
    if [[ ${#data_pairs[@]} -eq 0 ]]; then
        echo "Error: must provide data as key=value pairs" >&2
        return 1
    fi
    
    # Build JSON
    local data_json="{$(IFS=','; echo "${data_pairs[*]}")}"
    local full_json="{\"data\":$data_json,\"metadata\":{\"created_time\":\"$(date -u +%Y-%m-%dT%H:%M:%S.000000Z)\",\"version\":1}}"
    
    # Store in mock state
    local storage_key="secret/data/$path"
    VAULT_MOCK_SECRETS[$storage_key]="$full_json"
    
    echo "Success! Data written to: secret/data/$path"
}

mock::vault::kv_list() {
    local path="${1:-}"
    local search_pattern="secret/data/"
    if [[ -n "$path" ]]; then
        search_pattern="secret/data/$path"
    fi
    
    echo "Keys"
    echo "----"
    for key in "${!VAULT_MOCK_SECRETS[@]}"; do
        if [[ "$key" =~ ^$search_pattern ]]; then
            local display_key="${key#secret/data/}"
            [[ -n "$path" ]] && display_key="${display_key#$path/}"
            echo "$display_key"
        fi
    done
}

mock::vault::kv_delete() {
    local path="${1:-}"
    if [[ -z "$path" ]]; then
        echo "Error: path is required" >&2
        return 1
    fi
    
    local storage_key="secret/data/$path"
    if [[ -n "${VAULT_MOCK_SECRETS[$storage_key]}" ]]; then
        unset VAULT_MOCK_SECRETS[$storage_key]
        echo "Success! Data deleted (if it existed) at: secret/data/$path"
    else
        echo "Success! Data deleted (if it existed) at: secret/data/$path"
    fi
}

mock::vault::kv_metadata() {
    local subcommand="${1:-}"
    local path="${2:-}"
    
    case "$subcommand" in
        "get")
            if [[ -z "$path" ]]; then
                echo "Error: path is required" >&2
                return 1
            fi
            local storage_key="secret/data/$path"
            if [[ -n "${VAULT_MOCK_SECRETS[$storage_key]}" ]]; then
                echo "${VAULT_MOCK_SECRETS[$storage_key]}" | jq '.metadata' 2>/dev/null || echo '{"version":1,"created_time":"2024-01-15T10:00:00.000000Z"}'
            else
                echo "No metadata found at $path" >&2
                return 1
            fi
            ;;
        *)
            echo "Usage: vault kv metadata <subcommand> [options] [args]" >&2
            return 1
            ;;
    esac
}

mock::vault::cmd_read() {
    local path="${1:-}"
    if [[ -z "$path" ]]; then
        echo "Error: path is required" >&2
        return 1
    fi
    
    # Handle different secret engine types
    if [[ "$path" =~ ^secret/ ]]; then
        # KV engine - convert to kv get
        local kv_path="${path#secret/data/}"
        mock::vault::kv_get "$kv_path"
    elif [[ -n "${VAULT_MOCK_SECRETS[$path]}" ]]; then
        echo "${VAULT_MOCK_SECRETS[$path]}"
    else
        echo "No value found at $path" >&2
        return 1
    fi
}

mock::vault::cmd_write() {
    local path="${1:-}"
    shift
    if [[ -z "$path" ]]; then
        echo "Error: path is required" >&2
        return 1
    fi
    
    # Handle different secret engine types
    if [[ "$path" =~ ^secret/ ]]; then
        # KV engine - convert to kv put
        local kv_path="${path#secret/data/}"
        mock::vault::kv_put "$kv_path" "$@"
    else
        # Generic secret write
        local data_pairs=()
        for arg in "$@"; do
            if [[ "$arg" =~ ^([^=]+)=(.*)$ ]]; then
                data_pairs+=("\"${BASH_REMATCH[1]}\":\"${BASH_REMATCH[2]}\"")
            fi
        done
        
        if [[ ${#data_pairs[@]} -gt 0 ]]; then
            local data_json="{$(IFS=','; echo "${data_pairs[*]}")}"
            VAULT_MOCK_SECRETS[$path]="$data_json"
            echo "Success! Data written to: $path"
        else
            echo "Error: must provide data as key=value pairs" >&2
            return 1
        fi
    fi
}

mock::vault::cmd_list() {
    local path="${1:-}"
    echo "Keys"
    echo "----"
    for key in "${!VAULT_MOCK_SECRETS[@]}"; do
        if [[ -z "$path" ]] || [[ "$key" =~ ^$path ]]; then
            echo "$key"
        fi
    done
}

mock::vault::cmd_delete() {
    local path="${1:-}"
    if [[ -z "$path" ]]; then
        echo "Error: path is required" >&2
        return 1
    fi
    
    if [[ "$path" =~ ^secret/ ]]; then
        # KV engine - convert to kv delete
        local kv_path="${path#secret/data/}"
        mock::vault::kv_delete "$kv_path"
    else
        unset VAULT_MOCK_SECRETS[$path]
        echo "Success! Deleted $path"
    fi
}

mock::vault::cmd_auth() {
    case "${1:-}" in
        "list")
            echo "Path         Type        Accessor                Description"
            echo "----         ----        --------                -----------"
            for path in "${!VAULT_MOCK_AUTH_METHODS[@]}"; do
                local config="${VAULT_MOCK_AUTH_METHODS[$path]}"
                local type=$(echo "$config" | jq -r '.type // "unknown"' 2>/dev/null || echo "unknown")
                local accessor=$(echo "$config" | jq -r '.accessor // "unknown"' 2>/dev/null || echo "unknown")
                local description=$(echo "$config" | jq -r '.description // ""' 2>/dev/null || echo "")
                printf "%-12s %-11s %-23s %s\n" "$path" "$type" "$accessor" "$description"
            done
            ;;
        "enable")
            local method="${2:-}"
            local path="${3:-$method/}"
            if [[ -z "$method" ]]; then
                echo "Error: auth method type is required" >&2
                return 1
            fi
            VAULT_MOCK_AUTH_METHODS[$path]="{\"type\":\"$method\",\"description\":\"$method auth method\",\"accessor\":\"auth_${method}_$(date +%s)\"}"
            echo "Success! Enabled $method auth method at: $path"
            ;;
        "disable")
            local path="${2:-}"
            if [[ -z "$path" ]]; then
                echo "Error: path is required" >&2
                return 1
            fi
            [[ "$path" != *"/" ]] && path="${path}/"
            unset VAULT_MOCK_AUTH_METHODS[$path]
            echo "Success! Disabled auth method at: $path"
            ;;
        *)
            echo "Usage: vault auth <subcommand> [options] [args]" >&2
            echo "" >&2
            echo "Subcommands:" >&2
            echo "    disable    Disables an auth method" >&2
            echo "    enable     Enables an auth method" >&2
            echo "    list       Lists enabled auth methods" >&2
            return 1
            ;;
    esac
}

mock::vault::cmd_policy() {
    case "${1:-}" in
        "list")
            echo "Policies"
            echo "--------"
            for policy in "${!VAULT_MOCK_POLICIES[@]}"; do
                echo "$policy"
            done
            ;;
        "read")
            local name="${2:-}"
            if [[ -z "$name" ]]; then
                echo "Error: policy name is required" >&2
                return 1
            fi
            if [[ -n "${VAULT_MOCK_POLICIES[$name]}" ]]; then
                echo "${VAULT_MOCK_POLICIES[$name]}"
            else
                echo "No policy named: $name" >&2
                return 1
            fi
            ;;
        "write")
            local name="${2:-}"
            local policy_file="${3:-}"
            if [[ -z "$name" ]]; then
                echo "Error: policy name is required" >&2
                return 1
            fi
            if [[ -f "$policy_file" ]]; then
                VAULT_MOCK_POLICIES[$name]="$(cat "$policy_file")"
            else
                # Assume policy content provided directly
                VAULT_MOCK_POLICIES[$name]="$policy_file"
            fi
            echo "Success! Uploaded policy: $name"
            ;;
        "delete")
            local name="${2:-}"
            if [[ -z "$name" ]]; then
                echo "Error: policy name is required" >&2
                return 1
            fi
            unset VAULT_MOCK_POLICIES[$name]
            echo "Success! Deleted policy: $name"
            ;;
        *)
            echo "Usage: vault policy <subcommand> [options] [args]" >&2
            return 1
            ;;
    esac
}

mock::vault::cmd_token() {
    case "${1:-}" in
        "create")
            # Generate a more consistent token format with microsecond precision
            local timestamp=$(date +%s%N)
            local token="s.$(printf "%024s" "$timestamp$RANDOM" | md5sum 2>/dev/null | cut -c1-24 || printf "%024d" "$timestamp")"
            local policies="${2:-default}"
            local ttl="${3:-24h}"
            local accessor="accessor_$(date +%s%N)$RANDOM"
            
            # Store token info
            VAULT_MOCK_TOKENS[$token]="{\"accessor\":\"$accessor\",\"policies\":[\"$policies\"],\"ttl\":\"$ttl\",\"type\":\"service\"}"
            
            echo "Key                  Value"
            echo "---                  -----"
            echo "token                $token"
            echo "token_accessor       $accessor"
            echo "token_duration       24h"
            echo "token_renewable      true"
            echo "token_policies       [\"$policies\"]"
            ;;
        "lookup")
            local token="${2:-${VAULT_MOCK_CONFIG[token]}}"
            
            # Check if token exists in our mock storage
            if [[ -n "${VAULT_MOCK_TOKENS[$token]+x}" ]]; then
                local token_info="${VAULT_MOCK_TOKENS[$token]}"
                echo "Key                 Value"
                echo "---                 -----"
                echo "accessor            $(echo "$token_info" | jq -r '.accessor // "unknown"' 2>/dev/null || echo "unknown")"
                echo "creation_time       $(date +%s)"
                echo "display_name        token"
                echo "entity_id           n/a"
                echo "expire_time         <nil>"
                echo "explicit_max_ttl    0s"
                echo "id                  $token"
                echo "policies            $(echo "$token_info" | jq -r '.policies // ["default"]' 2>/dev/null || echo '["default"]')"
                echo "renewable           true"
                echo "ttl                 $(echo "$token_info" | jq -r '.ttl // "24h"' 2>/dev/null || echo "24h")"
                echo "type                $(echo "$token_info" | jq -r '.type // "service"' 2>/dev/null || echo "service")"
            else
                echo "Error: token not found" >&2
                return 1
            fi
            ;;
        "revoke")
            local token="${2:-}"
            if [[ -z "$token" ]]; then
                echo "Error: token is required" >&2
                return 1
            fi
            unset VAULT_MOCK_TOKENS[$token]
            echo "Success! Revoked token"
            ;;
        *)
            echo "Usage: vault token <subcommand> [options] [args]" >&2
            return 1
            ;;
    esac
}

mock::vault::cmd_secrets() {
    case "${1:-}" in
        "list")
            echo "Path         Type         Accessor              Description"
            echo "----         ----         --------              -----------"
            for path in "${!VAULT_MOCK_SECRET_ENGINES[@]}"; do
                local config="${VAULT_MOCK_SECRET_ENGINES[$path]}"
                local type=$(echo "$config" | jq -r '.type // "unknown"' 2>/dev/null || echo "unknown")
                local accessor=$(echo "$config" | jq -r '.accessor // "unknown"' 2>/dev/null || echo "unknown")
                local description=$(echo "$config" | jq -r '.description // ""' 2>/dev/null || echo "")
                printf "%-12s %-12s %-21s %s\n" "$path" "$type" "$accessor" "$description"
            done
            ;;
        "enable")
            local engine_type="${2:-}"
            local path="${3:-$engine_type/}"
            if [[ -z "$engine_type" ]]; then
                echo "Error: engine type is required" >&2
                return 1
            fi
            VAULT_MOCK_SECRET_ENGINES[$path]="{\"type\":\"$engine_type\",\"description\":\"$engine_type secret engine\",\"accessor\":\"${engine_type}_$(date +%s)\"}"
            echo "Success! Enabled the $engine_type secrets engine at: $path"
            ;;
        "disable")
            local path="${2:-}"
            if [[ -z "$path" ]]; then
                echo "Error: path is required" >&2
                return 1
            fi
            [[ "$path" != *"/" ]] && path="${path}/"
            unset VAULT_MOCK_SECRET_ENGINES[$path]
            echo "Success! Disabled the secrets engine at: $path"
            ;;
        *)
            echo "Usage: vault secrets <subcommand> [options] [args]" >&2
            return 1
            ;;
    esac
}

mock::vault::cmd_operator() {
    case "${1:-}" in
        "init")
            VAULT_MOCK_CONFIG[initialized]="true"
            VAULT_MOCK_CONFIG[sealed]="false"
            echo "Unseal Key 1: mock-unseal-key-1"
            echo ""
            echo "Initial Root Token: ${VAULT_MOCK_CONFIG[token]}"
            echo ""
            echo "Vault initialized with 1 key share and a key threshold of 1. Please"
            echo "securely distribute the key share printed above."
            ;;
        "unseal")
            if [[ "${VAULT_MOCK_CONFIG[initialized]}" != "true" ]]; then
                echo "Error: Vault is not initialized" >&2
                return 1
            fi
            VAULT_MOCK_CONFIG[sealed]="false"
            echo "Key                Value"
            echo "---                -----"
            echo "Seal Type          ${VAULT_MOCK_CONFIG[seal_type]}"
            echo "Initialized        true"
            echo "Sealed             false"
            echo "Total Shares       ${VAULT_MOCK_CONFIG[seal_shares]}"
            echo "Threshold          ${VAULT_MOCK_CONFIG[seal_threshold]}"
            echo "Version            ${VAULT_MOCK_CONFIG[version]}"
            echo "Storage Type       ${VAULT_MOCK_CONFIG[storage_type]}"
            echo "Cluster Name       ${VAULT_MOCK_CONFIG[cluster_name]}"
            ;;
        "seal")
            VAULT_MOCK_CONFIG[sealed]="true"
            echo "Success! Vault is sealed."
            ;;
        *)
            echo "Usage: vault operator <subcommand> [options] [args]" >&2
            return 1
            ;;
    esac
}

mock::vault::cmd_server() {
    case "${1:-}" in
        "dev"|"-dev")
            echo "==> Vault server configuration:"
            echo ""
            echo "             Api Address: http://127.0.0.1:8200"
            echo "                     Cgo: disabled"
            echo "         Cluster Address: https://127.0.0.1:8201"
            echo "              Go Version: go1.21.0"
            echo "              Listener 1: tcp (addr: \"127.0.0.1:8200\", cluster address: \"127.0.0.1:8201\", max_request_duration: \"1m30s\", max_request_size: \"33554432\", tls: \"disabled\")"
            echo "               Log Level: info"
            echo "                   Mlock: supported: false, enabled: false"
            echo "           Recovery Mode: false"
            echo "                 Storage: inmem"
            echo "                 Version: Vault v${VAULT_MOCK_CONFIG[version]}"
            echo "             Version Sha: mock-sha"
            echo ""
            echo "==> Vault server started! Log data will stream in below:"
            echo ""
            echo "WARNING! dev mode is enabled! In this mode, Vault runs entirely"
            echo "in-memory and starts unsealed with a single unseal key. The root"
            echo "token is already authenticated to the CLI, so you can immediately"
            echo "begin using Vault."
            echo ""
            echo "You may need to set the following environment variable:"
            echo ""
            echo "    $ export VAULT_ADDR='http://127.0.0.1:8200'"
            echo ""
            echo "The unseal key and root token are displayed below in case you"
            echo "want to seal/unseal the Vault or re-authenticate."
            echo ""
            echo "Unseal Key: mock-dev-unseal-key"
            echo "Root Token: ${VAULT_MOCK_CONFIG[token]}"
            echo ""
            echo "Development mode should NOT be used in production installations!"
            ;;
        *)
            echo "Usage: vault server [options]" >&2
            return 1
            ;;
    esac
}

mock::vault::cmd_help() {
    echo "Usage: vault <command> [args]"
    echo ""
    echo "Common commands:"
    echo "    read        Read data and retrieves secrets"
    echo "    write       Write data, configuration, and secrets"
    echo "    delete      Delete secrets and configuration"
    echo "    list        List data or secrets"
    echo "    login       Authenticate locally"
    echo "    agent       Start a Vault agent"
    echo "    server      Start a Vault server"
    echo "    status      Print seal and HA status"
    echo "    unwrap      Unwrap a wrapped secret"
    echo ""
    echo "Other commands:"
    echo "    audit          Interact with audit devices"
    echo "    auth           Interact with auth methods"
    echo "    debug          Runs the debug command"
    echo "    kv             Interact with Vault's Key-Value storage"
    echo "    lease          Interact with leases"
    echo "    monitor        Stream log messages from a Vault server"
    echo "    namespace      Interact with namespaces"
    echo "    operator       Perform operator-specific tasks"
    echo "    path-help      Retrieve API help for paths"
    echo "    plugin         Interact with Vault plugins and catalog"
    echo "    policy         Interact with policies"
    echo "    print          Prints runtime configurations"
    echo "    secrets        Interact with secrets engines"
    echo "    ssh            Initiate an SSH session"
    echo "    token          Interact with tokens"
    echo "    transform      Interact with Vault's Transform engine"
    echo "    transit        Interact with Vault's Transit engine"
    echo "    version        Prints the Vault version"
}

# Test helper functions
mock::vault::reset() {
    local save_state_param="${1:-true}"
    
    # Clear all data
    VAULT_MOCK_SECRETS=()
    VAULT_MOCK_POLICIES=()
    VAULT_MOCK_TOKENS=()
    VAULT_MOCK_AUTH_METHODS=()
    VAULT_MOCK_SECRET_ENGINES=()
    
    # Reset configuration to defaults
    VAULT_MOCK_CONFIG=(
        [addr]="http://127.0.0.1:8200"
        [token]="root-token"
        [initialized]="true"
        [sealed]="false"
        [version]="1.15.4"
        [cluster_name]="vault-cluster-mock"
        [cluster_id]="mock-cluster-123"
        [standby]="false"
        [error_mode]=""
        [seal_type]="shamir"
        [seal_threshold]="1"
        [seal_shares]="1"
        [storage_type]="inmem"
    )
    
    # Manually reinitialize default state without auto-saving
    # Default auth methods
    VAULT_MOCK_AUTH_METHODS["token/"]='{"type":"token","description":"token based credentials","accessor":"auth_token_abc123"}'
    VAULT_MOCK_AUTH_METHODS["userpass/"]='{"type":"userpass","description":"Username and password auth","accessor":"auth_userpass_def456"}'
    
    # Default secret engines
    VAULT_MOCK_SECRET_ENGINES["secret/"]='{"type":"kv","description":"key/value secret storage","accessor":"kv_abc123","options":{"version":"2"}}'
    VAULT_MOCK_SECRET_ENGINES["database/"]='{"type":"database","description":"Database secret engine","accessor":"database_def456"}'
    
    # Default policies
    VAULT_MOCK_POLICIES["root"]='path "*" { capabilities = ["create", "read", "update", "delete", "list", "sudo"] }'
    VAULT_MOCK_POLICIES["default"]='path "auth/token/lookup-self" { capabilities = ["read"] } path "auth/token/renew-self" { capabilities = ["update"] } path "auth/token/revoke-self" { capabilities = ["update"] }'
    
    # Default root token
    VAULT_MOCK_TOKENS["${VAULT_MOCK_CONFIG[token]}"]='{"accessor":"token_accessor_root","policies":["root"],"ttl":0,"type":"service","orphan":true}'
    
    # Default test secrets
    VAULT_MOCK_SECRETS["secret/data/test"]='{"data":{"username":"testuser","password":"testpass123","api_key":"test-api-key-abc123"},"metadata":{"created_time":"2024-01-15T10:00:00.000000Z","version":1}}'
    
    # Save the reset state only if requested
    if [[ "$save_state_param" == "true" ]]; then
        mock::vault::save_state
    fi
    
    mock::log_state "vault" "Vault mock reset to initial state"
}

mock::vault::set_error() {
    local error_mode="$1"
    VAULT_MOCK_CONFIG[error_mode]="$error_mode"
    mock::vault::save_state
    mock::log_state "vault" "Set Vault error mode: $error_mode"
}

mock::vault::set_sealed() {
    local sealed="$1"
    VAULT_MOCK_CONFIG[sealed]="$sealed"
    mock::vault::save_state
    mock::log_state "vault" "Set Vault sealed: $sealed"
}

mock::vault::set_initialized() {
    local initialized="$1"
    VAULT_MOCK_CONFIG[initialized]="$initialized"
    mock::vault::save_state
    mock::log_state "vault" "Set Vault initialized: $initialized"
}

mock::vault::set_config() {
    local key="$1"
    local value="$2"
    VAULT_MOCK_CONFIG[$key]="$value"
    mock::vault::save_state
    mock::log_state "vault" "Set Vault config: $key=$value"
}

# Test assertions
mock::vault::assert_secret_exists() {
    local path="$1"
    # Check both KV v2 format and direct path
    local kv2_path="secret/data/$path"
    if [[ -n "${VAULT_MOCK_SECRETS[$path]}" ]] || [[ -n "${VAULT_MOCK_SECRETS[$kv2_path]}" ]]; then
        return 0
    else
        echo "Assertion failed: Secret '$path' does not exist" >&2
        return 1
    fi
}

mock::vault::assert_secret_value() {
    local path="$1"
    local key="$2"
    local expected_value="$3"
    
    local kv2_path="secret/data/$path"
    local secret_data=""
    
    if [[ -n "${VAULT_MOCK_SECRETS[$kv2_path]}" ]]; then
        secret_data="${VAULT_MOCK_SECRETS[$kv2_path]}"
    elif [[ -n "${VAULT_MOCK_SECRETS[$path]}" ]]; then
        secret_data="${VAULT_MOCK_SECRETS[$path]}"
    else
        echo "Assertion failed: Secret '$path' does not exist" >&2
        return 1
    fi
    
    local actual_value
    if command -v jq >/dev/null 2>&1; then
        actual_value=$(echo "$secret_data" | jq -r ".data.$key // .${key}" 2>/dev/null || echo "")
    else
        # Fallback without jq
        actual_value=$(echo "$secret_data" | grep -o "\"$key\":\"[^\"]*\"" | sed "s/\"$key\":\"\([^\"]*\)\"/\1/")
    fi
    
    if [[ "$actual_value" == "$expected_value" ]]; then
        return 0
    else
        echo "Assertion failed: Secret '$path' key '$key' value mismatch" >&2
        echo "  Expected: '$expected_value'" >&2
        echo "  Actual: '$actual_value'" >&2
        return 1
    fi
}

mock::vault::assert_policy_exists() {
    local policy_name="$1"
    if [[ -n "${VAULT_MOCK_POLICIES[$policy_name]}" ]]; then
        return 0
    else
        echo "Assertion failed: Policy '$policy_name' does not exist" >&2
        return 1
    fi
}

mock::vault::assert_auth_method_enabled() {
    local path="$1"
    [[ "$path" != *"/" ]] && path="${path}/"
    if [[ -n "${VAULT_MOCK_AUTH_METHODS[$path]}" ]]; then
        return 0
    else
        echo "Assertion failed: Auth method at '$path' is not enabled" >&2
        return 1
    fi
}

mock::vault::assert_secret_engine_enabled() {
    local path="$1"
    [[ "$path" != *"/" ]] && path="${path}/"
    if [[ -n "${VAULT_MOCK_SECRET_ENGINES[$path]}" ]]; then
        return 0
    else
        echo "Assertion failed: Secret engine at '$path' is not enabled" >&2
        return 1
    fi
}

mock::vault::assert_sealed() {
    local expected_sealed="${1:-true}"
    if [[ "${VAULT_MOCK_CONFIG[sealed]}" == "$expected_sealed" ]]; then
        return 0
    else
        echo "Assertion failed: Vault sealed state mismatch" >&2
        echo "  Expected: $expected_sealed" >&2
        echo "  Actual: ${VAULT_MOCK_CONFIG[sealed]}" >&2
        return 1
    fi
}

mock::vault::assert_initialized() {
    local expected_initialized="${1:-true}"
    if [[ "${VAULT_MOCK_CONFIG[initialized]}" == "$expected_initialized" ]]; then
        return 0
    else
        echo "Assertion failed: Vault initialized state mismatch" >&2
        echo "  Expected: $expected_initialized" >&2
        echo "  Actual: ${VAULT_MOCK_CONFIG[initialized]}" >&2
        return 1
    fi
}

# Debug functions
mock::vault::dump_state() {
    echo "=== Vault Mock State ==="
    echo "Configuration:"
    for key in "${!VAULT_MOCK_CONFIG[@]}"; do
        echo "  $key: ${VAULT_MOCK_CONFIG[$key]}"
    done
    
    echo "Secrets:"
    for path in "${!VAULT_MOCK_SECRETS[@]}"; do
        echo "  $path: ${VAULT_MOCK_SECRETS[$path]:0:100}..."
    done
    
    echo "Policies:"
    for name in "${!VAULT_MOCK_POLICIES[@]}"; do
        echo "  $name: ${VAULT_MOCK_POLICIES[$name]:0:50}..."
    done
    
    echo "Tokens:"
    for token in "${!VAULT_MOCK_TOKENS[@]}"; do
        echo "  $token: ${VAULT_MOCK_TOKENS[$token]:0:50}..."
    done
    
    echo "Auth Methods:"
    for path in "${!VAULT_MOCK_AUTH_METHODS[@]}"; do
        echo "  $path: ${VAULT_MOCK_AUTH_METHODS[$path]}"
    done
    
    echo "Secret Engines:"
    for path in "${!VAULT_MOCK_SECRET_ENGINES[@]}"; do
        echo "  $path: ${VAULT_MOCK_SECRET_ENGINES[$path]}"
    done
    
    echo "========================"
}

# Export all functions
export -f vault
export -f mock::vault::save_state
export -f mock::vault::load_state
export -f mock::vault::init_default_state
export -f mock::vault::cmd_status
export -f mock::vault::cmd_kv
export -f mock::vault::kv_get
export -f mock::vault::kv_put
export -f mock::vault::kv_list
export -f mock::vault::kv_delete
export -f mock::vault::kv_metadata
export -f mock::vault::cmd_read
export -f mock::vault::cmd_write
export -f mock::vault::cmd_list
export -f mock::vault::cmd_delete
export -f mock::vault::cmd_auth
export -f mock::vault::cmd_policy
export -f mock::vault::cmd_token
export -f mock::vault::cmd_secrets
export -f mock::vault::cmd_operator
export -f mock::vault::cmd_server
export -f mock::vault::cmd_help
export -f mock::vault::reset
export -f mock::vault::set_error
export -f mock::vault::set_sealed
export -f mock::vault::set_initialized
export -f mock::vault::set_config
export -f mock::vault::assert_secret_exists
export -f mock::vault::assert_secret_value
export -f mock::vault::assert_policy_exists
export -f mock::vault::assert_auth_method_enabled
export -f mock::vault::assert_secret_engine_enabled
export -f mock::vault::assert_sealed
export -f mock::vault::assert_initialized
export -f mock::vault::dump_state

# Save initial state
mock::vault::save_state

echo "[VAULT_MOCK] Vault mock implementation loaded"