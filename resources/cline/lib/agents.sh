#!/usr/bin/env bash
################################################################################
# Agent Registry Management
# 
# Tracks running Cline agents for coordination and management
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Agent registry location
CLINE_AGENTS_REGISTRY="${APP_ROOT}/.vrooli/cline-agents.json"

#######################################
# Initialize agent registry if not exists
# Returns:
#   0 on success
#######################################
agents::init_registry() {
    local registry_dir
    registry_dir=$(dirname "$CLINE_AGENTS_REGISTRY")
    
    # Create .vrooli directory if needed
    if [[ ! -d "$registry_dir" ]]; then
        mkdir -p "$registry_dir" || return 1
    fi
    
    # Create empty registry if it doesn't exist
    if [[ ! -f "$CLINE_AGENTS_REGISTRY" ]]; then
        echo '{"agents": {}}' > "$CLINE_AGENTS_REGISTRY" || return 1
    fi
    
    return 0
}

#######################################
# Generate unique agent ID
# Returns:
#   Agent ID string
#######################################
agents::generate_id() {
    echo "cline-agent-$(date +%s)-$$"
}

#######################################
# Register new agent in registry
# Arguments:
#   $1 - Agent ID
#   $2 - Process ID 
#   $3 - Command string
# Returns:
#   0 on success, 1 on error
#######################################
agents::register() {
    local agent_id="$1"
    local pid="$2"
    local command="$3"
    local start_time
    
    if [[ -z "$agent_id" || -z "$pid" || -z "$command" ]]; then
        log::error "agents::register: Missing required parameters"
        return 1
    fi
    
    agents::init_registry || return 1
    
    start_time=$(date -Iseconds)
    
    # Create temporary file for atomic update
    local temp_file="${CLINE_AGENTS_REGISTRY}.tmp.$$"
    
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
            }' "$CLINE_AGENTS_REGISTRY" > "$temp_file"; then
        log::error "Failed to update agent registry"
        rm -f "$temp_file"
        return 1
    fi
    
    # Atomically replace registry file
    if ! mv "$temp_file" "$CLINE_AGENTS_REGISTRY"; then
        log::error "Failed to save agent registry"
        rm -f "$temp_file"
        return 1
    fi
    
    log::debug "Registered agent: $agent_id (PID: $pid)"
    return 0
}

#######################################
# Unregister agent from registry
# Arguments:
#   $1 - Agent ID
# Returns:
#   0 on success, 1 on error
#######################################
agents::unregister() {
    local agent_id="$1"
    
    if [[ -z "$agent_id" ]]; then
        log::error "agents::unregister: Agent ID required"
        return 1
    fi
    
    [[ -f "$CLINE_AGENTS_REGISTRY" ]] || return 0
    
    # Create temporary file for atomic update
    local temp_file="${CLINE_AGENTS_REGISTRY}.tmp.$$"
    
    # Remove agent from registry
    if ! jq --arg id "$agent_id" 'del(.agents[$id])' "$CLINE_AGENTS_REGISTRY" > "$temp_file"; then
        log::error "Failed to update agent registry"
        rm -f "$temp_file"
        return 1
    fi
    
    # Atomically replace registry file
    if ! mv "$temp_file" "$CLINE_AGENTS_REGISTRY"; then
        log::error "Failed to save agent registry"
        rm -f "$temp_file"
        return 1
    fi
    
    log::debug "Unregistered agent: $agent_id"
    return 0
}

#######################################
# Update agent last seen timestamp
# Arguments:
#   $1 - Agent ID
# Returns:
#   0 on success, 1 on error
#######################################
agents::heartbeat() {
    local agent_id="$1"
    local current_time
    
    if [[ -z "$agent_id" ]]; then
        log::error "agents::heartbeat: Agent ID required"
        return 1
    fi
    
    [[ -f "$CLINE_AGENTS_REGISTRY" ]] || return 1
    
    current_time=$(date -Iseconds)
    
    # Create temporary file for atomic update
    local temp_file="${CLINE_AGENTS_REGISTRY}.tmp.$$"
    
    # Update last_seen timestamp
    if ! jq --arg id "$agent_id" \
            --arg time "$current_time" \
            '.agents[$id].last_seen = $time' "$CLINE_AGENTS_REGISTRY" > "$temp_file"; then
        log::error "Failed to update agent registry"
        rm -f "$temp_file"
        return 1
    fi
    
    # Atomically replace registry file
    if ! mv "$temp_file" "$CLINE_AGENTS_REGISTRY"; then
        log::error "Failed to save agent registry" 
        rm -f "$temp_file"
        return 1
    fi
    
    return 0
}

#######################################
# Check if process is still running
# Arguments:
#   $1 - Process ID
# Returns:
#   0 if running, 1 if not
#######################################
agents::is_pid_running() {
    local pid="$1"
    
    if [[ -z "$pid" ]]; then
        return 1
    fi
    
    # Use kill -0 to check if PID exists without sending signal
    kill -0 "$pid" 2>/dev/null
}

