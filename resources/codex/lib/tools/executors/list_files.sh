#!/usr/bin/env bash
################################################################################
# List Files Tool Executor
# 
# Safely lists files and directories with workspace sandboxing
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

################################################################################
# List Files Tool Implementation
################################################################################

#######################################
# Execute list_files tool
# Arguments:
#   $1 - Tool arguments (JSON)
#   $2 - Execution context (sandbox, local)
# Returns:
#   Execution result (JSON)
#######################################
tool_list_files::execute() {
    local arguments="$1"
    local context="${2:-sandbox}"
    
    log::debug "Executing list_files tool in $context context"
    
    # Extract arguments
    local path show_hidden recursive max_depth
    path=$(echo "$arguments" | jq -r '.path // "."')
    show_hidden=$(echo "$arguments" | jq -r '.show_hidden // false')
    recursive=$(echo "$arguments" | jq -r '.recursive // false')
    max_depth=$(echo "$arguments" | jq -r '.max_depth // 3')
    
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
        if [[ "$path" == "." ]]; then
            final_path="$workspace_dir"
        elif [[ "$path" =~ \.\./|^/ ]]; then
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
        # Local context allows more flexibility
        if [[ "$path" =~ ^/ ]]; then
            final_path="$path"
        elif [[ "$path" == "." ]]; then
            final_path="$workspace_dir"
        else
            final_path="$workspace_dir/$path"
        fi
    fi
    
    log::debug "Listing directory: $final_path"
    
    # Check if directory exists
    if [[ ! -d "$final_path" ]]; then
        echo "{\"success\": false, \"error\": \"Directory not found: $final_path\"}"
        return 1
    fi
    
    # Check directory permissions
    if [[ ! -r "$final_path" ]]; then
        echo '{"success": false, "error": "Directory is not readable - permission denied"}'
        return 1
    fi
    
    # Build ls/find command based on options
    local files_json="[]"
    local total_files=0
    local total_dirs=0
    
    if [[ "$recursive" == "true" ]]; then
        # Use find for recursive listing
        local find_cmd=("find" "$final_path" "-maxdepth" "$max_depth" "-type" "f,d")
        
        if [[ "$show_hidden" != "true" ]]; then
            find_cmd+=("-not" "-path" "*/.*")
        fi
        
        # Execute find and process results
        while IFS= read -r -d '' item; do
            [[ -z "$item" ]] && continue
            
            local item_info
            item_info=$(tool_list_files::get_item_info "$item" "$final_path")
            
            if [[ -n "$item_info" ]]; then
                files_json=$(echo "$files_json" | jq ". + [$item_info]")
                
                local item_type
                item_type=$(echo "$item_info" | jq -r '.type')
                if [[ "$item_type" == "file" ]]; then
                    ((total_files++))
                elif [[ "$item_type" == "directory" ]]; then
                    ((total_dirs++))
                fi
            fi
        done < <(find "$final_path" -maxdepth "$max_depth" \( -type f -o -type d \) \
                $(if [[ "$show_hidden" != "true" ]]; then echo "-not -path '*/.*'"; fi) \
                -print0 2>/dev/null)
    else
        # Use ls for non-recursive listing
        local ls_opts="-la"
        if [[ "$show_hidden" != "true" ]]; then
            ls_opts="-l"
        fi
        
        # Get directory contents
        while IFS= read -r line; do
            [[ -z "$line" ]] && continue
            
            # Skip total line from ls -l
            [[ "$line" =~ ^total ]] && continue
            
            # Parse ls output
            local permissions links owner group size date time name
            read -r permissions links owner group size date time name <<< "$line"
            
            # Skip . and .. entries unless explicitly requested
            if [[ "$name" == "." || "$name" == ".." ]]; then
                continue
            fi
            
            # Build full path
            local full_item_path="$final_path/$name"
            
            local item_info
            item_info=$(tool_list_files::get_item_info "$full_item_path" "$final_path" "$permissions" "$size")
            
            if [[ -n "$item_info" ]]; then
                files_json=$(echo "$files_json" | jq ". + [$item_info]")
                
                local item_type
                item_type=$(echo "$item_info" | jq -r '.type')
                if [[ "$item_type" == "file" ]]; then
                    ((total_files++))
                elif [[ "$item_type" == "directory" ]]; then
                    ((total_dirs++))
                fi
            fi
        done < <(ls $ls_opts "$final_path" 2>/dev/null)
    fi
    
    # Build response
    cat << EOF
{
  "success": true,
  "path": "$final_path",
  "files": $files_json,
  "summary": {
    "total_files": $total_files,
    "total_directories": $total_dirs,
    "total_items": $((total_files + total_dirs)),
    "recursive": $recursive,
    "show_hidden": $show_hidden
  }
}
EOF
    
    return 0
}

