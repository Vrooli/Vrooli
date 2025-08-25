#!/usr/bin/env bash
# LiteLLM Adapter Disconnection Manager
# Disconnects from LiteLLM and reverts to native Claude Code

set -euo pipefail

# T using cached value or compute once (4 levels up: resources/claude-code/adapters/litellm/disconnect.sh)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../../.." && builtin pwd)}"
ADAPTER_DIR="${APP_ROOT}/resources/claude-code/adapters/litellm"

# Source required libraries
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${ADAPTER_DIR}/state.sh"

#######################################
# Disconnect from LiteLLM backend
# Arguments:
#   $1 - Reason for disconnection (optional)
# Returns: 0 on success, 1 on failure
#######################################
litellm::disconnect() {
    local reason="${1:-manual}"
    
    log::header "🔌 Disconnecting from LiteLLM Backend"
    
    # Check if connected
    if ! litellm::is_connected; then
        log::info "Not currently connected to LiteLLM"
        return 0
    fi
    
    # Get current state for summary
    local state_json
    state_json=$(litellm::get_state)
    
    local requests_handled
    requests_handled=$(echo "$state_json" | jq -r '.requests_handled // 0')
    
    local requests_failed
    requests_failed=$(echo "$state_json" | jq -r '.requests_failed // 0')
    
    local connection_time
    connection_time=$(echo "$state_json" | jq -r '.connection_time // "unknown"')
    
    # Calculate session duration
    local duration="N/A"
    if [[ "$connection_time" != "unknown" && "$connection_time" != "null" ]]; then
        local start_seconds
        start_seconds=$(date -d "$connection_time" +%s 2>/dev/null || echo 0)
        local current_seconds
        current_seconds=$(date +%s)
        if [[ $start_seconds -gt 0 ]]; then
            local duration_seconds=$((current_seconds - start_seconds))
            local hours=$((duration_seconds / 3600))
            local minutes=$(((duration_seconds % 3600) / 60))
            duration="${hours}h ${minutes}m"
        fi
    fi
    
    # Update state to disconnected
    litellm::set_connected "false" "$reason"
    
    log::success "✅ Successfully disconnected from LiteLLM backend"
    echo
    log::info "📊 Session Summary:"
    log::info "   Duration: $duration"
    log::info "   Requests Handled: $requests_handled"
    log::info "   Requests Failed: $requests_failed"
    if [[ $((requests_handled + requests_failed)) -gt 0 ]]; then
        local success_rate
        success_rate=$(echo "scale=2; $requests_handled * 100 / ($requests_handled + $requests_failed)" | bc)
        log::info "   Success Rate: ${success_rate}%"
    fi
    log::info "   Disconnect Reason: $reason"
    echo
    log::info "💡 Claude Code will now use the native Claude CLI"
    
    # Check if this was due to rate limit recovery
    if [[ "$reason" == "rate_limit_recovered" ]]; then
        log::info "✨ Rate limits have reset - native Claude is available again"
    fi
    
    return 0
}

# Main execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    litellm::disconnect "$@"
fi

# Export function
export -f litellm::disconnect