#######################################
# Clean up dead agents from registry
# Returns:
#   Number of cleaned agents
#######################################
agents::cleanup() {
    local cleaned=0
    local temp_file
    
    [[ -f "$CLINE_AGENTS_REGISTRY" ]] || return 0
    
    temp_file="${CLINE_AGENTS_REGISTRY}.tmp.$$"
    
    # Get list of all agents and check their PIDs
    local agent_ids
    agent_ids=$(jq -r '.agents | keys[]' "$CLINE_AGENTS_REGISTRY" 2>/dev/null || echo "")
    
    if [[ -z "$agent_ids" ]]; then
        return 0
    fi
    
    # Start with current registry
    cp "$CLINE_AGENTS_REGISTRY" "$temp_file" || return 0
    
    while IFS= read -r agent_id; do
        [[ -n "$agent_id" ]] || continue
        
        local pid
        pid=$(jq -r --arg id "$agent_id" '.agents[$id].pid // empty' "$temp_file" 2>/dev/null)
        
        if [[ -n "$pid" ]]; then
            if ! agents::is_pid_running "$pid"; then
                # Mark as crashed and remove
                log::debug "Cleaning up dead agent: $agent_id (PID: $pid)"
                jq --arg id "$agent_id" 'del(.agents[$id])' "$temp_file" > "${temp_file}.new" && \
                    mv "${temp_file}.new" "$temp_file"
                ((cleaned++))
            fi
        fi
    done <<< "$agent_ids"
    
    # Atomically replace registry file if changes were made
    if [[ $cleaned -gt 0 ]]; then
        mv "$temp_file" "$CLINE_AGENTS_REGISTRY" || rm -f "$temp_file"
    else
        rm -f "$temp_file"
    fi
    
    return $cleaned
}

