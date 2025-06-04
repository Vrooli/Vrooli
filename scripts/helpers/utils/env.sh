#!/usr/bin/env bash
set -euo pipefail

UTILS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/exit_codes.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/system.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/vault.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/var.sh"

env::dev_file_exists() {
    [ -f "$var_ENV_DEV_FILE" ]
}

env::prod_file_exists() {
    [ -f "$var_ENV_PROD_FILE" ]
}

# Load secrets from an environment file
env::load_env_file() {
    ENV_FILE="$var_ENV_PROD_FILE"
    if env::in_development "$NODE_ENV"; then
        ENV_FILE="$var_ENV_DEV_FILE"
    fi

    if [ ! -f "$ENV_FILE" ]; then
        log::error "Error: Environment file $ENV_FILE does not exist."
        exit "${ERROR_ENV_FILE_MISSING}"
    fi

    log::info "Sourcing environment file $ENV_FILE"
    set -a
    # shellcheck source=/dev/null
    . "$ENV_FILE"
    set +a
}

# Checks if the HashiCorp Vault CLI ('vault') is installed.
# If not, prints instructions and exits.
env::check_vault_cli() {
    log::info "Checking for Vault CLI..."
    if system::is_command "vault"; then
        log::success "Vault CLI is already installed."
        return 0
    fi

    # Vault CLI installation is typically manual (downloading binary)
    # Attempting auto-install via package managers is unreliable
    log::error "HashiCorp Vault CLI ('vault') not found. Please install it manually from https://developer.hashicorp.com/vault/downloads and ensure it's in your system PATH."
    exit "${ERROR_DEPENDENCY_MISSING:-5}"
}

# Main function to check for the Vault CLI.
env::check_vault_deps() {
    log::header "⚙️ Checking Vault client dependencies..."
    env::check_vault_cli
    log::success "✅ Vault client dependencies checked."
    return 0
} 

# Authenticates to Vault using the token method.
env::authenticate_with_token() {
    : "${VAULT_TOKEN:?Required environment variable VAULT_TOKEN is not set}"
    log::info "Authenticating to Vault using token"
    export VAULT_TOKEN
    return 0
}

# Authenticates to Vault using the AppRole method.
env::authenticate_with_approle() {
    : "${VAULT_ROLE_ID:?Required environment variable VAULT_ROLE_ID is not set}"
    : "${VAULT_SECRET_ID:?Required environment variable VAULT_SECRET_ID is not set}"
    log::info "Authenticating to Vault using AppRole"
    local resp
    resp=$(vault write -format=json auth/approle/login role_id="$VAULT_ROLE_ID" secret_id="$VAULT_SECRET_ID")
    VAULT_TOKEN=$(echo "$resp" | jq -r '.auth.client_token')
    export VAULT_TOKEN
    log::success "Authenticated to Vault with AppRole"
    return 0
}

# Authenticates to Vault using the Kubernetes method.
env::authenticate_with_kubernetes() {
    : "${VAULT_K8S_ROLE:?Required environment variable VAULT_K8S_ROLE is not set}"
    : "${K8S_JWT_PATH:?Required environment variable K8S_JWT_PATH is not set}"
    log::info "Authenticating to Vault using Kubernetes auth"
    local jwt
    jwt=$(cat "$K8S_JWT_PATH")
    local resp
    resp=$(vault write -format=json auth/kubernetes/login role="$VAULT_K8S_ROLE" jwt="$jwt")
    VAULT_TOKEN=$(echo "$resp" | jq -r '.auth.client_token')
    export VAULT_TOKEN
    log::success "Authenticated to Vault with Kubernetes"
    return 0
}

# Fetches secrets from the configured Vault path after successful authentication.
env::fetch_secrets_from_vault() {
    log::info "Fetching secrets from Vault at path: $VAULT_SECRET_PATH"
    check_vault_dependencies
    local endpoint="${VAULT_ADDR}/v1/${VAULT_SECRET_PATH}"
    local resp
    resp=$(curl -s -H "X-Vault-Token: $VAULT_TOKEN" -k -w "\n%{http_code}" "$endpoint")
    local status
    status=$(echo "$resp" | tail -n1)
    local body
    body=$(echo "$resp" | sed '$d')
    validate_vault_response "$status" "$body"
    local secret_json
    secret_json=$(handle_kv_version "$body" "$VAULT_SECRET_PATH")
    extract_secrets "$secret_json"
    log::success "Secrets fetched and exported from Vault"
    return 0
}

