#!/usr/bin/env bash
################################################################################
# Function Calling Capability
# 
# Handles function calling and tool execution workflows
# Coordinates between APIs and tool executors
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

################################################################################
# Function Calling Interface
################################################################################

#######################################
# Execute function calling workflow
# Arguments:
#   $1 - Model configuration (JSON)
#   $2 - User request
#   $3 - Execution context (cli, sandbox, text)
# Returns:
#   Execution result
#######################################
function_calling::execute() {
    local model_config="$1"
    local request="$2"
    local context="$3"
    
    log::debug "Executing function calling capability in $context context"
    
    # Extract model configuration
    local model_name api_endpoint
    model_name=$(echo "$model_config" | jq -r '.model_name // "gpt-5-nano"')
    api_endpoint=$(echo "$model_config" | jq -r '.api_endpoint // "completions"')
    
    # Route based on context - each context handles function calling differently
    case "$context" in
        cli)
            # CLI context handles function calling via external Codex CLI
            function_calling::execute_via_cli "$model_config" "$request"
            ;;
        sandbox)
            # Sandbox context uses our internal function calling implementation
            function_calling::execute_via_sandbox "$model_config" "$request"
            ;;
        text)
            # Text context falls back to text generation
            log::warn "Function calling requested in text-only context, falling back to text generation"
            source "${APP_ROOT}/resources/codex/lib/capabilities/text-generation.sh"
            text_generation::execute "$model_config" "$request"
            ;;
        *)
            log::error "Unknown context for function calling: $context"
            return 1
            ;;
    esac
}

#######################################
# Execute function calling via CLI context
# Arguments:
#   $1 - Model configuration (JSON)
#   $2 - User request
# Returns:
#   CLI execution result
#######################################
function_calling::execute_via_cli() {
    local model_config="$1"
    local request="$2"
    
    # Source CLI context
    source "${APP_ROOT}/resources/codex/lib/execution-contexts/external-cli.sh"
    
    # CLI context handles the execution
    cli_context::execute "function-calling" "$model_config" "$request"
}

#######################################
# Execute function calling via sandbox context
# Arguments:
#   $1 - Model configuration (JSON)
#   $2 - User request
# Returns:
#   Sandbox execution result
#######################################
function_calling::execute_via_sandbox() {
    local model_config="$1"
    local request="$2"
    
    log::info "Executing function calling in sandbox"
    
    # Source required components
    source "${APP_ROOT}/resources/codex/lib/apis/completions.sh" 2>/dev/null || true
    source "${APP_ROOT}/resources/codex/lib/apis/responses.sh" 2>/dev/null || true
    
    # Get model info
    local model_name api_endpoint
    model_name=$(echo "$model_config" | jq -r '.model_name')
    api_endpoint=$(echo "$model_config" | jq -r '.api_endpoint // "completions"')
    
    # Get available tools
    local tools_definitions
    tools_definitions=$(function_calling::get_basic_tools)
    
    # Execute based on API type
    case "$api_endpoint" in
        responses)
            function_calling::execute_responses_loop "$model_name" "$request" "$tools_definitions"
            ;;
        completions)
            function_calling::execute_completions_loop "$model_name" "$request" "$tools_definitions"
            ;;
        *)
            log::error "Unsupported API endpoint for function calling: $api_endpoint"
            return 1
            ;;
    esac
}

################################################################################
# Execution Loops
################################################################################

#######################################
# Execute function calling loop with completions API
# Arguments:
#   $1 - Model name
#   $2 - User request
#   $3 - Tools definitions (JSON)
# Returns:
#   Final result
#######################################
function_calling::execute_completions_loop() {
    local model_name="$1"
    local request="$2"
    local tools="$3"
    
    if type -t completions_api::function_calling_loop &>/dev/null; then
        completions_api::function_calling_loop "$model_name" "$request" "$tools" "function_calling::execute_tool"
    else
        log::error "Completions API function calling not available"
        return 1
    fi
}

#######################################
# Execute function calling loop with responses API
# Arguments:
#   $1 - Model name
#   $2 - User request
#   $3 - Tools definitions (JSON)
# Returns:
#   Final result
#######################################
function_calling::execute_responses_loop() {
    local model_name="$1"
    local request="$2"
    local tools="$3"
    
    if type -t responses_api::function_calling_with_reasoning &>/dev/null; then
        responses_api::function_calling_with_reasoning "$model_name" "$request" "$tools" "function_calling::execute_tool"
    else
        log::error "Responses API function calling not available"
        return 1
    fi
}

################################################################################
# Tool Execution
################################################################################

#######################################
# Execute a function call
# Arguments:
#   $1 - Tool name
#   $2 - Tool arguments (JSON)
# Returns:
#   Tool execution result (JSON)
#######################################
function_calling::execute_tool() {
    local tool_name="$1"
    local arguments="$2"
    
    log::debug "Executing tool: $tool_name"
    
    # Safety workspace
    local workspace="${CODEX_WORKSPACE:-/tmp/codex-workspace}"
    mkdir -p "$workspace"
    
    case "$tool_name" in
        write_file)
            function_calling::tool_write_file "$arguments" "$workspace"
            ;;
        read_file)
            function_calling::tool_read_file "$arguments" "$workspace"
            ;;
        execute_command)
            function_calling::tool_execute_command "$arguments" "$workspace"
            ;;
        list_files)
            function_calling::tool_list_files "$arguments" "$workspace"
            ;;
        *)
            echo "{\"success\": false, \"error\": \"Unknown tool: $tool_name\"}"
            ;;
    esac
}

