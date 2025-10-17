#!/usr/bin/env bash
################################################################################
# HTTP Client - Shared HTTP utilities for API communication
# 
# Provides common HTTP functionality for OpenAI API calls including:
# - Authentication
# - Error handling
# - Retry logic
# - Request/response processing
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/resources/codex/lib/common.sh"

################################################################################
# Core HTTP Functions
################################################################################

#######################################
# Make authenticated HTTP request to OpenAI API
# Arguments:
#   $1 - HTTP method (GET, POST, PUT, DELETE)
#   $2 - API endpoint path (e.g., "/chat/completions")
#   $3 - Request body (JSON string, optional)
#   $4 - Additional headers (JSON object, optional)
# Returns:
#   HTTP response body
#######################################
http_client::request() {
    local method="$1"
    local endpoint="$2"
    local body="${3:-}"
    local extra_headers="${4:-{}}"
    
    # Get API key
    local api_key
    api_key=$(codex::get_api_key)
    if [[ -z "$api_key" ]]; then
        log::error "No API key available"
        echo '{"error": {"message": "No API key configured"}}'
        return 1
    fi
    
    # Build full URL
    local base_url="${CODEX_API_ENDPOINT:-https://api.openai.com/v1}"
    local url="${base_url}${endpoint}"
    
    # Build headers
    local headers=(
        "Authorization: Bearer $api_key"
        "Content-Type: application/json"
        "User-Agent: resource-codex/1.0"
    )
    
    # Add extra headers if provided
    if [[ "$extra_headers" != "{}" && -n "$extra_headers" ]]; then
        while IFS= read -r header; do
            [[ -n "$header" ]] && headers+=("$header")
        done < <(echo "$extra_headers" | jq -r 'to_entries[] | "\(.key): \(.value)"' 2>/dev/null || true)
    fi
    
    # Build curl command
    local curl_args=(
        --silent
        --show-error
        --request "$method"
        --max-time "${CODEX_TIMEOUT:-30}"
        --connect-timeout 10
    )
    
    # Add headers
    for header in "${headers[@]}"; do
        curl_args+=(--header "$header")
    done
    
    # Add body for non-GET requests
    if [[ "$method" != "GET" && -n "$body" ]]; then
        curl_args+=(--data "$body")
    fi
    
    # Add URL last
    curl_args+=("$url")
    
    log::debug "Making $method request to $endpoint"
    
    # Make request with retry
    http_client::request_with_retry "${curl_args[@]}"
}

#######################################
# Make HTTP request with exponential backoff retry
# Arguments:
#   All arguments are passed to curl
# Returns:
#   HTTP response body
#######################################
http_client::request_with_retry() {
    local max_retries="${CODEX_RETRY_COUNT:-3}"
    local base_delay="${CODEX_RETRY_DELAY:-2}"
    local attempt=1

    while [[ $attempt -le $max_retries ]]; do
        log::debug "HTTP request attempt $attempt/$max_retries"

        # Make the request and capture HTTP status code
        local temp_file=$(mktemp)
        local response
        local http_code
        local exit_code

        response=$(curl --write-out "\n%{http_code}" "$@" 2>&1 | tee "$temp_file")
        exit_code=$?

        # Extract HTTP code from last line
        http_code=$(tail -n1 "$temp_file" 2>/dev/null || echo "0")
        response=$(head -n-1 "$temp_file" 2>/dev/null || echo "$response")
        rm -f "$temp_file"

        # If successful, return response
        if [[ $exit_code -eq 0 ]]; then
            # Check for rate limit (HTTP 429)
            if [[ "$http_code" == "429" ]]; then
                log::debug "Rate limit detected (HTTP 429)"

                # Detect and record rate limit
                local rate_info
                rate_info=$(codex::detect_rate_limit "$response" "$http_code")
                codex::record_rate_limit "$rate_info"

                # Extract retry time
                local retry_after
                retry_after=$(echo "$rate_info" | jq -r '.retry_after_seconds // 300')

                # Don't retry automatically - let caller handle rate limits
                echo "$response"
                return 1
            fi

            # Check for API-level errors
            if echo "$response" | jq -e '.error' &>/dev/null; then
                local error_type=$(echo "$response" | jq -r '.error.type // "unknown"')
                local error_code=$(echo "$response" | jq -r '.error.code // "unknown"')

                # Check for rate limit error
                if [[ "$error_type" == "rate_limit_exceeded" ]] || [[ "$error_code" == "rate_limit_exceeded" ]]; then
                    log::debug "Rate limit detected in API response"

                    # Detect and record rate limit
                    local rate_info
                    rate_info=$(codex::detect_rate_limit "$response" "429")
                    codex::record_rate_limit "$rate_info"

                    # Don't retry on rate limits - return error to caller
                    echo "$response"
                    return 1
                fi

                # Retry on server errors, not client errors
                if [[ "$error_type" =~ (server_error|timeout) ]]; then
                    log::debug "API error (retryable): $error_type"
                else
                    # Client error, don't retry
                    log::debug "API error (non-retryable): $error_type"
                    echo "$response"
                    return 0
                fi
            else
                # Success
                echo "$response"
                return 0
            fi
        else
            log::debug "HTTP request failed with exit code: $exit_code"
        fi

        # Calculate delay for next attempt
        if [[ $attempt -lt $max_retries ]]; then
            local delay=$((base_delay ** attempt))
            log::debug "Retrying in ${delay} seconds..."
            sleep "$delay"
        fi

        ((attempt++))
    done

    # All retries failed
    log::error "HTTP request failed after $max_retries attempts"
    echo '{"error": {"message": "Request failed after retries", "type": "network_error"}}'
    return 1
}