# Load secrets from HashiCorp Vault.
# Orchestrates authentication and secret retrieval based on environment variables.
env::load_vault_secrets() {
    log::info "Loading secrets from HashiCorp Vault..."

    # Ensure Vault address is set
    : "${VAULT_ADDR:?Required environment variable VAULT_ADDR is not set}"

    # 1. Check if Vault is healthy and ready
    if ! check_vault_health; then
        log::error "Aborting secret loading due to Vault health check failure."
        exit "${ERROR_VAULT_CONNECTION_FAILED:-30}"
    fi

    # Orchestrate authentication and fetching of secrets

    # 2. Check required base Vault env vars (VAULT_SECRET_PATH)
    : "${VAULT_SECRET_PATH:?Required environment variable VAULT_SECRET_PATH is not set}"

    # 3. Check client dependencies (curl, jq)
    # Note: vault CLI is not strictly needed if using curl/API only
    system::assert_command "curl"
    system::assert_command "jq"

    # 4. Determine VAULT_AUTH_METHOD (default to token)
    local auth_method
    auth_method=$(echo "${VAULT_AUTH_METHOD:-token}" | tr '[:upper:]' '[:lower:]')

    # 5. Call appropriate authenticate_with_* function
    case "$auth_method" in
        token)
            env::authenticate_with_token
            ;;
        approle)
            env::authenticate_with_approle
            ;;
        kubernetes)
            env::authenticate_with_kubernetes
            ;;
        *)
            log::error "Unsupported VAULT_AUTH_METHOD: $auth_method"
            exit "${ERROR_USAGE:-1}"
            ;;
    esac

    # 6. Call env::fetch_secrets_from_vault
    env::fetch_secrets_from_vault

    log::info "Raw secrets fetched from Vault successfully."
    return 0
}

# Internal helper to load a JWT key from a PEM file if the environment variable is not already set.
# Arguments:
#   $1: The name of the environment variable to set (e.g., JWT_PRIV)
#   $2: The full path to the PEM file.
env::load_jwt_key_from_pem_if_unset() {
    local var_name="$1"
    local file_path="$2"
    local current_value
    current_value="${!var_name:-}"

    if [ -n "$current_value" ]; then
        log::info "$var_name is already set. Skipping load from PEM file $file_path."
        return 0
    fi

    log::info "$var_name not set by primary source; attempting to load from $file_path..."
    if [ ! -f "$file_path" ]; then
        log::warning "$var_name: PEM file $file_path not found."
        return 1
    fi

    local raw_content
    raw_content=$(cat "$file_path")

    if [ -z "$raw_content" ]; then
        log::warning "$var_name: PEM file $file_path is empty. $var_name remains unset/empty."
        # Optionally, explicitly set to empty: export "$var_name"=""
        return 1
    fi

    local escaped_content
    escaped_content=$(echo -n "$raw_content" | sed ':a;N;$!ba;s/\n/\\\\n/g')
    
    export "$var_name"="$escaped_content"
    log::info "$var_name loaded and newline-escaped from $file_path."
    return 0
}

