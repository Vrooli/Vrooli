#!/usr/bin/env bash
################################################################################
# Responses API - OpenAI Responses Endpoint Handler
# 
# Handles communication with the /responses endpoint (2025)
# Optimized for codex-mini-latest and other specialized models
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/resources/codex/lib/apis/http-client.sh"

################################################################################
# Responses API Interface
################################################################################

#######################################
# Call responses API for text generation
# Arguments:
#   $1 - Model name
#   $2 - User prompt OR messages array (JSON)
#   $3 - Options (JSON, optional)
# Returns:
#   API response
#######################################
responses_api::call() {
    local model="$1"
    local input="$2"
    local options="${3:-{}}"
    
    log::debug "Calling responses API: model=$model"
    
    # Determine if input is messages array or simple prompt
    local messages
    if echo "$input" | jq -e 'type == "array"' &>/dev/null 2>&1; then
        # Input is already a messages array
        messages="$input"
    else
        # Input is a simple prompt, wrap it in messages
        messages=$(jq -n --arg content "$input" '[{"role": "user", "content": $content}]')
    fi
    
    # Build request for responses API
    local request_body
    request_body=$(responses_api::build_request "$model" "$messages" "$options")
    
    # Make API call
    local response
    response=$(http_client::post "/responses" "$request_body")
    
    # Check for errors
    if http_client::has_error "$response"; then
        local error_msg
        error_msg=$(http_client::get_error "$response")
        log::error "Responses API error: $error_msg"
    fi
    
    echo "$response"
}

#######################################
# Call responses API with function calling
# Arguments:
#   $1 - Model name
#   $2 - Messages array (JSON)
#   $3 - Tools array (JSON)
#   $4 - Options (JSON, optional)
# Returns:
#   API response
#######################################
responses_api::call_with_tools() {
    local model="$1"
    local messages="$2"
    local tools="$3"
    local options="${4:-{}}"
    
    log::debug "Calling responses API with tools: model=$model, tools_count=$(echo "$tools" | jq 'length')"
    
    # Build request with tools for responses API
    local request_body
    request_body=$(responses_api::build_request_with_tools "$model" "$messages" "$tools" "$options")
    
    # Make API call
    local response
    response=$(http_client::post "/responses" "$request_body")
    
    # Check for errors
    if http_client::has_error "$response"; then
        local error_msg
        error_msg=$(http_client::get_error "$response")
        log::error "Responses API (with tools) error: $error_msg"
    fi
    
    echo "$response"
}

################################################################################
# Request Building Functions
################################################################################

