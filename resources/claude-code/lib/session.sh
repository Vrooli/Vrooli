#!/usr/bin/env bash
# Claude Code Session Management Functions
# Handles session listing and resumption

# Set CLAUDE_CODE_SCRIPT_DIR if not already set (for BATS test compatibility)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLAUDE_CODE_SCRIPT_DIR="${CLAUDE_CODE_SCRIPT_DIR:-${APP_ROOT}/resources/claude-code}"

# Source required utilities
# shellcheck disable=SC1091
source "${CLAUDE_CODE_SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${CLAUDE_CODE_SCRIPT_DIR}/../../../lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

#######################################
# Manage Claude sessions
#######################################
claude_code::session() {
    log::header "🔄 Managing Claude Sessions"
    
    if ! claude_code::is_installed; then
        log::error "Claude Code is not installed. Run: $0 --action install"
        return 1
    fi
    
    if [[ -z "${SESSION_ID:-}" ]]; then
        # List sessions
        claude_code::session_list
    else
        # Resume specific session
        claude_code::session_resume "$SESSION_ID"
    fi
}

#######################################
# List available Claude sessions
#######################################
claude_code::session_list() {
    log::info "Listing recent sessions..."
    
    # Check for session files
    if [[ -d "$CLAUDE_SESSIONS_DIR" ]]; then
        local sessions
        sessions=$(ls -t "$CLAUDE_SESSIONS_DIR" 2>/dev/null | head -10)
        if [[ -n "$sessions" ]]; then
            echo "Recent sessions:"
            echo "$sessions" | nl
        else
            log::info "No sessions found"
        fi
    else
        log::info "No session directory found"
    fi
    
    echo
    log::info "To resume a session: $0 --action session --session-id <id>"
}

#######################################
# Resume a specific Claude session
# Arguments:
#   $1 - session ID
#######################################
claude_code::session_resume() {
    local session_id="$1"
    
    log::info "Resuming session: $session_id"
    
    local cmd="claude --print --resume \"$session_id\""
    
    # Add max turns if specified
    if [[ "$MAX_TURNS" != "$DEFAULT_MAX_TURNS" ]]; then
        cmd="$cmd --max-turns $MAX_TURNS"
    else
        cmd="$cmd --max-turns 5"
    fi
    
    log::info "Executing: $cmd"
    eval "$cmd"
}

#######################################
# Delete a Claude session
# Arguments:
#   $1 - session ID
#######################################
claude_code::session_delete() {
    local session_id="$1"
    
    if [[ -z "$session_id" ]]; then
        log::error "No session ID provided"
        return 1
    fi
    
    local session_file="$CLAUDE_SESSIONS_DIR/$session_id"
    
    if [[ -f "$session_file" ]]; then
        if confirm "Delete session $session_id?"; then
            trash::safe_remove "$session_file" --temp
            log::success "✓ Session deleted: $session_id"
        else
            log::info "Session deletion cancelled"
        fi
    else
        log::warn "Session not found: $session_id"
    fi
}

#######################################
# View details of a Claude session
# Arguments:
#   $1 - session ID
#######################################
claude_code::session_view() {
    local session_id="$1"
    
    if [[ -z "$session_id" ]]; then
        log::error "No session ID provided"
        return 1
    fi
    
    local session_file="$CLAUDE_SESSIONS_DIR/$session_id"
    
    if [[ -f "$session_file" ]]; then
        log::info "Session details for: $session_id"
        echo
        if system::is_command jq; then
            cat "$session_file" | jq .
        else
            cat "$session_file"
        fi
    else
        log::error "Session not found: $session_id"
        return 1
    fi
}

# Export functions for subshell availability
export -f claude_code::session
export -f claude_code::session_list
export -f claude_code::session_resume
export -f claude_code::session_delete
export -f claude_code::session_view