#!/usr/bin/env bash
# Vault Mock - Tier 2 (Stateful)
# 
# Provides stateful HashiCorp Vault mocking for testing:
# - Secret management (KV v2 operations)
# - Token authentication and lifecycle
# - Policy management (basic CRUD)
# - Administrative operations (seal/unseal, status)
# - Error injection for resilience testing
#
# Coverage: ~80% of common Vault operations in 450 lines

# === Configuration ===
declare -gA VAULT_SECRETS=()           # Path -> secret_data (JSON)
declare -gA VAULT_POLICIES=()          # Policy_name -> policy_content
declare -gA VAULT_TOKENS=()            # Token -> "ttl|policies|meta"
declare -gA VAULT_SECRET_ENGINES=()    # Engine_path -> "type|config"
declare -gA VAULT_AUTH_METHODS=()      # Auth_path -> "type|config"
declare -gA VAULT_CONFIG=(             # Global Vault configuration
    [sealed]="false"
    [initialized]="true"
    [server_address]="http://127.0.0.1:8200"
    [root_token]="hvs.AAAAAAAAAAAAAAAA"
    [current_token]="${VAULT_TOKEN:-hvs.AAAAAAAAAAAAAAAA}"
    [error_mode]=""
    [version]="1.15.2"
)

# Debug mode
declare -g VAULT_DEBUG="${VAULT_DEBUG:-}"

# === Helper Functions ===
vault_debug() {
    [[ -n "$VAULT_DEBUG" ]] && echo "[MOCK:VAULT] $*" >&2
}

vault_check_error() {
    case "${VAULT_CONFIG[error_mode]}" in
        "connection_failed")
            echo "Error connecting to vault: connection refused" >&2
            return 1
            ;;
        "unauthorized")
            echo "Error: permission denied" >&2
            return 2
            ;;
        "forbidden")
            echo "Error: 1 error occurred: * permission denied" >&2
            return 2
            ;;
        "sealed")
            echo "Error: Vault is sealed" >&2
            return 2
            ;;
        "uninitialized")
            echo "Error: Vault is not initialized" >&2
            return 2
            ;;
    esac
    return 0
}

vault_check_auth() {
    local required_policy="${1:-}"
    
    # Check if Vault is sealed
    if [[ "${VAULT_CONFIG[sealed]}" == "true" ]]; then
        echo "Error: Vault is sealed" >&2
        return 2
    fi
    
    # Check if token exists
    local token="${VAULT_CONFIG[current_token]}"
    if [[ -z "${VAULT_TOKENS[$token]}" && "$token" != "${VAULT_CONFIG[root_token]}" ]]; then
        echo "Error: invalid token" >&2
        return 2
    fi
    
    return 0
}

vault_generate_token() {
    local prefix="${1:-hvs}"
    printf "%s.%08x%08x%08x%08x" "$prefix" $RANDOM $RANDOM $RANDOM $RANDOM
}

vault_normalize_path() {
    local path="$1"
    # Remove leading/trailing slashes and normalize
    path="${path#/}"
    path="${path%/}"
    echo "$path"
}

vault_kv_path() {
    local path="$1"
    # Convert user path to internal KV v2 path
    if [[ "$path" =~ ^secret/ ]]; then
        echo "${path/secret\//secret/data/}"
    else
        echo "secret/data/$path"
    fi
}

vault_metadata_path() {
    local path="$1"
    # Convert to metadata path
    if [[ "$path" =~ ^secret/data/ ]]; then
        echo "${path/secret\/data\//secret/metadata/}"
    else
        echo "secret/metadata/$path"
    fi
}

