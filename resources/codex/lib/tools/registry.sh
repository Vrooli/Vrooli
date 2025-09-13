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
    
    # Load all tool definitions
    local tools_json="[]"
    
    if [[ -d "$TOOL_DEFINITIONS_DIR" ]]; then
        for def_file in "$TOOL_DEFINITIONS_DIR"/*.json; do
            if [[ -f "$def_file" ]]; then
                local tool_def
                tool_def=$(cat "$def_file" 2>/dev/null || echo '{}')
                
                # Apply category filter if specified
                if [[ -n "$category_filter" ]]; then
                    local tool_category
                    tool_category=$(echo "$tool_def" | jq -r '.category // "general"')
                    
                    if [[ "$tool_category" != "$category_filter" ]]; then
                        continue
                    fi
                fi
                
                # Check if executor is available
                local tool_name executor_available
                tool_name=$(echo "$tool_def" | jq -r '.function.name // ""')
                executor_available=$(tool_registry::check_executor "$tool_name")
                
                # Add availability info
                tool_def=$(echo "$tool_def" | jq --argjson available "$executor_available" '. + {available: $available}')
                
                # Add to tools array
                tools_json=$(echo "$tools_json" | jq ". + [$tool_def]")
            fi
        done
    fi
    
    echo "$tools_json"
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
# Validate tool definition structure
# Arguments:
#   $1 - Tool definition (JSON)
# Returns:
#   0 if valid, 1 if invalid
#######################################
tool_registry::validate_definition() {
    local tool_def="$1"
    
    # Required fields check
    local required_fields=(
        ".type"
        ".function.name"
        ".function.description"
        ".function.parameters"
    )
    
    for field in "${required_fields[@]}"; do
        if ! echo "$tool_def" | jq -e "$field" &>/dev/null; then
            log::debug "Tool definition missing required field: $field"
            return 1
        fi
    done
    
    # Validate type is "function"
    local tool_type
    tool_type=$(echo "$tool_def" | jq -r '.type')
    if [[ "$tool_type" != "function" ]]; then
        log::debug "Tool definition has invalid type: $tool_type"
        return 1
    fi
    
    return 0
}

#######################################
# Validate tool arguments against schema
# Arguments:
#   $1 - Tool name
#   $2 - Arguments (JSON)
# Returns:
#   0 if valid, 1 if invalid
#######################################
tool_registry::validate_arguments() {
    local tool_name="$1"
    local arguments="$2"
    
    # Get tool definition
    local tool_def
    tool_def=$(tool_registry::get_tool "$tool_name")
    
    if [[ $(echo "$tool_def" | jq 'keys | length') -eq 0 ]]; then
        return 1
    fi
    
    # Extract required parameters
    local required_params
    required_params=$(echo "$tool_def" | jq -r '.function.parameters.required[]? // empty' 2>/dev/null)
    
    # Check required parameters are present
    while IFS= read -r param; do
        [[ -z "$param" ]] && continue
        
        if ! echo "$arguments" | jq -e --arg param "$param" 'has($param)' &>/dev/null; then
            log::debug "Missing required parameter: $param"
            return 1
        fi
    done <<< "$required_params"
    
    return 0
}

################################################################################
# Tool Categories
################################################################################

#######################################
# Get tools by category
# Arguments:
#   $1 - Category name
# Returns:
#   JSON array of tools in category
#######################################
tool_registry::get_category_tools() {
    local category="$1"
    
    tool_registry::list_tools "$category"
}

#######################################
# Get available categories
# Returns:
#   JSON array of category names
#######################################
tool_registry::get_categories() {
    local categories="[]"
    
    if [[ -d "$TOOL_DEFINITIONS_DIR" ]]; then
        for def_file in "$TOOL_DEFINITIONS_DIR"/*.json; do
            if [[ -f "$def_file" ]]; then
                local category
                category=$(cat "$def_file" 2>/dev/null | jq -r '.category // "general"')
                
                # Add to categories if not already present
                if ! echo "$categories" | jq -e --arg cat "$category" 'index($cat)' &>/dev/null; then
                    categories=$(echo "$categories" | jq ". + [\"$category\"]")
                fi
            fi
        done
    fi
    
    echo "$categories"
}

################################################################################
# Built-in Tools Registration
################################################################################

#######################################
# Initialize built-in tools
# Returns:
#   0 if successful
#######################################
tool_registry::initialize_builtin_tools() {
    log::info "Initializing built-in tools..."
    
    mkdir -p "$TOOL_DEFINITIONS_DIR"
    mkdir -p "$TOOL_EXECUTORS_DIR"
    
    # Register core file system tools
    tool_registry::register_builtin_file_tools
    
    # Register core command execution tools  
    tool_registry::register_builtin_command_tools
    
    # Register core code analysis tools
    tool_registry::register_builtin_code_tools
    
    log::info "Built-in tools initialized"
    return 0
}

#######################################
# Register built-in file system tools
#######################################
tool_registry::register_builtin_file_tools() {
    # write_file tool
    local write_file_def='
{
  "type": "function",
  "category": "file",
  "function": {
    "name": "write_file",
    "description": "Write content to a file",
    "parameters": {
      "type": "object",
      "properties": {
        "path": {"type": "string", "description": "File path to write to"},
        "content": {"type": "string", "description": "Content to write to file"}
      },
      "required": ["path", "content"]
    }
  }
}'
    
    # read_file tool
    local read_file_def='
{
  "type": "function", 
  "category": "file",
  "function": {
    "name": "read_file",
    "description": "Read content from a file",
    "parameters": {
      "type": "object",
      "properties": {
        "path": {"type": "string", "description": "File path to read from"}
      },
      "required": ["path"]
    }
  }
}'
    
    # list_files tool
    local list_files_def='
{
  "type": "function",
  "category": "file", 
  "function": {
    "name": "list_files",
    "description": "List files in a directory",
    "parameters": {
      "type": "object",
      "properties": {
        "path": {"type": "string", "description": "Directory path", "default": "."}
      }
    }
  }
}'
    
    echo "$write_file_def" | jq . > "$TOOL_DEFINITIONS_DIR/write_file.json"
    echo "$read_file_def" | jq . > "$TOOL_DEFINITIONS_DIR/read_file.json"
    echo "$list_files_def" | jq . > "$TOOL_DEFINITIONS_DIR/list_files.json"
}

#######################################
# Register built-in command execution tools
#######################################
tool_registry::register_builtin_command_tools() {
    # execute_command tool
    local execute_command_def='
{
  "type": "function",
  "category": "command",
  "function": {
    "name": "execute_command", 
    "description": "Execute a shell command",
    "parameters": {
      "type": "object",
      "properties": {
        "command": {"type": "string", "description": "Shell command to execute"},
        "timeout": {"type": "number", "description": "Timeout in seconds", "default": 30},
        "working_dir": {"type": "string", "description": "Working directory", "default": "."}
      },
      "required": ["command"]
    }
  }
}'
    
    echo "$execute_command_def" | jq . > "$TOOL_DEFINITIONS_DIR/execute_command.json"
}

#######################################
# Register built-in code analysis tools  
#######################################
tool_registry::register_builtin_code_tools() {
    # analyze_code tool
    local analyze_code_def='
{
  "type": "function",
  "category": "code",
  "function": {
    "name": "analyze_code",
    "description": "Analyze code for patterns, issues, and metrics",
    "parameters": {
      "type": "object", 
      "properties": {
        "code": {"type": "string", "description": "Code to analyze"},
        "language": {"type": "string", "description": "Programming language"},
        "analysis_type": {"type": "string", "description": "Type of analysis", "enum": ["syntax", "style", "complexity", "security"], "default": "syntax"}
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
EOF
}

#######################################
# Test tool registry functionality
# Returns:
#   0 if working correctly
#######################################
tool_registry::test() {
    log::info "Testing tool registry..."
    
    # Initialize built-in tools
    tool_registry::initialize_builtin_tools
    
    # Test listing tools
    local tools
    tools=$(tool_registry::list_tools)
    local tool_count
    tool_count=$(echo "$tools" | jq 'length')
    
    if [[ $tool_count -gt 0 ]]; then
        log::success "Tool registry test passed: $tool_count tools found"
        return 0
    else
        log::error "Tool registry test failed: no tools found"
        return 1
    fi
}

# Export functions
export -f tool_registry::list_tools
export -f tool_registry::get_tool
export -f tool_registry::execute_tool
export -f tool_registry::initialize_builtin_tools