#######################################
# Build responses API request
# Arguments:
#   $1 - Model name
#   $2 - Messages array (JSON)
#   $3 - Options object (JSON, optional)
# Returns:
#   Request body JSON
#######################################
responses_api::build_request() {
    local model="$1"
    local messages="$2"
    local options="${3:-{}}"
    
    # Default options for responses API
    local default_options='{
        "temperature": 1.0,
        "max_completion_tokens": 8192
    }'
    
    # Responses API has different structure than chat completions
    local request_body
    request_body=$(jq -n \
        --arg model "$model" \
        --argjson messages "$messages" \
        --argjson options "$options" \
        --argjson defaults "$default_options" \
        '($defaults + $options) as $opts |
        {
            model: $model,
            messages: $messages,
            temperature: ($opts.temperature // 1.0),
            max_completion_tokens: ($opts.max_completion_tokens // 8192)
        }')
    
    # Add reasoning controls if model supports them
    if [[ "$model" == "codex-mini-latest" ]]; then
        request_body=$(echo "$request_body" | jq '. + {
            "reasoning_effort": "medium",
            "reasoning_summary": true
        }')
    fi
    
    echo "$request_body"
}

#######################################
# Build responses API request with tools
# Arguments:
#   $1 - Model name
#   $2 - Messages array (JSON)
#   $3 - Tools array (JSON)
#   $4 - Options object (JSON, optional)
# Returns:
#   Request body JSON
#######################################
responses_api::build_request_with_tools() {
    local model="$1"
    local messages="$2"
    local tools="$3"
    local options="${4:-{}}"
    
    # Build base request
    local base_request
    base_request=$(responses_api::build_request "$model" "$messages" "$options")
    
    # Add tools - responses API might have different tool format
    local request_with_tools
    request_with_tools=$(echo "$base_request" | jq --argjson tools "$tools" '. + {
        tools: $tools,
        tool_choice: "auto"
    }')
    
    echo "$request_with_tools"
}

################################################################################
# Response Processing
################################################################################

#######################################
# Extract content from responses API response
# Arguments:
#   $1 - API response (JSON)
# Returns:
#   Content string
#######################################
responses_api::extract_content() {
    local response="$1"
    
    # Responses API may have different response structure
    echo "$response" | jq -r '.message.content // .content // .choices[0].message.content // ""'
}

#######################################
# Extract reasoning from responses API response
# Arguments:
#   $1 - API response (JSON)
# Returns:
#   Reasoning content (if available)
#######################################
responses_api::extract_reasoning() {
    local response="$1"
    
    echo "$response" | jq -r '.reasoning // .reasoning_summary // ""'
}

#######################################
# Extract tool calls from responses API response
# Arguments:
#   $1 - API response (JSON)
# Returns:
#   Tool calls array (JSON)
#######################################
responses_api::extract_tool_calls() {
    local response="$1"
    
    echo "$response" | jq '.message.tool_calls // .tool_calls // .choices[0].message.tool_calls // []'
}

################################################################################
# Specialized Functions for Responses API
################################################################################

#######################################
# Generate code with reasoning
# Arguments:
#   $1 - Model name
#   $2 - Code request
#   $3 - Show reasoning (true/false, optional)
# Returns:
#   Generated code with optional reasoning
#######################################
responses_api::generate_code_with_reasoning() {
    local model="$1"
    local request="$2"
    local show_reasoning="${3:-false}"
    
    local system_message="You are an expert programmer. Generate clean, efficient, well-commented code. Think through the problem step by step."
    
    # Build messages
    local messages
    messages=$(jq -n \
        --arg system "$system_message" \
        --arg user "$request" \
        '[{"role": "system", "content": $system}, {"role": "user", "content": $user}]')
    
    # Set options to include reasoning
    local options='{"reasoning_summary": true}'
    
    # Make API call
    local response
    response=$(responses_api::call "$model" "$messages" "$options")
    
    if ! http_client::has_error "$response"; then
        local content
        content=$(responses_api::extract_content "$response")
        
        if [[ "$show_reasoning" == "true" ]]; then
            local reasoning
            reasoning=$(responses_api::extract_reasoning "$response")
            
            if [[ -n "$reasoning" ]]; then
                echo "## Reasoning:"
                echo "$reasoning"
                echo ""
                echo "## Generated Code:"
            fi
        fi
        
        echo "$content"
    else
        return 1
    fi
}

#######################################
# Execute function calling with reasoning preservation
# Arguments:
#   $1 - Model name
#   $2 - Initial user message
#   $3 - Available tools (JSON array)
#   $4 - Tool executor function name
#   $5 - Max iterations (optional, default 5)
# Returns:
#   Final response with reasoning chain
#######################################
responses_api::function_calling_with_reasoning() {
    local model="$1"
    local user_message="$2"
    local tools="$3"
    local executor_function="$4"
    local max_iterations="${5:-5}"
    
    log::info "Starting responses API function calling with reasoning preservation"
    
    # Initialize conversation
    local messages
    messages=$(jq -n --arg content "$user_message" '[{"role": "user", "content": $content}]')
    
    # Track reasoning chain
    local reasoning_chain=()
    
    local env_max_turns="${CODEX_MAX_TURNS:-}"
    if [[ -n "$env_max_turns" && "$env_max_turns" =~ ^[0-9]+$ && "$env_max_turns" -gt 0 ]]; then
        max_iterations="$env_max_turns"
    fi

    local max_duration="${CODEX_TIMEOUT:-}"
    local start_time=$(date +%s)

    local iteration=0
    while [[ $iteration -lt $max_iterations ]]; do
        if [[ -n "$max_duration" && "$max_duration" =~ ^[0-9]+$ ]]; then
            local now=$(date +%s)
            if (( now - start_time >= max_duration )); then
                log::error "Responses API function calling exceeded timeout (${max_duration}s)"
                return 1
            fi
        fi
        ((iteration++))
        log::debug "Responses API iteration $iteration/$max_iterations"
        
        # Make API call with tools and reasoning
        local options='{"reasoning_summary": true, "reasoning_effort": "medium"}'
        local response
        response=$(responses_api::call_with_tools "$model" "$messages" "$tools" "$options")
        
        # Check for API errors
        if http_client::has_error "$response"; then
            log::error "API error in responses function calling loop"
            return 1
        fi
        
        # Extract and store reasoning
        local reasoning
        reasoning=$(responses_api::extract_reasoning "$response")
        if [[ -n "$reasoning" ]]; then
            reasoning_chain+=("Step $iteration: $reasoning")
        fi
        
        # Get assistant message
        local assistant_message
        assistant_message=$(echo "$response" | jq '.message // .choices[0].message')
        
        # Add assistant message to conversation
        messages=$(echo "$messages" | jq ". + [$assistant_message]")
        
        # Check for tool calls
        local tool_calls
        tool_calls=$(responses_api::extract_tool_calls "$response")
        
        if [[ $(echo "$tool_calls" | jq 'length') -eq 0 ]]; then
            # No tool calls, return final response with reasoning
            local final_content
            final_content=$(responses_api::extract_content "$response")
            
            # Include reasoning chain if available
            if [[ ${#reasoning_chain[@]} -gt 0 ]]; then
                echo "## Reasoning Chain:"
                for reason in "${reasoning_chain[@]}"; do
                    echo "- $reason"
                done
                echo ""
                echo "## Final Result:"
            fi
            
            echo "$final_content"
            return 0
        fi
        
        # Execute tool calls (similar to completions API)
        while IFS= read -r tool_call; do
            [[ -z "$tool_call" ]] && continue
            
            local tool_id tool_name tool_args
            tool_id=$(echo "$tool_call" | jq -r '.id')
            tool_name=$(echo "$tool_call" | jq -r '.function.name')
            tool_args=$(echo "$tool_call" | jq -r '.function.arguments')
            
            log::debug "Executing tool: $tool_name"
            
            # Execute tool
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
    
    log::warn "Responses API function calling reached max iterations"
    return 1
}

################################################################################
# Utility Functions
################################################################################

#######################################
# Check if model is available for responses API
# Arguments:
#   $1 - Model name
# Returns:
#   0 if available, 1 if not
#######################################
responses_api::check_model() {
    local model="$1"
    
    # Known models that work with responses API
    local supported_models=(
        "codex-mini-latest"
        "o1-mini"
        "o1-preview"
    )
    
    for supported in "${supported_models[@]}"; do
        if [[ "$model" == "$supported" ]]; then
            return 0
        fi
    done
    
    return 1
}

#######################################
# Test responses API connectivity
# Returns:
#   0 if API is working, 1 if not
#######################################
responses_api::test() {
    log::info "Testing responses API connectivity..."
    
    # Use a known working model
    local response
    response=$(responses_api::call "codex-mini-latest" "Hello, respond with just 'OK'")
    
    if ! http_client::has_error "$response"; then
        local content
        content=$(responses_api::extract_content "$response")
        log::success "Responses API test successful: $content"
        return 0
    else
        log::error "Responses API test failed"
        return 1
    fi
}

#######################################
# Get responses API specific capabilities
# Arguments:
#   $1 - Model name
# Returns:
#   JSON object with capabilities
#######################################
responses_api::get_capabilities() {
    local model="$1"
    
    case "$model" in
        codex-mini-latest)
            cat << 'EOF'
{
  "supports_reasoning": true,
  "supports_tools": true,
  "reasoning_effort_levels": ["minimal", "low", "medium", "high"],
  "max_context": 200000,
  "max_output": 100000,
  "optimized_for": "coding"
}
EOF
            ;;
        o1-mini|o1-preview)
            cat << 'EOF'
{
  "supports_reasoning": true,
  "supports_tools": false,
  "reasoning_effort_levels": ["medium", "high"],
  "max_context": 128000,
  "max_output": 65536,
  "optimized_for": "reasoning"
}
EOF
            ;;
        *)
            echo '{}'
            ;;
    esac
}

# Export functions
export -f responses_api::call
export -f responses_api::call_with_tools
export -f responses_api::extract_content
export -f responses_api::extract_reasoning
export -f responses_api::generate_code_with_reasoning
