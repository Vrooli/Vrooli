#!/usr/bin/env bash
################################################################################
# Git Commit Tool Executor
# 
# Handles git commit operations with staging and validation
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

#######################################
# Execute git commit tool
# Arguments:
#   $1 - Tool arguments (JSON)
#   $2 - Execution context
# Returns:
#   JSON result with commit information
#######################################
tool_git_commit::execute() {
    local arguments="$1"
    local context="${2:-sandbox}"
    
    # Parse arguments
    local message files all path
    message=$(echo "$arguments" | jq -r '.message')
    files=$(echo "$arguments" | jq -r '.files[]? // empty' 2>/dev/null)
    all=$(echo "$arguments" | jq -r '.all // false')
    path=$(echo "$arguments" | jq -r '.path // "."')
    
    # Validate required parameters
    if [[ "$message" == "null" || -z "$message" ]]; then
        echo '{"success": false, "error": "Commit message is required"}'
        return 1
    fi
    
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
    
    # Stage files if specified
    local staging_output=""
    
    if [[ "$all" == "true" ]]; then
        staging_output=$(git add -A 2>&1)
        if [[ $? -ne 0 ]]; then
            echo '{"success": false, "error": "Failed to stage all files", "details": "'"$staging_output"'"}'
            return 1
        fi
    elif [[ -n "$files" ]]; then
        local staging_errors=()
        while IFS= read -r file; do
            [[ -z "$file" ]] && continue
            if ! git add "$file" 2>/dev/null; then
                staging_errors+=("$file")
            fi
        done <<< "$files"
        
        if [[ ${#staging_errors[@]} -gt 0 ]]; then
            local error_list=$(printf '"%s",' "${staging_errors[@]}" | sed 's/,$//')
            echo '{"success": false, "error": "Failed to stage some files", "failed_files": ['"$error_list"']}'
            return 1
        fi
    fi
    
    # Check if there are staged changes
    if ! git diff --staged --quiet; then
        # Proceed with commit
        local commit_output
        commit_output=$(git commit -m "$message" 2>&1)
        local commit_status=$?
        
        if [[ $commit_status -eq 0 ]]; then
            # Extract commit hash
            local commit_hash
            commit_hash=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
            
            # Get commit info
            local commit_info
            commit_info=$(git show --stat --format="Author: %an <%ae>%nDate: %ad%nHash: %H" HEAD 2>/dev/null || echo "")
            
            cat << EOF
{
  "success": true,
  "commit_hash": "$commit_hash",
  "message": "$message",
  "commit_output": $(echo "$commit_output" | jq -R -s .),
  "commit_info": $(echo "$commit_info" | jq -R -s .)
}
EOF
        else
            echo '{"success": false, "error": "Commit failed", "details": "'"$commit_output"'"}'
            return 1
        fi
    else
        # Check if working directory is clean
        if git diff --quiet; then
            echo '{"success": false, "error": "No changes to commit - working directory is clean"}'
        else
            echo '{"success": false, "error": "No staged changes to commit. Use git add or set all=true to stage changes"}'
        fi
        return 1
    fi
    
    return 0
}

# Export function
export -f tool_git_commit::execute