#!/usr/bin/env bash
# Vault Common Functions
# Core utility functions shared across Vault operations

#######################################
# Check if Vault is installed (container exists)
# Returns: 0 if installed, 1 if not
#######################################
vault::is_installed() {
    docker container inspect "$VAULT_CONTAINER_NAME" >/dev/null 2>&1
}

#######################################
# Check if Vault is running
# Returns: 0 if running, 1 if not
#######################################
vault::is_running() {
    vault::is_installed && [[ "$(docker container inspect --format='{{.State.Status}}'  "$VAULT_CONTAINER_NAME" 2>/dev/null)" == "running" ]]
}

#######################################
# Check if Vault is healthy (running and responding)
# Returns: 0 if healthy, 1 if not
#######################################
vault::is_healthy() {
    if ! vault::is_running; then
        return 1
    fi
    
    # Check if Vault API is responding
    curl -sf "${VAULT_BASE_URL}/v1/sys/health" >/dev/null 2>&1
}

#######################################
# Check if Vault is initialized
# Returns: 0 if initialized, 1 if not
#######################################
vault::is_initialized() {
    if ! vault::is_healthy; then
        return 1
    fi
    
    local init_status
    init_status=$(curl -sf "${VAULT_BASE_URL}/v1/sys/init" 2>/dev/null | jq -r '.initialized // false' 2>/dev/null)
    [[ "$init_status" == "true" ]]
}

#######################################
# Check if Vault is sealed
# Returns: 0 if sealed, 1 if unsealed
#######################################
vault::is_sealed() {
    if ! vault::is_healthy; then
        return 1
    fi
    
    local seal_status
    local api_response
    api_response=$(curl -sf "${VAULT_BASE_URL}/v1/sys/seal-status" 2>/dev/null)
    
    # If API call failed, assume sealed for safety
    if [[ $? -ne 0 ]] || [[ -z "$api_response" ]]; then
        return 0  # Consider sealed if we can't determine status
    fi
    
    seal_status=$(echo "$api_response" | jq -r '.sealed' 2>/dev/null)
    
    # If jq parsing failed, assume sealed for safety
    if [[ $? -ne 0 ]] || [[ -z "$seal_status" ]]; then
        return 0
    fi
    
    [[ "$seal_status" == "true" ]]
}

#######################################
# Get Vault status string
# Returns: not_installed, stopped, unhealthy, sealed, healthy
#######################################
vault::get_status() {
    if ! vault::is_installed; then
        echo "not_installed"
    elif ! vault::is_running; then
        echo "stopped"
    elif ! vault::is_healthy; then
        echo "unhealthy"
    elif vault::is_sealed; then
        echo "sealed"
    else
        echo "healthy"
    fi
}

#######################################
# Wait for Vault to be healthy
# Arguments:
#   $1 - max wait time in seconds (optional, defaults to VAULT_STARTUP_MAX_WAIT)
#   $2 - check interval in seconds (optional, defaults to VAULT_STARTUP_WAIT_INTERVAL)
# Returns: 0 if healthy, 1 if timeout
#######################################
vault::wait_for_health() {
    local max_wait="${1:-$VAULT_STARTUP_MAX_WAIT}"
    local check_interval="${2:-$VAULT_STARTUP_WAIT_INTERVAL}"
    local elapsed=0
    
    while [[ $elapsed -lt $max_wait ]]; do
        if vault::is_healthy; then
            return 0
        fi
        
        sleep "$check_interval"
        elapsed=$((elapsed + check_interval))
    done
    
    return 1
}

#######################################
# Ensure required directories exist
#######################################
vault::ensure_directories() {
    local dirs=(
        "$VAULT_DATA_DIR"
        "$VAULT_CONFIG_DIR"
        "$VAULT_LOGS_DIR"
    )
    
    for dir in "${dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            mkdir -p "$dir"
            log::info "Created directory: $dir"
        fi
    done
}

#######################################
# Create Vault configuration file
# Arguments:
#   $1 - configuration mode (dev or prod)
#######################################
vault::create_config() {
    local mode="${1:-$VAULT_MODE}"
    local config_file="${VAULT_CONFIG_DIR}/vault.hcl"
    
    vault::ensure_directories
    
    case "$mode" in
        "dev")
            cat > "$config_file" << EOF
