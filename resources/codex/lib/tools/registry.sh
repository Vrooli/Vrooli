#!/usr/bin/env bash
################################################################################
# Tool Registry - Central catalog of available tools
# 
# Manages tool definitions, executors, and availability
# Provides unified interface for tool discovery and execution
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Source permission system
source "${APP_ROOT}/resources/codex/lib/permissions.sh"

# Tool directories
TOOL_DEFINITIONS_DIR="${APP_ROOT}/resources/codex/lib/tools/definitions"
TOOL_EXECUTORS_DIR="${APP_ROOT}/resources/codex/lib/tools/executors"

################################################################################
# Tool Registry Interface
################################################################################

#######################################
# Get all available tools
# Arguments:
#   $1 - Category filter (optional: file, command, code, api, analysis)
# Returns:
#   JSON array of tool definitions
#######################################
tool_registry::list_tools() {
    local category_filter="${1:-}"

    log::debug "Listing tools, category filter: $category_filter"

    local tools_json
    tools_json=$(tool_registry::get_definitions)

    if [[ -n "$category_filter" ]]; then
        if command -v jq &>/dev/null; then
            echo "$tools_json" | jq --arg category "$category_filter" '[ .[] | select((.category // "general") == $category) ]'
        else
            log::warn "jq not available; returning unfiltered tool list"
            echo "$tools_json"
        fi
    else
        echo "$tools_json"
    fi
}

