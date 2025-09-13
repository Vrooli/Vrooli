#!/usr/bin/env bash
################################################################################
# Text-Only Execution Context
# 
# Provides simple text generation with no code execution
# This is Tier 3 - safe, fast, and cost-effective
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/resources/codex/lib/common.sh"

################################################################################
# Text Context Interface
################################################################################

#######################################
# Check if text context is available
# Returns:
#   0 if available (always true), 1 if not
#######################################
text_context::is_available() {
    # Text generation is always available if we have API access
    codex::is_configured
}

#######################################
# Get text context status
# Returns:
#   JSON status object
#######################################
text_context::status() {
    local available="false"
    local api_configured="false"
    local models_available=0
    
    if codex::is_configured; then
        available="true"
        api_configured="true"
        
        # Count available models (basic estimation)
        models_available=8  # gpt-5 series + gpt-4o series + o1 series
    fi
    
    cat << EOF
{
  "context": "text-only",
  "tier": 3,
  "available": $available,
  "api_configured": $api_configured,
  "models_available": $models_available,
  "capabilities": ["text-generation", "reasoning"]
}
EOF
}

################################################################################
# Execution Interface
################################################################################

#######################################
# Execute request through text context
# Arguments:
#   $1 - Capability type (text-generation, reasoning)
#   $2 - Model config (JSON)
#   $3 - User request
# Returns:
#   0 on success, 1 on failure
#######################################
text_context::execute() {
    local capability="$1"
    local model_config="$2"
    local request="$3"
    
    if ! text_context::is_available; then
        log::error "Text context not available - API not configured"
        return 1
    fi
    
    log::info "Executing via text-only context..."
    log::debug "Capability: $capability"
    
    case "$capability" in
        text-generation)
            text_context::generate_text "$model_config" "$request"
            ;;
        reasoning)
            text_context::reasoning "$model_config" "$request"
            ;;
        *)
            log::error "Unsupported capability for text context: $capability"
            return 1
            ;;
    esac
}

#######################################
# Generate text using specified model
# Arguments:
#   $1 - Model config (JSON)
#   $2 - User request
# Returns:
#   0 on success, 1 on failure
#######################################
text_context::generate_text() {
    local model_config="$1"
    local request="$2"
    
    # Extract model configuration
    local model_name=$(echo "$model_config" | jq -r '.model_name // "gpt-5-nano"' 2>/dev/null || echo "gpt-5-nano")
    local api_endpoint=$(echo "$model_config" | jq -r '.api_endpoint // "completions"' 2>/dev/null || echo "completions")
    local temperature=$(echo "$model_config" | jq -r '.temperature // 0.2' 2>/dev/null || echo "0.2")
    local max_tokens=$(echo "$model_config" | jq -r '.max_tokens // 8192' 2>/dev/null || echo "8192")
    
    log::debug "Text generation: model=$model_name, api=$api_endpoint, temp=$temperature, max=$max_tokens"
    
    # Get API key
    local api_key
    api_key=$(codex::get_api_key)
    if [[ -z "$api_key" ]]; then
        log::error "No API key available"
        return 1
    fi
    
    # Build system message for code generation
    local system_message="You are an expert programmer. Generate clean, efficient, well-commented code."
    
    # Prepare request based on API endpoint
    local response
    if [[ "$api_endpoint" == "responses" ]]; then
        # Use responses API
        response=$(text_context::call_responses_api "$api_key" "$model_name" "$system_message" "$request" "$temperature" "$max_tokens")
    else
        # Use chat completions API
        response=$(text_context::call_completions_api "$api_key" "$model_name" "$system_message" "$request" "$temperature" "$max_tokens")
    fi
    
    if [[ $? -ne 0 ]]; then
        log::error "API call failed"
        return 1
    fi
    
    # Extract generated text
    local generated_text
    if [[ "$api_endpoint" == "responses" ]]; then
        generated_text=$(echo "$response" | jq -r '.message.content // .content // ""' 2>/dev/null)
    else
        generated_text=$(echo "$response" | jq -r '.choices[0].message.content // ""' 2>/dev/null)
    fi
    
    if [[ -z "$generated_text" || "$generated_text" == "null" ]]; then
        log::error "No text generated"
        return 1
    fi
    
    # Output the generated text
    echo "$generated_text"
    return 0
}

#######################################
# Handle reasoning tasks
# Arguments:
#   $1 - Model config (JSON)
#   $2 - User request
# Returns:
#   0 on success, 1 on failure
#######################################
text_context::reasoning() {
    local model_config="$1"
    local request="$2"
    
    # For reasoning tasks, prefer o1 models if available in config
    local model_name=$(echo "$model_config" | jq -r '.model_name // "gpt-5"' 2>/dev/null || echo "gpt-5")
    
    # Override with reasoning model if current model isn't optimized for reasoning
    if [[ ! "$model_name" =~ ^(o1|gpt-5)$ ]]; then
        # Try to use a reasoning-optimized model
        local reasoning_config=$(echo "$model_config" | jq '.model_name = "gpt-5"' 2>/dev/null || echo "$model_config")
        model_config="$reasoning_config"
    fi
    
    # Add reasoning prompt context
    local reasoning_request="Think step by step and show your reasoning process. $request"
    
    text_context::generate_text "$model_config" "$reasoning_request"
}

