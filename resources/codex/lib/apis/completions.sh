#!/usr/bin/env bash
################################################################################
# Completions API - OpenAI Chat Completions Endpoint Handler
# 
# Handles communication with the /chat/completions endpoint
# Supports both text generation and function calling
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/resources/codex/lib/apis/http-client.sh"

################################################################################
# Completions API Interface
################################################################################

#######################################
# Call chat completions API for text generation
# Arguments:
#   $1 - Model name
#   $2 - User prompt OR messages array (JSON)
#   $3 - Options (JSON, optional)
# Returns:
#   API response
#######################################
completions_api::call() {
    local model="$1"
    local input="$2"
    local options="${3:-{}}"
    
    log::debug "Calling completions API: model=$model"
    
    # Determine if input is messages array or simple prompt
    local messages
    if echo "$input" | jq -e 'type == "array"' &>/dev/null 2>&1; then
        # Input is already a messages array
        messages="$input"
    else
        # Input is a simple prompt, wrap it in messages
        messages=$(jq -n --arg content "$input" '[{"role": "user", "content": $content}]')
    fi
    
    # Build request
    local request_body
    request_body=$(http_client::build_chat_request "$model" "$messages" "$options")
    
    # Make API call
    local response
    response=$(http_client::post "/chat/completions" "$request_body")
    
    # Check for errors
    if http_client::has_error "$response"; then
        local error_msg
        error_msg=$(http_client::get_error "$response")
        log::error "Completions API error: $error_msg"
    fi
    
    echo "$response"
}

#######################################
# Call chat completions API with function calling
# Arguments:
#   $1 - Model name
#   $2 - Messages array (JSON)
#   $3 - Tools array (JSON)
#   $4 - Options (JSON, optional)
# Returns:
#   API response
#######################################
completions_api::call_with_tools() {
    local model="$1"
    local messages="$2"
    local tools="$3"
    local options="${4:-{}}"
    
    log::debug "Calling completions API with tools: model=$model, tools_count=$(echo "$tools" | jq 'length')"
    
    # Build request with tools
    local request_body
    request_body=$(http_client::build_chat_request_with_tools "$model" "$messages" "$tools" "$options")
    
    # Make API call
    local response
    response=$(http_client::post "/chat/completions" "$request_body")
    
    # Check for errors
    if http_client::has_error "$response"; then
        local error_msg
        error_msg=$(http_client::get_error "$response")
        log::error "Completions API (with tools) error: $error_msg"
    fi
    
    echo "$response"
}

#######################################
# Simple text generation
# Arguments:
#   $1 - Model name
#   $2 - Prompt text
#   $3 - System message (optional)
# Returns:
#   Generated text
#######################################
completions_api::generate_text() {
    local model="$1"
    local prompt="$2"
    local system_message="${3:-You are an expert programmer. Generate clean, efficient code.}"
    
    # Build messages array
    local messages
    messages=$(jq -n \
        --arg system "$system_message" \
        --arg user "$prompt" \
        '[{"role": "system", "content": $system}, {"role": "user", "content": $user}]')
    
    # Make API call
    local response
    response=$(completions_api::call "$model" "$messages")
    
    # Extract and return content
    if ! http_client::has_error "$response"; then
        http_client::extract_content "$response"
    else
        return 1
    fi
}

################################################################################
# Specialized Functions
################################################################################

#######################################
# Code generation with language-specific context
# Arguments:
#   $1 - Model name
#   $2 - Code request
#   $3 - Programming language (optional)
# Returns:
#   Generated code
#######################################
completions_api::generate_code() {
    local model="$1"
    local request="$2"
    local language="${3:-}"
    
    # Build language-specific system message
    local system_message="You are an expert programmer. Generate clean, efficient, well-commented code."
    if [[ -n "$language" ]]; then
        system_message="$system_message Focus on $language best practices and idioms."
    fi
    
    completions_api::generate_text "$model" "$request" "$system_message"
}

#######################################
# Code review and suggestions
# Arguments:
#   $1 - Model name
#   $2 - Code to review
# Returns:
#   Review and suggestions
#######################################
completions_api::review_code() {
    local model="$1"
    local code="$2"
    
    local system_message="You are an expert code reviewer. Analyze code for readability, performance, security, and best practices. Provide specific, actionable feedback."
    local prompt="Review this code and suggest improvements:\n\n$code"
    
    completions_api::generate_text "$model" "$prompt" "$system_message"
}

#######################################
# Explain code functionality
# Arguments:
#   $1 - Model name
#   $2 - Code to explain
# Returns:
#   Code explanation
#######################################
completions_api::explain_code() {
    local model="$1"
    local code="$2"
    
    local system_message="You are an expert programmer. Explain code clearly and comprehensively, including its purpose, logic, and any important details."
    local prompt="Explain what this code does:\n\n$code"
    
    completions_api::generate_text "$model" "$prompt" "$system_message"
}

