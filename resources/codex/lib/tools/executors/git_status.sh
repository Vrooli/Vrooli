#!/usr/bin/env bash
################################################################################
# Git Status Tool Executor
# 
# Provides structured git status information with proper error handling
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

#######################################
# Execute git status tool
# Arguments:
#   $1 - Tool arguments (JSON)
#   $2 - Execution context
# Returns:
#   JSON result with git status information
#######################################
tool_git_status::execute() {
    local arguments="$1"
    local context="${2:-sandbox}"
    
    # Parse arguments
    local path porcelain
    path=$(echo "$arguments" | jq -r '.path // "."')
    porcelain=$(echo "$arguments" | jq -r '.porcelain // true')
    
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
    
    # Get git status
    local status_output branch_info
    
    if [[ "$porcelain" == "true" ]]; then
        status_output=$(git status --porcelain 2>/dev/null || echo "")
        branch_info=$(git branch --show-current 2>/dev/null || echo "HEAD")
    else
        status_output=$(git status 2>/dev/null || echo "")
        branch_info=$(git branch --show-current 2>/dev/null || echo "HEAD")
    fi
    
    # Parse porcelain output if enabled
    if [[ "$porcelain" == "true" ]]; then
        local staged_files=()
        local modified_files=()
        local untracked_files=()
        
        while IFS= read -r line; do
            [[ -z "$line" ]] && continue
            
            local status_code="${line:0:2}"
            local file_path="${line:3}"
            
            case "$status_code" in
                "M "*)  staged_files+=("\"$file_path\"") ;;
                " M"*)  modified_files+=("\"$file_path\"") ;;
                "A "*)  staged_files+=("\"$file_path\"") ;;
                "??"*)  untracked_files+=("\"$file_path\"") ;;
                *) ;;
            esac
        done <<< "$status_output"
        
        # Build JSON response
        local staged_json="[$(IFS=,; echo "${staged_files[*]}")]"
        local modified_json="[$(IFS=,; echo "${modified_files[*]}")]"
        local untracked_json="[$(IFS=,; echo "${untracked_files[*]}")]"
        
        cat << EOF
{
  "success": true,
  "current_branch": "$branch_info",
  "staged_files": $staged_json,
  "modified_files": $modified_json,
  "untracked_files": $untracked_json,
  "clean": $([ ${#staged_files[@]} -eq 0 ] && [ ${#modified_files[@]} -eq 0 ] && [ ${#untracked_files[@]} -eq 0 ] && echo "true" || echo "false"),
  "raw_output": $(echo "$status_output" | jq -R -s .)
}
EOF
    else
        # Return human-readable status
        cat << EOF
{
  "success": true,
  "current_branch": "$branch_info",
  "status_output": $(echo "$status_output" | jq -R -s .)
}
EOF
    fi
    
    return 0
}

# Export function
export -f tool_git_status::execute