################################################################################
# API Call Functions
################################################################################

#######################################
# Call OpenAI Chat Completions API
# Arguments:
#   $1 - API key
#   $2 - Model name
#   $3 - System message
#   $4 - User request
#   $5 - Temperature
#   $6 - Max tokens
# Returns:
#   API response JSON
#######################################
text_context::call_completions_api() {
    local api_key="$1"
    local model="$2"
    local system_message="$3"
    local user_request="$4"
    local temperature="$5"
    local max_tokens="$6"
    
    # Handle model-specific parameters
    local request_body
    if [[ "$model" == gpt-5* ]]; then
        # GPT-5 models: use max_completion_tokens and temperature must be 1
        request_body=$(jq -n \
            --arg model "$model" \
            --arg system "$system_message" \
            --arg user "$user_request" \
            --argjson max_tokens "$max_tokens" \
            '{
                model: $model,
                messages: [
                    {role: "system", content: $system},
                    {role: "user", content: $user}
                ],
                temperature: 1,
                max_completion_tokens: $max_tokens
            }')
    else
        # GPT-4 and older models: use max_tokens and custom temperature
        request_body=$(jq -n \
            --arg model "$model" \
            --arg system "$system_message" \
            --arg user "$user_request" \
            --argjson temp "$temperature" \
            --argjson max_tokens "$max_tokens" \
            '{
                model: $model,
                messages: [
                    {role: "system", content: $system},
                    {role: "user", content: $user}
                ],
                temperature: $temp,
                max_tokens: $max_tokens
            }')
    fi
    
    # Make API call
    local response
    response=$(curl -s -X POST "${CODEX_API_ENDPOINT}/chat/completions" \
        -H "Authorization: Bearer $api_key" \
        -H "Content-Type: application/json" \
        -d "$request_body" \
        --max-time "${CODEX_TIMEOUT:-30}")
    
    if [[ $? -ne 0 ]]; then
        log::error "Failed to connect to OpenAI API"
        return 1
    fi
    
    # Check for API errors
    if echo "$response" | jq -e '.error' &>/dev/null; then
        local error_msg=$(echo "$response" | jq -r '.error.message // "Unknown error"')
        log::error "API error: $error_msg"
        return 1
    fi
    
    echo "$response"
}

#######################################
# Call OpenAI Responses API (placeholder)
# Arguments:
#   $1 - API key
#   $2 - Model name
#   $3 - System message
#   $4 - User request
#   $5 - Temperature
#   $6 - Max tokens
# Returns:
#   API response JSON
#######################################
text_context::call_responses_api() {
    local api_key="$1"
    local model="$2"
    local system_message="$3"
    local user_request="$4"
    local temperature="$5"
    local max_tokens="$6"
    
    # TODO: Implement responses API call
    # For now, fall back to completions API
    log::debug "Responses API not yet implemented, falling back to completions"
    text_context::call_completions_api "$@"
}

################################################################################
# Utility Functions
################################################################################

#######################################
# Format output for specific languages
# Arguments:
#   $1 - Generated text
#   $2 - Language hint (optional)
# Returns:
#   Formatted text
#######################################
text_context::format_output() {
    local text="$1"
    local language="${2:-}"
    
    # Basic formatting based on language
    case "$language" in
        python|py)
            # Add Python-specific formatting if needed
            echo "$text"
            ;;
        javascript|js)
            # Add JavaScript-specific formatting if needed
            echo "$text"
            ;;
        *)
            # Default formatting
            echo "$text"
            ;;
    esac
}

#######################################
# Estimate cost for text generation
# Arguments:
#   $1 - Model name
#   $2 - Input tokens (estimate)
#   $3 - Output tokens (estimate)
# Returns:
#   Cost estimate in USD
#######################################
text_context::estimate_cost() {
    local model="$1"
    local input_tokens="$2"
    local output_tokens="$3"
    
    # Cost per 1M tokens (input/output)
    local input_cost output_cost
    case "$model" in
        gpt-5-nano)
            input_cost=0.05
            output_cost=0.40
            ;;
        gpt-5-mini)
            input_cost=0.25
            output_cost=2.00
            ;;
        gpt-5)
            input_cost=1.25
            output_cost=10.00
            ;;
        gpt-4o-mini)
            input_cost=0.15
            output_cost=0.60
            ;;
        gpt-4o)
            input_cost=2.50
            output_cost=10.00
            ;;
        codex-mini-latest)
            input_cost=1.50
            output_cost=6.00
            ;;
        *)
            input_cost=1.00
            output_cost=3.00
            ;;
    esac
    
    # Calculate cost: (tokens / 1,000,000) * cost_per_million
    local total_cost
    total_cost=$(echo "scale=6; ($input_tokens / 1000000) * $input_cost + ($output_tokens / 1000000) * $output_cost" | bc -l 2>/dev/null || echo "0.001")
    
    echo "$total_cost"
}

# Export functions
export -f text_context::is_available
export -f text_context::execute
export -f text_context::generate_text