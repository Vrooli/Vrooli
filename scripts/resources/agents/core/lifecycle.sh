#!/usr/bin/env bash
################################################################################
# Agent Lifecycle Management
# 
# Generic process lifecycle functions for all agent-capable resources
# Handles process checking, cleanup, and agent termination
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
if ! declare -F agents::registry::create_temp_file >/dev/null 2>&1; then
    source "${APP_ROOT}/scripts/resources/agents/core/registry.sh"
fi


#######################################
# Check if process is still running
# Arguments:
#   $1 - Process ID
# Returns:
#   0 if running, 1 if not
#######################################
agents::lifecycle::is_pid_running() {
    local pid="$1"
    
    if [[ -z "$pid" ]]; then
        return 1
    fi
    
    # Use kill -0 to check if PID exists without sending signal
    kill -0 "$pid" 2>/dev/null
}

#######################################
# Clean up dead agents from registry
# Arguments:
#   $1 - Configuration array name (passed by reference)
# Returns:
#   Number of cleaned agents
#######################################
agents::lifecycle::cleanup() {
    local -n config_ref=$1
    local registry_file="${config_ref[registry_file]}"
    local resource_name="${config_ref[resource_name]}"
    local cleaned=0
    local temp_file
    
    [[ -f "$registry_file" ]] || return 0
    
    temp_file=$(agents::registry::create_temp_file "$registry_file") || return 0
    
    # Get list of all agents and check their PIDs
    local agent_ids
    agent_ids=$(jq -r '.agents | keys[]' "$registry_file" 2>/dev/null || echo "")
    
    if [[ -z "$agent_ids" ]]; then
        return 0
    fi
    
    # Start with current registry
    cp "$registry_file" "$temp_file" || return 0
    
    while IFS= read -r agent_id; do
        [[ -n "$agent_id" ]] || continue
        
        local pid
        pid=$(jq -r --arg id "$agent_id" '.agents[$id].pid // empty' "$temp_file" 2>/dev/null)
        
        if [[ -n "$pid" ]]; then
            if ! agents::lifecycle::is_pid_running "$pid"; then
                # Mark as crashed and remove
                log::debug "Cleaning up dead $resource_name agent: $agent_id (PID: $pid)"
                jq --arg id "$agent_id" 'del(.agents[$id])' "$temp_file" > "${temp_file}.new" && \
                    mv "${temp_file}.new" "$temp_file"
                ((cleaned++))
            fi
        fi
    done <<< "$agent_ids"
    
    # Atomically replace registry file if changes were made
    if [[ $cleaned -gt 0 ]]; then
        mv "$temp_file" "$registry_file" || rm -f "$temp_file"
    else
        rm -f "$temp_file"
    fi
    
    return $cleaned
}

#######################################
# Stop agent by ID or all agents
# Arguments:
#   $1 - Configuration array name (passed by reference)
#   $2 - Agent ID or "all"
#   $3 - Force flag (true/false)
# Returns:
#   0 on success, 1 on error
#######################################
agents::lifecycle::stop() {
    local -n config_ref=$1
    local target="$2"
    local force="${3:-false}"
    local registry_file="${config_ref[registry_file]}"
    local resource_name="${config_ref[resource_name]}"
    local signal="TERM"
    
    if [[ -z "$target" ]]; then
        log::error "agents::lifecycle::stop: Target required (agent-id or 'all')"
        return 1
    fi
    
    [[ -f "$registry_file" ]] || {
        log::warn "No $resource_name agents registry found"
        return 0
    }
    
    # Use KILL signal if force is true
    if [[ "$force" == "true" ]]; then
        signal="KILL"
    fi
    
    if [[ "$target" == "all" ]]; then
        # Stop all agents
        local agent_ids stopped=0
        agent_ids=$(jq -r '.agents | keys[]' "$registry_file" 2>/dev/null || echo "")
        
        while IFS= read -r agent_id; do
            [[ -n "$agent_id" ]] || continue
            
            local pid
            pid=$(jq -r --arg id "$agent_id" '.agents[$id].pid // empty' "$registry_file" 2>/dev/null)
            
            if [[ -n "$pid" ]] && agents::lifecycle::is_pid_running "$pid"; then
                log::info "Stopping $resource_name agent: $agent_id (PID: $pid)"
                if kill -s "$signal" "$pid" 2>/dev/null; then
                    ((stopped++))
                    # Use the registry module to unregister
                    source "${APP_ROOT}/scripts/resources/agents/core/registry.sh"
                    agents::registry::unregister config_ref "$agent_id"
                else
                    log::warn "Failed to stop $resource_name agent: $agent_id"
                fi
            else
                # Remove dead agent from registry
                source "${APP_ROOT}/scripts/resources/agents/core/registry.sh"
                agents::registry::unregister config_ref "$agent_id"
            fi
        done <<< "$agent_ids"
        
        log::info "Stopped $stopped $resource_name agents"
        return 0
    else
        # Stop specific agent
        local pid
        pid=$(jq -r --arg id "$target" '.agents[$id].pid // empty' "$registry_file" 2>/dev/null)
        
        if [[ -z "$pid" ]]; then
            log::error "$resource_name agent not found: $target"
            return 1
        fi
        
        if agents::lifecycle::is_pid_running "$pid"; then
            log::info "Stopping $resource_name agent: $target (PID: $pid)"
            if kill -s "$signal" "$pid" 2>/dev/null; then
                source "${APP_ROOT}/scripts/resources/agents/core/registry.sh"
                agents::registry::unregister config_ref "$target"
                log::info "$resource_name agent stopped successfully"
                return 0
            else
                log::error "Failed to stop $resource_name agent: $target"
                return 1
            fi
        else
            log::warn "$resource_name agent process not running, cleaning up registry"
            source "${APP_ROOT}/scripts/resources/agents/core/registry.sh"
            agents::registry::unregister config_ref "$target"
            return 0
        fi
    fi
}

#######################################
# Setup cleanup trap for agent on EXIT
# Arguments:
#   $1 - Configuration array name (passed by reference)
#   $2 - Agent ID to clean up
# Returns:
#   0 on success
#######################################
agents::lifecycle::setup_cleanup() {
    local -n config_ref=$1
    local agent_id="$2"
    local resource_name="${config_ref[resource_name]}"
    
    if [[ -z "$agent_id" ]]; then
        log::error "agents::lifecycle::setup_cleanup: Agent ID required"
        return 1
    fi
    
    # Set up cleanup trap
    trap "source '${APP_ROOT}/scripts/resources/agents/core/registry.sh'; agents::registry::unregister config_ref '$agent_id' 2>/dev/null || true" EXIT INT TERM
    
    return 0
}

# Export functions for use by resource-specific implementations
export -f agents::lifecycle::is_pid_running
export -f agents::lifecycle::cleanup
export -f agents::lifecycle::stop
export -f agents::lifecycle::setup_cleanup