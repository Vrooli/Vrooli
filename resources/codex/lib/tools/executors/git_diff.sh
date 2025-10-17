#!/usr/bin/env bash
################################################################################
# Git Diff Tool Executor
# 
# Shows git differences with structured output and analysis
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

#######################################
# Execute git diff tool
# Arguments:
#   $1 - Tool arguments (JSON)
#   $2 - Execution context
# Returns:
#   JSON result with diff information
#######################################
tool_git_diff::execute() {
    local arguments="$1"
    local context="${2:-sandbox}"
    
    # Parse arguments
    local target commit1 commit2 file_path context_lines path
    target=$(echo "$arguments" | jq -r '.target // "unstaged"')
    commit1=$(echo "$arguments" | jq -r '.commit1 // empty')
    commit2=$(echo "$arguments" | jq -r '.commit2 // empty')
    file_path=$(echo "$arguments" | jq -r '.file_path // empty')
    context_lines=$(echo "$arguments" | jq -r '.context_lines // 3')
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
    
    local diff_command=""
    local diff_output=""
    local diff_description=""
    
    # Build diff command based on target
    case "$target" in
        "staged")
            diff_command="git diff --cached"
            diff_description="Changes in staging area"
            ;;
        "unstaged")
            diff_command="git diff"
            diff_description="Unstaged changes in working directory"
            ;;
        "commit")
            if [[ -n "$commit1" ]]; then
                if [[ -n "$commit2" ]]; then
                    diff_command="git diff $commit1 $commit2"
                    diff_description="Diff between $commit1 and $commit2"
                else
                    diff_command="git diff $commit1"
                    diff_description="Diff between $commit1 and working directory"
                fi
            else
                diff_command="git diff HEAD~1 HEAD"
                diff_description="Last commit changes"
            fi
            ;;
        *)
            # Treat as file path
            if [[ -f "$target" ]]; then
                diff_command="git diff $target"
                diff_description="Changes in file: $target"
                file_path="$target"
            else
                echo '{"success": false, "error": "Invalid target: '"$target"'. Use staged, unstaged, commit, or file path"}'
                return 1
            fi
            ;;
    esac
    
    # Add file path if specified
    if [[ -n "$file_path" && "$target" != "commit" ]]; then
        diff_command="$diff_command $file_path"
        diff_description="$diff_description (file: $file_path)"
    fi
    
    # Add context lines option
    diff_command="$diff_command -U$context_lines"
    
    # Execute diff command
    diff_output=$(eval "$diff_command" 2>&1)
    local diff_status=$?
    
    if [[ $diff_status -eq 0 ]]; then
        # Parse diff statistics
        local stats_output
        stats_output=$(eval "$diff_command --stat" 2>/dev/null || echo "")
        
        # Count additions and deletions
        local additions=0
        local deletions=0
        local files_changed=0
        
        if [[ -n "$stats_output" ]]; then
            # Extract stats from last line (e.g., "2 files changed, 15 insertions(+), 3 deletions(-)")
            local stats_line
            stats_line=$(echo "$stats_output" | tail -n 1)
            
            if [[ "$stats_line" =~ ([0-9]+)[[:space:]]*(insertion|insertions) ]]; then
                additions="${BASH_REMATCH[1]}"
            fi
            
            if [[ "$stats_line" =~ ([0-9]+)[[:space:]]*(deletion|deletions) ]]; then
                deletions="${BASH_REMATCH[1]}"
            fi
            
            if [[ "$stats_line" =~ ([0-9]+)[[:space:]]*(file|files)[[:space:]]*changed ]]; then
                files_changed="${BASH_REMATCH[1]}"
            fi
        fi
        
        # Determine if there are changes
        local has_changes="false"
        if [[ -n "$diff_output" && "$diff_output" != "" ]]; then
            has_changes="true"
        fi
        
        cat << EOF
{
  "success": true,
  "target": "$target",
  "description": "$diff_description",
  "has_changes": $has_changes,
  "statistics": {
    "files_changed": $files_changed,
    "additions": $additions,
    "deletions": $deletions
  },
  "diff_output": $(echo "$diff_output" | jq -R -s .),
  "stats_output": $(echo "$stats_output" | jq -R -s .)
}
EOF
    else
        # Check if it's a "no changes" scenario (exit code 1 with empty output is normal)
        if [[ $diff_status -eq 1 && -z "$diff_output" ]]; then
            cat << EOF
{
  "success": true,
  "target": "$target",
  "description": "$diff_description",
  "has_changes": false,
  "statistics": {
    "files_changed": 0,
    "additions": 0,
    "deletions": 0
  },
  "diff_output": "",
  "stats_output": ""
}
EOF
        else
            echo '{"success": false, "error": "Git diff command failed", "details": "'"$diff_output"'"}'
            return 1
        fi
    fi
    
    return 0
}

# Export function
export -f tool_git_diff::execute