#!/bin/bash
# Gemini injection functionality

# Get script directory
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
GEMINI_INJECT_DIR="${RESOURCE_DIR}/lib"

# Source dependencies
source "${GEMINI_INJECT_DIR}/core.sh"
source "${REPO_ROOT}/scripts/lib/utils/log.sh"

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