# === Main Vault Command ===
vault() {
    vault_debug "vault called with: $*"
    
    if ! vault_check_error; then
        return $?
    fi
    
    if [[ $# -eq 0 ]]; then
        echo "Usage: vault <command> [options] [args]"
        echo ""
        echo "Common commands:"
        echo "    kv          Interact with Vault's Key-Value store"
        echo "    read        Read data and retrieves secrets"
        echo "    write       Write data, configuration, and secrets"
        echo "    delete      Delete secrets and configuration"
        echo "    list        List data or secrets"
        echo "    status      Print seal and HA status"
        echo "    token       Interact with tokens"
        echo "    policy      Interact with policies"
        echo "    auth        Interact with auth methods"
        echo "    secrets     Interact with secrets engines"
        return 1
    fi
    
    local command="$1"
    shift
    
    case "$command" in
        status)
            vault_cmd_status "$@"
            ;;
        kv)
            vault_cmd_kv "$@"
            ;;
        read)
            vault_cmd_read "$@"
            ;;
        write)
            vault_cmd_write "$@"
            ;;
        delete)
            vault_cmd_delete "$@"
            ;;
        list)
            vault_cmd_list "$@"
            ;;
        token)
            vault_cmd_token "$@"
            ;;
        policy)
            vault_cmd_policy "$@"
            ;;
        auth)
            vault_cmd_auth "$@"
            ;;
        secrets)
            vault_cmd_secrets "$@"
            ;;
        operator)
            vault_cmd_operator "$@"
            ;;
        --version)
            echo "Vault v${VAULT_CONFIG[version]} (mock)"
            ;;
        *)
            echo "Error: unknown command \"$command\"" >&2
            return 1
            ;;
    esac
}

# === Command Implementations ===

vault_cmd_status() {
    vault_debug "Status command"
    
    local sealed="${VAULT_CONFIG[sealed]}"
    local initialized="${VAULT_CONFIG[initialized]}"
    
    cat <<EOF
Key             Value
---             -----
Seal Type       mock
Initialized     $initialized
Sealed          $sealed
Total Shares    1
Threshold       1
Version         ${VAULT_CONFIG[version]}
Build Date      2023-10-27T15:20:00Z
Storage Type    mock
Cluster Name    vault-cluster-mock
Cluster ID      mock-cluster-id
HA Enabled      false
EOF
    
    # Return appropriate exit code
    [[ "$sealed" == "true" ]] && return 2 || return 0
}

vault_cmd_kv() {
    if [[ $# -eq 0 ]]; then
        echo "Usage: vault kv <subcommand> [options] [args]"
        return 1
    fi
    
    local subcommand="$1"
    shift
    
    case "$subcommand" in
        get)
            vault_kv_get "$@"
            ;;
        put)
            vault_kv_put "$@"
            ;;
        delete)
            vault_kv_delete "$@"
            ;;
        list)
            vault_kv_list "$@"
            ;;
        metadata)
            vault_kv_metadata "$@"
            ;;
        *)
            echo "Error: unknown kv subcommand \"$subcommand\"" >&2
            return 1
            ;;
    esac
}

vault_kv_get() {
    vault_debug "KV get: $*"
    vault_check_auth || return $?
    
    # Parse options
    local format="" field="" path=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -format) format="$2"; shift 2 ;;
            -field) field="$2"; shift 2 ;;
            -*) shift ;;
            *) path="$1"; shift ;;
        esac
    done
    
    if [[ -z "$path" ]]; then
        echo "Error: missing path" >&2
        return 1
    fi
    
    local kv_path=$(vault_kv_path "$path")
    local secret_data="${VAULT_SECRETS[$kv_path]}"
    
    if [[ -z "$secret_data" ]]; then
        echo "No value found at $path" >&2
        return 2
    fi
    
    # Return specific field if requested
    if [[ -n "$field" ]]; then
        echo "$secret_data" | jq -r ".data.data.\"$field\"" 2>/dev/null || echo "null"
        return 0
    fi
    
    # Format output based on requested format
    case "$format" in
        json)
            echo "$secret_data"
            ;;
        *)
            # Default table format
            echo "======= Metadata ======="
            echo "$secret_data" | jq -r '.data.metadata | to_entries[] | "\(.key)\t\(.value)"' 2>/dev/null || echo "Key\tValue"
            echo ""
            echo "====== Data ======"
            echo "$secret_data" | jq -r '.data.data | to_entries[] | "\(.key)\t\(.value)"' 2>/dev/null || echo "Key\tValue"
            ;;
    esac
}

