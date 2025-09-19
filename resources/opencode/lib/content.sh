#!/bin/bash
# OpenCode Content Functions - Manage AI models and configurations

# Source common functions
source "${BASH_SOURCE[0]%/*}/common.sh"

opencode::content::add() {
    local content_type="${1:-config}"
    local name="${2:-}"
    
    case "${content_type}" in
        config|configuration)
            opencode::content::add_config "$@"
            ;;
        model)
            opencode::content::add_model "$@"
            ;;
        *)
            log::error "Unknown content type: ${content_type}"
            log::info "Supported types: config, model"
            return 1
            ;;
    esac
}

opencode::content::add_config() {
    local config_name="${2:-default}"
    local provider="${3:-ollama}"
    local chat_model="${4:-llama3.2:3b}"
    local completion_model="${5:-qwen2.5-coder:3b}"

    log::info "Adding OpenCode configuration: ${config_name}"

    # Ensure directories exist
    opencode_ensure_dirs
    opencode_load_secrets || true

    # Create or update configuration
    local config_file="${OPENCODE_DATA_DIR}/config-${config_name}.json"
    local timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

    if command -v jq &>/dev/null; then
        local gateway_url=""
        local slug="${CLOUDFLARE_AI_GATEWAY_SLUG:-}"
        if [[ -n "${CLOUDFLARE_ACCOUNT_ID:-}" ]]; then
            if [[ -z "$slug" ]]; then
                slug="vrooli"
            fi
            gateway_url="https://gateway.ai.cloudflare.com/v1/${CLOUDFLARE_ACCOUNT_ID}/${slug}"
        fi

        jq -n \
            --arg name "${config_name}" \
            --arg provider "${provider}" \
            --arg chat "${chat_model}" \
            --arg completion "${completion_model}" \
            --arg created "${timestamp}" \
            --arg openrouter_key "${OPENROUTER_API_KEY:-}" \
            --arg cloudflare_token "${CLOUDFLARE_API_TOKEN:-}" \
            --arg cloudflare_account "${CLOUDFLARE_ACCOUNT_ID:-}" \
            --arg cloudflare_slug "$slug" \
            --arg cloudflare_url "$gateway_url" \
            --argjson port ${OPENCODE_PORT} \
            '{
                name: $name,
                provider: $provider,
                chat_model: $chat,
                completion_model: $completion,
                port: $port,
                auto_suggest: true,
                enable_logging: true,
                created: $created
            }
            | if $openrouter_key != "" then .openrouter_api_key = $openrouter_key else . end
            | if $cloudflare_token != "" then .cloudflare_api_token = $cloudflare_token else . end
            | if $cloudflare_account != "" then .cloudflare_account_id = $cloudflare_account else . end
            | if $cloudflare_slug != "" then .cloudflare_gateway_slug = $cloudflare_slug else . end
            | if $cloudflare_url != "" then .cloudflare_gateway_url = $cloudflare_url else . end' > "${config_file}"
    else
        cat > "${config_file}" <<EOF
{
  "name": "${config_name}",
  "provider": "${provider}",
  "chat_model": "${chat_model}",
  "completion_model": "${completion_model}",
  "port": ${OPENCODE_PORT},
  "auto_suggest": true,
  "enable_logging": true,
  "created": "${timestamp}"
}
EOF
    fi

    log::success "Configuration '${config_name}' saved to ${config_file}"
    log::info "To activate this configuration, run: opencode content activate ${config_name}"
}

opencode::content::add_model() {
    local model_name="${2:-}"
    
    if [[ -z "${model_name}" ]]; then
        log::error "Model name is required"
        log::info "Usage: opencode content add model <model_name>"
        return 1
    fi
    
    log::info "Adding model reference: ${model_name}"
    
    # Add to available models list
    local models_file="${OPENCODE_DATA_DIR}/available-models.json"
    local models="[]"
    
    if [[ -f "${models_file}" ]]; then
        models=$(cat "${models_file}")
    fi
    
    # Add model if not already present
    models=$(echo "${models}" | jq --arg model "${model_name}" '. + [$model] | unique')
    echo "${models}" > "${models_file}"
    
    log::success "Model '${model_name}' added to available models"
}