################################################################################
# Specialized OpenAI API Functions
################################################################################

#######################################
# Make POST request to OpenAI API
# Arguments:
#   $1 - API endpoint path
#   $2 - Request body (JSON)
# Returns:
#   API response
#######################################
http_client::post() {
    local endpoint="$1"
    local body="$2"
    
    http_client::request "POST" "$endpoint" "$body"
}

#######################################
# Make GET request to OpenAI API
# Arguments:
#   $1 - API endpoint path
# Returns:
#   API response
#######################################
http_client::get() {
    local endpoint="$1"
    
    http_client::request "GET" "$endpoint"
}

#######################################
# Validate API connectivity
# Returns:
#   0 if API is accessible, 1 otherwise
#######################################
http_client::validate_api() {
    local response
    response=$(http_client::get "/models")
    
    if echo "$response" | jq -e '.data' &>/dev/null; then
        return 0
    else
        return 1
    fi
}

################################################################################
# Request/Response Processing
################################################################################

#######################################
# Build standard chat completions request
# Arguments:
#   $1 - Model name
#   $2 - Messages array (JSON)
#   $3 - Options object (JSON, optional)
# Returns:
#   Request body JSON
#######################################
http_client::build_chat_request() {
    local model="$1"
    local messages="$2"
    local options="${3:-{}}"
    
    # Default options
    local default_options='{
        "temperature": 0.2,
        "max_tokens": 8192
    }'
    
    # Handle model-specific parameters
    local request_body
    if [[ "$model" == gpt-5* ]]; then
        # GPT-5 models: use max_completion_tokens and temperature must be 1
        request_body=$(jq -n \
            --arg model "$model" \
            --argjson messages "$messages" \
            --argjson options "$options" \
            --argjson defaults "$default_options" \
            '($defaults + $options) as $opts |
            {
                model: $model,
                messages: $messages,
                temperature: 1,
                max_completion_tokens: ($opts.max_tokens // 8192)
            }')
    else
        # Other models: standard parameters
        request_body=$(jq -n \
            --arg model "$model" \
            --argjson messages "$messages" \
            --argjson options "$options" \
            --argjson defaults "$default_options" \
            '($defaults + $options) as $opts |
            {
                model: $model,
                messages: $messages,
                temperature: ($opts.temperature // 0.2),
                max_tokens: ($opts.max_tokens // 8192)
            }')
    fi
    
    echo "$request_body"
}

#######################################
# Build chat request with tools
# Arguments:
#   $1 - Model name
#   $2 - Messages array (JSON)
#   $3 - Tools array (JSON)
#   $4 - Options object (JSON, optional)
# Returns:
#   Request body JSON
#######################################
http_client::build_chat_request_with_tools() {
    local model="$1"
    local messages="$2"
    local tools="$3"
    local options="${4:-{}}"
    
    # Build base request
    local base_request
    base_request=$(http_client::build_chat_request "$model" "$messages" "$options")
    
    # Add tools
    local request_with_tools
    request_with_tools=$(echo "$base_request" | jq --argjson tools "$tools" '. + {tools: $tools, tool_choice: "auto"}')
    
    echo "$request_with_tools"
}

#######################################
# Extract content from chat response
# Arguments:
#   $1 - API response (JSON)
# Returns:
#   Content string
#######################################
http_client::extract_content() {
    local response="$1"
    
    echo "$response" | jq -r '.choices[0].message.content // .message.content // ""'
}

#######################################
# Extract tool calls from chat response
# Arguments:
#   $1 - API response (JSON)
# Returns:
#   Tool calls array (JSON)
#######################################
http_client::extract_tool_calls() {
    local response="$1"
    
    echo "$response" | jq '.choices[0].message.tool_calls // .message.tool_calls // []'
}

#######################################
# Check if response has error
# Arguments:
#   $1 - API response (JSON)
# Returns:
#   0 if has error, 1 if no error
#######################################
http_client::has_error() {
    local response="$1"
    
    echo "$response" | jq -e '.error' &>/dev/null
}

#######################################
# Get error message from response
# Arguments:
#   $1 - API response (JSON)
# Returns:
#   Error message string
#######################################
http_client::get_error() {
    local response="$1"
    
    echo "$response" | jq -r '.error.message // "Unknown error"'
}

################################################################################
# Utility Functions
################################################################################

#######################################
# Get token usage from response
# Arguments:
#   $1 - API response (JSON)
# Returns:
#   Usage object (JSON)
#######################################
http_client::get_usage() {
    local response="$1"
    
    echo "$response" | jq '.usage // {}'
}

#######################################
# Calculate request cost estimate
# Arguments:
#   $1 - Model name
#   $2 - Usage object (JSON)
# Returns:
#   Cost estimate in USD
#######################################
http_client::calculate_cost() {
    local model="$1"
    local usage="$2"
    
    local input_tokens=$(echo "$usage" | jq -r '.prompt_tokens // .input_tokens // 0')
    local output_tokens=$(echo "$usage" | jq -r '.completion_tokens // .output_tokens // 0')
    
    # Get model costs (this could be enhanced to use model registry)
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
        codex-mini-latest)
            input_cost=1.50
            output_cost=6.00
            ;;
        *)
            input_cost=1.00
            output_cost=3.00
            ;;
    esac
    
    # Calculate total cost
    local total_cost
    total_cost=$(echo "scale=6; ($input_tokens / 1000000) * $input_cost + ($output_tokens / 1000000) * $output_cost" | bc -l 2>/dev/null || echo "0.001")
    
    echo "$total_cost"
}

# Export functions
export -f http_client::request
export -f http_client::post
export -f http_client::get
export -f http_client::validate_api
export -f http_client::build_chat_request
export -f http_client::extract_content