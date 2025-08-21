#!/usr/bin/env bash
# LiteLLM Adapter Execution Handler
# Routes Claude Code prompts through LiteLLM backend

set -euo pipefail

# Get script directory
ADAPTER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source required libraries
# shellcheck disable=SC1091
source "${ADAPTER_DIR}/../../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${ADAPTER_DIR}/state.sh"
# shellcheck disable=SC1091
source "${ADAPTER_DIR}/status.sh"
# shellcheck disable=SC1091
source "${ADAPTER_DIR}/translate.sh" 2>/dev/null || true

#######################################
# Execute a prompt through LiteLLM
# Arguments:
#   $1 - Prompt text
#   $2 - Max turns (optional, default 1)
#   $3 - Output format (optional, default "text")
#   $4 - Allowed tools (optional)
# Returns: 0 on success, 1 on failure
# Outputs: Response from LiteLLM
#######################################
litellm::execute() {
    local prompt="$1"
    local max_turns="${2:-1}"
    local output_format="${3:-text}"
    local allowed_tools="${4:-}"
    
    # Check if connected
    if ! litellm::is_connected; then
        log::error "Not connected to LiteLLM. Use: resource-claude-code for litellm connect"
        return 1
    fi
    
    # Check if LiteLLM is available
    if ! litellm::check_resource_available; then
        log::error "LiteLLM resource is not available"
        litellm::record_error "LiteLLM resource not available"
        litellm::increment_request_count "failure"
        return 1
    fi
    
    # Get model mapping
    local claude_model="${CLAUDE_MODEL:-claude-3-5-sonnet-latest}"
    local litellm_model
    litellm_model=$(litellm::get_model_mapping "$claude_model")
    
    log::debug "Executing through LiteLLM with model: $litellm_model"
    
    # Build LiteLLM command
    local litellm_cmd="resource-litellm run"
    litellm_cmd+=" --model '$litellm_model'"
    litellm_cmd+=" --prompt '$prompt'"
    
    # Add optional parameters
    if [[ -n "$max_turns" && "$max_turns" != "1" ]]; then
        # LiteLLM might not support max-turns directly, adapt as needed
        litellm_cmd+=" --max-tokens $((max_turns * 1000))"
    else
        litellm_cmd+=" --max-tokens 4000"
    fi
    
    # Handle tools if specified
    if [[ -n "$allowed_tools" ]]; then
        # Translate Claude tools to LiteLLM format
        local translated_tools
        translated_tools=$(litellm::translate_tools "$allowed_tools" 2>/dev/null || echo "")
        if [[ -n "$translated_tools" ]]; then
            litellm_cmd+=" --tools '$translated_tools'"
        fi
    fi
    
    # Add output format if JSON
    if [[ "$output_format" == "json" || "$output_format" == "stream-json" ]]; then
        litellm_cmd+=" --format json"
    fi
    
    # Execute the command
    local response
    local exit_code
    
    log::debug "Executing: $litellm_cmd"
    
    # Create a temp file for output
    local temp_output
    temp_output=$(mktemp)
    
    # Execute with timeout
    if timeout "${TIMEOUT:-600}" bash -c "$litellm_cmd" > "$temp_output" 2>&1; then
        exit_code=0
        response=$(cat "$temp_output")
        
        # Track successful request
        litellm::increment_request_count "success"
        
        # Estimate and track cost (rough estimate)
        local input_chars=${#prompt}
        local output_chars=${#response}
        local estimated_cost=$(echo "scale=6; ($input_chars + $output_chars) * 0.000001" | bc)
        litellm::track_cost "litellm" "$estimated_cost"
        
        # Output the response
        echo "$response"
    else
        exit_code=$?
        response=$(cat "$temp_output" 2>/dev/null || echo "Execution failed")
        
        # Track failed request
        litellm::increment_request_count "failure"
        litellm::record_error "LiteLLM execution failed: $response"
        
        log::error "LiteLLM execution failed with exit code: $exit_code"
        log::debug "Error output: $response"
        
        # Clean up and return error
        rm -f "$temp_output"
        return $exit_code
    fi
    
    # Clean up
    rm -f "$temp_output"
    
    # Check if we should auto-disconnect
    if litellm::should_auto_disconnect; then
        log::info "Auto-disconnect time reached, disconnecting from LiteLLM"
        # Source disconnect script and disconnect
        # shellcheck disable=SC1091
        source "${ADAPTER_DIR}/disconnect.sh"
        litellm::disconnect "auto_disconnect"
    fi
    
    return 0
}

#######################################
# Execute with automatic fallback
# This is called from main execute.sh when connected to LiteLLM
# Arguments: Same as claude_code::run environment variables
# Returns: 0 on success, 1 on failure
#######################################
litellm::execute_with_fallback() {
    local prompt="${PROMPT:-}"
    local max_turns="${MAX_TURNS:-5}"
    local output_format="${OUTPUT_FORMAT:-text}"
    local allowed_tools="${ALLOWED_TOOLS:-}"
    
    if [[ -z "$prompt" ]]; then
        log::error "No prompt provided"
        return 1
    fi
    
    log::info "🔄 Routing through LiteLLM backend"
    
    # Try to execute through LiteLLM
    if litellm::execute "$prompt" "$max_turns" "$output_format" "$allowed_tools"; then
        return 0
    else
        # If LiteLLM fails, we could fall back to native Claude if needed
        # But that would require checking if rate limits have cleared
        log::error "LiteLLM execution failed"
        
        # Check if we should disconnect due to repeated failures
        local state_json
        state_json=$(litellm::get_state)
        local failed_count
        failed_count=$(echo "$state_json" | jq -r '.requests_failed // 0')
        
        if [[ $failed_count -gt 3 ]]; then
            log::warn "Multiple failures detected, disconnecting from LiteLLM"
            # shellcheck disable=SC1091
            source "${ADAPTER_DIR}/disconnect.sh"
            litellm::disconnect "repeated_failures"
        fi
        
        return 1
    fi
}

#######################################
# Handle automatic fallback on rate limit
# Called when a rate limit is detected
# Arguments:
#   $1 - Rate limit info JSON (from detect_rate_limit)
# Returns: 0 if fallback activated, 1 otherwise
#######################################
litellm::auto_fallback_on_rate_limit() {
    local rate_info="${1:-}"
    
    # Check if auto-fallback is enabled
    local auto_fallback
    auto_fallback=$(litellm::get_config "auto_fallback_enabled")
    
    if [[ "$auto_fallback" != "true" ]]; then
        log::debug "Auto-fallback is disabled"
        return 1
    fi
    
    local auto_fallback_on_rate_limit
    auto_fallback_on_rate_limit=$(litellm::get_config "auto_fallback_on_rate_limit")
    
    if [[ "$auto_fallback_on_rate_limit" != "true" ]]; then
        log::debug "Auto-fallback on rate limit is disabled"
        return 1
    fi
    
    # Check if already connected
    if litellm::is_connected; then
        log::debug "Already connected to LiteLLM"
        return 0
    fi
    
    # Check if LiteLLM is available
    if ! litellm::check_resource_available; then
        log::warn "LiteLLM resource not available for fallback"
        return 1
    fi
    
    log::info "🔄 Activating LiteLLM fallback due to rate limit"
    
    # Extract reset time from rate info if available
    local reset_time
    reset_time=$(echo "$rate_info" | jq -r '.reset_time // ""' 2>/dev/null || echo "")
    
    # Calculate auto-disconnect time based on rate limit reset
    local auto_hours=5  # Default
    if [[ -n "$reset_time" && "$reset_time" != "null" ]]; then
        # Try to calculate hours until reset
        local reset_seconds
        reset_seconds=$(date -d "$reset_time" +%s 2>/dev/null || echo 0)
        local current_seconds
        current_seconds=$(date +%s)
        
        if [[ $reset_seconds -gt $current_seconds ]]; then
            local hours_until_reset=$(( (reset_seconds - current_seconds) / 3600 + 1 ))
            auto_hours=$hours_until_reset
            log::info "Setting auto-disconnect to $auto_hours hours (when rate limit resets)"
        fi
    fi
    
    # Connect to LiteLLM
    # shellcheck disable=SC1091
    source "${ADAPTER_DIR}/connect.sh"
    if litellm::connect --auto-disconnect-in "$auto_hours" --reason "rate_limit"; then
        log::success "✅ Successfully activated LiteLLM fallback"
        return 0
    else
        log::error "Failed to activate LiteLLM fallback"
        return 1
    fi
}

# Export functions
export -f litellm::execute
export -f litellm::execute_with_fallback
export -f litellm::auto_fallback_on_rate_limit