opencode::content::list() {
    local content_type="${1:-all}"
    
    case "${content_type}" in
        config|configurations)
            opencode::content::list_configs
            ;;
        model|models)
            opencode::content::list_models
            ;;
        all)
            opencode::content::list_configs
            echo ""
            opencode::content::list_models
            ;;
        *)
            log::error "Unknown content type: ${content_type}"
            log::info "Supported types: config, model, all"
            return 1
            ;;
    esac
}

opencode::content::list_configs() {
    log::info "OpenCode Configurations:"
    
    if [[ -f "${OPENCODE_CONFIG_FILE}" ]]; then
        echo "📋 Active configuration:"
        if command -v jq &>/dev/null; then
            local provider=$(jq -r '.provider // "unknown"' "${OPENCODE_CONFIG_FILE}")
            local chat_model=$(jq -r '.chat_model // "unknown"' "${OPENCODE_CONFIG_FILE}")
            local completion_model=$(jq -r '.completion_model // "unknown"' "${OPENCODE_CONFIG_FILE}")
            echo "   Provider: ${provider}"
            echo "   Chat Model: ${chat_model}" 
            echo "   Completion Model: ${completion_model}"
        else
            echo "   ${OPENCODE_CONFIG_FILE}"
        fi
        echo ""
    fi
    
    echo "💾 Saved configurations:"
    local config_files=(${OPENCODE_DATA_DIR}/config-*.json)
    if [[ -f "${config_files[0]}" ]]; then
        for config_file in "${config_files[@]}"; do
            local config_name=$(basename "${config_file}" .json | sed 's/^config-//')
            if command -v jq &>/dev/null; then
                local provider=$(jq -r '.provider // "unknown"' "${config_file}")
                local chat_model=$(jq -r '.chat_model // "unknown"' "${config_file}")
                echo "   ${config_name}: ${provider}/${chat_model}"
            else
                echo "   ${config_name}: ${config_file}"
            fi
        done
    else
        echo "   No saved configurations found"
    fi
}

opencode::content::list_models() {
    log::info "Available AI Models:"
    
    # Get models from Ollama if available
    local ollama_models=$(opencode_get_models ollama)
    local ollama_count=$(echo "${ollama_models}" | jq '. | length')
    
    if [[ "${ollama_count}" -gt 0 ]]; then
        echo "🦙 Ollama models (${ollama_count}):"
        echo "${ollama_models}" | jq -r '.[]' | sed 's/^/   - /'
    else
        echo "🦙 Ollama models: None found (is Ollama running?)"
    fi
    
    # Get OpenRouter models if configured
    local openrouter_models=$(opencode_get_models openrouter) 
    local openrouter_count=$(echo "${openrouter_models}" | jq '. | length')
    
    if [[ "${openrouter_count}" -gt 0 ]]; then
        echo "🔄 OpenRouter models (${openrouter_count}):"
        echo "${openrouter_models}" | jq -r '.[]' | sed 's/^/   - /'
    else
        echo "🔄 OpenRouter models: None configured"
    fi
    
    # Show custom models list if it exists
    local models_file="${OPENCODE_DATA_DIR}/available-models.json"
    if [[ -f "${models_file}" ]]; then
        local custom_models=$(cat "${models_file}")
        local custom_count=$(echo "${custom_models}" | jq '. | length')
        if [[ "${custom_count}" -gt 0 ]]; then
            echo "📝 Custom model references (${custom_count}):"
            echo "${custom_models}" | jq -r '.[]' | sed 's/^/   - /'
        fi
    fi
}

opencode::content::get() {
    local content_type="${1:-}"
    local name="${2:-}"
    
    if [[ -z "${content_type}" ]]; then
        log::error "Content type is required"
        log::info "Usage: opencode content get <type> <name>"
        return 1
    fi
    
    case "${content_type}" in
        config|configuration)
            opencode::content::get_config "${name}"
            ;;
        model)
            opencode::content::get_model "${name}"
            ;;
        *)
            log::error "Unknown content type: ${content_type}"
            return 1
            ;;
    esac
}

