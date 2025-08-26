#!/usr/bin/env bash

#######################################
# Browser Session Manager
# 
# Manages persistent browser sessions for browserless.
# Allows reusing browser contexts across multiple operations.
#######################################

set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SESSION_MANAGER_DIR="${APP_ROOT}/resources/browserless/lib"

# Source log utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Source common utilities
source "${SESSION_MANAGER_DIR}/common.sh"

# Session storage directory
SESSION_STORE="${BROWSERLESS_DATA_DIR}/sessions"
mkdir -p "$SESSION_STORE"

#######################################
# Create a new browser session
# Arguments:
#   $1 - Session ID
#   $2 - Options (optional JSON object)
# Returns:
#   0 on success, 1 on failure
#######################################
session::create() {
    local session_id="${1:?Session ID required}"
    local options="${2:-{}}"
    local browserless_port="${BROWSERLESS_PORT:-4110}"
    
    log::debug "Creating browser session: $session_id"
    
    # Check if session already exists
    if session::exists "$session_id"; then
        log::warn "Session already exists: $session_id"
        return 0
    fi
    
    # Create session metadata
    local session_file="${SESSION_STORE}/${session_id}.json"
    local created_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    cat > "$session_file" <<EOF
{
    "id": "$session_id",
    "created_at": "$created_at",
    "status": "active",
    "options": $options,
    "browserless_port": $browserless_port
}
EOF
    
    log::success "Session created: $session_id"
    return 0
}

#######################################
# Destroy a browser session
# Arguments:
#   $1 - Session ID
# Returns:
#   0 on success, 1 on failure
#######################################
session::destroy() {
    local session_id="${1:?Session ID required}"
    
    log::debug "Destroying browser session: $session_id"
    
    local session_file="${SESSION_STORE}/${session_id}.json"
    
    if [[ ! -f "$session_file" ]]; then
        log::warn "Session not found: $session_id"
        return 1
    fi
    
    # Mark session as destroyed
    local destroyed_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    jq --arg destroyed_at "$destroyed_at" \
        '.status = "destroyed" | .destroyed_at = $destroyed_at' \
        "$session_file" > "${session_file}.tmp" && \
        mv "${session_file}.tmp" "$session_file"
    
    # TODO: Actually close the browser context in browserless
    # This would require browserless to support persistent sessions
    
    log::success "Session destroyed: $session_id"
    return 0
}

