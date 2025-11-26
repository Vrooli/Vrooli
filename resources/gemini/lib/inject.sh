#!/bin/bash
# Gemini injection functionality

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
GEMINI_INJECT_DIR="${APP_ROOT}/resources/gemini/lib"

# Source dependencies
source "${GEMINI_INJECT_DIR}/core.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Inject Gemini configuration into other resources
gemini::inject() {
    local target="${1:-}"
    local data="${2:-}"
    
    if [[ -z "$target" ]]; then
        log::error "Target resource required for injection"
        return 1
    fi
    
    # Initialize to get API key
    gemini::init >/dev/null 2>&1
    
    case "$target" in
        n8n|huginn|node-red)
            # Inject API credentials for automation platforms
            local injection_data="{
                \"type\": \"gemini\",
                \"name\": \"Google Gemini API\",
                \"credentials\": {
                    \"apiKey\": \"${GEMINI_API_KEY}\",
                    \"baseUrl\": \"${GEMINI_API_BASE}\",
                    \"defaultModel\": \"${GEMINI_DEFAULT_MODEL}\"
                }
            }"
            
            # Call target's injection handler if available
            if command -v "resource-${target}" >/dev/null 2>&1; then
                echo "$injection_data" | resource-"${target}" inject gemini -
                log::success "Injected Gemini credentials into ${target}"
            else
                log::warn "${target} resource not available for injection"
            fi
            ;;
        ollama)
            # Gemini can't be injected into Ollama (different API types)
            log::error "Gemini cannot be injected into Ollama (incompatible APIs)"
            return 1
            ;;
        *)
            log::error "Unknown injection target: ${target}"
            return 1
            ;;
    esac
    
    return 0
}

# Export function
export -f gemini::inject
