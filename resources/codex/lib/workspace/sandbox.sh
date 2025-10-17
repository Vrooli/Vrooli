#!/usr/bin/env bash
################################################################################
# Workspace Sandbox - Main interface for sandbox operations
# 
# This file provides the main interface expected by internal-sandbox.sh
# It aggregates functionality from manager.sh, security.sh, and isolation.sh
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
WORKSPACE_DIR="${APP_ROOT}/resources/codex/lib/workspace"

# Source all workspace components
source "${WORKSPACE_DIR}/manager.sh"
source "${WORKSPACE_DIR}/security.sh" 
source "${WORKSPACE_DIR}/isolation.sh"

################################################################################
# Sandbox Interface Functions
################################################################################

#######################################
# Create a sandbox workspace
# Arguments:
#   $1 - Workspace name (optional)
#   $2 - Security level (strict, moderate, relaxed)
# Returns:
#   Workspace ID
#######################################
sandbox::create_workspace() {
    local name="${1:-sandbox-$(date +%s)}"
    local security_level="${2:-moderate}"
    
    local options='{
        "description": "Sandbox execution workspace",
        "auto_cleanup": true,
        "backup_on_delete": false,
        "monitoring": true
    }'
    
    local result
    result=$(workspace_manager::create "$name" "$security_level" "$options")
    
    if [[ $(echo "$result" | jq -r '.success // false') == "true" ]]; then
        echo "$result" | jq -r '.workspace_id'
        return 0
    else
        echo ""
        return 1
    fi
}

#######################################
# Execute command in sandbox
# Arguments:
#   $1 - Workspace ID
#   $2 - Command to execute
#   $3 - Execution options (JSON, optional)
# Returns:
#   Execution result JSON
#######################################
sandbox::execute() {
    local workspace_id="$1"
    local command="$2"
    local options="${3:-{}}"
    
    # Use isolation system for execution
    workspace_isolation::execute "$workspace_id" "$command" "$options"
}

#######################################
# Get sandbox workspace status
# Arguments:
#   $1 - Workspace ID
# Returns:
#   Status JSON
#######################################
sandbox::get_status() {
    local workspace_id="$1"
    
    # Get workspace info from manager
    local workspace_info
    workspace_info=$(workspace_manager::get_info "$workspace_id")
    
    # Get security status
    local security_status
    security_status=$(workspace_security::get_status "$workspace_id")
    
    # Get isolation status
    local isolation_status
    isolation_status=$(workspace_isolation::get_status "$workspace_id")
    
    # Combine all status information
    jq -n \
        --argjson workspace "$workspace_info" \
        --argjson security "$security_status" \
        --argjson isolation "$isolation_status" \
        '{
            workspace: $workspace,
            security: $security,
            isolation: $isolation,
            combined_at: (now | todate)
        }'
}

#######################################
# Cleanup sandbox workspace
# Arguments:
#   $1 - Workspace ID
#   $2 - Force cleanup (true/false)
#######################################
sandbox::cleanup() {
    local workspace_id="$1"
    local force="${2:-false}"
    
    # Stop security monitoring
    workspace_security::cleanup_monitoring "$workspace_id"
    
    # Delete workspace
    workspace_manager::delete "$workspace_id" "$force"
}

#######################################
# List active sandbox workspaces
# Returns:
#   JSON array of active workspaces
#######################################
sandbox::list_workspaces() {
    workspace_manager::list "active"
}

#######################################
# Check if sandbox system is ready
# Returns:
#   0 if ready, 1 if not
#######################################
sandbox::is_ready() {
    # Check if all required components are available
    if type -t workspace_manager::create &>/dev/null && \
       type -t workspace_security::apply_policy &>/dev/null && \
       type -t workspace_isolation::create_environment &>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Export main interface functions
export -f sandbox::create_workspace
export -f sandbox::execute
export -f sandbox::get_status
export -f sandbox::cleanup
export -f sandbox::list_workspaces
export -f sandbox::is_ready