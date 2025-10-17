#!/usr/bin/env bash
################################################################################
# Direct System Execution Context
# 
# Makes real tool calls to the actual system - no sandboxing
# This is for getting actual work done, like Claude Code
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/resources/codex/lib/common.sh"

################################################################################
# Direct System Context Interface
################################################################################

#######################################
# Check if direct system context is available
# Returns:
#   0 if available, 1 if not
#######################################
direct_system_context::is_available() {
    # Check if required components exist
    local required_dirs=(
        "${APP_ROOT}/resources/codex/lib/tools"
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
# Get direct system context status
# Returns:
#   JSON status object
#######################################
direct_system_context::status() {
    local available="false"
    local tools_count=0
    local apis_ready="false"
    
    if direct_system_context::is_available; then
        available="true"
        
        # Count available tools
        if [[ -d "${APP_ROOT}/resources/codex/lib/tools/executors" ]]; then
            tools_count=$(find "${APP_ROOT}/resources/codex/lib/tools/executors" -name "*.sh" | wc -l)
        fi
        
        # Check APIs
        if [[ -f "${APP_ROOT}/resources/codex/lib/apis/completions.sh" ]]; then
            apis_ready="true"
        fi
    fi
    
    cat << EOF
{
  "context": "direct-system",
  "tier": 1.5,
  "available": $available,
  "description": "Direct system tool execution - no sandboxing",
  "tools_count": $tools_count,
  "apis_ready": $apis_ready,
  "working_directory": "$(pwd)",
  "system_access": "full"
}
EOF
}

#######################################
# Initialize direct system context
# Returns:
#   0 on success, 1 on failure
#######################################
direct_system_context::init() {
    if [[ "${CODEX_DIRECT_INIT:-}" == "true" ]]; then
        return 0
    fi
    
    # Source required components
    local required_files=(
        "${APP_ROOT}/resources/codex/lib/tools/registry.sh"
        "${APP_ROOT}/resources/codex/lib/apis/completions.sh"
    )
    
    for file in "${required_files[@]}"; do
        if [[ -f "$file" ]]; then
            source "$file"
        else
            log::warn "Required component not found: $file"
        fi
    done
    
    # Initialize tools registry
    if type -t tool_registry::initialize_builtin_tools &>/dev/null; then
        tool_registry::initialize_builtin_tools
    fi
    
    export CODEX_DIRECT_INIT="true"
    return 0
}

################################################################################
# Execution Interface
################################################################################

#######################################
# Execute request through direct system context
# Arguments:
#   $1 - Capability type (function-calling, text-generation)
#   $2 - Model config (JSON)
#   $3 - User request
# Returns:
#   0 on success, 1 on failure
#######################################
direct_system_context::execute() {
    local capability="$1"
    local model_config="$2" 
    local request="$3"
    
    if ! direct_system_context::is_available; then
        log::error "Direct system context not available"
        return 1
    fi
    
    # Initialize if not already done
    direct_system_context::init
    
    log::info "Executing via direct system context..."
    log::debug "Capability: $capability"
    log::debug "Working directory: $(pwd)"
    
    case "$capability" in
        function-calling)
            direct_system_context::execute_with_tools "$model_config" "$request"
            ;;
        text-generation)
            direct_system_context::execute_text_only "$model_config" "$request"
            ;;
        *)
            log::error "Unsupported capability for direct system context: $capability"
            return 1
            ;;
    esac
}

#######################################
# Execute with function calling tools on real system
# Arguments:
#   $1 - Model config (JSON)
#   $2 - User request
# Returns:
#   0 on success, 1 on failure
#######################################
direct_system_context::execute_with_tools() {
    local model_config="$1"
    local request="$2"
    
    log::info "Starting direct system execution with function calling..."
    log::info "Working directory: $(pwd)"
    
    # Initialize tool registry
    if type -t tool_registry::initialize_builtin_tools &>/dev/null; then
        tool_registry::initialize_builtin_tools
    fi
    
    # Get available tools
    local available_tools
    available_tools=$(tool_registry::list_tools | jq '[.[] | select(.available == true)]')
    local tool_count
    tool_count=$(echo "$available_tools" | jq 'length')
    
    log::debug "Available tools: $tool_count"
    
    if [[ $tool_count -eq 0 ]]; then
        log::warn "No tools available, falling back to text-only generation"
        direct_system_context::execute_text_only "$model_config" "$request"
        return $?
    fi
    
    # Start function calling conversation loop
    direct_system_context::function_calling_loop "$model_config" "$request" "$available_tools"
}