#######################################
# Check if a session exists and is active
# Arguments:
#   $1 - Session ID
# Returns:
#   0 if exists and active, 1 otherwise
#######################################
session::exists() {
    local session_id="${1:?Session ID required}"
    
    local session_file="${SESSION_STORE}/${session_id}.json"
    
    if [[ ! -f "$session_file" ]]; then
        return 1
    fi
    
    local status=$(jq -r '.status' "$session_file")
    
    if [[ "$status" == "active" ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# List all sessions
# Arguments:
#   $1 - Filter (optional: "active", "destroyed", "all")
# Returns:
#   JSON array of sessions
#######################################
session::list() {
    local filter="${1:-active}"
    
    local sessions="[]"
    
    for session_file in "${SESSION_STORE}"/*.json; do
        if [[ ! -f "$session_file" ]]; then
            continue
        fi
        
        local session=$(cat "$session_file")
        local status=$(echo "$session" | jq -r '.status')
        
        case "$filter" in
            active)
                if [[ "$status" == "active" ]]; then
                    sessions=$(echo "$sessions" | jq ". += [$session]")
                fi
                ;;
            destroyed)
                if [[ "$status" == "destroyed" ]]; then
                    sessions=$(echo "$sessions" | jq ". += [$session]")
                fi
                ;;
            all)
                sessions=$(echo "$sessions" | jq ". += [$session]")
                ;;
        esac
    done
    
    echo "$sessions"
}

#######################################
# Reuse an existing session or create if doesn't exist
# Arguments:
#   $1 - Session ID
#   $2 - Options (optional JSON object)
# Returns:
#   0 on success, 1 on failure
#######################################
session::reuse() {
    local session_id="${1:?Session ID required}"
    local options="${2:-{}}"
    
    if session::exists "$session_id"; then
        log::debug "Reusing existing session: $session_id"
        
        # Update last_used timestamp
        local session_file="${SESSION_STORE}/${session_id}.json"
        local last_used=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        
        jq --arg last_used "$last_used" \
            '.last_used = $last_used' \
            "$session_file" > "${session_file}.tmp" && \
            mv "${session_file}.tmp" "$session_file"
        
        return 0
    else
        log::debug "Creating new session: $session_id"
        session::create "$session_id" "$options"
        return $?
    fi
}

#######################################
# Get session metadata
# Arguments:
#   $1 - Session ID
# Returns:
#   JSON object with session metadata
#######################################
session::get_metadata() {
    local session_id="${1:?Session ID required}"
    
    local session_file="${SESSION_STORE}/${session_id}.json"
    
    if [[ ! -f "$session_file" ]]; then
        echo "{}"
        return 1
    fi
    
    cat "$session_file"
}

#######################################
# Update session metadata
# Arguments:
#   $1 - Session ID
#   $2 - Key to update
#   $3 - Value to set
# Returns:
#   0 on success, 1 on failure
#######################################
session::update_metadata() {
    local session_id="${1:?Session ID required}"
    local key="${2:?Key required}"
    local value="${3:?Value required}"
    
    local session_file="${SESSION_STORE}/${session_id}.json"
    
    if [[ ! -f "$session_file" ]]; then
        log::error "Session not found: $session_id"
        return 1
    fi
    
    # Update the metadata
    jq --arg key "$key" --arg value "$value" \
        '.[$key] = $value' \
        "$session_file" > "${session_file}.tmp" && \
        mv "${session_file}.tmp" "$session_file"
    
    return 0
}

#######################################
# Clean up old sessions
# Arguments:
#   $1 - Max age in minutes (optional, default: 60)
# Returns:
#   Number of sessions cleaned
#######################################
session::cleanup() {
    local max_age_minutes="${1:-60}"
    
    log::debug "Cleaning up sessions older than ${max_age_minutes} minutes"
    
    local cleaned=0
    local current_time=$(date +%s)
    
    for session_file in "${SESSION_STORE}"/*.json; do
        if [[ ! -f "$session_file" ]]; then
            continue
        fi
        
        local created_at=$(jq -r '.created_at' "$session_file")
        local session_time=$(date -d "$created_at" +%s 2>/dev/null || echo 0)
        local age_minutes=$(( (current_time - session_time) / 60 ))
        
        if [[ $age_minutes -gt $max_age_minutes ]]; then
            local session_id=$(jq -r '.id' "$session_file")
            log::debug "Cleaning up old session: $session_id (${age_minutes} minutes old)"
            session::destroy "$session_id"
            rm -f "$session_file"
            cleaned=$((cleaned + 1))
        fi
    done
    
    log::info "Cleaned up $cleaned old sessions"
    return 0
}

#######################################
# Execute a command in a session context
# This ensures the session exists before running
# Arguments:
#   $1 - Session ID
#   $2 - Command to execute
#   $@ - Additional arguments for the command
# Returns:
#   Command exit status
#######################################
session::with() {
    local session_id="${1:?Session ID required}"
    local command="${2:?Command required}"
    shift 2
    
    # Ensure session exists
    session::reuse "$session_id"
    
    # Execute the command with session ID
    "$command" "$@" "$session_id"
    local exit_code=$?
    
    # Update last command status
    session::update_metadata "$session_id" "last_command" "$command"
    session::update_metadata "$session_id" "last_command_status" "$exit_code"
    
    return $exit_code
}

# Export all functions
export -f session::create
export -f session::destroy
export -f session::exists
export -f session::list
export -f session::reuse
export -f session::get_metadata
export -f session::update_metadata
export -f session::cleanup
export -f session::with