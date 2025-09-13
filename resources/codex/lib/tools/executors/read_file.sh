#!/usr/bin/env bash
################################################################################
# Read File Tool Executor
# 
# Safely reads content from files with workspace sandboxing
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

################################################################################
# Read File Tool Implementation
################################################################################

#######################################
# Execute read_file tool
# Arguments:
#   $1 - Tool arguments (JSON)
#   $2 - Execution context (sandbox, local)
# Returns:
#   Execution result (JSON)
#######################################
tool_read_file::execute() {
    local arguments="$1"
    local context="${2:-sandbox}"
    
    log::debug "Executing read_file tool in $context context"
    
    # Extract arguments
    local path
    path=$(echo "$arguments" | jq -r '.path // ""')
    
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
            ;;
        local)
            workspace_dir="${PWD}"
            ;;
        *)
            workspace_dir="${CODEX_WORKSPACE:-/tmp/codex-workspace}"
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
        # Local context allows more flexibility but still validate
        if [[ "$path" =~ ^/ ]]; then
            final_path="$path"
        else
            final_path="$workspace_dir/$path"
        fi
    fi
    
    log::debug "Reading from file: $final_path"
    
    # Check if file exists
    if [[ ! -f "$final_path" ]]; then
        echo "{\"success\": false, \"error\": \"File not found: $final_path\"}"
        return 1
    fi
    
    # Check file permissions
    if [[ ! -r "$final_path" ]]; then
        echo '{"success": false, "error": "File is not readable - permission denied"}'
        return 1
    fi
    
    # Get file information
    local file_size file_type
    file_size=$(wc -c < "$final_path" 2>/dev/null || echo "0")
    file_type=$(file -b --mime-type "$final_path" 2>/dev/null || echo "unknown")
    
    # Check if file is too large (limit to 1MB for safety)
    local max_size=$((1024 * 1024))  # 1MB
    if [[ $file_size -gt $max_size ]]; then
        echo "{\"success\": false, \"error\": \"File too large: $file_size bytes (max: $max_size bytes)\"}"
        return 1
    fi
    
    # Check if file is binary
    if [[ "$file_type" != text/* && "$file_type" != "application/json" && "$file_type" != "application/xml" ]]; then
        echo "{\"success\": false, \"error\": \"Cannot read binary file (type: $file_type)\"}"
        return 1
    fi
    
    # Read file content
    local content
    if content=$(cat "$final_path" 2>/dev/null); then
        # Escape content for JSON
        local escaped_content
        escaped_content=$(echo "$content" | jq -Rs .)
        
        echo "{\"success\": true, \"content\": $escaped_content, \"path\": \"$final_path\", \"size\": $file_size, \"type\": \"$file_type\"}"
        return 0
    else
        echo '{"success": false, "error": "Failed to read file content"}'
        return 1
    fi
}

#######################################
# Read file with line range (for large files)
# Arguments:
#   $1 - Tool arguments (JSON with path, start_line, end_line)
#   $2 - Execution context
# Returns:
#   Execution result (JSON)
#######################################
tool_read_file::read_range() {
    local arguments="$1"
    local context="${2:-sandbox}"
    
    # Extract arguments
    local path start_line end_line
    path=$(echo "$arguments" | jq -r '.path // ""')
    start_line=$(echo "$arguments" | jq -r '.start_line // 1')
    end_line=$(echo "$arguments" | jq -r '.end_line // 100')
    
    # Validate line numbers
    if [[ $start_line -lt 1 || $end_line -lt $start_line ]]; then
        echo '{"success": false, "error": "Invalid line range"}'
        return 1
    fi
    
    # Get file path using same logic as regular read
    local workspace_dir
    case "$context" in
        sandbox)
            workspace_dir="${CODEX_WORKSPACE:-/tmp/codex-workspace}"
            ;;
        local)
            workspace_dir="${PWD}"
            ;;
        *)
            workspace_dir="${CODEX_WORKSPACE:-/tmp/codex-workspace}"
            ;;
    esac
    
    local final_path
    if [[ "$context" == "sandbox" ]]; then
        if [[ "$path" =~ \.\./|^/ ]]; then
            final_path="$workspace_dir/$(basename "$path")"
        elif [[ "$path" =~ ^$workspace_dir/ ]]; then
            final_path="$path"
        else
            final_path="$workspace_dir/$path"
        fi
    else
        if [[ "$path" =~ ^/ ]]; then
            final_path="$path"
        else
            final_path="$workspace_dir/$path"
        fi
    fi
    
    # Check if file exists
    if [[ ! -f "$final_path" ]]; then
        echo "{\"success\": false, \"error\": \"File not found: $final_path\"}"
        return 1
    fi
    
    # Read specified line range
    local content
    if content=$(sed -n "${start_line},${end_line}p" "$final_path" 2>/dev/null); then
        local escaped_content
        escaped_content=$(echo "$content" | jq -Rs .)
        
        local total_lines
        total_lines=$(wc -l < "$final_path" 2>/dev/null || echo "0")
        
        echo "{\"success\": true, \"content\": $escaped_content, \"path\": \"$final_path\", \"start_line\": $start_line, \"end_line\": $end_line, \"total_lines\": $total_lines}"
        return 0
    else
        echo '{"success": false, "error": "Failed to read file range"}'
        return 1
    fi
}

#######################################
# Validate read_file arguments
# Arguments:
#   $1 - Tool arguments (JSON)
# Returns:
#   0 if valid, 1 if invalid
#######################################
tool_read_file::validate() {
    local arguments="$1"
    
    # Check required fields
    if ! echo "$arguments" | jq -e '.path' &>/dev/null; then
        log::debug "read_file: missing path parameter"
        return 1
    fi
    
    # Validate path is not empty
    local path
    path=$(echo "$arguments" | jq -r '.path // ""')
    if [[ -z "$path" ]]; then
        log::debug "read_file: path parameter is empty"
        return 1
    fi
    
    # Basic security check - prevent reading sensitive system files
    if [[ "$path" =~ ^(/etc/shadow|/etc/passwd|/proc/|/sys/) ]]; then
        log::debug "read_file: attempt to read restricted system path"
        return 1
    fi
    
    return 0
}

#######################################
# Get tool information
# Returns:
#   Tool info JSON
#######################################
tool_read_file::info() {
    cat << 'EOF'
{
  "name": "read_file",
  "category": "file",
  "description": "Read content from files with workspace sandboxing",
  "security_level": "medium",
  "supports_contexts": ["sandbox", "local"],
  "limitations": [
    "Maximum file size: 1MB",
    "Text files only (no binary files)",
    "Sensitive system files are restricted"
  ],
  "restrictions": [
    "Sandbox context limits to workspace directory",
    "Path traversal attempts are prevented",
    "Binary files are blocked for security"
  ]
}
EOF
}

# Export functions
export -f tool_read_file::execute
export -f tool_read_file::read_range
export -f tool_read_file::validate