#######################################
# Execute text-only generation (no tools)
# Arguments:
#   $1 - Model config (JSON)
#   $2 - User request
# Returns:
#   0 on success, 1 on failure
#######################################
direct_system_context::execute_text_only() {
    local model_config="$1"
    local request="$2"
    
    log::info "Executing via direct system context (text-only)..."
    
    # Extract model info
    local model_name api_endpoint
    model_name=$(echo "$model_config" | jq -r '.model_name // "gpt-4o-mini"')
    api_endpoint=$(echo "$model_config" | jq -r '.api_endpoint // "completions"')
    
    # Prepare enhanced API request for coding
    local system_msg="You are a helpful coding assistant. When writing code, provide complete, working examples with clear explanations. Working directory: $(pwd)"
    local api_request
    api_request=$(jq -n \
        --arg model "$model_name" \
        --arg user_msg "$request" \
        --arg system "$system_msg" \
        '{
            model: $model,
            messages: [
                {
                    role: "system",
                    content: $system
                },
                {
                    role: "user",
                    content: $user_msg
                }
            ],
            temperature: 0.1,
            max_tokens: 2000
        }')
    
    # Get API key
    local api_key
    api_key=$(codex::get_api_key)
    if [[ -z "$api_key" ]]; then
        log::error "No API key available"
        return 1
    fi
    
    # Make API call using http client directly
    local api_response
    if [[ "$api_endpoint" == "completions" ]]; then
        api_response=$(http_client::request "POST" "/chat/completions" "$api_request")
    else
        log::error "Unsupported API endpoint: $api_endpoint"
        return 1
    fi
    
    # Extract and display response
    local content
    content=$(echo "$api_response" | jq -r '.choices[0].message.content // "No response"')
    echo "$content"
    
    return 0
}

################################################################################
# Function Calling Implementation
################################################################################

