#!/usr/bin/env bash
################################################################################
# Git Branch Tool Executor
# 
# Handles git branch operations (list, create, switch, delete)
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

#######################################
# Execute git branch tool
# Arguments:
#   $1 - Tool arguments (JSON)
#   $2 - Execution context
# Returns:
#   JSON result with branch operation information
#######################################
tool_git_branch::execute() {
    local arguments="$1"
    local context="${2:-sandbox}"
    
    # Parse arguments
    local action branch_name base_branch force path
    action=$(echo "$arguments" | jq -r '.action')
    branch_name=$(echo "$arguments" | jq -r '.branch_name // empty')
    base_branch=$(echo "$arguments" | jq -r '.base_branch // "main"')
    force=$(echo "$arguments" | jq -r '.force // false')
    path=$(echo "$arguments" | jq -r '.path // "."')
    
    # Validate required parameters
    if [[ "$action" == "null" || -z "$action" ]]; then
        echo '{"success": false, "error": "Action is required (list, create, switch, delete)"}'
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
    
    case "$action" in
        "list")
            local branches_output current_branch
            branches_output=$(git branch -a 2>/dev/null || echo "")
            current_branch=$(git branch --show-current 2>/dev/null || echo "HEAD")
            
            # Parse branch list
            local local_branches=() remote_branches=()
            while IFS= read -r line; do
                [[ -z "$line" ]] && continue
                
                # Remove leading spaces and markers
                local branch_name="${line#* }"
                branch_name="${branch_name#remotes/}"
                
                if [[ "$line" =~ remotes/ ]]; then
                    remote_branches+=("\"$branch_name\"")
                else
                    local_branches+=("\"$branch_name\"")
                fi
            done <<< "$branches_output"
            
            local local_json="[$(IFS=,; echo "${local_branches[*]}")]"
            local remote_json="[$(IFS=,; echo "${remote_branches[*]}")]"
            
            cat << EOF
{
  "success": true,
  "action": "list",
  "current_branch": "$current_branch",
  "local_branches": $local_json,
  "remote_branches": $remote_json
}
EOF
            ;;
            
        "create")
            if [[ -z "$branch_name" ]]; then
                echo '{"success": false, "error": "branch_name is required for create action"}'
                return 1
            fi
            
            # Check if branch already exists
            if git show-ref --verify --quiet "refs/heads/$branch_name"; then
                if [[ "$force" != "true" ]]; then
                    echo '{"success": false, "error": "Branch already exists: '"$branch_name"'"}'
                    return 1
                fi
            fi
            
            local create_output
            if [[ "$force" == "true" ]]; then
                create_output=$(git checkout -B "$branch_name" "$base_branch" 2>&1)
            else
                create_output=$(git checkout -b "$branch_name" "$base_branch" 2>&1)
            fi
            
            if [[ $? -eq 0 ]]; then
                cat << EOF
{
  "success": true,
  "action": "create",
  "branch_name": "$branch_name",
  "base_branch": "$base_branch",
  "output": $(echo "$create_output" | jq -R -s .)
}
EOF
            else
                echo '{"success": false, "error": "Failed to create branch", "details": "'"$create_output"'"}'
                return 1
            fi
            ;;
            
        "switch")
            if [[ -z "$branch_name" ]]; then
                echo '{"success": false, "error": "branch_name is required for switch action"}'
                return 1
            fi
            
            local switch_output
            switch_output=$(git checkout "$branch_name" 2>&1)
            
            if [[ $? -eq 0 ]]; then
                local current_branch
                current_branch=$(git branch --show-current 2>/dev/null || echo "HEAD")
                
                cat << EOF
{
  "success": true,
  "action": "switch",
  "branch_name": "$branch_name",
  "current_branch": "$current_branch",
  "output": $(echo "$switch_output" | jq -R -s .)
}
EOF
            else
                echo '{"success": false, "error": "Failed to switch branch", "details": "'"$switch_output"'"}'
                return 1
            fi
            ;;
            
        "delete")
            if [[ -z "$branch_name" ]]; then
                echo '{"success": false, "error": "branch_name is required for delete action"}'
                return 1
            fi
            
            # Check if trying to delete current branch
            local current_branch
            current_branch=$(git branch --show-current 2>/dev/null || echo "HEAD")
            if [[ "$current_branch" == "$branch_name" ]]; then
                echo '{"success": false, "error": "Cannot delete current branch. Switch to another branch first."}'
                return 1
            fi
            
            local delete_output
            if [[ "$force" == "true" ]]; then
                delete_output=$(git branch -D "$branch_name" 2>&1)
            else
                delete_output=$(git branch -d "$branch_name" 2>&1)
            fi
            
            if [[ $? -eq 0 ]]; then
                cat << EOF
{
  "success": true,
  "action": "delete",
  "branch_name": "$branch_name",
  "force": $force,
  "output": $(echo "$delete_output" | jq -R -s .)
}
EOF
            else
                echo '{"success": false, "error": "Failed to delete branch", "details": "'"$delete_output"'"}'
                return 1
            fi
            ;;
            
        *)
            echo '{"success": false, "error": "Invalid action: '"$action"'. Valid actions are: list, create, switch, delete"}'
            return 1
            ;;
    esac
    
    return 0
}

# Export function
export -f tool_git_branch::execute