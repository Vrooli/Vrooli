#!/usr/bin/env bash
################################################################################
# Write File Tool Executor
# 
# Safely writes content to files with workspace sandboxing
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

################################################################################
# Write File Tool Implementation
################################################################################

#######################################
# Execute write_file tool
# Arguments:
#   $1 - Tool arguments (JSON)
#   $2 - Execution context (sandbox, local)
# Returns:
#   Execution result (JSON)
#######################################
tool_write_file::execute() {
    local arguments="$1"
    local context="${2:-sandbox}"
    
    log::debug "Executing write_file tool in $context context"
    
    # Extract arguments
    local path content
    path=$(echo "$arguments" | jq -r '.path // ""')
    content=$(echo "$arguments" | jq -r '.content // ""')
    
    # Validate arguments
    if [[ -z "$path" ]]; then
        echo '{"success": false, "error": "Missing required parameter: path"}'
        return 1
    fi
    
    # Get workspace directory based on context
    local workspace_dir
    case "$context" in
        sandbox)
            workspace_dir="${CODEX_WORKSPACE:-/tmp/codex-workspace}"
            mkdir -p "$workspace_dir"
            ;;
        local)
            workspace_dir="${PWD}"
            ;;
        *)
            workspace_dir="${CODEX_WORKSPACE:-/tmp/codex-workspace}"
            mkdir -p "$workspace_dir"
            ;;
    esac
    
    # Ensure path is within workspace for sandbox mode
    local final_path
    if [[ "$context" == "sandbox" ]]; then
        # Prevent path traversal attacks
        if [[ "$path" =~ \.\./|^/ ]]; then
            # If path contains ../ or starts with /, make it relative to workspace
            final_path="$workspace_dir/$(basename "$path")"
        elif [[ "$path" =~ ^$workspace_dir/ ]]; then
            # Path already within workspace
            final_path="$path"
        else
            # Make path relative to workspace
            final_path="$workspace_dir/$path"
        fi
    else
        # Local context allows absolute paths but still validate
        if [[ "$path" =~ ^/ ]]; then
            final_path="$path"
        else
            final_path="$workspace_dir/$path"
        fi
    fi
    
    log::debug "Writing to file: $final_path"
    
    # Create directory structure if needed
    local parent_dir
    parent_dir=$(dirname "$final_path")
    if ! mkdir -p "$parent_dir" 2>/dev/null; then
        echo '{"success": false, "error": "Failed to create parent directory"}'
        return 1
    fi
    
    # Write content to file
    if echo "$content" > "$final_path" 2>/dev/null; then
        local file_size
        file_size=$(wc -c < "$final_path" 2>/dev/null || echo "unknown")
        
        echo "{\"success\": true, \"message\": \"File written successfully\", \"path\": \"$final_path\", \"size\": $file_size}"
        return 0
    else
        echo '{"success": false, "error": "Failed to write file - permission denied or invalid path"}'
        return 1
    fi
}

#######################################
# Validate write_file arguments
# Arguments:
#   $1 - Tool arguments (JSON)
# Returns:
#   0 if valid, 1 if invalid
#######################################
tool_write_file::validate() {
    local arguments="$1"
    
    # Check required fields
    if ! echo "$arguments" | jq -e '.path' &>/dev/null; then
        log::debug "write_file: missing path parameter"
        return 1
    fi
    
    if ! echo "$arguments" | jq -e '.content' &>/dev/null; then
        log::debug "write_file: missing content parameter"
        return 1
    fi
    
    # Validate path is not empty
    local path
    path=$(echo "$arguments" | jq -r '.path // ""')
    if [[ -z "$path" ]]; then
        log::debug "write_file: path parameter is empty"
        return 1
    fi
    
    # Basic security check - prevent writing to sensitive system files
    if [[ "$path" =~ ^(/etc/|/bin/|/sbin/|/usr/bin/|/usr/sbin/|/boot/) ]]; then
        log::debug "write_file: attempt to write to restricted system path"
        return 1
    fi
    
    return 0
}

#######################################
# Get tool information
# Returns:
#   Tool info JSON
#######################################
tool_write_file::info() {
    cat << 'EOF'
{
  "name": "write_file",
  "category": "file",
  "description": "Write content to a file with workspace sandboxing",
  "security_level": "medium",
  "supports_contexts": ["sandbox", "local"],
  "restrictions": [
    "System paths are restricted in all contexts",
    "Sandbox context limits to workspace directory",
    "Path traversal attempts are prevented"
  ]
}
EOF
}

# Export functions
export -f tool_write_file::execute
export -f tool_write_file::validate