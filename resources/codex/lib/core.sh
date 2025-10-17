#!/usr/bin/env bash
################################################################################
# Codex Core Functions - Modern GPT-based Code Generation
# Uses GPT-3.5-turbo/GPT-4 for code generation (Codex API deprecated March 2023)
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CODEX_LIB_DIR="${APP_ROOT}/resources/codex/lib"

# Source dependencies
source "${APP_ROOT}/resources/codex/config/defaults.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${CODEX_LIB_DIR}/common.sh"

################################################################################
# Core API Functions
################################################################################

#######################################
# Generate code using OpenAI GPT models
# Arguments:
#   $1 - Prompt for code generation
#   $2 - Optional: Language (python, javascript, go, etc.)
#   $3 - Optional: Output file path
# Returns:
#   0 on success, 1 on failure
# Outputs:
#   Generated code to stdout or file
#######################################
codex::generate_code() {
    local prompt="$1"
    local language="${2:-}"
    local output_file="${3:-}"
    
    if ! codex::is_configured; then
        log::error "Codex not configured. Please set OPENAI_API_KEY"
        return 1
    fi
    
    local api_key=$(codex::get_api_key)
    local model="${CODEX_DEFAULT_MODEL:-gpt-3.5-turbo}"
    
    # Build the system message for code generation
    local system_message="You are an expert programmer. Generate clean, efficient, well-commented code."
    if [[ -n "$language" ]]; then
        system_message="$system_message Generate $language code."
    fi
    
    # Prepare the API request
    # Note: GPT-5 models have different parameters
    local request_body
    if [[ "$model" == gpt-5* ]]; then
        # GPT-5 models: use max_completion_tokens and temperature must be 1
        request_body=$(jq -n \
            --arg model "$model" \
            --arg system "$system_message" \
            --arg user "$prompt" \
            --argjson max_tokens "${CODEX_DEFAULT_MAX_TOKENS:-8192}" \
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
            --arg user "$prompt" \
            --argjson temp "${CODEX_DEFAULT_TEMPERATURE:-0.2}" \
            --argjson max_tokens "${CODEX_DEFAULT_MAX_TOKENS:-4096}" \
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
    
    # Make the API request
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
    
    # Extract the generated code
    local generated_code=$(echo "$response" | jq -r '.choices[0].message.content // ""')
    
    if [[ -z "$generated_code" ]]; then
        log::error "No code generated"
        return 1
    fi
    
    # Output the code
    if [[ -n "$output_file" ]]; then
        echo "$generated_code" > "$output_file"
        log::success "Code generated and saved to: $output_file"
    else
        echo "$generated_code"
    fi
    
    return 0
}

#######################################
# Complete partial code using GPT
# Arguments:
#   $1 - Code context (existing code)
#   $2 - Completion request
#   $3 - Optional: Output file
# Returns:
#   0 on success, 1 on failure
#######################################
codex::complete_code() {
    local context="$1"
    local request="$2"
    local output_file="${3:-}"
    
    local prompt="Given this code context:\n\n$context\n\nComplete the following: $request"
    
    codex::generate_code "$prompt" "" "$output_file"
}

#######################################
# Review code and suggest improvements
# Arguments:
#   $1 - Code to review
#   $2 - Optional: Output file
# Returns:
#   0 on success, 1 on failure
#######################################
codex::review_code() {
    local code="$1"
    local output_file="${2:-}"
    
    local prompt="Review this code and suggest improvements for readability, performance, and best practices:\n\n$code"
    
    codex::generate_code "$prompt" "" "$output_file"
}

#######################################
# Generate unit tests for code
# Arguments:
#   $1 - Code to test
#   $2 - Optional: Test framework (pytest, jest, etc.)
#   $3 - Optional: Output file
# Returns:
#   0 on success, 1 on failure
#######################################
codex::generate_tests() {
    local code="$1"
    local framework="${2:-}"
    local output_file="${3:-}"
    
    local prompt="Generate comprehensive unit tests for this code"
    if [[ -n "$framework" ]]; then
        prompt="$prompt using $framework"
    fi
    prompt="$prompt:\n\n$code"
    
    codex::generate_code "$prompt" "" "$output_file"
}

#######################################
# Translate code between languages
# Arguments:
#   $1 - Source code
#   $2 - Target language
#   $3 - Optional: Output file
# Returns:
#   0 on success, 1 on failure
#######################################
codex::translate_code() {
    local source_code="$1"
    local target_language="$2"
    local output_file="${3:-}"
    
    if [[ -z "$target_language" ]]; then
        log::error "Target language required for translation"
        return 1
    fi
    
    local prompt="Translate this code to $target_language, maintaining the same functionality:\n\n$source_code"
    
    codex::generate_code "$prompt" "$target_language" "$output_file"
}

#######################################
# Explain code in natural language
# Arguments:
#   $1 - Code to explain
#   $2 - Optional: Output file
# Returns:
#   0 on success, 1 on failure
#######################################
codex::explain_code() {
    local code="$1"
    local output_file="${2:-}"
    
    local prompt="Explain what this code does in clear, simple terms:\n\n$code"
    
    codex::generate_code "$prompt" "" "$output_file"
}

#######################################
# Get token usage from last request
# Returns:
#   Token usage information
#######################################
codex::get_token_usage() {
    # This would need to track the last response
    # For now, return placeholder
    echo "Token tracking not yet implemented"
}

#######################################
# Validate API connectivity
# Returns:
#   0 if API is accessible, 1 otherwise
#######################################
codex::validate_api() {
    if ! codex::is_configured; then
        return 1
    fi
    
    local api_key=$(codex::get_api_key)
    
    # Test the models endpoint
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $api_key" \
        "${CODEX_API_ENDPOINT}/models" \
        --max-time 5)
    
    if [[ "$response" == "200" ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Process a script file with Codex
# Arguments:
#   $1 - Script file path
#   $2 - Operation (generate, complete, review, test, explain)
#   $3 - Optional: Additional parameters
# Returns:
#   0 on success, 1 on failure
#######################################
codex::process_script() {
    local script_file="$1"
    local operation="${2:-generate}"
    local params="${3:-}"
    
    if [[ ! -f "$script_file" ]]; then
        log::error "Script file not found: $script_file"
        return 1
    fi
    
    local content=$(<"$script_file")
    local output_file="${CODEX_OUTPUT_DIR}/$(basename "$script_file").out"
    
    case "$operation" in
        generate)
            codex::generate_code "$content" "$params" "$output_file"
            ;;
        complete)
            codex::complete_code "$content" "$params" "$output_file"
            ;;
        review)
            codex::review_code "$content" "$output_file"
            ;;
        test)
            codex::generate_tests "$content" "$params" "$output_file"
            ;;
        explain)
            codex::explain_code "$content" "$output_file"
            ;;
        *)
            log::error "Unknown operation: $operation"
            return 1
            ;;
    esac
}