vault_kv_put() {
    vault_debug "KV put: $*"
    vault_check_auth || return $?
    
    if [[ $# -lt 2 ]]; then
        echo "Usage: vault kv put <path> <key=value> [key=value...]" >&2
        return 1
    fi
    
    local path="$1"
    shift
    
    local kv_path=$(vault_kv_path "$path")
    local data_obj="{"
    local first=true
    
    # Parse key=value pairs
    for arg in "$@"; do
        if [[ "$arg" =~ ^([^=]+)=(.*)$ ]]; then
            local key="${BASH_REMATCH[1]}"
            local value="${BASH_REMATCH[2]}"
            
            if [[ "$first" != "true" ]]; then
                data_obj+=","
            fi
            data_obj+="\"$key\":\"$value\""
            first=false
        fi
    done
    data_obj+="}"
    
    # Create full secret structure
    local secret_json="{\"data\":{\"data\":$data_obj,\"metadata\":{\"created_time\":\"$(date -Iseconds)\",\"custom_metadata\":null,\"deletion_time\":\"\",\"destroyed\":false,\"version\":1}}}"
    
    VAULT_SECRETS[$kv_path]="$secret_json"
    vault_debug "Stored secret at: $kv_path"
    
    echo "======= Secret Path ======="
    echo "$path"
    echo ""
    echo "======= Metadata ======="
    echo "Key                Value"
    echo "---                -----"
    echo "created_time       $(date -Iseconds)"
    echo "custom_metadata    <nil>"
    echo "deletion_time      n/a"
    echo "destroyed          false"
    echo "version            1"
}

vault_kv_delete() {
    vault_debug "KV delete: $*"
    vault_check_auth || return $?
    
    if [[ $# -eq 0 ]]; then
        echo "Error: missing path" >&2
        return 1
    fi
    
    local path="$1"
    local kv_path=$(vault_kv_path "$path")
    
    if [[ -z "${VAULT_SECRETS[$kv_path]}" ]]; then
        echo "No value found at $path" >&2
        return 2
    fi
    
    unset VAULT_SECRETS[$kv_path]
    vault_debug "Deleted secret at: $kv_path"
    echo "Success! Data deleted (if it existed) at: $path"
}

vault_kv_list() {
    vault_debug "KV list: $*"
    vault_check_auth || return $?
    
    local path="${1:-/}"
    local search_prefix="secret/data/"
    
    if [[ "$path" != "/" ]]; then
        search_prefix="secret/data/$(vault_normalize_path "$path")/"
    fi
    
    echo "Keys"
    echo "----"
    
    local found=false
    for secret_path in "${!VAULT_SECRETS[@]}"; do
        if [[ "$secret_path" =~ ^$search_prefix([^/]+)/?(.*)$ ]]; then
            local key="${BASH_REMATCH[1]}"
            local remainder="${BASH_REMATCH[2]}"
            
            if [[ -n "$remainder" ]]; then
                echo "$key/"
            else
                echo "$key"
            fi
            found=true
        fi
    done
    
    if [[ "$found" != "true" ]]; then
        echo "No entries found"
        return 0
    fi
}

vault_cmd_read() {
    vault_debug "Read command: $*"
    vault_check_auth || return $?
    
    if [[ $# -eq 0 ]]; then
        echo "Error: missing path" >&2
        return 1
    fi
    
    local path="$1"
    local secret_data="${VAULT_SECRETS[$path]}"
    
    if [[ -z "$secret_data" ]]; then
        echo "No value found at $path" >&2
        return 2
    fi
    
    echo "$secret_data"
}

vault_cmd_write() {
    vault_debug "Write command: $*"
    vault_check_auth || return $?
    
    if [[ $# -lt 2 ]]; then
        echo "Usage: vault write <path> <key=value> [key=value...]" >&2
        return 1
    fi
    
    local path="$1"
    shift
    
    local data_obj="{"
    local first=true
    
    for arg in "$@"; do
        if [[ "$arg" =~ ^([^=]+)=(.*)$ ]]; then
            local key="${BASH_REMATCH[1]}"
            local value="${BASH_REMATCH[2]}"
            
            if [[ "$first" != "true" ]]; then
                data_obj+=","
            fi
            data_obj+="\"$key\":\"$value\""
            first=false
        fi
    done
    data_obj+="}"
    
    VAULT_SECRETS[$path]="$data_obj"
    vault_debug "Wrote data to: $path"
    echo "Success! Data written to: $path"
}

vault_cmd_delete() {
    vault_debug "Delete command: $*"
    vault_check_auth || return $?
    
    if [[ $# -eq 0 ]]; then
        echo "Error: missing path" >&2
        return 1
    fi
    
    local path="$1"
    
    if [[ -z "${VAULT_SECRETS[$path]}" ]]; then
        echo "No value found at $path" >&2
        return 2
    fi
    
    unset VAULT_SECRETS[$path]
    vault_debug "Deleted: $path"
    echo "Success! Data deleted (if it existed) at: $path"
}

vault_cmd_list() {
    vault_debug "List command: $*"
    vault_check_auth || return $?
    
    local path="${1:-/}"
    
    echo "Keys"
    echo "----"
    
    local found=false
    for secret_path in "${!VAULT_SECRETS[@]}"; do
        if [[ "$secret_path" =~ ^$path(.*)$ ]]; then
            echo "${BASH_REMATCH[1]:-$secret_path}"
            found=true
        fi
    done
    
    if [[ "$found" != "true" ]]; then
        echo "No entries found"
    fi
}

vault_cmd_token() {
    vault_debug "Token command: $*"
    
    if [[ $# -eq 0 ]]; then
        echo "Usage: vault token <subcommand>" >&2
        return 1
    fi
    
    local subcommand="$1"
    shift
    
    case "$subcommand" in
        create)
            vault_token_create "$@"
            ;;
        lookup)
            vault_token_lookup "$@"
            ;;
        revoke)
            vault_token_revoke "$@"
            ;;
        *)
            echo "Error: unknown token subcommand \"$subcommand\"" >&2
            return 1
            ;;
    esac
}

vault_token_create() {
    vault_debug "Token create: $*"
    vault_check_auth || return $?
    
    local ttl="768h" policies="default" display_name=""
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -ttl) ttl="$2"; shift 2 ;;
            -policy) policies="$2"; shift 2 ;;
            -display-name) display_name="$2"; shift 2 ;;
            -*) shift ;;
            *) shift ;;
        esac
    done
    
    local token=$(vault_generate_token)
    VAULT_TOKENS[$token]="$ttl|$policies|$display_name"
    
    vault_debug "Created token: $token"
    
    cat <<EOF
Key                  Value
---                  -----
token                $token
token_accessor       $(vault_generate_token acc)
token_duration       $ttl
token_renewable      true
token_policies       [$policies]
identity_policies    []
policies             [$policies]
EOF
}

vault_token_lookup() {
    vault_debug "Token lookup: $*"
    
    local token="${1:-${VAULT_CONFIG[current_token]}}"
    local token_data="${VAULT_TOKENS[$token]}"
    
    if [[ -z "$token_data" && "$token" != "${VAULT_CONFIG[root_token]}" ]]; then
        echo "Error: invalid token" >&2
        return 2
    fi
    
    if [[ "$token" == "${VAULT_CONFIG[root_token]}" ]]; then
        token_data="0s|root|root"
    fi
    
    IFS='|' read -r ttl policies display_name <<< "$token_data"
    
    cat <<EOF
Key                 Value
---                 -----
accessor            $(vault_generate_token acc)
creation_time       $(date '+%s')
creation_ttl        $ttl
display_name        $display_name
entity_id           n/a
expire_time         $(date -d "+$ttl" '+%Y-%m-%dT%H:%M:%S.%NZ' 2>/dev/null || echo 'null')
explicit_max_ttl    0s
id                  $token
issue_time          $(date '+%Y-%m-%dT%H:%M:%S.%NZ')
meta                <nil>
num_uses            0
orphan              false
path                auth/token/create
policies            [$policies]
renewable           true
ttl                 $ttl
type                service
EOF
}

vault_token_revoke() {
    vault_debug "Token revoke: $*"
    vault_check_auth || return $?
    
    local token="${1:-${VAULT_CONFIG[current_token]}}"
    
    if [[ "$token" == "${VAULT_CONFIG[root_token]}" ]]; then
        echo "Error: cannot revoke root token" >&2
        return 1
    fi
    
    if [[ -n "${VAULT_TOKENS[$token]}" ]]; then
        unset VAULT_TOKENS[$token]
        vault_debug "Revoked token: $token"
        echo "Success! Revoked token (if it existed)"
    else
        echo "Success! Revoked token (if it existed)"
    fi
}

vault_cmd_policy() {
    vault_debug "Policy command: $*"
    
    if [[ $# -eq 0 ]]; then
        echo "Usage: vault policy <subcommand>" >&2
        return 1
    fi
    
    local subcommand="$1"
    shift
    
    case "$subcommand" in
        read)
            vault_policy_read "$@"
            ;;
        write)
            vault_policy_write "$@"
            ;;
        list)
            vault_policy_list "$@"
            ;;
        delete)
            vault_policy_delete "$@"
            ;;
        *)
            echo "Error: unknown policy subcommand \"$subcommand\"" >&2
            return 1
            ;;
    esac
}