# Main function to load secrets based on the SECRETS_SOURCE environment variable.
# NOTE: Make sure to call env::construct_derived_secrets() after this function and after any other secrets are loaded (e.g. check_location_if_not_set())
env::load_secrets() {
    log::header "🔑 Loading secrets..."

    : "${ENVIRONMENT:?Required environment variable ENVIRONMENT is not set}"

    # Determine Node Env
    NODE_ENV="${NODE_ENV:-development}"
    if env::in_production; then
        NODE_ENV="production"
    fi

    # Determine the correct .env file to source
    local env_file_to_source="$var_ENV_PROD_FILE"
    if env::in_development "$NODE_ENV"; then
        env_file_to_source="$var_ENV_DEV_FILE"
    fi

    # Source the primary .env file first to load base config (like VAULT_ADDR, etc.)
    # This makes these variables available regardless of SECRETS_SOURCE
    if [ -f "$env_file_to_source" ]; then
        log::info "Sourcing base environment file: $env_file_to_source"
        set -a
        # shellcheck source=/dev/null
        . "$env_file_to_source"
        set +a
    else
        # In containerized environments, the file might not exist, relying solely on injected vars
        log::info "Base environment file not found: $env_file_to_source. Relying on existing environment variables."
    fi

    # Default SECRETS_SOURCE to 'file' if not set, to avoid fatal errors
    : "${SECRETS_SOURCE:=file}"
    export SECRETS_SOURCE

    : "${SECRETS_SOURCE:?Required environment variable SECRETS_SOURCE is not set}"

    # Now, load secrets based on the source type
    case "$(echo "$SECRETS_SOURCE" | tr '[:upper:]' '[:lower:]')" in
        e|env|environment|f|file)
            log::info "Using secrets from sourced environment file."
            # Check if required variables are present from the file
            : "${DB_USER:?DB_USER not found in environment/file when SECRETS_SOURCE=file}"
            : "${DB_PASSWORD:?DB_PASSWORD not found in environment/file when SECRETS_SOURCE=file}"
            : "${REDIS_PASSWORD:?REDIS_PASSWORD not found in environment/file when SECRETS_SOURCE=file}"
            # env::load_env_file function is now redundant as sourcing is done above
            ;;
        v|vault|hashicorp|hashicorp-vault)
            env::check_vault_deps
            # Call the function to load secrets from Vault
            # This function assumes VAULT_* vars are now set (from file or injected env)
            # It will fetch and export secrets like DB_USER, DB_PASSWORD, etc., potentially overwriting sourced values.
            env::load_vault_secrets
            ;;
        *)
            log::error "Invalid secrets source: $SECRETS_SOURCE"
            exit "${ERROR_USAGE:-1}"
            ;;
    esac

    # Load JWT keys from their respective PEM files if they haven't been set by the primary source.
    log::info "Checking/Loading JWT keys from PEM files if not already set..."
    if env::in_development; then
        env::load_jwt_key_from_pem_if_unset "JWT_PRIV" "${var_STAGING_JWT_PRIV_KEY_FILE}"
        env::load_jwt_key_from_pem_if_unset "JWT_PUB" "${var_STAGING_JWT_PUB_KEY_FILE}"
    else
        env::load_jwt_key_from_pem_if_unset "JWT_PRIV" "${var_PRODUCTION_JWT_PRIV_KEY_FILE}"
        env::load_jwt_key_from_pem_if_unset "JWT_PUB" "${var_PRODUCTION_JWT_PUB_KEY_FILE}"
    fi

    log::success "Secrets loaded and processed."
}

env::construct_derived_secrets() {
    log::info "Constructing derived secrets (DB_URL, REDIS_URL) and checking required secrets..."
    : "${LOCATION:?FATAL: LOCATION is not set. This should have been set by the location-checking script.}"
    : "${DB_USER:?DB_USER was not set by environment file or Vault}"
    : "${DB_PASSWORD:?DB_PASSWORD was not set by environment file or Vault}"
    : "${REDIS_PASSWORD:?REDIS_PASSWORD was not set by environment file or Vault}"
    : "${JWT_PRIV:?FATAL: JWT_PRIV is not set. Check .env file, Vault, or ensure jwt_priv.pem exists at project root.}"
    : "${JWT_PUB:?FATAL: JWT_PUB is not set. Check .env file, Vault, or ensure jwt_pub.pem exists at project root.}"
    
    export SERVER_LOCATION="$LOCATION"
    export DB_URL="postgresql://${DB_USER}:${DB_PASSWORD}@postgres:${PORT_DB:-5432}"
    export REDIS_URL="redis://:${REDIS_PASSWORD}@redis:${PORT_REDIS:-6379}"
    export WORKER_ID=0 # This is fine for single-node deployments, but should be set to the pod ordinal for multi-node deployments.
}

env::is_location_remote() {
    # Use LOCATION or provided argument
    local location="${LOCATION:-}"
    if [ -z "$location" ]; then
        location="$LOCATION"
    fi
    location=$(echo "$location" | tr '[:upper:]' '[:lower:]')

    case "$location" in
        r*) return 0 ;;
        *) return 1 ;;
    esac
}

env::is_location_local() {
    ! env::is_location_remote
}

env::in_production() {
    # Use provided argument (preferred), otherwise default to ENVIRONMENT
    local environment="${1:-${ENVIRONMENT:-}}"
    environment=$(echo "$environment" | tr '[:upper:]' '[:lower:]')

    case "$environment" in
        p*) return 0 ;;
        *) return 1 ;;
    esac
}

env::in_development() {
    # Use provided argument (preferred), otherwise default to ENVIRONMENT
    local environment="${1:-${ENVIRONMENT:-}}"
    environment=$(echo "$environment" | tr '[:upper:]' '[:lower:]')

    case "$environment" in
        d*) return 0 ;;
        *) return 1 ;;
    esac
}