#######################################
# Get detailed information about a file or directory
# Arguments:
#   $1 - Item path
#   $2 - Base directory
#   $3 - Permissions (optional, from ls output)
#   $4 - Size (optional, from ls output)
# Returns:
#   Item info JSON
#######################################
tool_list_files::get_item_info() {
    local item_path="$1"
    local base_dir="$2"
    local permissions="$3"
    local ls_size="$4"
    
    # Check if item exists
    if [[ ! -e "$item_path" ]]; then
        return 1
    fi
    
    # Get item name relative to base directory
    local item_name
    item_name=$(basename "$item_path")
    
    local relative_path
    relative_path=${item_path#$base_dir/}
    
    # Determine type
    local item_type
    if [[ -d "$item_path" ]]; then
        item_type="directory"
    elif [[ -f "$item_path" ]]; then
        item_type="file"
    elif [[ -L "$item_path" ]]; then
        item_type="symlink"
    else
        item_type="other"
    fi
    
    # Get size
    local size
    if [[ -n "$ls_size" && "$ls_size" != "-" ]]; then
        size="$ls_size"
    elif [[ -f "$item_path" ]]; then
        size=$(stat -f%z "$item_path" 2>/dev/null || stat -c%s "$item_path" 2>/dev/null || echo "0")
    else
        size="0"
    fi
    
    # Get permissions if not provided
    if [[ -z "$permissions" ]]; then
        permissions=$(ls -ld "$item_path" 2>/dev/null | cut -d' ' -f1)
    fi
    
    # Get modification time
    local modified
    modified=$(stat -f%Sm "$item_path" 2>/dev/null || stat -c%Y "$item_path" 2>/dev/null || echo "0")
    
    # Get file type for files
    local mime_type=""
    if [[ "$item_type" == "file" ]]; then
        mime_type=$(file -b --mime-type "$item_path" 2>/dev/null || echo "unknown")
    fi
    
    # Build JSON
    jq -n \
        --arg name "$item_name" \
        --arg path "$relative_path" \
        --arg full_path "$item_path" \
        --arg type "$item_type" \
        --arg size "$size" \
        --arg permissions "$permissions" \
        --arg modified "$modified" \
        --arg mime_type "$mime_type" \
        '{
            name: $name,
            path: $path,
            full_path: $full_path,
            type: $type,
            size: ($size | tonumber),
            permissions: $permissions,
            modified: $modified,
            mime_type: (if $mime_type != "" then $mime_type else null end)
        }'
}

#######################################
# Get directory tree structure
# Arguments:
#   $1 - Tool arguments (JSON with path, max_depth)
#   $2 - Execution context
# Returns:
#   Directory tree JSON
#######################################
tool_list_files::get_tree() {
    local arguments="$1"
    local context="${2:-sandbox}"
    
    # Extract arguments
    local path max_depth
    path=$(echo "$arguments" | jq -r '.path // "."')
    max_depth=$(echo "$arguments" | jq -r '.max_depth // 2')
    
    # Use recursive listing with tree format
    local updated_args
    updated_args=$(echo "$arguments" | jq --argjson recursive true '. + {recursive: $recursive}')
    
    tool_list_files::execute "$updated_args" "$context"
}

#######################################
# Validate list_files arguments
# Arguments:
#   $1 - Tool arguments (JSON)
# Returns:
#   0 if valid, 1 if invalid
#######################################
tool_list_files::validate() {
    local arguments="$1"
    
    # Path is optional, defaults to "."
    local path
    path=$(echo "$arguments" | jq -r '.path // "."')
    
    # Basic security check - prevent listing sensitive system directories
    if [[ "$path" =~ ^(/proc|/sys|/dev|/etc/shadow|/etc/passwd) ]]; then
        log::debug "list_files: attempt to list restricted system path"
        return 1
    fi
    
    # Validate max_depth if provided
    local max_depth
    max_depth=$(echo "$arguments" | jq -r '.max_depth // 3')
    if [[ ! "$max_depth" =~ ^[0-9]+$ ]] || [[ $max_depth -gt 10 ]]; then
        log::debug "list_files: invalid max_depth (must be number ≤ 10)"
        return 1
    fi
    
    return 0
}

#######################################
# Get tool information
# Returns:
#   Tool info JSON
#######################################
tool_list_files::info() {
    cat << 'EOF'
{
  "name": "list_files",
  "category": "file",
  "description": "List files and directories with detailed metadata",
  "security_level": "low",
  "supports_contexts": ["sandbox", "local"],
  "features": [
    "Recursive directory traversal",
    "Hidden files support",
    "Detailed file metadata",
    "Directory tree structure",
    "MIME type detection"
  ],
  "restrictions": [
    "Maximum depth limited to 10 levels",
    "Sensitive system directories blocked",
    "Sandbox context limits to workspace"
  ]
}
EOF
}

# Export functions
export -f tool_list_files::execute
export -f tool_list_files::get_tree
export -f tool_list_files::validate