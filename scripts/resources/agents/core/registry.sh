#!/usr/bin/env bash
################################################################################
# Agent Registry Core Functions
# 
# Generic registry management for all agent-capable resources
# Uses configuration passed by reference to customize behavior
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

#######################################
# Initialize agent registry if not exists
# Arguments:
#   $1 - Registry file path
# Returns:
#   0 on success
#######################################
agents::registry::init() {
    local registry_file="$1"
    local registry_dir
    
    registry_dir=$(dirname "$registry_file")
    
    # Create .vrooli directory if needed
    if [[ ! -d "$registry_dir" ]]; then
        mkdir -p "$registry_dir" || return 1
    fi
    
    # Create empty registry if it doesn't exist
    # Use pretty-printed JSON format for consistency
    if [[ ! -f "$registry_file" ]]; then
        cat > "$registry_file" <<-'EOF'
{
  "agents": {}
}
EOF
        if [[ $? -ne 0 ]]; then
            return 1
        fi
    fi
    
    return 0
}

#######################################
# Generate unique agent ID
# Arguments:
#   $1 - Configuration array name (passed by reference)
# Returns:
#   Agent ID string
#######################################
agents::registry::generate_id() {
    local -n config_ref=$1
    local prefix="${config_ref[agent_id_prefix]}"
    
    echo "${prefix}-$(date +%s)-$$"
}

#######################################
# Register new agent in registry
# Arguments:
#   $1 - Registry file path
#   $2 - Resource name
#   $3 - Agent ID
#   $4 - Process ID 
#   $5 - Command string
# Returns:
#   0 on success, 1 on error
#######################################
agents::registry::register() {
    local registry_file="$1"
    local resource_name="$2"
    local agent_id="$3"
    local pid="$4"
    local command="$5"
    local start_time
    
    if [[ -z "$agent_id" || -z "$pid" || -z "$command" ]]; then
        log::error "agents::registry::register: Missing required parameters"
        return 1
    fi
    
    agents::registry::init "$registry_file" || return 1
    
    start_time=$(date -Iseconds)
    
    # Create temporary file for atomic update
    local temp_file="${registry_file}.tmp.$$"
    
    # Add agent to registry using jq
    if ! jq --arg id "$agent_id" \
            --arg pid "$pid" \
            --arg cmd "$command" \
            --arg start "$start_time" \
            '.agents[$id] = {
                "id": $id,
                "pid": ($pid | tonumber),
                "status": "running",
                "start_time": $start,
                "command": $cmd,
                "last_seen": $start
            }' "$registry_file" > "$temp_file"; then
        log::error "Failed to update $resource_name agent registry"
        rm -f "$temp_file"
        return 1
    fi
    
    # Atomically replace registry file
    if ! mv "$temp_file" "$registry_file"; then
        log::error "Failed to save $resource_name agent registry"
        rm -f "$temp_file"
        return 1
    fi
    
    log::debug "Registered $resource_name agent: $agent_id (PID: $pid)"
    return 0
}

#######################################
# Unregister agent from registry
# Arguments:
#   $1 - Configuration array name (passed by reference)
#   $2 - Agent ID
# Returns:
#   0 on success, 1 on error
#######################################
agents::registry::unregister() {
    local -n config_ref=$1
    local agent_id="$2"
    local registry_file="${config_ref[registry_file]}"
    local resource_name="${config_ref[resource_name]}"
    
    if [[ -z "$agent_id" ]]; then
        log::error "agents::registry::unregister: Agent ID required"
        return 1
    fi
    
    [[ -f "$registry_file" ]] || return 0
    
    # Create temporary file for atomic update
    local temp_file="${registry_file}.tmp.$$"
    
    # Remove agent from registry
    if ! jq --arg id "$agent_id" 'del(.agents[$id])' "$registry_file" > "$temp_file"; then
        log::error "Failed to update $resource_name agent registry"
        rm -f "$temp_file"
        return 1
    fi
    
    # Atomically replace registry file
    if ! mv "$temp_file" "$registry_file"; then
        log::error "Failed to save $resource_name agent registry"
        rm -f "$temp_file"
        return 1
    fi
    
    log::debug "Unregistered $resource_name agent: $agent_id"
    return 0
}

