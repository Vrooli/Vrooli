#!/usr/bin/env bash

# Ollama API Functions
# This file contains API interaction functions and information display

# Agent management is now handled via unified system (see cli.sh)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Source messages if not already loaded
if [[ -z "${MSG_MODEL_NOT_INSTALLED:-}" ]]; then
    OLLAMA_CONFIG_DIR="${APP_ROOT}/resources/ollama/config"
    # shellcheck disable=SC1090,SC1091
    source "${OLLAMA_CONFIG_DIR}/messages.sh" 2>/dev/null || true
fi

#######################################
# Send a prompt to Ollama with model selection and advanced parameters
# Arguments:
#   $1 - prompt text
#   $2 - model name (optional, uses type-based selection if not provided)
#   $3 - model type (general|code|reasoning|vision, defaults to general)
# Uses global variables for additional parameters
#######################################
ollama::send_prompt() {
    local prompt_text="$1"
    local model="$2"
    local model_type="$3"
    
    # Validate Ollama is available
    if ! ollama::is_healthy; then
        log::error "$MSG_OLLAMA_API_UNAVAILABLE"
        log::info "$MSG_START_OLLAMA"
        return 1
    fi
    
    # Validate prompt text
    if [[ -z "$prompt_text" ]]; then
        log::error "$MSG_PROMPT_NO_TEXT"
        log::info "$MSG_PROMPT_USAGE"
        return 1
    fi
    
    # Select model based on priority: specific model > type-based selection
    if [[ -n "$model" ]]; then
        # Validate specified model is available
        if ! ollama::validate_model_available "$model"; then
            log::error "$MSG_MODEL_NOT_INSTALLED"
            log::info "$MSG_AVAILABLE_MODELS"
            ollama::get_installed_models | tr ' ' '\n' | sed 's/^/  • /'
            log::info "$MSG_INSTALL_MODEL"
            return 1
        fi
        if [[ "$OUTPUT_FORMAT" != "json" ]]; then
            log::info "$MSG_MODEL_USING"
        fi
    else
        # Use type-based selection (defaults to general if not specified)
        local use_case="${model_type:-general}"
        if [[ "$OUTPUT_FORMAT" != "json" ]]; then
            log::info "$MSG_MODEL_SELECTING"
        fi
        
        model=$(ollama::get_best_available_model "$use_case")
        if [[ -z "$model" ]]; then
            log::error "$MSG_MODEL_NONE_SUITABLE"
            log::info "$MSG_MODEL_INSTALL_FIRST"
            return 1
        fi
        if [[ "$OUTPUT_FORMAT" != "json" ]]; then
            log::info "$MSG_MODEL_SELECTED"
        fi
    fi
    
    # Build options object
    local options="{}"
    
    # Add temperature if specified
    if [[ -n "$TEMPERATURE" ]]; then
        options=$(echo "$options" | jq --arg temp "$TEMPERATURE" '. + {temperature: ($temp | tonumber)}')
    fi
    
    # Add max tokens if specified
    if [[ -n "$MAX_TOKENS" ]]; then
        options=$(echo "$options" | jq --arg max "$MAX_TOKENS" '. + {num_predict: ($max | tonumber)}')
    fi
    
    # Add top_p if specified
    if [[ -n "$TOP_P" ]]; then
        options=$(echo "$options" | jq --arg p "$TOP_P" '. + {top_p: ($p | tonumber)}')
    fi
    
    # Add top_k if specified
    if [[ -n "$TOP_K" ]]; then
        options=$(echo "$options" | jq --arg k "$TOP_K" '. + {top_k: ($k | tonumber)}')
    fi
    
    # Add seed if specified
    if [[ -n "$SEED" ]]; then
        options=$(echo "$options" | jq --arg seed "$SEED" '. + {seed: ($seed | tonumber)}')
    fi
    
    # Build the complete prompt with system message if provided
    local full_prompt="$prompt_text"
    if [[ -n "$SYSTEM_PROMPT" ]]; then
        full_prompt="${SYSTEM_PROMPT}\n\n${prompt_text}"
    fi
    
    # Build request JSON
    local request_json
    request_json=$(jq -n \
        --arg model "$model" \
        --arg prompt "$full_prompt" \
        --argjson options "$options" \
        '{
            model: $model,
            prompt: $prompt,
            stream: false,
            options: $options
        }')
    
    # Send the prompt
    if [[ "$OUTPUT_FORMAT" != "json" ]]; then
        log::info "$MSG_PROMPT_SENDING"
        echo
        log::header "$MSG_PROMPT_RESPONSE_HEADER"
    fi
    
    local start_time
    start_time=$(date +%s)
    
    # Make the API call
    local response
    if response=$(curl -s -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d "$request_json"); then
        
        local end_time
        end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        # Handle different output formats
        if [[ "$OUTPUT_FORMAT" == "json" ]]; then
            # For JSON format, output the full response
            echo "$response"
            return 0
        else
            # Parse response for text format
            if system::is_command "jq"; then
                local response_text
                response_text=$(echo "$response" | jq -r '.response // empty')
                local error_msg
                error_msg=$(echo "$response" | jq -r '.error // empty')
                
                if [[ -n "$error_msg" && "$error_msg" != "null" ]]; then
                    log::error "$MSG_PROMPT_API_ERROR"
                    return 1
                elif [[ -n "$response_text" && "$response_text" != "null" ]]; then
                    echo "$response_text"
                    echo
                    log::info "$MSG_PROMPT_RESPONSE_TIME"
                    
                    # Show token statistics if available
                    local total_duration prompt_eval_count eval_count
                    total_duration=$(echo "$response" | jq -r '.total_duration // empty' 2>/dev/null)
                    prompt_eval_count=$(echo "$response" | jq -r '.prompt_eval_count // empty' 2>/dev/null)
                    eval_count=$(echo "$response" | jq -r '.eval_count // empty' 2>/dev/null)
                    
                    if [[ -n "$prompt_eval_count" && "$prompt_eval_count" != "null" && -n "$eval_count" && "$eval_count" != "null" ]]; then
                        log::info "$MSG_PROMPT_TOKEN_COUNT"
                    fi
                    
                    # Show generation parameters if they differ from defaults
                    local params_info=""
                    if [[ "$TEMPERATURE" != "0.8" ]]; then
                        params_info="${params_info}temperature=$TEMPERATURE "
                    fi
                    if [[ -n "$MAX_TOKENS" ]]; then
                        params_info="${params_info}max_tokens=$MAX_TOKENS "
                    fi
                    if [[ "$TOP_P" != "0.9" ]]; then
                        params_info="${params_info}top_p=$TOP_P "
                    fi
                    if [[ "$TOP_K" != "40" ]]; then
                        params_info="${params_info}top_k=$TOP_K "
                    fi
                    if [[ -n "$SEED" ]]; then
                        params_info="${params_info}seed=$SEED "
                    fi
                    if [[ -n "$params_info" ]]; then
                        log::info "$MSG_PROMPT_PARAMETERS"
                    fi
                    
                    return 0
                else
                    log::error "$MSG_PROMPT_NO_RESPONSE"
                    return 1
                fi
            else
                # Fallback if jq is not available - just show raw response
                log::warn "$MSG_JQ_UNAVAILABLE"
                echo "$response"
                return 0
            fi
        fi
    else
        log::error "$MSG_FAILED_API_REQUEST"
        log::info "$MSG_CHECK_STATUS"
        return 1
    fi
}

