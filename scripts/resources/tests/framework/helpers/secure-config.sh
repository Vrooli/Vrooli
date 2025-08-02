#!/bin/bash
# ====================================================================
# Secure Configuration Helper
# ====================================================================
#
# Provides secure configuration management for test scenarios, including:
# - Environment variable handling with secure defaults
# - Secret retrieval from configuration files
# - Service URL configuration
# - Production vs development environment handling
#
# ====================================================================

# Configuration file locations
VROOLI_CONFIG_DIR="${VROOLI_CONFIG_DIR:-$HOME/.vrooli}"
VROOLI_TEST_CONFIG="${VROOLI_TEST_CONFIG:-$VROOLI_CONFIG_DIR/test-config.json}"
VROOLI_SECRETS_FILE="${VROOLI_SECRETS_FILE:-$VROOLI_CONFIG_DIR/.secrets}"

# Environment detection
VROOLI_ENV="${VROOLI_ENV:-development}"
IS_CI="${CI:-false}"

# Source logging if available
SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
if [[ -f "$SCRIPT_DIR/framework/helpers/logging.sh" ]]; then
    source "$SCRIPT_DIR/framework/helpers/logging.sh"
fi

# Get secure value from environment or config file
get_secure_value() {
    local key="$1"
    local default_value="$2"
    local is_secret="${3:-false}"
    
    # First check environment variable
    local env_var_name="VROOLI_${key^^}"
    env_var_name="${env_var_name//-/_}"
    local env_value="${!env_var_name}"
    
    if [[ -n "$env_value" ]]; then
        echo "$env_value"
        return 0
    fi
    
    # Check test configuration file
    if [[ -f "$VROOLI_TEST_CONFIG" ]]; then
        local config_value
        config_value=$(jq -r ".${key} // empty" "$VROOLI_TEST_CONFIG" 2>/dev/null)
        if [[ -n "$config_value" && "$config_value" != "null" ]]; then
            echo "$config_value"
            return 0
        fi
    fi
    
    # Check secrets file for sensitive values
    if [[ "$is_secret" == "true" && -f "$VROOLI_SECRETS_FILE" ]]; then
        local secret_value
        secret_value=$(grep "^${key}=" "$VROOLI_SECRETS_FILE" 2>/dev/null | cut -d'=' -f2-)
        if [[ -n "$secret_value" ]]; then
            echo "$secret_value"
            return 0
        fi
    fi
    
    # Return default value
    echo "$default_value"
}

# Get service URL with environment-aware defaults
get_service_url() {
    local service="$1"
    local default_port="$2"
    
    # Check for service-specific URL override
    local url_key="${service}_url"
    local url_value
    url_value=$(get_secure_value "$url_key" "")
    
    if [[ -n "$url_value" ]]; then
        echo "$url_value"
        return 0
    fi
    
    # Build URL from components
    local host
    host=$(get_secure_value "${service}_host" "localhost")
    local port
    port=$(get_secure_value "${service}_port" "$default_port")
    local protocol
    protocol=$(get_secure_value "${service}_protocol" "http")
    
    echo "${protocol}://${host}:${port}"
}

# Get authentication token for a service
get_auth_token() {
    local service="$1"
    local default_token="$2"
    
    # For production, never use default tokens
    if [[ "$VROOLI_ENV" == "production" ]]; then
        default_token=""
    fi
    
    # Check for service-specific token
    local token_key="${service}_token"
    local token
    token=$(get_secure_value "$token_key" "$default_token" true)
    
    if [[ -z "$token" && "$VROOLI_ENV" == "production" ]]; then
        log_error "No ${service} token configured for production environment"
        return 1
    fi
    
    echo "$token"
}

# Initialize secure configuration
init_secure_config() {
    # Create config directory if it doesn't exist
    mkdir -p "$VROOLI_CONFIG_DIR"
    
    # Create default test configuration if it doesn't exist
    if [[ ! -f "$VROOLI_TEST_CONFIG" ]]; then
        cat > "$VROOLI_TEST_CONFIG" <<EOF
{
  "environment": "$VROOLI_ENV",
  "service_defaults": {
    "timeout": 300,
    "retry_count": 3
  },
  "services": {
    "vault": {
      "host": "localhost",
      "port": 8200,
      "protocol": "http"
    },
    "n8n": {
      "host": "localhost", 
      "port": 5678,
      "protocol": "http"
    },
    "ollama": {
      "host": "localhost",
      "port": 11434,
      "protocol": "http"
    },
    "qdrant": {
      "host": "localhost",
      "port": 6333,
      "protocol": "http"
    }
  }
}
EOF
        log_info "Created default test configuration at $VROOLI_TEST_CONFIG"
    fi
    
    # Create secrets template if it doesn't exist
    if [[ ! -f "$VROOLI_SECRETS_FILE" ]]; then
        cat > "$VROOLI_SECRETS_FILE" <<EOF
# Vrooli Test Secrets Configuration
# Add your secret values here (do not commit this file!)
# Format: KEY=VALUE

# Development tokens (replace with secure values in production)
vault_token=
n8n_api_key=
openrouter_api_key=

# Database credentials
postgres_password=
redis_password=
EOF
        chmod 600 "$VROOLI_SECRETS_FILE"
        log_info "Created secrets template at $VROOLI_SECRETS_FILE"
    fi
}

# Check if we're in a secure environment
is_secure_environment() {
    if [[ "$VROOLI_ENV" == "production" ]]; then
        # Check for required production configurations
        local vault_token
        vault_token=$(get_auth_token "vault" "")
        if [[ -z "$vault_token" ]]; then
            return 1
        fi
    fi
    return 0
}

# Export common service URLs for convenience
export_service_urls() {
    export VAULT_BASE_URL=$(get_service_url "vault" "8200")
    export N8N_BASE_URL=$(get_service_url "n8n" "5678")
    export OLLAMA_BASE_URL=$(get_service_url "ollama" "11434")
    export QDRANT_BASE_URL=$(get_service_url "qdrant" "6333")
    export MINIO_BASE_URL=$(get_service_url "minio" "9000")
    export WHISPER_BASE_URL=$(get_service_url "whisper" "8090")
    export AGENT_S2_BASE_URL=$(get_service_url "agent_s2" "4113")
    export BROWSERLESS_BASE_URL=$(get_service_url "browserless" "4110")
    export SEARXNG_BASE_URL=$(get_service_url "searxng" "8100")
    export UNSTRUCTURED_BASE_URL=$(get_service_url "unstructured" "8000")
}

# Log configuration status
log_config_status() {
    log_info "Environment: $VROOLI_ENV"
    log_info "Config directory: $VROOLI_CONFIG_DIR"
    log_info "Test config: $VROOLI_TEST_CONFIG"
    log_info "Secrets file: $VROOLI_SECRETS_FILE"
    log_info "CI environment: $IS_CI"
}

# Initialize on source
init_secure_config

# Export functions for use in tests
export -f get_secure_value
export -f get_service_url
export -f get_auth_token
export -f is_secure_environment
export -f export_service_urls