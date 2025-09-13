#!/usr/bin/env bash
################################################################################
# Codex Tools Example - Enhanced execution with GPT-5 function calling
# 
# THIS IS A CONCEPT DEMONSTRATION - NOT PRODUCTION READY
# Shows how resource-codex could be enhanced to execute code like Claude Code
################################################################################

# Source dependencies
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/resources/codex/config/defaults.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/resources/codex/lib/common.sh"

################################################################################
# Tool Definitions for GPT-5
################################################################################

# Define available tools that GPT-5 can request
CODEX_TOOLS='[
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
]'

################################################################################
# Tool Execution Functions
################################################################################

#######################################
# Execute a tool call from GPT-5
# Arguments:
#   $1 - Tool name
#   $2 - Tool arguments (JSON)
# Returns:
#   Tool result as JSON
#######################################
codex::execute_tool() {
    local tool_name="$1"
    local arguments="$2"
    
    case "$tool_name" in
        write_file)
            local path=$(echo "$arguments" | jq -r '.path')
            local content=$(echo "$arguments" | jq -r '.content')
            
            # Safety check - only allow writing to /tmp/codex/ for demo
            if [[ ! "$path" =~ ^/tmp/codex/ ]]; then
                path="/tmp/codex/$path"
            fi
            
            mkdir -p "$(dirname "$path")"
            echo "$content" > "$path"
            
            if [[ $? -eq 0 ]]; then
                echo "{\"success\": true, \"message\": \"File written to $path\"}"
            else
                echo "{\"success\": false, \"error\": \"Failed to write file\"}"
            fi
            ;;
            
        read_file)
            local path=$(echo "$arguments" | jq -r '.path')
            
            # Safety check
            if [[ ! "$path" =~ ^/tmp/codex/ ]] && [[ ! -f "$path" ]]; then
                path="/tmp/codex/$path"
            fi
            
            if [[ -f "$path" ]]; then
                local content=$(cat "$path" | jq -Rs .)
                echo "{\"success\": true, \"content\": $content}"
            else
                echo "{\"success\": false, \"error\": \"File not found\"}"
            fi
            ;;
            
        execute_command)
            local command=$(echo "$arguments" | jq -r '.command')
            local timeout=$(echo "$arguments" | jq -r '.timeout // 30')
            
            # Safety check - only allow safe commands for demo
            if [[ "$command" =~ (rm|sudo|reboot|shutdown|mv|dd) ]]; then
                echo "{\"success\": false, \"error\": \"Command not allowed for safety\"}"
                return
            fi
            
            # Execute with timeout
            local output
            output=$(timeout "$timeout" bash -c "$command" 2>&1)
            local exit_code=$?
            
            # Escape output for JSON
            output=$(echo "$output" | jq -Rs .)
            
            echo "{\"success\": $([ $exit_code -eq 0 ] && echo true || echo false), \"output\": $output, \"exit_code\": $exit_code}"
            ;;
            
        list_files)
            local path=$(echo "$arguments" | jq -r '.path // "."')
            
            # Safety check
            if [[ ! "$path" =~ ^/tmp/codex/ ]] && [[ "$path" != "." ]]; then
                path="/tmp/codex/$path"
            fi
            
            if [[ -d "$path" ]]; then
                local files=$(ls -la "$path" | tail -n +2 | jq -Rs .)
                echo "{\"success\": true, \"files\": $files}"
            else
                echo "{\"success\": false, \"error\": \"Directory not found\"}"
            fi
            ;;
            
        *)
            echo "{\"success\": false, \"error\": \"Unknown tool: $tool_name\"}"
            ;;
    esac
}

################################################################################
# Main Execution Loop with Tools
################################################################################