vault_policy_read() {
    vault_debug "Policy read: $*"
    vault_check_auth || return $?
    
    if [[ $# -eq 0 ]]; then
        echo "Error: missing policy name" >&2
        return 1
    fi
    
    local policy_name="$1"
    local policy_content="${VAULT_POLICIES[$policy_name]}"
    
    if [[ -z "$policy_content" ]]; then
        echo "No policy named: $policy_name" >&2
        return 2
    fi
    
    echo "$policy_content"
}

vault_policy_write() {
    vault_debug "Policy write: $*"
    vault_check_auth || return $?
    
    if [[ $# -lt 2 ]]; then
        echo "Usage: vault policy write <name> <path-or-content>" >&2
        return 1
    fi
    
    local policy_name="$1"
    local policy_source="$2"
    local policy_content=""
    
    # Check if it's a file path or direct content
    if [[ -f "$policy_source" ]]; then
        policy_content=$(cat "$policy_source")
    else
        policy_content="$policy_source"
    fi
    
    VAULT_POLICIES[$policy_name]="$policy_content"
    vault_debug "Wrote policy: $policy_name"
    echo "Success! Uploaded policy: $policy_name"
}

vault_policy_list() {
    vault_debug "Policy list"
    vault_check_auth || return $?
    
    echo "default"
    echo "root"
    for policy_name in "${!VAULT_POLICIES[@]}"; do
        echo "$policy_name"
    done
}

vault_policy_delete() {
    vault_debug "Policy delete: $*"
    vault_check_auth || return $?
    
    if [[ $# -eq 0 ]]; then
        echo "Error: missing policy name" >&2
        return 1
    fi
    
    local policy_name="$1"
    
    if [[ -z "${VAULT_POLICIES[$policy_name]}" ]]; then
        echo "No policy named: $policy_name" >&2
        return 2
    fi
    
    unset VAULT_POLICIES[$policy_name]
    vault_debug "Deleted policy: $policy_name"
    echo "Success! Deleted policy: $policy_name"
}

vault_cmd_operator() {
    vault_debug "Operator command: $*"
    
    if [[ $# -eq 0 ]]; then
        echo "Usage: vault operator <subcommand>" >&2
        return 1
    fi
    
    local subcommand="$1"
    shift
    
    case "$subcommand" in
        seal)
            VAULT_CONFIG[sealed]="true"
            vault_debug "Vault sealed"
            echo "Success! Vault is sealed."
            ;;
        unseal)
            VAULT_CONFIG[sealed]="false"
            vault_debug "Vault unsealed"
            echo "Key                Value"
            echo "---                -----"
            echo "Sealed             false"
            echo "Total Shares       1"
            echo "Threshold          1"
            echo "Version            ${VAULT_CONFIG[version]}"
            echo "Build Date         2023-10-27T15:20:00Z"
            echo "Storage Type       mock"
            echo "Cluster Name       vault-cluster-mock"
            echo "Cluster ID         mock-cluster-id"
            echo "HA Enabled         false"
            ;;
        init)
            VAULT_CONFIG[initialized]="true"
            VAULT_CONFIG[sealed]="false"
            vault_debug "Vault initialized"
            echo "Unseal Key 1: mock-unseal-key"
            echo ""
            echo "Initial Root Token: ${VAULT_CONFIG[root_token]}"
            echo ""
            echo "Vault initialized with 1 key share and a key threshold of 1."
            ;;
        *)
            echo "Error: unknown operator subcommand \"$subcommand\"" >&2
            return 1
            ;;
    esac
}

# === Mock Configuration Functions ===
vault_mock_set_error() {
    VAULT_CONFIG[error_mode]="$1"
    vault_debug "Set error mode: $1"
}

vault_mock_set_token() {
    VAULT_CONFIG[current_token]="$1"
    vault_debug "Set current token: $1"
}

vault_mock_seal() {
    VAULT_CONFIG[sealed]="true"
    vault_debug "Vault sealed"
}

vault_mock_unseal() {
    VAULT_CONFIG[sealed]="false"
    vault_debug "Vault unsealed"
}

# === Convention-based Test Functions ===
test_vault_connection() {
    vault_debug "Testing connection..."
    
    # Test status command
    local result
    result=$(vault status 2>&1)
    
    if [[ "$result" =~ "Version" ]]; then
        vault_debug "Connection test passed"
        return 0
    else
        vault_debug "Connection test failed: $result"
        return 1
    fi
}

test_vault_health() {
    vault_debug "Testing health..."
    
    # Test connection
    test_vault_connection || return 1
    
    # Test basic operations
    vault kv put secret/test key=value >/dev/null 2>&1 || return 1
    vault kv get secret/test >/dev/null 2>&1 || return 1
    vault kv delete secret/test >/dev/null 2>&1 || return 1
    
    # Test token operations
    vault token create >/dev/null 2>&1 || return 1
    
    vault_debug "Health test passed"
    return 0
}

test_vault_basic() {
    vault_debug "Testing basic operations..."
    
    # Test KV operations
    vault kv put secret/basic-test username=testuser password=testpass >/dev/null 2>&1 || return 1
    local result
    result=$(vault kv get -field=username secret/basic-test 2>&1)
    [[ "$result" == "testuser" ]] || return 1
    
    # Test policy operations
    vault policy write test-policy 'path "secret/*" { capabilities = ["read"] }' >/dev/null 2>&1 || return 1
    vault policy read test-policy >/dev/null 2>&1 || return 1
    
    # Test token operations
    local token_output
    token_output=$(vault token create -policy=test-policy 2>&1)
    [[ "$token_output" =~ "token" ]] || return 1
    
    # Cleanup
    vault kv delete secret/basic-test >/dev/null 2>&1
    vault policy delete test-policy >/dev/null 2>&1
    
    vault_debug "Basic test passed"
    return 0
}

# === State Management ===
vault_mock_reset() {
    vault_debug "Resetting mock state (called from: ${BASH_SOURCE[1]:-unknown}:${BASH_LINENO[0]:-unknown})"
    
    VAULT_SECRETS=()
    VAULT_POLICIES=()
    VAULT_TOKENS=()
    VAULT_SECRET_ENGINES=()
    VAULT_AUTH_METHODS=()
    VAULT_CONFIG[error_mode]=""
    VAULT_CONFIG[sealed]="false"
    VAULT_CONFIG[initialized]="true"
    
    # Initialize defaults
    vault_mock_init_defaults
}

vault_mock_init_defaults() {
    # Default secret engines
    VAULT_SECRET_ENGINES["secret/"]="kv|version=2"
    VAULT_SECRET_ENGINES["sys/"]="system|builtin"
    
    # Default auth methods
    VAULT_AUTH_METHODS["token/"]="token|builtin"
    
    # Default policies
    VAULT_POLICIES["default"]='path "secret/data/*" { capabilities = ["create", "update", "read", "delete", "list"] }'
    VAULT_POLICIES["root"]='path "*" { capabilities = ["create", "read", "update", "delete", "list", "sudo"] }'
    
    # Sample secrets for testing
    VAULT_SECRETS["secret/data/demo"]='{"data":{"data":{"username":"admin","password":"secret123"},"metadata":{"created_time":"2023-01-01T00:00:00Z","version":1}}}'
}

vault_mock_dump_state() {
    echo "=== Vault Mock State ==="
    echo "Sealed: ${VAULT_CONFIG[sealed]}"
    echo "Initialized: ${VAULT_CONFIG[initialized]}"
    echo "Current Token: ${VAULT_CONFIG[current_token]:0:20}..."
    echo "Error Mode: ${VAULT_CONFIG[error_mode]:-none}"
    echo "Secrets: ${#VAULT_SECRETS[@]}"
    for path in "${!VAULT_SECRETS[@]}"; do
        echo "  $path: ${VAULT_SECRETS[$path]:0:50}..."
    done
    echo "Policies: ${#VAULT_POLICIES[@]}"
    for policy in "${!VAULT_POLICIES[@]}"; do
        echo "  $policy: ${VAULT_POLICIES[$policy]:0:50}..."
    done
    echo "Tokens: ${#VAULT_TOKENS[@]}"
    for token in "${!VAULT_TOKENS[@]}"; do
        echo "  ${token:0:20}...: ${VAULT_TOKENS[$token]}"
    done
    echo "==================="
}

# === Export Functions ===
export -f vault
export -f test_vault_connection test_vault_health test_vault_basic
export -f vault_mock_reset vault_mock_set_error vault_mock_set_token
export -f vault_mock_seal vault_mock_unseal vault_mock_dump_state
export -f vault_debug vault_check_error

# Initialize with defaults
vault_mock_reset
vault_debug "Vault Tier 2 mock initialized"