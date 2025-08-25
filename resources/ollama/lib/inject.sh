#!/usr/bin/env bash
################################################################################
# Ollama Injection Framework
#
# Allows scenarios to inject model requirements into Ollama
# Supports .json files with model pull instructions
#
# Injection file format:
# {
#   "models": [
#     {"name": "llama3.2:3b", "alias": "llama-small"},
#     {"name": "qwen2.5-coder:7b"},
#     {"name": "llava:7b", "alias": "vision"}
#   ]
# }
################################################################################

set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OLLAMA_INJECT_DIR="${APP_ROOT}/resources/ollama/lib"

# Source dependencies
# shellcheck disable=SC1091
source "${OLLAMA_INJECT_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${OLLAMA_INJECT_DIR}/core.sh"

#######################################
# Main injection handler
# Arguments:
#   $1 - Path to injection file (.json)
# Returns:
#   0 on success, 1 on failure
#######################################
ollama::inject() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        log::error "Injection file not found: $file"
        return 1
    fi
    
    # Validate JSON
    if ! jq empty "$file" 2>/dev/null; then
        log::error "Invalid JSON in injection file: $file"
        return 1
    fi
    
    # Check if Ollama is running
    if ! ollama::is_running; then
        log::warn "Ollama is not running. Starting it..."
        if ! ollama::start; then
            log::error "Failed to start Ollama"
            return 1
        fi
    fi
    
    log::info "Processing Ollama injection from: $(basename "$file")"
    
    # Extract models array
    local models
    models=$(jq -r '.models[]? | @json' "$file" 2>/dev/null || echo "")
    
    if [[ -z "$models" ]]; then
        log::warn "No models specified in injection file"
        return 0
    fi
    
    local success_count=0
    local fail_count=0
    
    # Process each model
    while IFS= read -r model_json; do
        local model_name
        local model_alias
        
        model_name=$(echo "$model_json" | jq -r '.name // empty')
        model_alias=$(echo "$model_json" | jq -r '.alias // empty')
        
        if [[ -z "$model_name" ]]; then
            log::warn "Skipping model with no name"
            ((fail_count++))
            continue
        fi
        
        log::info "📦 Pulling model: $model_name"
        
        # Pull the model
        if timeout 600 docker exec -i ollama ollama pull "$model_name" 2>&1 | while IFS= read -r line; do
            # Show progress for large downloads
            if [[ "$line" == *"pulling"* ]] || [[ "$line" == *"%"* ]]; then
                echo -ne "\r    $line" >&2
            fi
        done; then
            echo "" >&2  # Clear progress line
            log::success "   ✅ Model pulled: $model_name"
            ((success_count++))
            
            # Create alias if specified
            if [[ -n "$model_alias" ]]; then
                log::info "   🏷️  Creating alias: $model_alias → $model_name"
                if docker exec -i ollama ollama cp "$model_name" "$model_alias" 2>/dev/null; then
                    log::success "   ✅ Alias created: $model_alias"
                else
                    log::warn "   ⚠️  Failed to create alias: $model_alias"
                fi
            fi
        else
            echo "" >&2  # Clear progress line
            log::error "   ❌ Failed to pull model: $model_name"
            ((fail_count++))
        fi
    done <<< "$models"
    
    # Summary
    echo
    log::info "📊 Injection Summary:"
    log::info "   ✅ Successful: $success_count"
    if [[ $fail_count -gt 0 ]]; then
        log::warn "   ❌ Failed: $fail_count"
    fi
    
    # List current models
    echo
    log::info "📋 Current models in Ollama:"
    ollama::list_models 2>/dev/null || true
    
    # Return success if at least one model was pulled
    [[ $success_count -gt 0 ]] && return 0 || return 1
}

#######################################
# List models (helper for validation)
#######################################
ollama::list_models() {
    if ! ollama::is_running; then
        log::error "Ollama is not running"
        return 1
    fi
    
    docker exec -i ollama ollama list 2>/dev/null | while IFS= read -r line; do
        echo "   $line"
    done
}

# Export functions for use by CLI
export -f ollama::inject
export -f ollama::list_models