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
# Remove orphaned temporary registry files
# Arguments:
#   $1 - Registry file path
# Cleans up files left behind by interrupted writes
#######################################
agents::registry::cleanup_temp_files() {
    local registry_file="$1"
    local registry_dir
    local registry_base
    local temp_file
    local pid_part
    local pid

    registry_dir=$(dirname "$registry_file")
    registry_base=$(basename "$registry_file")

    if [[ ! -d "$registry_dir" ]]; then
        return 0
    fi

    shopt -s nullglob
    for temp_file in "${registry_dir}/${registry_base}.tmp."*; do
        [[ -f "$temp_file" ]] || continue
        pid_part="${temp_file##*.tmp.}"
        pid="${pid_part%%.*}"
        if [[ -n "$pid" && "$pid" =~ ^[0-9]+$ ]] && kill -0 "$pid" 2>/dev/null; then
            continue
        fi
        rm -f "$temp_file"
    done
    shopt -u nullglob
}

#######################################
# Create temporary file for atomic updates
# Arguments:
#   $1 - Registry file path
# Returns:
#   Path to temporary file
#######################################
agents::registry::create_temp_file() {
    local registry_file="$1"
    local template

    agents::registry::cleanup_temp_files "$registry_file"
    template="${registry_file}.tmp.${BASHPID}.XXXXXX"
    mktemp "$template"
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

    local lock_fd
    exec {lock_fd}>"${registry_file}.lock" || return 1
    if ! flock -w 5 "$lock_fd"; then
        exec {lock_fd}>&-
        log::error "Failed to lock $resource_name agent registry"
        return 1
    fi

    local temp_file
    local rc=0
    temp_file=$(agents::registry::create_temp_file "$registry_file") || rc=1

    if [[ $rc -eq 0 ]]; then
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
            rc=1
        fi
    fi

    if [[ $rc -eq 0 ]]; then
        if ! mv "$temp_file" "$registry_file"; then
            log::error "Failed to save $resource_name agent registry"
            rc=1
        fi
    fi

    [[ -n "$temp_file" && -f "$temp_file" ]] && rm -f "$temp_file"

    flock -u "$lock_fd"
    exec {lock_fd}>&-

    if [[ $rc -eq 0 ]]; then
        log::debug "Registered $resource_name agent: $agent_id (PID: $pid)"
    fi

    return $rc
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
    
    local lock_fd
    exec {lock_fd}>"${registry_file}.lock" || return 1
    if ! flock -w 5 "$lock_fd"; then
        exec {lock_fd}>&-
        log::error "Failed to lock $resource_name agent registry"
        return 1
    fi

    local temp_file
    local rc=0
    temp_file=$(agents::registry::create_temp_file "$registry_file") || rc=1

    if [[ $rc -eq 0 ]]; then
        if ! jq --arg id "$agent_id" 'del(.agents[$id])' "$registry_file" > "$temp_file"; then
            log::error "Failed to update $resource_name agent registry"
            rc=1
        fi
    fi

    if [[ $rc -eq 0 ]]; then
        if ! mv "$temp_file" "$registry_file"; then
            log::error "Failed to save $resource_name agent registry"
            rc=1
        fi
    fi

    [[ -n "$temp_file" && -f "$temp_file" ]] && rm -f "$temp_file"

    flock -u "$lock_fd"
    exec {lock_fd}>&-

    if [[ $rc -eq 0 ]]; then
        log::debug "Unregistered $resource_name agent: $agent_id"
    fi

    return $rc
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
    
    local lock_fd
    exec {lock_fd}>"${registry_file}.lock" || return 1
    if ! flock -w 5 "$lock_fd"; then
        exec {lock_fd}>&-
        log::error "Failed to lock $resource_name agent registry"
        return 1
    fi

    local temp_file
    local rc=0
    temp_file=$(agents::registry::create_temp_file "$registry_file") || rc=1

    if [[ $rc -eq 0 ]]; then
        if ! jq --arg id "$agent_id" \
                --arg time "$current_time" \
                '.agents[$id].last_seen = $time' "$registry_file" > "$temp_file"; then
            log::error "Failed to update $resource_name agent registry"
            rc=1
        fi
    fi

    if [[ $rc -eq 0 ]]; then
        if ! mv "$temp_file" "$registry_file"; then
            log::error "Failed to save $resource_name agent registry"
            rc=1
        fi
    fi

    [[ -n "$temp_file" && -f "$temp_file" ]] && rm -f "$temp_file"

    flock -u "$lock_fd"
    exec {lock_fd}>&-

    return $rc
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
