#!/bin/bash
# LiteLLM core functionality

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
LITELLM_CORE_DIR="${APP_ROOT}/resources/litellm/lib"
LITELLM_RESOURCE_DIR="${APP_ROOT}/resources/litellm"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${LITELLM_RESOURCE_DIR}/config/defaults.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/credentials-utils.sh"

# Initialize LiteLLM
litellm::init() {
    local verbose="${1:-false}"
    
    # Create required directories
    mkdir -p "$LITELLM_CONFIG_DIR" "$LITELLM_LOG_DIR" "$LITELLM_DATA_DIR"
    
    # Set up default configuration if not exists
    if [[ ! -f "$LITELLM_CONFIG_FILE" ]]; then
        litellm::create_default_config "$verbose"
    fi
    
    # Load environment variables
    if [[ -f "$LITELLM_ENV_FILE" ]]; then
        set -a  # automatically export all variables
        source "$LITELLM_ENV_FILE"
        set +a
    fi
    
    [[ "$verbose" == "true" ]] && log::info "LiteLLM initialized"
    return 0
}

# Create default configuration
litellm::create_default_config() {
    local verbose="${1:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Creating default LiteLLM configuration"
    
    # Create config.yaml
    cat > "$LITELLM_CONFIG_FILE" <<EOF
# LiteLLM Configuration
# See https://docs.litellm.ai/docs/proxy/configs for full documentation

model_list:
  # OpenAI Models
  - model_name: gpt-3.5-turbo
    litellm_params:
      model: openai/gpt-3.5-turbo
      api_key: \${OPENAI_API_KEY}
      
  - model_name: gpt-4
    litellm_params:
      model: openai/gpt-4
      api_key: \${OPENAI_API_KEY}
      
  # Anthropic Models
  - model_name: claude-3-haiku
    litellm_params:
      model: anthropic/claude-3-haiku-20240307
      api_key: \${ANTHROPIC_API_KEY}
      
  - model_name: claude-3-sonnet
    litellm_params:
      model: anthropic/claude-3-5-sonnet-20241022
      api_key: \${ANTHROPIC_API_KEY}
      
  # OpenRouter Models (fallback provider)
  - model_name: openrouter-gpt-4
    litellm_params:
      model: openai/gpt-4
      api_base: https://openrouter.ai/api/v1
      api_key: \${OPENROUTER_API_KEY}
      
  - model_name: openrouter-claude
    litellm_params:
      model: anthropic/claude-3-5-sonnet
      api_base: https://openrouter.ai/api/v1
      api_key: \${OPENROUTER_API_KEY}

# Router configuration
router_settings:
  routing_strategy: "${LITELLM_DEFAULT_ROUTING_STRATEGY}"
  enable_pre_call_checks: true
  enable_fallbacks: ${LITELLM_ENABLE_FALLBACKS}
  num_retries: ${LITELLM_MAX_RETRIES}
  
# Logging
litellm_settings:
  set_verbose: false
  json_logs: true
  log_file: "${LITELLM_LOG_FILE}"
  
# General settings  
general_settings:
  master_key: "\${LITELLM_MASTER_KEY}"
  database_url: "sqlite:///\${LITELLM_DATA_DIR}/litellm.db"
  
# Budget controls
budget_settings:
  enable_budget_control: ${LITELLM_ENABLE_BUDGET_CONTROL}
  max_budget: ${LITELLM_DEFAULT_MAX_BUDGET}
  
# Rate limiting
rate_limit_settings:
  default_tpm: ${LITELLM_DEFAULT_TPM}
  default_rpm: ${LITELLM_DEFAULT_RPM}
EOF

    # Create environment file with placeholders
    cat > "$LITELLM_ENV_FILE" <<EOF
# LiteLLM Environment Variables
# Configure your API keys here or in Vault

# Master key for LiteLLM proxy authentication
LITELLM_MASTER_KEY=${LITELLM_MASTER_KEY}

# Provider API Keys (set these or store in Vault)
# OPENAI_API_KEY=sk-your-openai-key
# ANTHROPIC_API_KEY=sk-ant-your-anthropic-key
# OPENROUTER_API_KEY=sk-or-your-openrouter-key
# GOOGLE_API_KEY=your-google-key

# Database configuration
DATABASE_URL=sqlite:///${LITELLM_DATA_DIR}/litellm.db

# Logging
LOG_LEVEL=${LITELLM_LOG_LEVEL}
EOF

    chmod 600 "$LITELLM_ENV_FILE"  # Secure the environment file
    
    [[ "$verbose" == "true" ]] && log::info "Default configuration created"
}