#######################################
# Update agent last seen timestamp
# Arguments:
#   $1 - Configuration array name (passed by reference)
#   $2 - Agent ID
# Returns:
#   0 on success, 1 on error
#######################################
agents::registry::heartbeat() {
    local -n config_ref=$1
    local agent_id="$2"
    local registry_file="${config_ref[registry_file]}"
    local resource_name="${config_ref[resource_name]}"
    local current_time
    
    if [[ -z "$agent_id" ]]; then
        log::error "agents::registry::heartbeat: Agent ID required"
        return 1
    fi
    
    [[ -f "$registry_file" ]] || return 1
    
    current_time=$(date -Iseconds)
    
    # Create temporary file for atomic update
    local temp_file="${registry_file}.tmp.$$"
    
    # Update last_seen timestamp
    if ! jq --arg id "$agent_id" \
            --arg time "$current_time" \
            '.agents[$id].last_seen = $time' "$registry_file" > "$temp_file"; then
        log::error "Failed to update $resource_name agent registry"
        rm -f "$temp_file"
        return 1
    fi
    
    # Atomically replace registry file
    if ! mv "$temp_file" "$registry_file"; then
        log::error "Failed to save $resource_name agent registry" 
        rm -f "$temp_file"
        return 1
    fi
    
    return 0
}

#######################################
# List all agents
# Arguments:
#   $1 - Configuration array name (passed by reference)
#   $2 - Optional status filter (running|stopped|crashed)
#   $3 - Output format (json|table)
# Returns:
#   0 on success
#######################################
agents::registry::list() {
    local -n config_ref=$1
    local status_filter="${2:-}"
    local output_format="${3:-table}"
    local registry_file="${config_ref[registry_file]}"
    
    [[ -f "$registry_file" ]] || {
        if [[ "$output_format" == "json" ]]; then
            echo '{"agents": {}}'
        else
            echo "No agents registered"
        fi
        return 0
    }
    
    # Clean up dead agents first
    source "${APP_ROOT}/scripts/resources/agents/core/lifecycle.sh"
    agents::lifecycle::cleanup config_ref >/dev/null
    
    if [[ "$output_format" == "json" ]]; then
        if [[ -n "$status_filter" ]]; then
            jq --arg status "$status_filter" '{
                agents: .agents | to_entries | map(select(.value.status == $status)) | from_entries
            }' "$registry_file"
        else
            cat "$registry_file"
        fi
    else
        # Table format
        local agents_data
        if [[ -n "$status_filter" ]]; then
            agents_data=$(jq -r --arg status "$status_filter" \
                '.agents | to_entries | map(select(.value.status == $status)) | .[] | 
                [.value.id, .value.pid, .value.status, .value.start_time] | @tsv' \
                "$registry_file" 2>/dev/null || echo "")
        else
            agents_data=$(jq -r '.agents | to_entries | .[] | 
                [.value.id, .value.pid, .value.status, .value.start_time] | @tsv' \
                "$registry_file" 2>/dev/null || echo "")
        fi
        
        if [[ -n "$agents_data" ]]; then
            echo "AGENT_ID                            PID     STATUS   START_TIME"
            echo "────────────────────────────────────────────────────────────────────"
            echo "$agents_data" | column -t -s $'\t'
        else
            echo "No agents found"
            if [[ -n "$status_filter" ]]; then
                echo "(filtered by status: $status_filter)"
            fi
        fi
    fi
}

#######################################
# Get agent information
# Arguments:
#   $1 - Configuration array name (passed by reference)
#   $2 - Agent ID
#   $3 - Output format (json|table)
# Returns:
#   0 on success, 1 if not found
#######################################
agents::registry::info() {
    local -n config_ref=$1
    local agent_id="$2"
    local output_format="${3:-table}"
    local registry_file="${config_ref[registry_file]}"
    
    if [[ -z "$agent_id" ]]; then
        log::error "agents::registry::info: Agent ID required"
        return 1
    fi
    
    [[ -f "$registry_file" ]] || {
        log::error "No agents registry found"
        return 1
    }
    
    local agent_data
    agent_data=$(jq --arg id "$agent_id" '.agents[$id] // empty' "$registry_file" 2>/dev/null)
    
    if [[ -z "$agent_data" || "$agent_data" == "null" ]]; then
        if [[ "$output_format" == "json" ]]; then
            echo '{"error": "Agent not found"}'
        else
            echo "Agent not found: $agent_id"
        fi
        return 1
    fi
    
    if [[ "$output_format" == "json" ]]; then
        echo "$agent_data"
    else
        echo "Agent Information:"
        echo "════════════════════"
        echo "$agent_data" | jq -r '
            "ID:           " + .id + "\n" +
            "PID:          " + (.pid | tostring) + "\n" + 
            "Status:       " + .status + "\n" +
            "Start Time:   " + .start_time + "\n" +
            "Last Seen:    " + .last_seen + "\n" +
            "Command:      " + .command
        '
    fi
    
    return 0
}

# Export functions for use by resource-specific implementations
export -f agents::registry::init
export -f agents::registry::generate_id
export -f agents::registry::register
export -f agents::registry::unregister
export -f agents::registry::heartbeat
export -f agents::registry::list
export -f agents::registry::info