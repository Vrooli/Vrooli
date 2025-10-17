#!/usr/bin/env bash
# LiteLLM Adapter Connection Manager
# Establishes connection to use LiteLLM as Claude Code backend

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
ADAPTER_DIR="${APP_ROOT}/resources/claude-code/adapters/litellm"

# Source required libraries
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${ADAPTER_DIR}/state.sh"
# shellcheck disable=SC1091
source "${ADAPTER_DIR}/status.sh"

#######################################
# Connect to LiteLLM backend
# Arguments:
#   --model <model>              - Model to use (optional)
#   --auto-disconnect-in <hours> - Hours until auto-disconnect (optional)
#   --reason <reason>            - Connection reason (optional)
#   --force                      - Force connection even if already connected
# Returns: 0 on success, 1 on failure
#######################################
litellm::connect() {
    local model=""
    local auto_hours=""
    local reason="manual"
    local force="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --model)
                model="$2"
                shift 2
                ;;
            --auto-disconnect-in)
                auto_hours="$2"
                shift 2
                ;;
            --reason)
                reason="$2"
                shift 2
                ;;
            --force)
                force="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    log::header "🔌 Connecting to LiteLLM Backend"
    
    # Check if already connected
    if litellm::is_connected; then
        if [[ "$force" != "true" ]]; then
            log::warn "Already connected to LiteLLM"
            log::info "Use --force to reconnect"
            return 0
        else
            log::info "Force reconnecting..."
            litellm::disconnect "reconnect"
        fi
    fi
    
    # Check LiteLLM availability
    log::info "Checking LiteLLM resource availability..."
    if ! litellm::check_resource_available; then
        log::error "LiteLLM resource is not available or not healthy"
        log::info "Please ensure LiteLLM is running:"
        log::info "  1. Check if installed: vrooli resource list"
        log::info "  2. Start the service: resource-litellm start"
        log::info "  3. Verify status: resource-litellm status"
        return 1
    fi
    log::success "✅ LiteLLM resource is available"
    
    # Test connection
    log::info "Testing LiteLLM connection..."
    if ! litellm::test_connection; then
        log::error "Failed to establish connection with LiteLLM"
        log::info "Please check LiteLLM configuration and API keys"
        return 1
    fi
    log::success "✅ Connection test successful"
    
    # Set model if specified
    if [[ -n "$model" ]]; then
        log::info "Setting preferred model: $model"
        litellm::set_config "preferred_model" "$model"
    fi
    
    # Set auto-disconnect hours if specified
    if [[ -n "$auto_hours" ]]; then
        log::info "Setting auto-disconnect: $auto_hours hours"
        litellm::set_config "auto_disconnect_after_hours" "$auto_hours"
    fi
    
    # Update state to connected
    litellm::set_connected "true" "$reason"
    
    # Get the auto-disconnect time
    local disconnect_time
    disconnect_time=$(jq -r '.auto_disconnect_at // null' "${CLAUDE_CONFIG_DIR:-$HOME/.claude}/litellm_state.json")
    
    log::success "✅ Successfully connected to LiteLLM backend"
    echo
    log::info "📝 Connection Details:"
    log::info "   Model: $(litellm::get_config preferred_model)"
    log::info "   Auto-disconnect: $disconnect_time"
    log::info "   Reason: $reason"
    echo
    log::info "💡 Usage:"
    log::info "   Claude Code will now route requests through LiteLLM"
    log::info "   To disconnect: resource-claude-code for litellm disconnect"
    log::info "   To check status: resource-claude-code for litellm status"
    
    return 0
}

# Main execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    litellm::connect "$@"
fi

# Export function
export -f litellm::connect