#######################################
# Show comprehensive Ollama information
#######################################
ollama::info() {
    cat << EOF
=== Ollama Resource Information ===

ID: ollama
Category: ai
Display Name: Ollama
Description: Local LLM inference engine

Service Details:
- Binary Location: $OLLAMA_INSTALL_DIR/ollama
- Service Port: $OLLAMA_PORT
- Service URL: $OLLAMA_BASE_URL
- Service Name: $OLLAMA_SERVICE_NAME
- Run User: $OLLAMA_USER

Endpoints:
- Health Check: $OLLAMA_BASE_URL/api/tags
- List Models: $OLLAMA_BASE_URL/api/tags
- Chat: $OLLAMA_BASE_URL/api/chat
- Generate: $OLLAMA_BASE_URL/api/generate
- Pull Model: $OLLAMA_BASE_URL/api/pull
- Show Model: $OLLAMA_BASE_URL/api/show

Configuration:
- Default Models (2025): ${DEFAULT_MODELS[@]}
- Model Catalog: $(echo "${!MODEL_CATALOG[@]}" | wc -w) models available
- Model Storage: ~/.ollama/models
- Total Default Size: $(ollama::calculate_default_size)GB

Ollama Features:
- Local LLM inference
- Multiple model support
- Streaming responses
- OpenAI-compatible API
- GPU acceleration support
- Model quantization
- Custom model creation
- Multi-modal support (vision models)

Example Usage:
# Show available models from catalog
$0 --action available

# List currently installed models
curl $OLLAMA_BASE_URL/api/tags

# Pull new models
ollama pull deepseek-r1:8b
ollama pull qwen2.5-coder:7b

# Run interactive chat with different models
ollama run llama3.1:8b      # General purpose
ollama run deepseek-r1:8b   # Advanced reasoning 
ollama run qwen2.5-coder:7b # Code generation

# Generate text via API
curl -X POST $OLLAMA_BASE_URL/api/generate \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "llama3.1:8b",
    "prompt": "Why is the sky blue?",
    "stream": false
  }'

# Chat completion (OpenAI-compatible)
curl -X POST $OLLAMA_BASE_URL/api/chat \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "llama3.1:8b",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'

For more information, visit: https://ollama.com/library
EOF
}

#######################################
# Get Ollama service URLs with health status
# Outputs JSON with service URLs and health information
#######################################
ollama::get_urls() {
    local health_status="unknown"
    local health_path="/api/tags"
    local primary_url="$OLLAMA_BASE_URL"
    local health_url="${OLLAMA_BASE_URL}${health_path}"
    
    # Check if Ollama is running and healthy
    if ollama::is_healthy >/dev/null 2>&1; then
        health_status="healthy"
    elif ollama::is_installed >/dev/null 2>&1; then
        # Installed but not running
        health_status="unhealthy"
    else
        # Not installed
        health_status="unavailable"
    fi
    
    # Output URL information as JSON
    cat << EOF
{
  "primary": "${primary_url}",
  "health": "${health_url}",
  "name": "Ollama AI",
  "type": "http",
  "status": "${health_status}",
  "port": ${OLLAMA_PORT},
  "endpoints": {
    "api": "${OLLAMA_BASE_URL}/api",
    "generate": "${OLLAMA_BASE_URL}/api/generate",
    "chat": "${OLLAMA_BASE_URL}/api/chat",
    "models": "${OLLAMA_BASE_URL}/api/tags"
  }
}
EOF
}

# Export functions for subshell availability
export -f ollama::send_prompt
export -f ollama::info
export -f ollama::get_urls