#######################################
# Execute function calling conversation loop
# Arguments:
#   $1 - Model config (JSON)
#   $2 - User request
#   $3 - Available tools (JSON array)
# Returns:
#   0 on success, 1 on failure
#######################################
direct_system_context::function_calling_loop() {
    local model_config="$1"
    local request="$2"
    local available_tools="$3"
    
    # Extract model info
    local model_name api_key
    model_name=$(echo "$model_config" | jq -r '.model_name // "gpt-4o-mini"')
    api_key=$(codex::get_api_key)
    
    if [[ -z "$api_key" ]]; then
        log::error "No API key available"
        return 1
    fi
    
    # Initialize conversation with system message
    local messages='[]'
    local system_message="You are a helpful AI assistant with access to tools. Use the available tools to complete the user's request. Be thorough and provide clear explanations of what you're doing. Working directory: $(pwd)"
    
    messages=$(echo "$messages" | jq ". + [{\"role\": \"system\", \"content\": $(echo "$system_message" | jq -Rs .)}]")
    messages=$(echo "$messages" | jq ". + [{\"role\": \"user\", \"content\": $(echo "$request" | jq -Rs .)}]")
    
    # Convert tools to OpenAI function format
    local functions
    functions=$(direct_system_context::convert_tools_to_functions "$available_tools")
    
    log::debug "Starting conversation loop with $(echo "$functions" | jq 'length') functions"
    
    local max_iterations=10
    local iteration=0
    
    while [[ $iteration -lt $max_iterations ]]; do
        iteration=$((iteration + 1))
        log::debug "Function calling iteration: $iteration"
        
        # Call model with current conversation and available functions
        local api_response
        api_response=$(direct_system_context::call_with_functions "$api_key" "$model_name" "$messages" "$functions")
        
        if [[ $? -ne 0 ]]; then
            log::error "API call failed in iteration $iteration"
            return 1
        fi
        
        # Extract assistant message
        local assistant_message
        assistant_message=$(echo "$api_response" | jq '.choices[0].message')
        
        if [[ -z "$assistant_message" || "$assistant_message" == "null" ]]; then
            log::error "No assistant message in API response"
            return 1
        fi
        
        # Add assistant message to conversation
        messages=$(echo "$messages" | jq ". + [$assistant_message]")
        
        # Check if assistant wants to call functions
        local function_calls
        function_calls=$(echo "$assistant_message" | jq '.tool_calls // []')
        
        if [[ $(echo "$function_calls" | jq 'length') -eq 0 ]]; then
            # No function calls - conversation is complete
            local content
            content=$(echo "$assistant_message" | jq -r '.content // ""')
            echo "$content"
            return 0
        fi
        
        # Execute function calls and collect results
        local has_function_results=false
        while read -r tool_call; do
            [[ -z "$tool_call" || "$tool_call" == "null" ]] && continue
            
            local call_id function_name function_args
            call_id=$(echo "$tool_call" | jq -r '.id // ""')
            function_name=$(echo "$tool_call" | jq -r '.function.name // ""')
            function_args=$(echo "$tool_call" | jq -r '.function.arguments // "{}"')
            
            if [[ -n "$function_name" ]]; then
                log::info "Executing tool: $function_name"
                
                # Execute the tool in local context (real system)
                local tool_result
                tool_result=$(tool_registry::execute_tool "$function_name" "$function_args" "local")
                
                # Add function result to conversation
                local function_message
                function_message=$(jq -n \
                    --arg call_id "$call_id" \
                    --arg name "$function_name" \
                    --arg content "$tool_result" \
                    '{
                        role: "tool",
                        tool_call_id: $call_id,
                        name: $name,
                        content: $content
                    }')
                
                messages=$(echo "$messages" | jq ". + [$function_message]")
                has_function_results=true
                
                log::debug "Tool $function_name executed, result added to conversation"
            fi
        done < <(echo "$function_calls" | jq -c '.[]')
        
        # If no function results were added, break to avoid infinite loop
        if [[ "$has_function_results" != "true" ]]; then
            local content
            content=$(echo "$assistant_message" | jq -r '.content // "Task completed."')
            echo "$content"
            return 0
        fi
        
        # Continue conversation loop with function results
    done
    
    log::warn "Function calling loop reached maximum iterations ($max_iterations)"
    echo "Task partially completed. Maximum conversation length reached."
    return 0
}

#######################################
# Convert tool registry format to OpenAI functions format
# Arguments:
#   $1 - Tools array (JSON)
# Returns:
#   OpenAI functions format (JSON array)
#######################################
direct_system_context::convert_tools_to_functions() {
    local tools="$1"
    
    echo "$tools" | jq '[.[] | {
        type: "function",
        function: {
            name: .function.name,
            description: .function.description,
            parameters: .function.parameters
        }
    }]'
}

#######################################
# Call OpenAI API with function calling support
# Arguments:
#   $1 - API key
#   $2 - Model name
#   $3 - Messages array (JSON)
#   $4 - Functions array (JSON)
# Returns:
#   API response JSON
#######################################
direct_system_context::call_with_functions() {
    local api_key="$1"
    local model="$2"
    local messages="$3"
    local functions="$4"
    
    # Prepare request with function calling
    local api_request
    api_request=$(jq -n \
        --arg model "$model" \
        --argjson messages "$messages" \
        --argjson tools "$functions" \
        '{
            model: $model,
            messages: $messages,
            tools: $tools,
            tool_choice: "auto",
            temperature: 0.1,
            max_tokens: 4000
        }')
    
    # Make API call using existing http client
    local response
    response=$(http_client::request "POST" "/chat/completions" "$api_request")
    
    if [[ $? -ne 0 ]]; then
        log::error "HTTP request failed for function calling"
        return 1
    fi
    
    # Check for API errors
    if echo "$response" | jq -e '.error' &>/dev/null; then
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error.message // "Unknown error"')
        log::error "API error in function calling: $error_msg"
        return 1
    fi
    
    echo "$response"
}

# Export functions
export -f direct_system_context::is_available
export -f direct_system_context::execute
export -f direct_system_context::status