opencode::content::get_config() {
    local config_name="${1:-active}"
    
    if [[ "${config_name}" == "active" ]]; then
        if [[ -f "${OPENCODE_CONFIG_FILE}" ]]; then
            log::info "Active OpenCode configuration:"
            cat "${OPENCODE_CONFIG_FILE}" | jq .
        else
            log::error "No active configuration found"
            return 1
        fi
    else
        local config_file="${OPENCODE_DATA_DIR}/config-${config_name}.json"
        if [[ -f "${config_file}" ]]; then
            log::info "OpenCode configuration '${config_name}':"
            cat "${config_file}" | jq .
        else
            log::error "Configuration '${config_name}' not found"
            return 1
        fi
    fi
}

opencode::content::get_model() {
    local model_name="${1:-}"
    
    if [[ -z "${model_name}" ]]; then
        log::error "Model name is required"
        return 1
    fi
    
    # Check if model exists in available models
    local all_models=$(opencode_get_models)
    if echo "${all_models}" | jq -e --arg model "${model_name}" 'any(. == $model)' >/dev/null; then
        log::success "Model '${model_name}' is available"
        log::info "Model: ${model_name}"
        
        # Show additional info if it's an Ollama model
        if command -v ollama &>/dev/null && ollama list 2>/dev/null | grep -q "^${model_name}"; then
            echo "Provider: Ollama"
            echo "Status: Installed locally"
            ollama show "${model_name}" 2>/dev/null | head -10 || true
        else
            echo "Provider: Remote/API"
        fi
    else
        log::error "Model '${model_name}' not found in available models"
        log::info "Use 'opencode content list models' to see available models"
        return 1
    fi
}

opencode::content::remove() {
    local content_type="${1:-}"
    local name="${2:-}"
    
    if [[ -z "${content_type}" || -z "${name}" ]]; then
        log::error "Content type and name are required"
        log::info "Usage: opencode content remove <type> <name>"
        return 1
    fi
    
    case "${content_type}" in
        config|configuration)
            opencode::content::remove_config "${name}"
            ;;
        model)
            opencode::content::remove_model "${name}"
            ;;
        *)
            log::error "Unknown content type: ${content_type}"
            return 1
            ;;
    esac
}

opencode::content::remove_config() {
    local config_name="${1}"
    
    if [[ "${config_name}" == "active" ]]; then
        log::error "Cannot remove active configuration"
        log::info "Create a new configuration and activate it first"
        return 1
    fi
    
    local config_file="${OPENCODE_DATA_DIR}/config-${config_name}.json"
    if [[ -f "${config_file}" ]]; then
        rm "${config_file}"
        log::success "Configuration '${config_name}' removed"
    else
        log::error "Configuration '${config_name}' not found"
        return 1
    fi
}

opencode::content::remove_model() {
    local model_name="${1}"
    
    # Remove from custom models list
    local models_file="${OPENCODE_DATA_DIR}/available-models.json"
    if [[ -f "${models_file}" ]]; then
        local models=$(cat "${models_file}")
        local updated_models=$(echo "${models}" | jq --arg model "${model_name}" 'map(select(. != $model))')
        echo "${updated_models}" > "${models_file}"
        log::success "Model '${model_name}' removed from custom models list"
    else
        log::error "No custom models list found"
        return 1
    fi
}

# Custom subcommand: activate configuration
opencode::content::activate() {
    local config_name="${1:-}"
    
    if [[ -z "${config_name}" ]]; then
        log::error "Configuration name is required"
        log::info "Usage: opencode content activate <config_name>"
        return 1
    fi
    
    local config_file="${OPENCODE_DATA_DIR}/config-${config_name}.json"
    if [[ -f "${config_file}" ]]; then
        cp "${config_file}" "${OPENCODE_CONFIG_FILE}"
        log::success "Configuration '${config_name}' activated"
        log::info "New active configuration:"
        opencode::content::get_config active
    else
        log::error "Configuration '${config_name}' not found"
        return 1
    fi
}