#######################################
# Enumerate all tool definitions (unfiltered)
# Returns:
#   JSON array of tool definitions
#######################################
tool_registry::enumerate_definitions() {
    local tools_json="[]"

    if [[ -d "$TOOL_DEFINITIONS_DIR" ]]; then
        for def_file in "$TOOL_DEFINITIONS_DIR"/*.json; do
            [[ -f "$def_file" ]] || continue

            local tool_def
            tool_def=$(cat "$def_file" 2>/dev/null || echo '{}')

            local tool_name executor_available
            tool_name=$(echo "$tool_def" | jq -r '.function.name // ""')
            executor_available=$(tool_registry::check_executor "$tool_name")

            tool_def=$(echo "$tool_def" | jq --argjson available "$executor_available" '. + {available: $available}')
            tools_json=$(echo "$tools_json" | jq ". + [$tool_def]")
        done
    fi

    echo "$tools_json"
}

#######################################
# Determine allowed tool patterns under current policy
# Returns:
#   Comma-separated pattern string
#######################################
tool_registry::effective_allowed_patterns() {
    if [[ "${CODEX_SKIP_PERMISSIONS:-}" == "true" ]]; then
        echo "*"
        return
    fi

    local allowed_patterns="${CODEX_ALLOWED_TOOLS:-}"
    if [[ -z "$allowed_patterns" || "$allowed_patterns" == "null" ]]; then
        local from_settings
        from_settings=$(codex_settings::get "tools.allowed" | jq -r '. | join(",")' 2>/dev/null)
        allowed_patterns="${from_settings:-}"
    fi

    if [[ -z "$allowed_patterns" || "$allowed_patterns" == "null" ]]; then
        allowed_patterns="read_file,list_files,analyze_code"
    fi

    echo "$allowed_patterns"
}

#######################################
# Check if a tool name is permitted by pattern list
# Arguments:
#   $1 - Tool name
#   $2 - Allowed pattern string
# Returns:
#   0 if allowed, 1 otherwise
#######################################
tool_registry::is_tool_name_allowed() {
    local tool_name="$1"
    local allowed_list="$2"

    if [[ "${CODEX_SKIP_PERMISSIONS:-}" == "true" ]]; then
        return 0
    fi

    IFS=',' read -ra patterns <<< "$allowed_list"
    for pattern in "${patterns[@]}"; do
        pattern=$(echo "$pattern" | xargs)
        [[ -z "$pattern" ]] && continue

        if [[ "$pattern" == "*" ]]; then
            return 0
        fi

        local base="${pattern%%(*}"

        if [[ "$pattern" == "$tool_name" ]]; then
            return 0
        fi

        if [[ "$pattern" == "$tool_name"* ]]; then
            return 0
        fi

        if [[ "$base" == "$tool_name" ]]; then
            return 0
        fi
    done

    return 1
}

#######################################
# Get tool definitions respecting permission policies
# Returns:
#   JSON array of allowed tool definitions
#######################################
tool_registry::get_definitions() {
    local allowed_patterns
    allowed_patterns=$(tool_registry::effective_allowed_patterns)

    local result="["
    local first=true

    if command -v jq &>/dev/null; then
        while IFS= read -r tool_def; do
            [[ -z "$tool_def" ]] && continue
            local tool_name
            tool_name=$(echo "$tool_def" | jq -r '.function.name // ""')

            if tool_registry::is_tool_name_allowed "$tool_name" "$allowed_patterns"; then
                continue
            fi

            if $first; then
                result+="$tool_def"
                first=false
            else
                result+=",$tool_def"
            fi
        done < <(tool_registry::enumerate_definitions | jq -c '.[]')
    fi

    result+=']'
    echo "$result"
}

# Backwards compatibility alias (older modules expect tools_registry namespace)
tools_registry::get_definitions() {
    tool_registry::get_definitions
}

#######################################
# Get specific tool definition
# Arguments:
#   $1 - Tool name
# Returns:
#   Tool definition JSON or empty object
#######################################
tool_registry::get_tool() {
    local tool_name="$1"
    
    log::debug "Getting tool definition: $tool_name"
    
    local def_file="$TOOL_DEFINITIONS_DIR/$tool_name.json"
    if [[ -f "$def_file" ]]; then
        local tool_def
        tool_def=$(cat "$def_file" 2>/dev/null || echo '{}')
        
        # Check executor availability
        local executor_available
        executor_available=$(tool_registry::check_executor "$tool_name")
        
        # Add availability info
        echo "$tool_def" | jq --argjson available "$executor_available" '. + {available: $available}'
    else
        echo '{}'
    fi
}

#######################################
# Check if tool executor is available
# Arguments:
#   $1 - Tool name
# Returns:
#   true/false (JSON boolean)
#######################################
tool_registry::check_executor() {
    local tool_name="$1"
    
    local executor_file="$TOOL_EXECUTORS_DIR/$tool_name.sh"
    if [[ -f "$executor_file" ]]; then
        echo "true"
    else
        echo "false"
    fi
}

#######################################
# Execute a tool
# Arguments:
#   $1 - Tool name
#   $2 - Tool arguments (JSON)
#   $3 - Execution context (optional: sandbox, local)
# Returns:
#   Tool execution result (JSON)
#######################################
tool_registry::execute_tool() {
    local tool_name="$1"
    local arguments="$2"
    local context="${3:-sandbox}"
    
    log::debug "Executing tool: $tool_name in $context context"
    
    # SECURITY CHECK: Validate permissions before execution
    if ! permissions::check_and_confirm "$tool_name" "$arguments"; then
        echo '{"success": false, "error": "Permission denied: tool not allowed by current security policy"}'
        return 1
    fi
    
    # Check if tool exists
    local tool_def
    tool_def=$(tool_registry::get_tool "$tool_name")
    
    if [[ $(echo "$tool_def" | jq 'keys | length') -eq 0 ]]; then
        echo '{"success": false, "error": "Tool not found"}'
        return 1
    fi
    
    # Check if executor is available
    local executor_available
    executor_available=$(echo "$tool_def" | jq -r '.available // false')
    
    if [[ "$executor_available" != "true" ]]; then
        echo '{"success": false, "error": "Tool executor not available"}'
        return 1
    fi
    
    # Validate arguments against tool schema
    if ! tool_registry::validate_arguments "$tool_name" "$arguments"; then
        echo '{"success": false, "error": "Invalid arguments for tool"}'
        return 1
    fi
    
    # Source and execute the tool
    local executor_file="$TOOL_EXECUTORS_DIR/$tool_name.sh"
    source "$executor_file"
    
    # Execute with context information
    local executor_function="tool_${tool_name}::execute"
    if type -t "$executor_function" &>/dev/null; then
        "$executor_function" "$arguments" "$context"
    else
        echo '{"success": false, "error": "Tool executor function not found"}'
        return 1
    fi
}

################################################################################
# Tool Management
################################################################################

#######################################
# Register a new tool
# Arguments:
#   $1 - Tool definition (JSON)
# Returns:
#   0 if successful, 1 if failed
#######################################
tool_registry::register_tool() {
    local tool_def="$1"
    
    # Extract tool name
    local tool_name
    tool_name=$(echo "$tool_def" | jq -r '.function.name // ""')
    
    if [[ -z "$tool_name" ]]; then
        log::error "Tool definition missing name"
        return 1
    fi
    
    # Validate tool definition structure
    if ! tool_registry::validate_definition "$tool_def"; then
        log::error "Invalid tool definition for $tool_name"
        return 1
    fi
    
    # Save tool definition
    local def_file="$TOOL_DEFINITIONS_DIR/$tool_name.json"
    mkdir -p "$TOOL_DEFINITIONS_DIR"
    
    if echo "$tool_def" | jq . > "$def_file"; then
        log::info "Registered tool: $tool_name"
        return 0
    else
        log::error "Failed to register tool: $tool_name"
        return 1
    fi
}

#######################################
# Validate a tool definition
# Arguments:
#   $1 - Tool definition (JSON)
# Returns:
#   0 if valid, 1 otherwise
#######################################
tool_registry::validate_definition() {
    local tool_def="$1"
    
    if [[ -z "$tool_def" ]]; then
        return 1
    fi
    
    if ! echo "$tool_def" | jq -e '.function.name and .function.parameters' >/dev/null 2>&1; then
        return 1
    fi
    
    return 0
}

#######################################
# Validate tool arguments against schema
# Arguments:
#   $1 - Tool name
#   $2 - Tool arguments (JSON)
# Returns:
#   0 if valid, 1 otherwise
#######################################
tool_registry::validate_arguments() {
    local tool_name="$1"
    local arguments="$2"
    
    local tool_def
    tool_def=$(tool_registry::get_tool "$tool_name")
    
    if [[ $(echo "$tool_def" | jq 'keys | length') -eq 0 ]]; then
        return 1
    fi
    
    local schema
    schema=$(echo "$tool_def" | jq '.function.parameters')
    
    # Use jq to validate arguments against schema (basic type checks)
    if echo "$arguments" | jq --argjson schema "$schema" 'def validate($schema):
        if $schema.type == "object" then
            if type != "object" then
                false
            else
                if ($schema.required // []) | map(has(.)) | all then
                    true
                else
                    false
                end
            end
        elif $schema.type == "string" then
            type == "string"
        elif $schema.type == "number" then
            (type == "number") or (type == "string" and test("^[0-9]+$"))
        else
            true
        end;
        validate($schema)' >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

#######################################
# Initialize built-in tool definitions
#######################################
tool_registry::initialize_builtin_tools() {
    mkdir -p "$TOOL_DEFINITIONS_DIR"
    
    local read_file_def='{
  "name": "read_file",
  "description": "Read the contents of a file",
  "category": "file",
  "function": {
    "name": "read_file",
    "parameters": {
      "type": "object",
      "properties": {
        "path": {"type": "string", "description": "Path to the file"}
      },
      "required": ["path"]
    }
  }
}'
    
    echo "$read_file_def" | jq . > "$TOOL_DEFINITIONS_DIR/read_file.json"
    
    local write_file_def='{
  "name": "write_file",
  "description": "Write content to a file",
  "category": "file",
  "function": {
    "name": "write_file",
    "parameters": {
      "type": "object",
      "properties": {
        "path": {"type": "string", "description": "Path to the file"},
        "content": {"type": "string", "description": "Content to write"}
      },
      "required": ["path", "content"]
    }
  }
}'
    
    echo "$write_file_def" | jq . > "$TOOL_DEFINITIONS_DIR/write_file.json"
    
    local execute_command_def='{
  "name": "execute_command",
  "description": "Execute a bash command",
  "category": "command",
  "function": {
    "name": "execute_command",
    "parameters": {
      "type": "object",
      "properties": {
        "command": {"type": "string", "description": "Command to execute"},
        "timeout": {"type": "number", "description": "Timeout in seconds", "default": 30},
        "working_dir": {"type": "string", "description": "Working directory"},
        "environment": {"type": "object", "description": "Environment variables"}
      },
      "required": ["command"]
    }
  }
}'
    
    echo "$execute_command_def" | jq . > "$TOOL_DEFINITIONS_DIR/execute_command.json"
    
    local list_files_def='{
  "name": "list_files",
  "description": "List files in a directory",
  "category": "file",
  "function": {
    "name": "list_files",
    "parameters": {
      "type": "object",
      "properties": {
        "path": {"type": "string", "description": "Directory path"}
      }
    }
  }
}'
    
    echo "$list_files_def" | jq . > "$TOOL_DEFINITIONS_DIR/list_files.json"
    
    local analyze_code_def='{
  "name": "analyze_code",
  "description": "Analyze code for potential issues",
  "category": "analysis",
  "function": {
    "name": "analyze_code",
    "parameters": {
      "type": "object",
      "properties": {
        "code": {"type": "string", "description": "Code to analyze"},
        "language": {"type": "string", "description": "Programming language"},
        "analysis_type": {"type": "string", "description": "Type of analysis"}
      },
      "required": ["code"]
    }
  }
}'
    
    echo "$analyze_code_def" | jq . > "$TOOL_DEFINITIONS_DIR/analyze_code.json"
}

################################################################################
# Utility Functions
################################################################################

#######################################
# Get tool registry status
# Returns:
#   JSON status object
#######################################
tool_registry::get_status() {
    local total_tools available_tools
    total_tools=$(tool_registry::list_tools | jq 'length')
    available_tools=$(tool_registry::list_tools | jq '[.[] | select(.available == true)] | length')
    
    cat << EOF
{
  "total_tools": $total_tools,
  "available_tools": $available_tools,
  "categories": $(tool_registry::get_categories),
  "definitions_dir": "$TOOL_DEFINITIONS_DIR",
  "executors_dir": "$TOOL_EXECUTORS_DIR"
}
