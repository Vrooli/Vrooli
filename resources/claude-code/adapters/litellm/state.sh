#!/usr/bin/env bash
# LiteLLM Adapter State Management
# Manages connection state and fallback configuration

set -euo pipefail

# State file location
LITELLM_STATE_FILE="${CLAUDE_CONFIG_DIR:-$HOME/.claude}/litellm_state.json"
LITELLM_CONFIG_FILE="${CLAUDE_CONFIG_DIR:-$HOME/.claude}/litellm_config.json"

#######################################
# Initialize state file if it doesn't exist
#######################################
litellm::init_state() {
    local state_dir
    state_dir=${LITELLM_STATE_FILE%/*}
    
    # Create directory if it doesn't exist
    mkdir -p "$state_dir"
    
    # Initialize state file with defaults
    if [[ ! -f "$LITELLM_STATE_FILE" ]]; then
        cat > "$LITELLM_STATE_FILE" <<-EOF
		{
		    "connected": false,
		    "connection_time": null,
		    "auto_disconnect_at": null,
		    "fallback_reason": null,
		    "fallback_triggered_at": null,
		    "model_mapping": "claude-3-5-sonnet-latest",
		    "requests_handled": 0,
		    "requests_failed": 0,
		    "last_request_time": null,
		    "last_error": null,
		    "last_error_time": null,
		    "connection_history": []
		}
		EOF
    fi
    
    # Initialize config file with defaults
    if [[ ! -f "$LITELLM_CONFIG_FILE" ]]; then
        cat > "$LITELLM_CONFIG_FILE" <<-EOF
		{
		    "auto_fallback_enabled": true,
		    "auto_fallback_on_rate_limit": true,
		    "auto_fallback_on_error": false,
		    "auto_disconnect_after_hours": 5,
		    "preferred_model": "qwen2.5-coder",
		    "model_mappings": {
		        "claude-3-5-sonnet": "qwen2.5-coder",
		        "claude-3-5-sonnet-latest": "qwen2.5-coder",
		        "claude-3-5-sonnet-20241022": "qwen2.5-coder",
		        "claude-3-haiku": "qwen2.5",
		        "claude-3-haiku-20241022": "qwen2.5",
		        "claude-3-opus": "llama3.1-8b",
		        "claude-3-opus-20240229": "llama3.1-8b",
		        "claude-3-opus-latest": "llama3.1-8b"
		    },
		    "litellm_endpoint": null,
		    "litellm_api_key_env": "LITELLM_API_KEY",
		    "cost_tracking": {
		        "enabled": true,
		        "native_claude_cost": 0.0,
		        "litellm_cost": 0.0,
		        "last_reset": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
		    },
		    "performance_metrics": {
		        "enabled": false,
		        "track_latency": true,
		        "track_quality": false
		    }
		}
		EOF
    fi
}

#######################################
# Get current connection state
# Returns: 0 if connected, 1 if not
#######################################
litellm::is_connected() {
    litellm::init_state
    
    local connected
    connected=$(jq -r '.connected // false' "$LITELLM_STATE_FILE" 2>/dev/null || echo "false")
    
    [[ "$connected" == "true" ]]
}

#######################################
# Set connection state
# Arguments:
#   $1 - true/false
#   $2 - reason (optional)
#######################################
litellm::set_connected() {
    local connected="$1"
    local reason="${2:-manual}"
    
    litellm::init_state
    
    local temp_file
    temp_file=$(mktemp)
    
    local current_time
    current_time=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    
    if [[ "$connected" == "true" ]]; then
        # Calculate auto-disconnect time
        local auto_hours
        auto_hours=$(jq -r '.auto_disconnect_after_hours // 5' "$LITELLM_CONFIG_FILE" 2>/dev/null || echo "5")
        local auto_disconnect
        auto_disconnect=$(date -u -d "+${auto_hours} hours" +%Y-%m-%dT%H:%M:%SZ)
        
        # Update connection state (avoiding complex jq syntax that causes bash parsing issues)
        jq --arg time "$current_time" \
           --arg disconnect "$auto_disconnect" \
           --arg reason "$reason" \
           '.connected = true |
            .connection_time = $time |
            .auto_disconnect_at = $disconnect |
            .fallback_reason = $reason |
            .fallback_triggered_at = $time |
            .requests_handled = 0 |
            .requests_failed = 0 |
            .last_error = null' \
           "$LITELLM_STATE_FILE" > "$temp_file"
        
        # Add new connection record to history (step by step to avoid bash parsing issues)
        # First create the new record
        echo "{\"connected_at\":\"$current_time\",\"reason\":\"$reason\",\"auto_disconnect_at\":\"$auto_disconnect\"}" > "${temp_file}.record"
        
        # Add it to the front of existing history
        jq --slurpfile new_record "${temp_file}.record" \
           '.connection_history = $new_record + (.connection_history // [])' \
           "$temp_file" > "${temp_file}.2"
        
        # Keep only last 50 records
        jq '.connection_history = .connection_history[0:50]' "${temp_file}.2" > "$LITELLM_STATE_FILE"
        
        rm -f "${temp_file}.record" "${temp_file}.2"
        
        rm -f "$temp_file"
    else
        # Record disconnection
        jq --arg time "$current_time" \
           --arg reason "$reason" \
           '.connected = false |
            .auto_disconnect_at = null |
            if .connection_time then
                .connection_history[0].disconnected_at = $time |
                .connection_history[0].disconnect_reason = $reason |
                .connection_history[0].total_requests = .requests_handled |
                .connection_history[0].failed_requests = .requests_failed
            else . end' \
           "$LITELLM_STATE_FILE" > "$temp_file" && mv "$temp_file" "$LITELLM_STATE_FILE"
    fi
}

#######################################
# Check if auto-disconnect time has passed
# Returns: 0 if should disconnect, 1 otherwise
#######################################
litellm::should_auto_disconnect() {
    litellm::init_state
    
    local auto_disconnect
    auto_disconnect=$(jq -r '.auto_disconnect_at // null' "$LITELLM_STATE_FILE" 2>/dev/null)
    
    if [[ "$auto_disconnect" == "null" || -z "$auto_disconnect" ]]; then
        return 1
    fi
    
    local current_time
    current_time=$(date +%s)
    local disconnect_time
    disconnect_time=$(date -d "$auto_disconnect" +%s 2>/dev/null || echo "0")
    
    [[ $current_time -ge $disconnect_time ]]
}

#######################################
# Increment request counter
# Arguments:
#   $1 - success/failure
#######################################
litellm::increment_request_count() {
    local status="$1"
    
    litellm::init_state
    
    local temp_file
    temp_file=$(mktemp)
    local current_time
    current_time=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    
    if [[ "$status" == "success" ]]; then
        jq --arg time "$current_time" \
           '.requests_handled = (.requests_handled + 1) |
            .last_request_time = $time' \
           "$LITELLM_STATE_FILE" > "$temp_file" && mv "$temp_file" "$LITELLM_STATE_FILE"
    else
        jq --arg time "$current_time" \
           '.requests_failed = (.requests_failed + 1) |
            .last_request_time = $time' \
           "$LITELLM_STATE_FILE" > "$temp_file" && mv "$temp_file" "$LITELLM_STATE_FILE"
    fi
}

#######################################
# Record an error
# Arguments:
#   $1 - Error message
#######################################
litellm::record_error() {
    local error_msg="$1"
    
    litellm::init_state
    
    local temp_file
    temp_file=$(mktemp)
    local current_time
    current_time=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    
    jq --arg error "$error_msg" \
       --arg time "$current_time" \
       '.last_error = $error |
        .last_error_time = $time' \
       "$LITELLM_STATE_FILE" > "$temp_file" && mv "$temp_file" "$LITELLM_STATE_FILE"
}

#######################################
# Get configuration value
# Arguments:
#   $1 - Config key (dot notation)
# Outputs: Config value
#######################################
litellm::get_config() {
    local key="$1"
    
    litellm::init_state
    
    # Use 'has' to check if key exists, then get its value
    # This properly handles false boolean values
    if jq -e "has(\"${key#*.}\") or (.$key | . != null)" "$LITELLM_CONFIG_FILE" &>/dev/null; then
        jq -r ".$key" "$LITELLM_CONFIG_FILE" 2>/dev/null
    else
        echo "null"
    fi
}

#######################################
# Set configuration value
# Arguments:
#   $1 - Config key (dot notation)
#   $2 - Value
#######################################
litellm::set_config() {
    local key="$1"
    local value="$2"
    
    litellm::init_state
    
    local temp_file
    temp_file=$(mktemp)
    
    # Handle different value types
    if [[ "$value" =~ ^[0-9]+$ ]] || [[ "$value" =~ ^[0-9]+\.[0-9]+$ ]]; then
        # Numeric value
        jq ".$key = $value" "$LITELLM_CONFIG_FILE" > "$temp_file"
    elif [[ "$value" == "true" ]] || [[ "$value" == "false" ]]; then
        # Boolean value
        jq ".$key = $value" "$LITELLM_CONFIG_FILE" > "$temp_file"
    else
        # String value
        jq --arg val "$value" ".$key = \$val" "$LITELLM_CONFIG_FILE" > "$temp_file"
    fi
    
    mv "$temp_file" "$LITELLM_CONFIG_FILE"
}

#######################################
# Get current state as JSON
# Outputs: Full state JSON
#######################################
litellm::get_state() {
    litellm::init_state
    
    # Merge state and config for comprehensive view
    local state_json
    state_json=$(cat "$LITELLM_STATE_FILE")
    local config_json
    config_json=$(cat "$LITELLM_CONFIG_FILE")
    
    echo "$state_json" | jq --argjson config "$config_json" '. + {config: $config}'
}

#######################################
# Get model mapping
# Arguments:
#   $1 - Claude model name
# Outputs: LiteLLM model name
#######################################
litellm::get_model_mapping() {
    local claude_model="$1"
    
    litellm::init_state
    
    # First check if there's a direct mapping
    local mapped_model
    mapped_model=$(jq -r ".model_mappings.\"$claude_model\" // null" "$LITELLM_CONFIG_FILE" 2>/dev/null)
    
    if [[ "$mapped_model" != "null" && -n "$mapped_model" ]]; then
        echo "$mapped_model"
    else
        # Use the preferred model as fallback
        litellm::get_config "preferred_model"
    fi
}

#######################################
# Update cost tracking
# Arguments:
#   $1 - Service type (native/litellm)
#   $2 - Cost amount
#######################################
litellm::track_cost() {
    local service="$1"
    local cost="$2"
    
    litellm::init_state
    
    local temp_file
    temp_file=$(mktemp)
    
    if [[ "$service" == "native" ]]; then
        jq ".cost_tracking.native_claude_cost = (.cost_tracking.native_claude_cost + $cost)" \
           "$LITELLM_CONFIG_FILE" > "$temp_file"
    else
        jq ".cost_tracking.litellm_cost = (.cost_tracking.litellm_cost + $cost)" \
           "$LITELLM_CONFIG_FILE" > "$temp_file"
    fi
    
    mv "$temp_file" "$LITELLM_CONFIG_FILE"
}

#######################################
# Reset state (for testing or manual reset)
#######################################
litellm::reset_state() {
    litellm::init_state
    
    # Reset state file
    cat > "$LITELLM_STATE_FILE" <<-EOF
	{
	    "connected": false,
	    "connection_time": null,
	    "auto_disconnect_at": null,
	    "fallback_reason": null,
	    "fallback_triggered_at": null,
	    "model_mapping": "claude-3-5-sonnet-latest",
	    "requests_handled": 0,
	    "requests_failed": 0,
	    "last_request_time": null,
	    "last_error": null,
	    "last_error_time": null,
	    "connection_history": []
	}
	EOF
}

# Export functions for use by other scripts
export -f litellm::init_state
export -f litellm::is_connected
export -f litellm::set_connected
export -f litellm::should_auto_disconnect
export -f litellm::increment_request_count
export -f litellm::record_error
export -f litellm::get_config
export -f litellm::set_config
export -f litellm::get_state
export -f litellm::get_model_mapping
export -f litellm::track_cost
export -f litellm::reset_state