#######################################
# Execute a prompt with tool calling
# Arguments:
#   $1 - User prompt
# Returns:
#   0 on success, 1 on failure
#######################################
codex::execute_with_tools() {
    local prompt="$1"
    local max_iterations=10
    local iteration=0
    
    if ! codex::is_configured; then
        log::error "Codex not configured. Please set OPENAI_API_KEY"
        return 1
    fi
    
    local api_key=$(codex::get_api_key)
    local model="${CODEX_DEFAULT_MODEL:-gpt-5-nano}"
    
    log::info "Starting execution with tools (model: $model)"
    echo "Safety: All files will be created in /tmp/codex/"
    echo "----------------------------------------"
    
    # Initialize conversation
    local messages='[
        {"role": "system", "content": "You are an expert programmer. Use the available tools to complete tasks. All files will be created in /tmp/codex/ for safety."},
        {"role": "user", "content": "'"$prompt"'"}
    ]'
    
    while [[ $iteration -lt $max_iterations ]]; do
        ((iteration++))
        log::info "Iteration $iteration/$max_iterations"
        
        # Build request with tools
        local request_body=$(jq -n \
            --arg model "$model" \
            --argjson messages "$messages" \
            --argjson tools "$CODEX_TOOLS" \
            '{
                model: $model,
                messages: $messages,
                tools: $tools,
                tool_choice: "auto",
                temperature: 1,
                max_completion_tokens: 8192
            }')
        
        # Make API call
        local response
        response=$(curl -s -X POST "${CODEX_API_ENDPOINT}/chat/completions" \
            -H "Authorization: Bearer $api_key" \
            -H "Content-Type: application/json" \
            -d "$request_body" \
            --max-time 30)
        
        # Check for errors
        if echo "$response" | jq -e '.error' &>/dev/null; then
            local error_msg=$(echo "$response" | jq -r '.error.message // "Unknown error"')
            log::error "API error: $error_msg"
            return 1
        fi
        
        # Get the assistant's message
        local assistant_message=$(echo "$response" | jq '.choices[0].message')
        
        # Add assistant's message to conversation
        messages=$(echo "$messages" | jq ". + [$assistant_message]")
        
        # Check if there are tool calls
        local tool_calls=$(echo "$assistant_message" | jq -r '.tool_calls // []')
        
        if [[ $(echo "$tool_calls" | jq 'length') -eq 0 ]]; then
            # No tool calls, we have the final response
            local final_content=$(echo "$assistant_message" | jq -r '.content // ""')
            echo ""
            log::success "Task completed!"
            echo "----------------------------------------"
            echo "$final_content"
            return 0
        fi
        
        # Execute each tool call
        echo "$tool_calls" | jq -c '.[]' | while read -r tool_call; do
            local tool_id=$(echo "$tool_call" | jq -r '.id')
            local tool_name=$(echo "$tool_call" | jq -r '.function.name')
            local tool_args=$(echo "$tool_call" | jq -r '.function.arguments')
            
            log::info "Executing tool: $tool_name"
            
            # Execute the tool
            local tool_result=$(codex::execute_tool "$tool_name" "$tool_args")
            
            # Show what happened
            echo "  → $tool_name: $(echo "$tool_result" | jq -r '.message // .error // "completed"')"
            
            # Add tool result to conversation
            local tool_message=$(jq -n \
                --arg role "tool" \
                --arg content "$tool_result" \
                --arg tool_call_id "$tool_id" \
                '{role: $role, content: $content, tool_call_id: $tool_call_id}')
            
            messages=$(echo "$messages" | jq ". + [$tool_message]")
        done
    done
    
    log::error "Max iterations reached without completion"
    return 1
}

################################################################################
# Example Usage Functions
################################################################################

#######################################
# Run example demonstrations
#######################################
codex::tools_demo() {
    echo "=== Codex Tools Demonstration ==="
    echo ""
    
    # Create demo directory
    mkdir -p /tmp/codex
    
    # Example 1: Simple file creation
    echo "Example 1: Create a Python hello world"
    echo "----------------------------------------"
    codex::execute_with_tools "Create a Python file called hello.py that prints 'Hello, World!' and then run it"
    echo ""
    
    # Example 2: Multi-step task
    echo "Example 2: Create and test a function"
    echo "----------------------------------------"
    codex::execute_with_tools "Create a Python file with a fibonacci function, then create a test file that tests it with n=10, and run the test"
    echo ""
    
    # Show created files
    echo "Files created during demo:"
    echo "----------------------------------------"
    ls -la /tmp/codex/
}

################################################################################
# Comparison Functions
################################################################################

#######################################
# Show difference between modes
#######################################
codex::compare_modes() {
    local prompt="Write a function to calculate factorial"
    
    echo "=== Comparison: Text Generation vs Tool Execution ==="
    echo ""
    echo "Prompt: $prompt"
    echo ""
    
    echo "1. CURRENT MODE (Text Generation Only):"
    echo "----------------------------------------"
    # This would call the existing codex::generate_code
    echo "def factorial(n):"
    echo "    if n <= 1:"
    echo "        return 1"
    echo "    return n * factorial(n - 1)"
    echo ""
    echo "→ Returns text only, no files created"
    echo ""
    
    echo "2. ENHANCED MODE (With Tool Execution):"
    echo "----------------------------------------"
    codex::execute_with_tools "$prompt and save it to factorial.py, then test it with n=5"
    echo ""
    echo "→ Creates actual files and runs tests"
    echo ""
    
    echo "Files created:"
    ls -la /tmp/codex/*.py 2>/dev/null || echo "None"
}

# Main entry point for testing
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-demo}" in
        demo)
            codex::tools_demo
            ;;
        compare)
            codex::compare_modes
            ;;
        execute)
            shift
            codex::execute_with_tools "$*"
            ;;
        *)
            echo "Usage: $0 {demo|compare|execute <prompt>}"
            echo ""
            echo "  demo     - Run demonstration examples"
            echo "  compare  - Compare text generation vs tool execution"
            echo "  execute  - Execute a prompt with tools"
            echo ""
            echo "Example:"
            echo "  $0 execute 'Create a Python script that sorts a list and test it'"
            exit 1
            ;;
    esac
fi