# Vault Development Configuration
# WARNING: This is for development only - not secure for production!
# In dev mode, most settings are handled via environment variables
# to avoid conflicts with the built-in dev server configuration
# The dev server ignores most config file settings anyway
EOF
            ;;
        "prod")
            cat > "$config_file" << EOF
# Vault Production Configuration

# Storage backend (file-based for persistence)
storage "file" {
  path = "/vault/data"
}

# HTTP listener
listener "tcp" {
  address       = "0.0.0.0:${VAULT_PORT}"
  tls_disable   = ${VAULT_TLS_DISABLE}
}

# API configuration
api_addr = "${VAULT_BASE_URL}"
cluster_addr = "https://127.0.0.1:8201"

# Disable memory locking for Docker
disable_mlock = true

# UI configuration
ui = true

# Audit logging
audit "file" {
  file_path = "/vault/logs/audit.log"
}

# Performance settings
max_lease_ttl = "768h"
default_lease_ttl = "768h"
EOF
            ;;
        *)
            log::error "Unknown Vault mode: $mode"
            return 1
            ;;
    esac
    
    log::info "Created Vault configuration: $config_file"
}

#######################################
# Get or create Vault root token
# Returns: root token string
#######################################
vault::get_root_token() {
    if [[ "$VAULT_MODE" == "dev" ]]; then
        echo "$VAULT_DEV_ROOT_TOKEN_ID"
    elif [[ -f "$VAULT_TOKEN_FILE" ]]; then
        cat "$VAULT_TOKEN_FILE"
    else
        log::error "Root token file not found: $VAULT_TOKEN_FILE"
        return 1
    fi
}

#######################################
# Validate secret path format
# Arguments:
#   $1 - secret path
# Returns: 0 if valid, 1 if invalid
#######################################
vault::validate_secret_path() {
    local path="$1"
    
    if [[ -z "$path" ]]; then
        log::error "Secret path cannot be empty"
        return 1
    fi
    
    # Check for invalid characters
    if [[ "$path" =~ [[:space:]] ]]; then
        log::error "Secret path cannot contain spaces: $path"
        return 1
    fi
    
    # Check path doesn't start with /
    if [[ "$path" =~ ^/ ]]; then
        log::error "Secret path should not start with '/': $path"
        return 1
    fi
    
    return 0
}

#######################################
# Construct full secret path with engine prefix
# Arguments:
#   $1 - secret path
# Returns: full path string
#######################################
vault::construct_secret_path() {
    local path="$1"
    
    if ! vault::validate_secret_path "$path"; then
        return 1
    fi
    
    # Add namespace prefix if configured
    if [[ -n "$VAULT_NAMESPACE_PREFIX" ]]; then
        path="${VAULT_NAMESPACE_PREFIX}/${path}"
    fi
    
    echo "${VAULT_SECRET_ENGINE}/data/${path}"
}

#######################################
# Make authenticated API request to Vault
# Arguments:
#   $1 - HTTP method (GET, POST, PUT, DELETE)
#   $2 - API endpoint (e.g., /v1/sys/health)
#   $3 - request body (optional, for POST/PUT)
# Returns: API response
#######################################
vault::api_request() {
    local method="$1"
    local endpoint="$2"
    local body="${3:-}"
    
    local token
    token=$(vault::get_root_token) || return 1
    
    local curl_args=(
        -s
        -X "$method"
        -H "X-Vault-Token: $token"
        -H "Content-Type: application/json"
    )
    
    if [[ -n "$body" ]]; then
        curl_args+=(-d "$body")
    fi
    
    curl "${curl_args[@]}" "${VAULT_BASE_URL}${endpoint}"
}

#######################################
# Generate secure random string
# Arguments:
#   $1 - length (optional, default 32)
# Returns: random string
#######################################
vault::generate_random_string() {
    local length="${1:-32}"
    openssl rand -base64 "$length" | tr -d "=+/" | cut -c1-"$length"
}

#######################################
# Cleanup function for graceful shutdown
#######################################
vault::cleanup() {
    log::info "Cleaning up Vault resources..."
    # Add cleanup logic here if needed
}