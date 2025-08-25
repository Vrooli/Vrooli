#!/usr/bin/env bash
# LiteLLM Adapter Status Check
# Verifies LiteLLM availability and connection status

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

#######################################
# Get LiteLLM resource status (uses enhanced resource-litellm status)
# Returns: JSON status from LiteLLM resource or error
#######################################
litellm::get_resource_status() {
    if ! command -v resource-litellm &>/dev/null; then
        echo '{"error": "LiteLLM CLI not available", "available": false}'
        return 1
    fi
    
    # Try to get JSON status without timeout first (avoids buffering issues)
    local status_json
    local temp_file
    temp_file=$(mktemp)
    
    # Run command in background with timeout
    (
        resource-litellm status --json 2>/dev/null > "$temp_file"
    ) &
    local pid=$!
    
    # Wait up to 25 seconds for the command
    local count=0
    while kill -0 $pid 2>/dev/null && [ $count -lt 25 ]; do
        sleep 1
        count=$((count + 1))
    done
    
    # Kill if still running
    if kill -0 $pid 2>/dev/null; then
        kill -9 $pid 2>/dev/null
        rm -f "$temp_file"
        echo '{"error": "Status command timed out", "available": false}'
        return 1
    fi
    
    # Read the output
    if [[ -f "$temp_file" ]] && [[ -s "$temp_file" ]]; then
        status_json=$(cat "$temp_file")
        rm -f "$temp_file"
        
        # Validate it's proper JSON and has expected fields
        if echo "$status_json" | jq -e '.container_running' >/dev/null 2>&1; then
            echo "$status_json"
            return 0
        fi
    fi
    
    rm -f "$temp_file"
    # Fallback: get basic status info
    echo '{"error": "Failed to get LiteLLM status", "available": false}'
    return 1
}

