#!/usr/bin/env bash
################################################################################
# Internal Sandbox Execution Context
# 
# Provides function calling and tool execution using our own implementation
# This is Tier 2 - powerful execution with safety controls
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/resources/codex/lib/common.sh"

################################################################################
# Sandbox Context Interface
################################################################################

#######################################
# Check if sandbox context is available
# Returns:
#   0 if available, 1 if not
#######################################
sandbox_context::is_available() {
    # Check if required components exist
    local required_dirs=(
        "${APP_ROOT}/resources/codex/lib/tools"
        "${APP_ROOT}/resources/codex/lib/workspace"
        "${APP_ROOT}/resources/codex/lib/apis"
    )
    
    for dir in "${required_dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            return 1
        fi
    done
    
    # Check if we have API access
    if ! codex::is_configured; then
        return 1
    fi
    
    return 0
}

#######################################
# Get sandbox context status
# Returns:
#   JSON status object
#######################################
sandbox_context::status() {
    local available="false"
    local tools_count=0
    local workspace_ready="false"
    local apis_ready="false"
    
    if sandbox_context::is_available; then
        available="true"
        
        # Count available tools
        if [[ -d "${APP_ROOT}/resources/codex/lib/tools/executors" ]]; then
            tools_count=$(find "${APP_ROOT}/resources/codex/lib/tools/executors" -name "*.sh" | wc -l)
        fi
        
        # Check workspace
        if [[ -f "${APP_ROOT}/resources/codex/lib/workspace/sandbox.sh" ]]; then
            workspace_ready="true"
        fi
        
        # Check APIs
        if [[ -f "${APP_ROOT}/resources/codex/lib/apis/completions.sh" ]]; then
            apis_ready="true"
        fi
    fi
    
    cat << EOF
{
  "context": "internal-sandbox",
  "tier": 2,
  "available": $available,
  "tools_count": $tools_count,
  "workspace_ready": $workspace_ready,
  "apis_ready": $apis_ready,
  "capabilities": ["function-calling", "text-generation"]
}
EOF
}

#######################################
# Initialize sandbox context
# Returns:
#   0 on success, 1 on failure
#######################################
sandbox_context::init() {
    log::debug "Initializing sandbox context"
    
    # Create workspace directory
    local workspace="${CODEX_WORKSPACE:-/tmp/codex-workspace}"
    mkdir -p "$workspace"
    
    # Source required components
    local required_files=(
        "${APP_ROOT}/resources/codex/lib/tools/registry.sh"
        "${APP_ROOT}/resources/codex/lib/workspace/sandbox.sh"
        "${APP_ROOT}/resources/codex/lib/apis/completions.sh"
    )
    
    for file in "${required_files[@]}"; do
        if [[ -f "$file" ]]; then
            source "$file"
        else
            log::warn "Required component not found: $file"
        fi
    done
    
    log::debug "Sandbox context initialized"
    return 0
}

################################################################################
# Execution Interface
################################################################################

#######################################
# Execute request through sandbox context
# Arguments:
#   $1 - Capability type (function-calling, text-generation)
#   $2 - Model config (JSON)
#   $3 - User request
# Returns:
#   0 on success, 1 on failure
#######################################
sandbox_context::execute() {
    local capability="$1"
    local model_config="$2" 
    local request="$3"
    
    if ! sandbox_context::is_available; then
        log::error "Sandbox context not available"
        return 1
    fi
    
    # Initialize if not already done
    sandbox_context::init
    
    log::info "Executing via internal sandbox..."
    log::debug "Capability: $capability"
    
    case "$capability" in
        function-calling)
            sandbox_context::execute_with_tools "$model_config" "$request"
            ;;
        text-generation)
            sandbox_context::execute_text_only "$model_config" "$request"
            ;;
        *)
            log::error "Unsupported capability for sandbox context: $capability"
            return 1
            ;;
    esac
}

