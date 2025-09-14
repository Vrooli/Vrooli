#!/usr/bin/env bash
################################################################################
# Git Clone Tool Executor
# 
# Handles git clone operations with validation and progress tracking
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

#######################################
# Execute git clone tool
# Arguments:
#   $1 - Tool arguments (JSON)
#   $2 - Execution context
# Returns:
#   JSON result with clone information
#######################################
tool_git_clone::execute() {
    local arguments="$1"
    local context="${2:-sandbox}"
    
    # Parse arguments
    local url destination branch depth recursive
    url=$(echo "$arguments" | jq -r '.url')
    destination=$(echo "$arguments" | jq -r '.destination // empty')
    branch=$(echo "$arguments" | jq -r '.branch // empty')
    depth=$(echo "$arguments" | jq -r '.depth // empty')
    recursive=$(echo "$arguments" | jq -r '.recursive // false')
    
    # Validate required parameters
    if [[ "$url" == "null" || -z "$url" ]]; then
        echo '{"success": false, "error": "Repository URL is required"}'
        return 1
    fi
    
    # Validate URL format
    if ! [[ "$url" =~ ^(https?|git|ssh)://.*|.*@.*:.*$ ]]; then
        echo '{"success": false, "error": "Invalid repository URL format"}'
        return 1
    fi
    
    # Determine destination directory
    if [[ -z "$destination" || "$destination" == "null" ]]; then
        # Extract repository name from URL
        destination=$(basename "$url" .git)
    fi
    
    # Check if destination already exists
    if [[ -d "$destination" ]]; then
        if [[ -d "$destination/.git" ]]; then
            echo '{"success": false, "error": "Directory already exists and appears to be a git repository: '"$destination"'"}'
            return 1
        else
            echo '{"success": false, "error": "Directory already exists: '"$destination"'"}'
            return 1
        fi
    fi
    
    # Build clone command
    local clone_command="git clone"
    
    # Add depth option for shallow clone
    if [[ -n "$depth" && "$depth" != "null" && "$depth" != "0" ]]; then
        clone_command="$clone_command --depth $depth"
    fi
    
    # Add branch option
    if [[ -n "$branch" && "$branch" != "null" ]]; then
        clone_command="$clone_command --branch $branch"
    fi
    
    # Add recursive option for submodules
    if [[ "$recursive" == "true" ]]; then
        clone_command="$clone_command --recursive"
    fi
    
    # Add URL and destination
    clone_command="$clone_command \"$url\" \"$destination\""
    
    # Execute clone command
    log::info "Cloning repository: $url"
    log::debug "Command: $clone_command"
    
    local clone_output clone_status start_time end_time
    start_time=$(date +%s)
    clone_output=$(eval "$clone_command" 2>&1)
    clone_status=$?
    end_time=$(date +%s)
    
    local duration=$((end_time - start_time))
    
    if [[ $clone_status -eq 0 ]]; then
        # Get repository information
        local repo_info=""
        if cd "$destination" 2>/dev/null; then
            local current_branch remote_url commit_hash
            current_branch=$(git branch --show-current 2>/dev/null || echo "HEAD")
            remote_url=$(git remote get-url origin 2>/dev/null || echo "unknown")
            commit_hash=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
            
            # Count commits (if not a shallow clone)
            local commit_count="unknown"
            if [[ -z "$depth" || "$depth" == "null" ]]; then
                commit_count=$(git rev-list --count HEAD 2>/dev/null || echo "unknown")
            fi
            
            # Check for submodules
            local has_submodules="false"
            if [[ -f ".gitmodules" ]]; then
                has_submodules="true"
            fi
            
            # Get repository size
            local repo_size
            repo_size=$(du -sh . 2>/dev/null | cut -f1 || echo "unknown")
            
            cd - >/dev/null
            
            repo_info=$(cat << EOF
{
  "current_branch": "$current_branch",
  "remote_url": "$remote_url",
  "commit_hash": "$commit_hash",
  "commit_count": "$commit_count",
  "has_submodules": $has_submodules,
  "repository_size": "$repo_size"
}
EOF
            )
        fi
        
        cat << EOF
{
  "success": true,
  "url": "$url",
  "destination": "$destination",
  "branch": $([ -n "$branch" ] && echo "\"$branch\"" || echo "null"),
  "depth": $([ -n "$depth" ] && echo "$depth" || echo "null"),
  "recursive": $recursive,
  "duration_seconds": $duration,
  "repository_info": $repo_info,
  "clone_output": $(echo "$clone_output" | jq -R -s .)
}
EOF
    else
        # Analyze common clone failures
        local error_type="unknown"
        local suggestions=""
        
        if [[ "$clone_output" =~ "authentication failed" || "$clone_output" =~ "Permission denied" ]]; then
            error_type="authentication"
            suggestions="Authentication failed. Check your credentials, SSH keys, or repository permissions."
        elif [[ "$clone_output" =~ "repository not found" || "$clone_output" =~ "does not exist" ]]; then
            error_type="not-found"
            suggestions="Repository not found. Check the URL or verify the repository exists."
        elif [[ "$clone_output" =~ "could not resolve host" ]]; then
            error_type="network"
            suggestions="Network error. Check your internet connection and DNS resolution."
        elif [[ "$clone_output" =~ "directory not empty" ]]; then
            error_type="directory-exists"
            suggestions="Destination directory already exists and is not empty."
        elif [[ "$clone_output" =~ "branch.*not found" ]]; then
            error_type="branch-not-found"
            suggestions="The specified branch '$branch' does not exist in the repository."
        fi
        
        # Clean up any partial clone
        if [[ -d "$destination" ]]; then
            rm -rf "$destination"
        fi
        
        cat << EOF
{
  "success": false,
  "error": "Clone failed",
  "error_type": "$error_type",
  "suggestions": "$suggestions",
  "duration_seconds": $duration,
  "details": $(echo "$clone_output" | jq -R -s .)
}
EOF
        return 1
    fi
    
    return 0
}

# Export function
export -f tool_git_clone::execute