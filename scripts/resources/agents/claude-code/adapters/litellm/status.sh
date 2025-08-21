#!/usr/bin/env bash
# LiteLLM Adapter Status Check
# Verifies LiteLLM availability and connection status

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

#######################################
# Check if LiteLLM resource is available
# Returns: 0 if available, 1 otherwise
#######################################
litellm::check_resource_available() {
    # Check if litellm resource CLI exists
    if command -v resource-litellm &>/dev/null; then
        # Check if it's actually running by looking for the health check
        local status_output
        if status_output=$(resource-litellm status 2>&1); then
            # Look for indicators that it's running
            if echo "$status_output" | grep -q "is running\|API is healthy"; then
                return 0
            fi
        fi
    fi
    
    return 1
}

#######################################
# Test LiteLLM connection with a simple prompt
# Returns: 0 if successful, 1 otherwise
#######################################
litellm::test_connection() {
    local test_prompt="${1:-Hello, please respond with 'OK' if you can hear me.}"
    
    if ! litellm::check_resource_available; then
        log::error "LiteLLM resource is not available"
        return 1
    fi
    
    # Get model mapping
    local model
    model=$(litellm::get_model_mapping "claude-3-5-sonnet-latest")
    
    # Try to execute a simple test prompt
    local response
    if response=$(resource-litellm run \
        --model "$model" \
        --prompt "$test_prompt" \
        --max-tokens 10 \
        --temperature 0 \
        --format json 2>/dev/null); then
        
        # Check if response contains expected output
        if echo "$response" | grep -q "OK\|ok\|Ok"; then
            return 0
        fi
    fi
    
    return 1
}

#######################################
# Get comprehensive status information
# Outputs: JSON status object
#######################################
litellm::get_full_status() {
    # Initialize state
    litellm::init_state
    
    # Check basic availability
    local resource_available="false"
    local resource_status="Not installed"
    local can_connect="false"
    local connection_test_result="Not tested"
    
    if litellm::check_resource_available; then
        resource_available="true"
        resource_status="Available"
        
        # Test actual connection
        if litellm::test_connection "test" &>/dev/null; then
            can_connect="true"
            connection_test_result="Success"
        else
            connection_test_result="Failed"
        fi
    fi
    
    # Get current state
    local state_json
    state_json=$(litellm::get_state)
    
    # Get connection info
    local is_connected
    is_connected=$(echo "$state_json" | jq -r '.connected // false')
    
    local connection_time
    connection_time=$(echo "$state_json" | jq -r '.connection_time // null')
    
    local auto_disconnect_at
    auto_disconnect_at=$(echo "$state_json" | jq -r '.auto_disconnect_at // null')
    
    local requests_handled
    requests_handled=$(echo "$state_json" | jq -r '.requests_handled // 0')
    
    local requests_failed
    requests_failed=$(echo "$state_json" | jq -r '.requests_failed // 0')
    
    # Calculate success rate
    local total_requests=$((requests_handled + requests_failed))
    local success_rate="N/A"
    if [[ $total_requests -gt 0 ]]; then
        success_rate=$(echo "scale=2; $requests_handled * 100 / $total_requests" | bc)
    fi
    
    # Check if should auto-disconnect
    local should_disconnect="false"
    if litellm::should_auto_disconnect; then
        should_disconnect="true"
    fi
    
    # Get configuration
    local auto_fallback
    auto_fallback=$(litellm::get_config "auto_fallback_enabled")
    
    local preferred_model
    preferred_model=$(litellm::get_config "preferred_model")
    
    # Build status JSON
    cat <<-EOF
	{
	    "resource_available": $resource_available,
	    "resource_status": "$resource_status",
	    "can_connect": $can_connect,
	    "connection_test": "$connection_test_result",
	    "is_connected": $is_connected,
	    "connection_time": $([ "$connection_time" = "null" ] && echo "null" || echo "\"$connection_time\""),
	    "auto_disconnect_at": $([ "$auto_disconnect_at" = "null" ] && echo "null" || echo "\"$auto_disconnect_at\""),
	    "should_auto_disconnect": $should_disconnect,
	    "requests_handled": $requests_handled,
	    "requests_failed": $requests_failed,
	    "success_rate": "$success_rate",
	    "auto_fallback_enabled": $auto_fallback,
	    "preferred_model": "$preferred_model",
	    "last_error": $(echo "$state_json" | jq '.last_error'),
	    "last_error_time": $(echo "$state_json" | jq '.last_error_time')
	}
	EOF
}

#######################################
# Display status in human-readable format
#######################################
litellm::display_status() {
    local status_json
    status_json=$(litellm::get_full_status)
    
    log::header "🔌 LiteLLM Adapter Status"
    echo
    
    # Resource availability
    local resource_available
    resource_available=$(echo "$status_json" | jq -r '.resource_available')
    
    if [[ "$resource_available" == "true" ]]; then
        log::success "✅ LiteLLM Resource: Available"
        
        local can_connect
        can_connect=$(echo "$status_json" | jq -r '.can_connect')
        if [[ "$can_connect" == "true" ]]; then
            log::success "✅ Connection Test: Passed"
        else
            log::warn "⚠️  Connection Test: Failed"
        fi
    else
        log::error "❌ LiteLLM Resource: Not available"
        log::info "   To install LiteLLM:"
        log::info "   1. Run: vrooli resource list"
        log::info "   2. Check if 'litellm' is available"
        log::info "   3. If available, run: resource-litellm start"
        log::info "   4. If not available, LiteLLM may need to be added to your Vrooli setup"
        echo
        return 1
    fi
    
    echo
    
    # Connection status
    local is_connected
    is_connected=$(echo "$status_json" | jq -r '.is_connected')
    
    if [[ "$is_connected" == "true" ]]; then
        log::info "🔗 Connection Status: Connected"
        
        local connection_time
        connection_time=$(echo "$status_json" | jq -r '.connection_time')
        if [[ "$connection_time" != "null" ]]; then
            log::info "   Connected since: $connection_time"
        fi
        
        local auto_disconnect_at
        auto_disconnect_at=$(echo "$status_json" | jq -r '.auto_disconnect_at')
        if [[ "$auto_disconnect_at" != "null" ]]; then
            log::info "   Auto-disconnect at: $auto_disconnect_at"
        fi
        
        echo
        log::info "📊 Request Statistics:"
        log::info "   Handled: $(echo "$status_json" | jq -r '.requests_handled')"
        log::info "   Failed: $(echo "$status_json" | jq -r '.requests_failed')"
        log::info "   Success Rate: $(echo "$status_json" | jq -r '.success_rate')%"
    else
        log::info "🔗 Connection Status: Not connected"
        log::info "   Connect with: resource-claude-code for litellm connect"
    fi
    
    echo
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   Auto-fallback: $(echo "$status_json" | jq -r '.auto_fallback_enabled')"
    log::info "   Preferred Model: $(echo "$status_json" | jq -r '.preferred_model')"
    
    # Last error
    local last_error
    last_error=$(echo "$status_json" | jq -r '.last_error // null')
    if [[ "$last_error" != "null" ]]; then
        echo
        log::warn "⚠️  Last Error: $last_error"
        log::info "   Time: $(echo "$status_json" | jq -r '.last_error_time')"
    fi
}

# Main execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Parse arguments
    FORMAT="${1:-text}"
    
    case "$FORMAT" in
        json|--json)
            litellm::get_full_status
            ;;
        text|--text|*)
            litellm::display_status
            ;;
    esac
fi

# Export functions
export -f litellm::check_resource_available
export -f litellm::test_connection
export -f litellm::get_full_status
export -f litellm::display_status