#######################################
# List all agents
# Arguments:
#   $1 - Optional status filter (running|stopped|crashed)
#   $2 - Output format (json|table)
# Returns:
#   0 on success
#######################################
agents::list() {
    local status_filter="${1:-}"
    local output_format="${2:-table}"
    
    [[ -f "$CLINE_AGENTS_REGISTRY" ]] || {
        if [[ "$output_format" == "json" ]]; then
            echo '{"agents": {}}'
        else
            echo "No agents registered"
        fi
        return 0
    }
    
    # Clean up dead agents first
    agents::cleanup >/dev/null
    
    if [[ "$output_format" == "json" ]]; then
        if [[ -n "$status_filter" ]]; then
            jq --arg status "$status_filter" '{
                agents: .agents | to_entries | map(select(.value.status == $status)) | from_entries
            }' "$CLINE_AGENTS_REGISTRY"
        else
            cat "$CLINE_AGENTS_REGISTRY"
        fi
    else
        # Table format
        local agents_data
        if [[ -n "$status_filter" ]]; then
            agents_data=$(jq -r --arg status "$status_filter" \
                '.agents | to_entries | map(select(.value.status == $status)) | .[] | 
                [.value.id, .value.pid, .value.status, .value.start_time] | @tsv' \
                "$CLINE_AGENTS_REGISTRY" 2>/dev/null || echo "")
        else
            agents_data=$(jq -r '.agents | to_entries | .[] | 
                [.value.id, .value.pid, .value.status, .value.start_time] | @tsv' \
                "$CLINE_AGENTS_REGISTRY" 2>/dev/null || echo "")
        fi
        
        if [[ -n "$agents_data" ]]; then
            echo "AGENT_ID                      PID     STATUS   START_TIME"
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
# Stop agent by ID or all agents
# Arguments:
#   $1 - Agent ID or "all"
#   $2 - Force flag (true/false)
# Returns:
#   0 on success, 1 on error
#######################################
agents::stop() {
    local target="$1"
    local force="${2:-false}"
    local signal="TERM"
    
    if [[ -z "$target" ]]; then
        log::error "agents::stop: Target required (agent-id or 'all')"
        return 1
    fi
    
    [[ -f "$CLINE_AGENTS_REGISTRY" ]] || {
        log::warn "No agents registry found"
        return 0
    }
    
    # Use KILL signal if force is true
    if [[ "$force" == "true" ]]; then
        signal="KILL"
    fi
    
    if [[ "$target" == "all" ]]; then
        # Stop all agents
        local agent_ids stopped=0
        agent_ids=$(jq -r '.agents | keys[]' "$CLINE_AGENTS_REGISTRY" 2>/dev/null || echo "")
        
        while IFS= read -r agent_id; do
            [[ -n "$agent_id" ]] || continue
            
            local pid
            pid=$(jq -r --arg id "$agent_id" '.agents[$id].pid // empty' "$CLINE_AGENTS_REGISTRY" 2>/dev/null)
            
            if [[ -n "$pid" ]] && agents::is_pid_running "$pid"; then
                log::info "Stopping agent: $agent_id (PID: $pid)"
                if kill -s "$signal" "$pid" 2>/dev/null; then
                    ((stopped++))
                    agents::unregister "$agent_id"
                else
                    log::warn "Failed to stop agent: $agent_id"
                fi
            else
                # Remove dead agent from registry
                agents::unregister "$agent_id"
            fi
        done <<< "$agent_ids"
        
        log::info "Stopped $stopped agents"
        return 0
    else
        # Stop specific agent
        local pid
        pid=$(jq -r --arg id "$target" '.agents[$id].pid // empty' "$CLINE_AGENTS_REGISTRY" 2>/dev/null)
        
        if [[ -z "$pid" ]]; then
            log::error "Agent not found: $target"
            return 1
        fi
        
        if agents::is_pid_running "$pid"; then
            log::info "Stopping agent: $target (PID: $pid)"
            if kill -s "$signal" "$pid" 2>/dev/null; then
                agents::unregister "$target"
                log::info "Agent stopped successfully"
                return 0
            else
                log::error "Failed to stop agent: $target"
                return 1
            fi
        else
            log::warn "Agent process not running, cleaning up registry"
            agents::unregister "$target"
            return 0
        fi
    fi
}

#######################################
# Get agent information
# Arguments:
#   $1 - Agent ID
#   $2 - Output format (json|table)
# Returns:
#   0 on success, 1 if not found
#######################################
agents::info() {
    local agent_id="$1"
    local output_format="${2:-table}"
    
    if [[ -z "$agent_id" ]]; then
        log::error "agents::info: Agent ID required"
        return 1
    fi
    
    [[ -f "$CLINE_AGENTS_REGISTRY" ]] || {
        log::error "No agents registry found"
        return 1
    }
    
    local agent_data
    agent_data=$(jq --arg id "$agent_id" '.agents[$id] // empty' "$CLINE_AGENTS_REGISTRY" 2>/dev/null)
    
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

#######################################
# CLI command handler for agent management
# Arguments:
#   $@ - Command arguments
# Returns:
#   Exit code from subcommand
#######################################
cline::agents::command() {
    local subcommand="${1:-list}"
    shift || true
    
    case "$subcommand" in
        "list"|"ls")
            local status_filter=""
            local json_output="false"
            
            # Parse arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --json)
                        json_output="true"
                        shift
                        ;;
                    --status)
                        status_filter="$2"
                        shift 2
                        ;;
                    *)
                        log::error "Unknown option: $1"
                        return 1
                        ;;
                esac
            done
            
            if [[ "$json_output" == "true" ]]; then
                agents::list "$status_filter" "json"
            else
                agents::list "$status_filter" "table"
            fi
            ;;
            
        "stop")
            local target="$1"
            local force="false"
            shift || true
            
            if [[ -z "$target" ]]; then
                log::error "Usage: agents stop <agent-id|all> [--force]"
                return 1
            fi
            
            # Parse remaining arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --force)
                        force="true"
                        shift
                        ;;
                    *)
                        log::error "Unknown option: $1"
                        return 1
                        ;;
                esac
            done
            
            agents::stop "$target" "$force"
            ;;
            
        "cleanup")
            local cleaned
            cleaned=$(agents::cleanup)
            if [[ $cleaned -gt 0 ]]; then
                log::info "Cleaned up $cleaned dead agents"
            else
                log::info "No dead agents to clean up"
            fi
            ;;
            
        "info")
            local agent_id="$1"
            local json_output="false"
            shift || true
            
            if [[ -z "$agent_id" ]]; then
                log::error "Usage: agents info <agent-id> [--json]"
                return 1
            fi
            
            # Parse remaining arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --json)
                        json_output="true"
                        shift
                        ;;
                    *)
                        log::error "Unknown option: $1"
                        return 1
                        ;;
                esac
            done
            
            if [[ "$json_output" == "true" ]]; then
                agents::info "$agent_id" "json"
            else
                agents::info "$agent_id" "table"
            fi
            ;;
            
        "help"|"-h"|"--help")
            echo "Usage: resource-cline agents <subcommand> [options]"
            echo ""
            echo "Subcommands:"
            echo "  list [--json] [--status STATUS]    List all agents (optionally filter by status)"
            echo "  stop <agent-id|all> [--force]      Stop specific agent or all agents"
            echo "  cleanup                             Remove dead agents from registry"
            echo "  info <agent-id> [--json]           Show detailed information about an agent"
            echo "  help                                Show this help message"
            echo ""
            echo "Status filters: running, stopped, crashed"
            ;;
            
        *)
            log::error "Unknown subcommand: $subcommand"
            log::error "Use 'resource-cline agents help' for usage information"
            return 1
            ;;
    esac
}

# Export functions
export -f agents::init_registry
export -f agents::generate_id
export -f agents::register
export -f agents::unregister
export -f agents::heartbeat
export -f agents::is_pid_running
export -f agents::cleanup
export -f agents::list
export -f agents::stop
export -f agents::info
export -f cline::agents::command