#######################################
# Execute with function calling tools
# Arguments:
#   $1 - Model config (JSON)
#   $2 - User request
# Returns:
#   0 on success, 1 on failure
#######################################
sandbox_context::execute_with_tools() {
    local model_config="$1"
    local request="$2"
    local max_iterations=10
    local iteration=0
    
    # Extract model info from config
    local model_name=$(echo "$model_config" | jq -r '.model_name // "gpt-5-nano"' 2>/dev/null || echo "gpt-5-nano")
    local api_endpoint=$(echo "$model_config" | jq -r '.api_endpoint // "completions"' 2>/dev/null || echo "completions")
    
    log::info "Starting function calling execution (model: $model_name, api: $api_endpoint)"
    
    # Get available tools
    local tools_definitions
    if type -t tools_registry::get_definitions &>/dev/null; then
        tools_definitions=$(tools_registry::get_definitions)
    else
        # Fallback minimal tool set
        tools_definitions='[
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
            }
        ]'
    fi
    
    # Initialize conversation
    local messages='[
        {"role": "system", "content": "You are an expert programmer. Use the available tools to complete tasks. All files will be created in the workspace for safety."},
        {"role": "user", "content": "'"$request"'"}
    ]'
    
    # Create workspace
    local workspace="${CODEX_WORKSPACE:-/tmp/codex-workspace}"
    mkdir -p "$workspace"
    
    log::info "Workspace: $workspace"
    log::info "Safety: All operations are sandboxed"
    
    # Execution loop
    while [[ $iteration -lt $max_iterations ]]; do
        ((iteration++))
        log::debug "Iteration $iteration/$max_iterations"
        
        # Make API call with tools
        local response
        if [[ "$api_endpoint" == "responses" ]]; then
            # Use responses API
            if type -t responses_api::call &>/dev/null; then
                response=$(responses_api::call "$model_name" "$messages" "$tools_definitions")
            else
                log::error "Responses API not available"
                return 1
            fi
        else
            # Use completions API
            if type -t completions_api::call_with_tools &>/dev/null; then
                response=$(completions_api::call_with_tools "$model_name" "$messages" "$tools_definitions")
            else
                log::error "Completions API with tools not available"
                return 1
            fi
        fi
        
        if [[ -z "$response" ]]; then
            log::error "No response from API"
            return 1
        fi
        
        # Check for API errors
        if echo "$response" | jq -e '.error' &>/dev/null; then
            local error_msg=$(echo "$response" | jq -r '.error.message // "Unknown error"')
            log::error "API error: $error_msg"
            return 1
        fi
        
        # Get assistant's message
        local assistant_message=$(echo "$response" | jq '.choices[0].message // .message' 2>/dev/null)
        
        if [[ -z "$assistant_message" || "$assistant_message" == "null" ]]; then
            log::error "No assistant message in response"
            return 1
        fi
        
        # Add to conversation
        messages=$(echo "$messages" | jq ". + [$assistant_message]")
        
        # Check for tool calls
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
        
        # Execute tool calls
        local tool_results=()
        while IFS= read -r tool_call; do
            [[ -z "$tool_call" ]] && continue
            
            local tool_id=$(echo "$tool_call" | jq -r '.id')
            local tool_name=$(echo "$tool_call" | jq -r '.function.name')
            local tool_args=$(echo "$tool_call" | jq -r '.function.arguments')
            
            log::info "Executing tool: $tool_name"
            
            # Execute the tool
            local tool_result
            if type -t "sandbox_context::execute_tool" &>/dev/null; then
                tool_result=$(sandbox_context::execute_tool "$tool_name" "$tool_args")
            else
                tool_result='{"success": false, "error": "Tool execution not implemented"}'
            fi
            
            # Show what happened
            echo "  → $tool_name: $(echo "$tool_result" | jq -r '.message // .error // "completed"')"
            
            # Add tool result to conversation
            local tool_message=$(jq -n \
                --arg role "tool" \
                --arg content "$tool_result" \
                --arg tool_call_id "$tool_id" \
                '{role: $role, content: $content, tool_call_id: $tool_call_id}')
            
            messages=$(echo "$messages" | jq ". + [$tool_message]")
            
        done < <(echo "$tool_calls" | jq -c '.[]')
    done
    
    log::error "Max iterations reached without completion"
    return 1
}

#######################################
# Execute text-only generation through sandbox
# Arguments:
#   $1 - Model config (JSON)
#   $2 - User request
# Returns:
#   0 on success, 1 on failure
#######################################
sandbox_context::execute_text_only() {
    local model_config="$1"
    local request="$2"
    
    # Extract model info
    local model_name=$(echo "$model_config" | jq -r '.model_name // "gpt-5-nano"' 2>/dev/null || echo "gpt-5-nano")
    local api_endpoint=$(echo "$model_config" | jq -r '.api_endpoint // "completions"' 2>/dev/null || echo "completions")
    
    log::info "Generating text via sandbox (model: $model_name)"
    
    # Simple text generation call
    if [[ "$api_endpoint" == "responses" ]]; then
        if type -t responses_api::call &>/dev/null; then
            responses_api::call "$model_name" "$request" ""
        else
            log::error "Responses API not available"
            return 1
        fi
    else
        if type -t completions_api::call &>/dev/null; then
            completions_api::call "$model_name" "$request"
        else
            log::error "Completions API not available"
            return 1
        fi
    fi
}

#######################################
# Execute a tool (basic implementation)
# Arguments:
#   $1 - Tool name
#   $2 - Tool arguments (JSON)
# Returns:
#   Tool result as JSON
#######################################
sandbox_context::execute_tool() {
    local tool_name="$1"
    local arguments="$2"
    
    local workspace="${CODEX_WORKSPACE:-/tmp/codex-workspace}"
    
    case "$tool_name" in
        write_file)
            local path=$(echo "$arguments" | jq -r '.path')
            local content=$(echo "$arguments" | jq -r '.content')
            
            # Safety: only allow writing to workspace
            if [[ ! "$path" =~ ^$workspace/ ]]; then
                path="$workspace/$path"
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
            
            # Safety: only allow reading from workspace
            if [[ ! "$path" =~ ^$workspace/ ]]; then
                path="$workspace/$path"
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
            
            # Safety: basic command filtering
            if [[ "$command" =~ (rm.*-rf|sudo|reboot|shutdown|format|dd) ]]; then
                echo "{\"success\": false, \"error\": \"Command not allowed for safety\"}"
                return
            fi
            
            # Execute in workspace
            cd "$workspace" || return 1
            
            local output
            output=$(timeout "$timeout" bash -c "$command" 2>&1)
            local exit_code=$?
            
            output=$(echo "$output" | jq -Rs .)
            
            echo "{\"success\": $([ $exit_code -eq 0 ] && echo true || echo false), \"output\": $output, \"exit_code\": $exit_code}"
            ;;
            
        *)
            echo "{\"success\": false, \"error\": \"Unknown tool: $tool_name\"}"
            ;;
    esac
}

# Initialize on source
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    sandbox_context::init
fi