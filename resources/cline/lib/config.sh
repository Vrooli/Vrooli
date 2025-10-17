#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLINE_CONFIG_SCRIPT_DIR="${APP_ROOT}/resources/cline/lib"

# Source common functions
source "$CLINE_CONFIG_SCRIPT_DIR/common.sh"

# View or update Cline configuration
cline::config() {
    local action="${1:-view}"
    
    case "$action" in
        view|show|get)
            cline::config_view
            ;;
        set|update)
            shift
            cline::config_set "$@"
            ;;
        provider)
            shift
            cline::config_provider "$@"
            ;;
        *)
            log::error "Unknown config action: $action"
            echo "Usage: resource-cline config [view|set|provider]"
            return 1
            ;;
    esac
}

# View current configuration
cline::config_view() {
    log::info "Cline Configuration:"
    echo ""
    echo "Provider: $(cline::get_provider)"
    echo "Endpoint: $(cline::get_endpoint)"
    echo "Config Dir: $CLINE_CONFIG_DIR"
    echo "Data Dir: $CLINE_DATA_DIR"
    
    if [[ -f "$CLINE_SETTINGS_FILE" ]]; then
        echo ""
        log::info "Settings file content:"
        cat "$CLINE_SETTINGS_FILE" | jq . 2>/dev/null || cat "$CLINE_SETTINGS_FILE"
    fi
    
    return 0
}

# Set configuration value
cline::config_set() {
    local key="${1:-}"
    local value="${2:-}"
    
    if [[ -z "$key" ]] || [[ -z "$value" ]]; then
        log::error "Usage: resource-cline config set <key> <value>"
        return 1
    fi
    
    # Ensure settings file exists
    if [[ ! -f "$CLINE_SETTINGS_FILE" ]]; then
        echo "{}" > "$CLINE_SETTINGS_FILE"
    fi
    
    # Update using jq
    if command -v jq >/dev/null 2>&1; then
        jq --arg key "$key" --arg val "$value" '.[$key] = $val' "$CLINE_SETTINGS_FILE" > "$CLINE_SETTINGS_FILE.tmp" && \
            mv "$CLINE_SETTINGS_FILE.tmp" "$CLINE_SETTINGS_FILE"
        log::success "Configuration updated: $key = $value"
    else
        log::error "jq is required to update configuration"
        return 1
    fi
    
    return 0
}

# Change provider
cline::config_provider() {
    local provider="${1:-}"
    
    if [[ -z "$provider" ]]; then
        log::info "Current provider: $(cline::get_provider)"
        log::info "Available providers: ollama, openrouter"
        return 0
    fi
    
    case "$provider" in
        ollama|openrouter)
            echo "$provider" > "$CLINE_CONFIG_DIR/provider.conf"
            log::success "Provider changed to: $provider"
            
            # Check if provider is available
            if [[ "$provider" == "ollama" ]]; then
                if ! curl -s http://localhost:11434/api/version >/dev/null 2>&1; then
                    log::warn "Ollama is not running. Please start Ollama."
                fi
            elif [[ "$provider" == "openrouter" ]]; then
                if [[ -z "$(cline::get_api_key openrouter)" ]]; then
                    log::warn "OpenRouter API key not configured"
                fi
            fi
            ;;
        *)
            log::error "Unknown provider: $provider"
            log::info "Available providers: ollama, openrouter"
            return 1
            ;;
    esac
    
    return 0
}

# Main - only run if called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cline::config "$@"
fi