#######################################
# Convert code between languages
# Arguments:
#   $1 - Model name
#   $2 - Source code
#   $3 - Target language
# Returns:
#   Converted code
#######################################
completions_api::convert_code() {
    local model="$1"
    local source_code="$2"
    local target_language="$3"
    
    if [[ -z "$target_language" ]]; then
        log::error "Target language required for conversion"
        return 1
    fi
    
    local system_message="You are an expert programmer fluent in multiple languages. Convert code accurately while maintaining functionality and following $target_language best practices."
    local prompt="Convert this code to $target_language:\n\n$source_code"
    
    completions_api::generate_text "$model" "$prompt" "$system_message"
}

################################################################################
# Function Calling Support
################################################################################

#######################################
# Execute function calling conversation
# Arguments:
#   $1 - Model name
#   $2 - Initial user message
#   $3 - Available tools (JSON array)
#   $4 - Tool executor function name
#   $5 - Max iterations (optional, default 5)
# Returns:
#   Final response
#######################################
completions_api::function_calling_loop() {
    local model="$1"
    local user_message="$2"
    local tools="$3"
    local executor_function="$4"
    local max_iterations="${5:-5}"
    
    log::info "Starting function calling loop (max_iterations: $max_iterations)"
    
    # Initialize conversation
    local messages
    messages=$(jq -n --arg content "$user_message" '[{"role": "user", "content": $content}]')
    
    local iteration=0
    while [[ $iteration -lt $max_iterations ]]; do
        ((iteration++))
        log::debug "Function calling iteration $iteration/$max_iterations"
        
        # Make API call with tools
        local response
        response=$(completions_api::call_with_tools "$model" "$messages" "$tools")
        
        # Check for API errors
        if http_client::has_error "$response"; then
            log::error "API error in function calling loop"
            return 1
        fi
        
        # Get assistant message
        local assistant_message
        assistant_message=$(echo "$response" | jq '.choices[0].message')
        
        # Add assistant message to conversation
        messages=$(echo "$messages" | jq ". + [$assistant_message]")
        
        # Check for tool calls
        local tool_calls
        tool_calls=$(http_client::extract_tool_calls "$response")
        
        if [[ $(echo "$tool_calls" | jq 'length') -eq 0 ]]; then
            # No tool calls, return final response
            local final_content
            final_content=$(http_client::extract_content "$response")
            echo "$final_content"
            return 0
        fi
        
        # Execute each tool call
        while IFS= read -r tool_call; do
            [[ -z "$tool_call" ]] && continue
            
            local tool_id tool_name tool_args
            tool_id=$(echo "$tool_call" | jq -r '.id')
            tool_name=$(echo "$tool_call" | jq -r '.function.name')
            tool_args=$(echo "$tool_call" | jq -r '.function.arguments')
            
            log::debug "Executing tool: $tool_name"
            
            # Execute tool via provided executor function
            local tool_result
            if type -t "$executor_function" &>/dev/null; then
                tool_result=$("$executor_function" "$tool_name" "$tool_args")
            else
                tool_result='{"error": "Tool executor not available"}'
            fi
            
            # Add tool result to conversation
            local tool_message
            tool_message=$(jq -n \
                --arg role "tool" \
                --arg content "$tool_result" \
                --arg tool_call_id "$tool_id" \
                '{role: $role, content: $content, tool_call_id: $tool_call_id}')
            
            messages=$(echo "$messages" | jq ". + [$tool_message]")
            
        done < <(echo "$tool_calls" | jq -c '.[]')
    done
    
    log::warn "Function calling loop reached max iterations"
    return 1
}

################################################################################
# Utility Functions
################################################################################

#######################################
# Check model availability
# Arguments:
#   $1 - Model name
# Returns:
#   0 if available, 1 if not
#######################################
completions_api::check_model() {
    local model="$1"
    
    # Try to get model info
    local response
    response=$(http_client::get "/models")
    
    if echo "$response" | jq -e --arg model "$model" '.data[] | select(.id == $model)' &>/dev/null; then
        return 0
    else
        return 1
    fi
}

#######################################
# Get supported models list
# Returns:
#   JSON array of available models
#######################################
completions_api::list_models() {
    local response
    response=$(http_client::get "/models")
    
    if ! http_client::has_error "$response"; then
        echo "$response" | jq '.data'
    else
        echo '[]'
    fi
}

#######################################
# Test API connectivity
# Returns:
#   0 if API is working, 1 if not
#######################################
completions_api::test() {
    log::info "Testing completions API connectivity..."
    
    local response
    response=$(completions_api::call "gpt-3.5-turbo" "Hello, respond with just 'OK'")
    
    if ! http_client::has_error "$response"; then
        local content
        content=$(http_client::extract_content "$response")
        log::success "API test successful: $content"
        return 0
    else
        log::error "API test failed"
        return 1
    fi
}

# Export functions
export -f completions_api::call
export -f completions_api::call_with_tools
export -f completions_api::generate_text
export -f completions_api::generate_code
export -f completions_api::function_calling_loop