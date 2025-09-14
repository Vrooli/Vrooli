#!/usr/bin/env bash
################################################################################
# Git Push Tool Executor
# 
# Handles git push operations with safety checks and confirmations
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

#######################################
# Execute git push tool
# Arguments:
#   $1 - Tool arguments (JSON)
#   $2 - Execution context
# Returns:
#   JSON result with push information
#######################################
tool_git_push::execute() {
    local arguments="$1"
    local context="${2:-sandbox}"
    
    # Parse arguments
    local remote branch force set_upstream path
    remote=$(echo "$arguments" | jq -r '.remote // "origin"')
    branch=$(echo "$arguments" | jq -r '.branch // empty')
    force=$(echo "$arguments" | jq -r '.force // false')
    set_upstream=$(echo "$arguments" | jq -r '.set_upstream // false')
    path=$(echo "$arguments" | jq -r '.path // "."')
    
    # Change to repository directory
    if ! cd "$path" 2>/dev/null; then
        echo '{"success": false, "error": "Cannot access directory: '"$path"'"}'
        return 1
    fi
    
    # Check if it's a git repository
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        echo '{"success": false, "error": "Not a git repository"}'
        return 1
    fi
    
    # Determine branch to push
    if [[ -z "$branch" || "$branch" == "null" ]]; then
        branch=$(git branch --show-current 2>/dev/null)
        if [[ -z "$branch" ]]; then
            echo '{"success": false, "error": "Cannot determine current branch"}'
            return 1
        fi
    fi
    
    # Check if remote exists
    if ! git remote get-url "$remote" >/dev/null 2>&1; then
        echo '{"success": false, "error": "Remote does not exist: '"$remote"'"}'
        return 1
    fi
    
    # Safety check for force push
    if [[ "$force" == "true" ]]; then
        log::warn "⚠️  WARNING: Force push requested - this can overwrite remote history!"
        
        # In non-interactive environments, reject force push unless explicitly confirmed
        if [[ "${CODEX_SKIP_CONFIRMATIONS:-}" != "true" && -t 0 ]]; then
            echo -n "Force push to $remote/$branch? This is DANGEROUS and can lose commits. Type 'yes' to continue: "
            read -r confirmation
            if [[ "$confirmation" != "yes" ]]; then
                echo '{"success": false, "error": "Force push cancelled by user"}'
                return 1
            fi
        elif [[ "${CODEX_SKIP_CONFIRMATIONS:-}" != "true" ]]; then
            echo '{"success": false, "error": "Force push requires interactive confirmation"}'
            return 1
        fi
    fi
    
    # Build push command
    local push_command="git push"
    
    if [[ "$set_upstream" == "true" ]]; then
        push_command="$push_command -u"
    fi
    
    if [[ "$force" == "true" ]]; then
        push_command="$push_command --force-with-lease"  # Safer than --force
    fi
    
    push_command="$push_command $remote $branch"
    
    # Check for unpushed commits
    local unpushed_commits
    unpushed_commits=$(git log "$remote/$branch..$branch" --oneline 2>/dev/null | wc -l || echo "unknown")
    
    # Execute push
    log::info "Executing: $push_command"
    local push_output
    push_output=$(eval "$push_command" 2>&1)
    local push_status=$?
    
    if [[ $push_status -eq 0 ]]; then
        # Get updated remote info
        local remote_hash
        remote_hash=$(git ls-remote "$remote" "refs/heads/$branch" 2>/dev/null | cut -f1 || echo "unknown")
        
        # Check if upstream was set
        local upstream_info=""
        local tracking_branch
        tracking_branch=$(git rev-parse --abbrev-ref --symbolic-full-name @{u} 2>/dev/null || echo "none")
        
        if [[ "$tracking_branch" != "none" ]]; then
            upstream_info="$tracking_branch"
        fi
        
        cat << EOF
{
  "success": true,
  "remote": "$remote",
  "branch": "$branch",
  "force_used": $force,
  "upstream_set": $([ -n "$upstream_info" ] && echo "true" || echo "false"),
  "upstream_branch": "$upstream_info",
  "commits_pushed": "$unpushed_commits",
  "remote_hash": "$remote_hash",
  "push_output": $(echo "$push_output" | jq -R -s .)
}
EOF
    else
        # Analyze common push failures
        local error_type="unknown"
        local suggestions=""
        
        if [[ "$push_output" =~ "rejected" ]]; then
            error_type="rejected"
            suggestions="The remote has changes you don't have locally. Try 'git pull' first or use force push if you're certain."
        elif [[ "$push_output" =~ "non-fast-forward" ]]; then
            error_type="non-fast-forward"
            suggestions="Your local branch is behind the remote. Pull the latest changes first."
        elif [[ "$push_output" =~ "authentication" ]]; then
            error_type="authentication"
            suggestions="Authentication failed. Check your credentials or SSH keys."
        elif [[ "$push_output" =~ "no such remote" ]]; then
            error_type="no-remote"
            suggestions="The remote '$remote' does not exist."
        fi
        
        cat << EOF
{
  "success": false,
  "error": "Push failed",
  "error_type": "$error_type",
  "suggestions": "$suggestions",
  "details": $(echo "$push_output" | jq -R -s .)
}
EOF
        return 1
    fi
    
    return 0
}

# Export function
export -f tool_git_push::execute