#######################################
# Check if LiteLLM resource is available (simplified check)
# Returns: 0 if available, 1 otherwise
#######################################
litellm::check_resource_available() {
    # First check if CLI is available
    if ! command -v resource-litellm &>/dev/null; then
        return 1
    fi
    
    # Quick check if container is running using docker directly
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^vrooli-litellm$"; then
        # Container is running, now do a simple API test
        # Use the resource's test command as it's more reliable than parsing JSON
        if timeout 15 resource-litellm test >/dev/null 2>&1; then
            return 0
        fi
    fi
    
    # If we get here, either container isn't running or test failed
    # Try the JSON status as a fallback (but don't rely on it)
    local status_json
    status_json=$(litellm::get_resource_status 2>/dev/null || echo '{"available": false}')
    
    # Check if container is running and API is healthy
    if echo "$status_json" | jq -e '.container_running == true and .api_healthy == true' >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

#######################################
# Test LiteLLM connection with a simple prompt
# Returns: 0 if successful, 1 otherwise
#######################################
litellm::test_connection() {
    local test_prompt="${1:-Hello, please respond with 'OK' if you can hear me.}"
    
    # Direct test without circular dependency on check_resource_available
    # First check if CLI is available
    if ! command -v resource-litellm &>/dev/null; then
        log::error "LiteLLM CLI not available"
        return 1
    fi
    
    # Check if container is running
    if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^vrooli-litellm$"; then
        log::error "LiteLLM container is not running"
        return 1
    fi
    
    # Use the resource's own test functionality with timeout
    if timeout 15 resource-litellm test >/dev/null 2>&1; then
        return 0
    fi
    
    log::error "LiteLLM test failed"
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
    
    # Get enhanced resource status information
    local resource_status_json="null"
    local available_models="null"
    local endpoint_health="null"
    local api_details="null"
    local api_response_time="null"
    
    if [[ "$resource_available" == "true" ]]; then
        # Get comprehensive status from LiteLLM resource
        local litellm_status
        if litellm_status=$(litellm::get_resource_status 2>/dev/null); then
            if ! echo "$litellm_status" | jq -e '.error' >/dev/null 2>&1; then
                resource_status_json="$litellm_status"
                
                # Extract specific information
                available_models=$(echo "$litellm_status" | jq '.available_models // null')
                endpoint_health=$(echo "$litellm_status" | jq '.endpoint_health // null')
                api_response_time=$(echo "$litellm_status" | jq '.api_response_time_ms // null')
                
                # Build API details from resource status
                local api_url api_key
                api_url=$(echo "$litellm_status" | jq -r '.api_url // ""')
                api_key=$(echo "$litellm_status" | jq -r '.master_key // ""')
                
                if [[ -n "$api_url" && -n "$api_key" ]]; then
                    api_details=$(jq -n \
                        --arg url "$api_url" \
                        --arg key "$api_key" \
                        '{
                            "url": $url,
                            "key": $key,
                            "available": true
                        }')
                fi
            fi
        fi
    fi
    
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
	    "last_error_time": $(echo "$state_json" | jq '.last_error_time'),
	    "api_details": $api_details,
	    "api_response_time_ms": $api_response_time,
	    "available_models": $available_models,
	    "endpoint_health": $endpoint_health,
	    "litellm_resource_status": $resource_status_json
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
    
    # API Details
    local api_details
    api_details=$(echo "$status_json" | jq -r '.api_details')
    if [[ "$api_details" != "null" ]]; then
        echo
        log::info "🔗 API Connection:"
        log::info "   URL: $(echo "$api_details" | jq -r '.url')"
        log::info "   Key: $(echo "$api_details" | jq -r '.key' | sed 's/.*-/***-/')"
        
        local response_time
        response_time=$(echo "$status_json" | jq -r '.api_response_time_ms')
        if [[ "$response_time" != "null" && -n "$response_time" ]]; then
            log::info "   Response Time: ${response_time}ms"
        fi
    fi
    
    # Available Models
    local available_models
    available_models=$(echo "$status_json" | jq -r '.available_models')
    if [[ "$available_models" != "null" ]]; then
        echo
        log::info "🤖 Available Models:"
        local model_count
        model_count=$(echo "$available_models" | jq '. | length')
        log::info "   Total: $model_count models"
        
        echo "$available_models" | jq -r '.[] | "   • " + .id + " (owned by: " + .owned_by + ")"' | head -10
        
        if [[ $model_count -gt 10 ]]; then
            log::info "   ... and $((model_count - 10)) more models"
        fi
        
        # Show model mappings
        echo
        log::info "🔄 Claude → LiteLLM Model Mappings:"
        local model_mappings
        model_mappings=$(litellm::get_config "model_mappings" 2>/dev/null || echo "{}")
        
        if [[ "$model_mappings" != "{}" && "$model_mappings" != "null" ]]; then
            echo "$model_mappings" | jq -r 'to_entries[] | "   " + .key + " → " + .value'
        else
            # Show default mappings
            log::info "   claude-3-5-sonnet → qwen2.5-coder"
            log::info "   claude-3-5-sonnet-latest → qwen2.5-coder"
            log::info "   claude-3-haiku → qwen2.5"
            log::info "   claude-3-opus → llama3.1-8b"
            log::info "   (default mappings - configure custom ones with 'config set')"
        fi
    fi
    
    # Endpoint Health
    local endpoint_health
    endpoint_health=$(echo "$status_json" | jq -r '.endpoint_health')
    if [[ "$endpoint_health" != "null" ]]; then
        echo
        log::info "💚 Endpoint Health:"
        
        local healthy_count unhealthy_count
        healthy_count=$(echo "$endpoint_health" | jq -r '.healthy_count // 0')
        unhealthy_count=$(echo "$endpoint_health" | jq -r '.unhealthy_count // 0')
        
        log::info "   Healthy: $healthy_count, Unhealthy: $unhealthy_count"
        
        # Show healthy endpoints
        if [[ $healthy_count -gt 0 ]]; then
            echo
            log::info "   Healthy Endpoints:"
            echo "$endpoint_health" | jq -r '.healthy_endpoints[]? | "   ✅ " + .model + (if .api_base then " (" + .api_base + ")" else "" end)'
        fi
        
        # Show unhealthy endpoints
        if [[ $unhealthy_count -gt 0 ]]; then
            echo
            log::warn "   Unhealthy Endpoints:"
            echo "$endpoint_health" | jq -r '.unhealthy_endpoints[]? | "   ❌ " + .model + (if .api_base then " (" + .api_base + ")" else "" end)'
        fi
    fi
    
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
export -f litellm::get_resource_status
export -f litellm::check_resource_available
export -f litellm::test_connection
export -f litellm::get_full_status
export -f litellm::display_status