#######################################
# Tool: Write file
# Arguments:
#   $1 - Arguments (JSON)
#   $2 - Workspace directory
# Returns:
#   Result (JSON)
#######################################
function_calling::tool_write_file() {
    local arguments="$1"
    local workspace="$2"
    
    local path content
    path=$(echo "$arguments" | jq -r '.path')
    content=$(echo "$arguments" | jq -r '.content')
    
    # Ensure path is within workspace
    if [[ ! "$path" =~ ^$workspace/ ]]; then
        path="$workspace/$path"
    fi
    
    # Create directory if needed
    mkdir -p "$(dirname "$path")"
    
    # Write file
    if echo "$content" > "$path"; then
        echo "{\"success\": true, \"message\": \"File written to $path\"}"
    else
        echo "{\"success\": false, \"error\": \"Failed to write file\"}"
    fi
}

#######################################
# Tool: Read file
# Arguments:
#   $1 - Arguments (JSON)
#   $2 - Workspace directory
# Returns:
#   Result (JSON)
#######################################
function_calling::tool_read_file() {
    local arguments="$1"
    local workspace="$2"
    
    local path
    path=$(echo "$arguments" | jq -r '.path')
    
    # Ensure path is within workspace
    if [[ ! "$path" =~ ^$workspace/ ]]; then
        path="$workspace/$path"
    fi
    
    if [[ -f "$path" ]]; then
        local content
        content=$(cat "$path" | jq -Rs .)
        echo "{\"success\": true, \"content\": $content}"
    else
        echo "{\"success\": false, \"error\": \"File not found: $path\"}"
    fi
}

#######################################
# Tool: Execute command
# Arguments:
#   $1 - Arguments (JSON)
#   $2 - Workspace directory
# Returns:
#   Result (JSON)
#######################################
function_calling::tool_execute_command() {
    local arguments="$1"
    local workspace="$2"
    
    local command timeout
    command=$(echo "$arguments" | jq -r '.command')
    timeout=$(echo "$arguments" | jq -r '.timeout // 30')
    
    # Basic safety checks
    if [[ "$command" =~ (rm.*-rf|sudo|reboot|shutdown|format|dd) ]]; then
        echo "{\"success\": false, \"error\": \"Command not allowed for safety\"}"
        return
    fi
    
    # Execute in workspace
    cd "$workspace" || return 1
    
    local output exit_code
    output=$(timeout "$timeout" bash -c "$command" 2>&1)
    exit_code=$?
    
    # Escape for JSON
    output=$(echo "$output" | jq -Rs .)
    
    echo "{\"success\": $([ $exit_code -eq 0 ] && echo true || echo false), \"output\": $output, \"exit_code\": $exit_code}"
}

#######################################
# Tool: List files
# Arguments:
#   $1 - Arguments (JSON)
#   $2 - Workspace directory
# Returns:
#   Result (JSON)
#######################################
function_calling::tool_list_files() {
    local arguments="$1"
    local workspace="$2"
    
    local path
    path=$(echo "$arguments" | jq -r '.path // "."')
    
    # Ensure path is within workspace
    if [[ "$path" == "." ]]; then
        path="$workspace"
    elif [[ ! "$path" =~ ^$workspace/ ]]; then
        path="$workspace/$path"
    fi
    
    if [[ -d "$path" ]]; then
        local files
        files=$(ls -la "$path" 2>/dev/null | jq -Rs .)
        echo "{\"success\": true, \"files\": $files}"
    else
        echo "{\"success\": false, \"error\": \"Directory not found: $path\"}"
    fi
}

################################################################################
# Tool Definitions
################################################################################

#######################################
# Get basic tool definitions
# Returns:
#   JSON array of tool definitions
#######################################
function_calling::get_basic_tools() {
    cat << 'EOF'
[
  {
    "type": "function",
    "function": {
      "name": "write_file",
      "description": "Write content to a file",
      "parameters": {
        "type": "object",
        "properties": {
          "path": {"type": "string", "description": "File path"},
          "content": {"type": "string", "description": "File content"}
        },
        "required": ["path", "content"]
      }
    }
  },
  {
    "type": "function",
    "function": {
      "name": "read_file",
      "description": "Read content from a file",
      "parameters": {
        "type": "object",
        "properties": {
          "path": {"type": "string", "description": "File path"}
        },
        "required": ["path"]
      }
    }
  },
  {
    "type": "function",
    "function": {
      "name": "execute_command",
      "description": "Execute a bash command",
      "parameters": {
        "type": "object",
        "properties": {
          "command": {"type": "string", "description": "Bash command to execute"},
          "timeout": {"type": "number", "description": "Timeout in seconds", "default": 30}
        },
        "required": ["command"]
      }
    }
  },
  {
    "type": "function",
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
  }
]
EOF
}

# Export functions
export -f function_calling::execute
export -f function_calling::execute_tool