# Test LiteLLM proxy connectivity
litellm::test_connection() {
    local timeout="${1:-$LITELLM_HEALTH_CHECK_TIMEOUT}"
    local verbose="${2:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Testing LiteLLM proxy connection"
    
    # Check if container is running
    if ! docker ps --format '{{.Names}}' | grep -q "^${LITELLM_CONTAINER_NAME}$"; then
        [[ "$verbose" == "true" ]] && log::warn "LiteLLM container is not running"
        return 1
    fi
    
    # Test health endpoint
    local response
    response=$(timeout "$timeout" curl -s "$LITELLM_HEALTH_CHECK_URL" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        [[ "$verbose" == "true" ]] && log::warn "Failed to connect to LiteLLM health endpoint"
        return 1
    fi
    
    # Check for healthy response
    if echo "$response" | grep -q '"status": "healthy"' || echo "$response" | grep -q '"health": "ok"'; then
        [[ "$verbose" == "true" ]] && log::info "LiteLLM proxy is healthy"
        return 0
    fi
    
    [[ "$verbose" == "true" ]] && log::warn "LiteLLM proxy health check failed"
    return 1
}

# Test model availability
litellm::test_model() {
    local model="${1:-$LITELLM_DEFAULT_MODEL}"
    local timeout="${2:-$LITELLM_TIMEOUT}"
    local verbose="${3:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Testing model: $model"
    
    local master_key
    master_key=$(litellm::get_master_key)
    
    if [[ -z "$master_key" ]]; then
        [[ "$verbose" == "true" ]] && log::warn "Master key not available"
        return 1
    fi
    
    local response
    response=$(timeout "$timeout" curl -s -X POST \
        -H "Authorization: Bearer $master_key" \
        -H "Content-Type: application/json" \
        -d "{\"model\": \"$model\", \"messages\": [{\"role\": \"user\", \"content\": \"test\"}], \"max_tokens\": 1}" \
        "${LITELLM_API_BASE}/chat/completions" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        [[ "$verbose" == "true" ]] && log::warn "Failed to test model $model"
        return 1
    fi
    
    # Check for error in response
    if echo "$response" | grep -q '"error"'; then
        [[ "$verbose" == "true" ]] && log::warn "Model test failed: $(echo "$response" | jq -r '.error.message // "unknown error"' 2>/dev/null)"
        return 1
    fi
    
    [[ "$verbose" == "true" ]] && log::info "Model $model is working"
    return 0
}

# Get available models
litellm::list_models() {
    local timeout="${1:-$LITELLM_TIMEOUT}"
    local verbose="${2:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Listing available models"
    
    local master_key
    master_key=$(litellm::get_master_key)
    
    if [[ -z "$master_key" ]]; then
        [[ "$verbose" == "true" ]] && log::warn "Master key not available"
        return 1
    fi
    
    timeout "$timeout" curl -s \
        -H "Authorization: Bearer $master_key" \
        "${LITELLM_API_BASE}/models" 2>/dev/null | \
        jq -r '.data[].id' 2>/dev/null || return 1
}

# Get proxy status and metrics
litellm::get_status() {
    local timeout="${1:-$LITELLM_TIMEOUT}"
    local verbose="${2:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Getting LiteLLM status"
    
    local master_key
    master_key=$(litellm::get_master_key)
    
    if [[ -z "$master_key" ]]; then
        [[ "$verbose" == "true" ]] && log::warn "Master key not available"
        return 1
    fi
    
    timeout "$timeout" curl -s \
        -H "Authorization: Bearer $master_key" \
        "${LITELLM_API_BASE}/health" 2>/dev/null
}

# Get master key from environment or Vault
litellm::get_master_key() {
    # Try environment variable first
    if [[ -n "${LITELLM_MASTER_KEY:-}" ]]; then
        echo "$LITELLM_MASTER_KEY"
        return 0
    fi
    
    # Try Vault
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^vault$'; then
        local vault_key
        vault_key=$(docker exec vault sh -c "export VAULT_TOKEN=myroot && vault kv get -field=master_key secret/vrooli/litellm 2>/dev/null" || true)
        if [[ -n "$vault_key" && "$vault_key" != "No value found at secret/vrooli/litellm" ]]; then
            echo "$vault_key"
            return 0
        fi
    elif command -v vault >/dev/null 2>&1; then
        local vault_key
        vault_key=$(vault kv get -field=master_key secret/vrooli/litellm 2>/dev/null || true)
        if [[ -n "$vault_key" ]]; then
            echo "$vault_key"
            return 0
        fi
    fi
    
    # Try environment file
    if [[ -f "$LITELLM_ENV_FILE" ]]; then
        local file_key
        file_key=$(grep "^LITELLM_MASTER_KEY=" "$LITELLM_ENV_FILE" | cut -d'=' -f2- | tr -d '"')
        if [[ -n "$file_key" ]]; then
            echo "$file_key"
            return 0
        fi
    fi
    
    return 1
}

# Load provider API keys from Vault
litellm::load_provider_keys() {
    local verbose="${1:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Loading provider API keys"
    
    # Check if Vault is available
    local vault_available=false
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^vault$'; then
        vault_available=true
    elif command -v vault >/dev/null 2>&1; then
        vault_available=true
    fi
    
    if [[ "$vault_available" == "true" ]]; then
        # Try to load common provider keys from Vault
        local providers=("openai" "anthropic" "openrouter" "google")
        
        for provider in "${providers[@]}"; do
            local key_var="${provider^^}_API_KEY"
            
            # Skip if already set
            if [[ -n "${!key_var:-}" ]]; then
                continue
            fi
            
            # Try to load from Vault
            local vault_key=""
            if docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^vault$'; then
                vault_key=$(docker exec vault sh -c "export VAULT_TOKEN=myroot && vault kv get -field=api_key secret/vrooli/$provider 2>/dev/null" || true)
            elif command -v vault >/dev/null 2>&1; then
                vault_key=$(vault kv get -field=api_key secret/vrooli/$provider 2>/dev/null || true)
            fi
            
            if [[ -n "$vault_key" && "$vault_key" != "No value found at secret/vrooli/$provider" ]]; then
                export "$key_var"="$vault_key"
                [[ "$verbose" == "true" ]] && log::info "Loaded $provider API key from Vault"
            fi
        done
    fi
}

# Check if LiteLLM container is running
litellm::is_running() {
    docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^${LITELLM_CONTAINER_NAME}$"
}

# Get container status
litellm::container_status() {
    if litellm::is_running; then
        echo "running"
    elif docker ps -a --format '{{.Names}}' 2>/dev/null | grep -q "^${LITELLM_CONTAINER_NAME}$"; then
        echo "stopped"
    else
        echo "not_created"
    fi
}