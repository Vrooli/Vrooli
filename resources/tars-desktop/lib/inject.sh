#!/bin/bash
# TARS-desktop injection functionality

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TARS_DESKTOP_INJECT_DIR="${APP_ROOT}/resources/tars-desktop/lib"

# Source dependencies
source "${TARS_DESKTOP_INJECT_DIR}/core.sh"

# Main inject function
tars_desktop::inject() {
    local target="${1:-}"
    local data="${2:-}"
    local verbose="${3:-false}"
    
    if [[ -z "$target" ]]; then
        log::error "Target resource not specified"
        return 1
    fi
    
    # Initialize
    tars_desktop::init >/dev/null 2>&1
    
    # Check if running
    if ! tars_desktop::is_running; then
        log::warn "TARS-desktop is not running. Starting it first..."
        source "${TARS_DESKTOP_INJECT_DIR}/start.sh"
        tars_desktop::start "$verbose" || return 1
    fi
    
    case "$target" in
        n8n|node-red|huginn)
            # Inject automation workflow
            tars_desktop::inject_automation "$target" "$data" "$verbose"
            ;;
        ollama|openrouter)
            # Configure AI provider for TARS
            tars_desktop::inject_ai_provider "$target" "$data" "$verbose"
            ;;
        *)
            log::error "Unsupported injection target: $target"
            return 1
            ;;
    esac
}

# Inject automation workflow
tars_desktop::inject_automation() {
    local target="$1"
    local workflow_file="$2"
    local verbose="${3:-false}"
    
    log::info "Injecting TARS-desktop workflow into $target"
    
    # Default workflow if not provided
    if [[ -z "$workflow_file" ]]; then
        workflow_file="${TARS_DESKTOP_RESOURCE_DIR}/examples/ui-automation-workflow.json"
        
        # Create example workflow
        cat > "$workflow_file" <<'EOF'
{
  "name": "TARS Desktop UI Automation",
  "description": "Execute UI automation tasks using TARS",
  "nodes": [
    {
      "id": "tars_trigger",
      "type": "webhook",
      "parameters": {
        "path": "/tars/execute",
        "method": "POST"
      }
    },
    {
      "id": "tars_action",
      "type": "http",
      "parameters": {
        "url": "http://localhost:11570/execute",
        "method": "POST",
        "headers": {
          "Content-Type": "application/json"
        },
        "body": "{{$json}}"
      }
    }
  ],
  "connections": [
    {
      "source": "tars_trigger",
      "target": "tars_action"
    }
  ]
}
EOF
    fi
    
    # Inject into target automation platform
    case "$target" in
        n8n)
            if command -v resource-n8n >/dev/null 2>&1; then
                resource-n8n inject "$workflow_file"
            else
                log::error "n8n resource CLI not found"
                return 1
            fi
            ;;
        huginn)
            if command -v resource-huginn >/dev/null 2>&1; then
                resource-huginn import-scenario "$workflow_file"
            else
                log::error "Huginn resource CLI not found"
                return 1
            fi
            ;;
        *)
            log::warn "Injection for $target not yet implemented"
            ;;
    esac
    
    log::success "TARS-desktop workflow injected into $target"
}

# Configure AI provider
tars_desktop::inject_ai_provider() {
    local provider="$1"
    local config_data="$2"
    local verbose="${3:-false}"
    
    log::info "Configuring TARS-desktop to use $provider"
    
    case "$provider" in
        ollama)
            # Configure to use local Ollama
            export TARS_DESKTOP_MODEL_PROVIDER="ollama"
            export TARS_DESKTOP_API_BASE="http://localhost:11434"
            
            # Check if Ollama has vision models
            if command -v resource-ollama >/dev/null 2>&1; then
                local vision_models
                vision_models=$(resource-ollama list-models | grep -E "llava|vision" | head -1 | awk '{print $1}')
                if [[ -n "$vision_models" ]]; then
                    export TARS_DESKTOP_MODEL="$vision_models"
                    log::info "Using Ollama vision model: $vision_models"
                else
                    log::warn "No vision models found in Ollama. TARS may have limited functionality"
                fi
            fi
            ;;
        openrouter)
            # Configure to use OpenRouter
            export TARS_DESKTOP_MODEL_PROVIDER="openrouter"
            export TARS_DESKTOP_API_BASE="https://openrouter.ai/api/v1"
            
            # Get API key from Vault or environment
            if [[ -z "$TARS_DESKTOP_API_KEY" ]]; then
                tars_desktop::init "$verbose"
            fi
            
            if [[ -z "$TARS_DESKTOP_API_KEY" || "$TARS_DESKTOP_API_KEY" == "sk-placeholder-key" ]]; then
                log::error "Valid OpenRouter API key required"
                return 1
            fi
            ;;
    esac
    
    # Restart to apply configuration
    source "${TARS_DESKTOP_INJECT_DIR}/start.sh"
    tars_desktop::restart "$verbose"
    
    log::success "TARS-desktop configured to use $provider"
}

# Export functions
export -f tars_desktop::inject
export -f tars_desktop::inject_automation
export -f tars_desktop::inject_ai_provider
# Wrapper function for compatibility
tars_desktop_inject() {
    tars_desktop::inject "$@"
}
export -f tars_desktop_inject
