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
            # Generate self-signed certificates if they don't exist
            # vault::generate_tls_certificates  # Disabled for now to test without TLS
            
            cat > "$config_file" << EOF
# Vault Production Configuration
# Note: TLS disabled for testing - enable for actual production use

# Storage backend with persistence
storage "file" {
  path = "/vault/data"
}

# HTTP listener (TLS disabled for testing)
listener "tcp" {
  address       = "0.0.0.0:${VAULT_PORT}"
  tls_disable   = 1
}

# API and cluster configuration
api_addr = "http://localhost:${VAULT_PORT}"
cluster_addr = "http://127.0.0.1:8201"

# Disable memory locking for Docker
disable_mlock = true

# UI configuration
ui = true

# Performance tuning
max_lease_ttl = "768h"
default_lease_ttl = "768h"
max_request_duration = "90s"
max_request_size = 33554432

# Log level
log_level = "info"
log_format = "json"

# Enable raw endpoint (required for some operations)
raw_storage_endpoint = false

# Telemetry configuration for monitoring
telemetry {
  prometheus_retention_time = "0s"
  disable_hostname = true
}
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
# Generate self-signed TLS certificates for production
#######################################
vault::generate_tls_certificates() {
    local tls_dir="${VAULT_CONFIG_DIR}/tls"
    local cert_file="${tls_dir}/server.crt"
    local key_file="${tls_dir}/server.key"
    
    # Create TLS directory if it doesn't exist
    mkdir -p "$tls_dir"
    
    # Generate certificates if they don't exist
    if [[ ! -f "$cert_file" ]] || [[ ! -f "$key_file" ]]; then
        log::info "Generating self-signed TLS certificates for Vault..."
        
        # Generate private key and certificate
        openssl req -x509 -newkey rsa:4096 -sha256 -days 365 -nodes \
            -keyout "$key_file" \
            -out "$cert_file" \
            -subj "/C=US/ST=State/L=City/O=Vrooli/CN=vault.local" \
            -addext "subjectAltName=DNS:localhost,DNS:vault.local,IP:127.0.0.1,IP:${SITE_IP:-127.0.0.1}" \
            2>/dev/null
        
        # Set appropriate permissions (readable by container)
        chmod 644 "$key_file"
        chmod 644 "$cert_file"
        
        log::info "TLS certificates generated successfully"
    else
        log::info "TLS certificates already exist"
    fi
}

#######################################
# Setup auto-unseal configuration
#######################################
vault::setup_auto_unseal() {
    local config_file="${VAULT_CONFIG_DIR}/vault.hcl"
    
    # Check if we have AWS KMS configuration
    if [[ -n "${AWS_KMS_KEY_ID:-}" ]] && [[ -n "${AWS_REGION:-}" ]]; then
        log::info "Configuring AWS KMS auto-unseal..."
        
        cat >> "$config_file" << EOF

# Auto-unseal configuration using AWS KMS
seal "awskms" {
  region     = "${AWS_REGION}"
  kms_key_id = "${AWS_KMS_KEY_ID}"
  
  # Optional: specify AWS credentials if not using IAM role
  # access_key = ""
  # secret_key = ""
}
EOF
        return 0
    fi
    
    # Check if we have Azure Key Vault configuration
    if [[ -n "${AZURE_KEY_VAULT_NAME:-}" ]] && [[ -n "${AZURE_KEY_NAME:-}" ]]; then
        log::info "Configuring Azure Key Vault auto-unseal..."
        
        cat >> "$config_file" << EOF

# Auto-unseal configuration using Azure Key Vault
seal "azurekeyvault" {
  vault_name = "${AZURE_KEY_VAULT_NAME}"
  key_name   = "${AZURE_KEY_NAME}"
}
EOF
        return 0
    fi
    
    log::warn "No auto-unseal configuration provided. Manual unseal will be required."
    return 1
}

#######################################
# Enable audit logging in production
#######################################
vault::enable_audit_logging() {
    if [[ "$VAULT_MODE" != "prod" ]]; then
        return 0
    fi
    
    log::info "Enabling audit logging..."
    
    # Get root token
    local token
    token=$(vault::get_root_token)
    
    if [[ -z "$token" ]]; then
        log::error "Cannot enable audit logging: no root token available"
        return 1
    fi
    
    # Enable file audit backend
    local audit_response
    audit_response=$(curl -sf -X PUT \
        -H "X-Vault-Token: $token" \
        -d '{
            "type": "file",
            "options": {
                "file_path": "/vault/logs/audit.log",
                "log_raw": "false",
                "hmac_accessor": "true",
                "mode": "0600",
                "format": "json"
            }
        }' \
        "${VAULT_BASE_URL}/v1/sys/audit/file" 2>/dev/null)
    
    if [[ $? -eq 0 ]] || echo "$audit_response" | grep -q "path already in use"; then
        log::info "Audit logging enabled successfully"
        return 0
    else
        log::error "Failed to enable audit logging"
        return 1
    fi
}

#######################################
# Setup AppRole authentication
#######################################
vault::setup_approle_auth() {
    log::info "Setting up AppRole authentication..."
    
    local token
    token=$(vault::get_root_token)
    
    if [[ -z "$token" ]]; then
        log::error "Cannot setup AppRole: no root token available"
        return 1
    fi
    
    # Enable AppRole auth method
    curl -sf -X POST \
        -H "X-Vault-Token: $token" \
        -d '{"type": "approle"}' \
        "${VAULT_BASE_URL}/v1/sys/auth/approle" 2>/dev/null || true
    
    # Create a policy for Vrooli services
    local policy_content='
    # Read-only access to Vrooli secrets
    path "secret/data/vrooli/*" {
      capabilities = ["read", "list"]
    }
    
    # Write access to ephemeral secrets
    path "secret/data/vrooli/ephemeral/*" {
      capabilities = ["create", "read", "update", "delete", "list"]
    }
    
    # Access to generate dynamic credentials
    path "auth/token/create" {
      capabilities = ["create", "update"]
    }
    '
    
    # Create the policy
    curl -sf -X PUT \
        -H "X-Vault-Token: $token" \
        -d "{\"policy\": $(echo "$policy_content" | jq -Rs .)}" \
        "${VAULT_BASE_URL}/v1/sys/policies/acl/vrooli-service" 2>/dev/null
    
    # Create AppRole for Vrooli services
    curl -sf -X POST \
        -H "X-Vault-Token: $token" \
        -d '{
            "token_policies": ["vrooli-service"],
            "token_ttl": "1h",
            "token_max_ttl": "24h",
            "secret_id_ttl": "0",
            "secret_id_num_uses": 0
        }' \
        "${VAULT_BASE_URL}/v1/auth/approle/role/vrooli-service" 2>/dev/null
    
    # Get Role ID
    local role_id
    role_id=$(curl -sf -X GET \
        -H "X-Vault-Token: $token" \
        "${VAULT_BASE_URL}/v1/auth/approle/role/vrooli-service/role-id" 2>/dev/null | jq -r '.data.role_id')
    
    # Generate Secret ID
    local secret_id
    secret_id=$(curl -sf -X POST \
        -H "X-Vault-Token: $token" \
        "${VAULT_BASE_URL}/v1/auth/approle/role/vrooli-service/secret-id" 2>/dev/null | jq -r '.data.secret_id')
    
    if [[ -n "$role_id" ]] && [[ -n "$secret_id" ]]; then
        log::info "AppRole authentication configured successfully"
        log::info "Role ID: $role_id"
        log::info "Secret ID: $secret_id"
        
        # Save to files for later use
        echo "$role_id" > "${VAULT_CONFIG_DIR}/approle-role-id"
        echo "$secret_id" > "${VAULT_CONFIG_DIR}/approle-secret-id"
        chmod 600 "${VAULT_CONFIG_DIR}/approle-"*
        
        return 0
    else
        log::error "Failed to setup AppRole authentication"
        return 1
    fi
}

#######################################
# Setup secret rotation policies
#######################################
vault::setup_secret_rotation() {
    log::info "Setting up secret rotation policies..."
    
    local token
    token=$(vault::get_root_token)
    
    if [[ -z "$token" ]]; then
        log::error "Cannot setup rotation: no root token available"
        return 1
    fi
    
    # Create a rotation policy
    local rotation_policy='{
        "rotation_period": "30d",
        "rotation_rules": [
            {
                "name": "api-keys",
                "path_pattern": "secret/data/vrooli/*/api-key",
                "rotation_period": "90d",
                "notification_period": "7d"
            },
            {
                "name": "database-passwords",
                "path_pattern": "secret/data/vrooli/*/db-password",
                "rotation_period": "30d",
                "notification_period": "3d"
            },
            {
                "name": "service-tokens",
                "path_pattern": "secret/data/vrooli/*/token",
                "rotation_period": "7d",
                "notification_period": "1d"
            }
        ]
    }'
    
    # Store rotation policy as metadata
    curl -sf -X POST \
        -H "X-Vault-Token: $token" \
        -d "{\"data\": $rotation_policy}" \
        "${VAULT_BASE_URL}/v1/secret/data/system/rotation-policy" 2>/dev/null
    
    log::info "Secret rotation policies configured"
    
    # Note: Actual rotation would be handled by a separate cron job or scheduler
    log::info "Note: Implement a cron job to check and rotate secrets based on these policies"
    
    return 0
}

# Removed duplicate vault::get_root_token function - keeping the more complete version below

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
# Get Vault root token using unified secrets system
# Returns: token string or empty if not found
#######################################
vault::get_root_token() {
    local vault_token="${VAULT_TOKEN:-}"
    
    # Try to get token from Vrooli secrets system
    if command -v secrets::resolve >/dev/null 2>&1; then
        if [[ -z "$vault_token" ]]; then
            vault_token=$(secrets::resolve "VAULT_TOKEN" 2>/dev/null || echo "")
        fi
    fi
    
    # Fallback to development default if still empty
    if [[ -z "$vault_token" ]]; then
        vault_token="${VAULT_DEV_ROOT_TOKEN_ID:-}"
    fi
    
    if [[ -z "$vault_token" ]]; then
        log::error "No Vault token available. Configure VAULT_TOKEN in Vrooli secrets."
        return 1
    fi
    
    echo "$vault_token"
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