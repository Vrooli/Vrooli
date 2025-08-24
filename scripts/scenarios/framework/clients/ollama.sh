#!/bin/bash
# Ollama Client - Provides convenient functions for interacting with Ollama

set -euo pipefail

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/scenarios/framework/clients"
if [[ -f "$SCRIPT_DIR/common.sh" ]]; then
    source "$SCRIPT_DIR/common.sh"
fi

# Get Ollama service URL
get_ollama_url() {
    get_resource_url "ollama"
}

# Check if Ollama is available
is_ollama_available() {
    local ollama_url
    ollama_url=$(get_ollama_url)
    
    if [[ -z "$ollama_url" ]]; then
        return 1
    fi
    
    check_url_health "$ollama_url" 10
}

# List available models
ollama_list_models() {
    local ollama_url
    ollama_url=$(get_ollama_url)
    
    if [[ -z "$ollama_url" ]]; then
        print_error "Ollama URL not configured"
        return 1
    fi
    
    print_debug "Listing Ollama models"
    
    local response
    if ! response=$(make_http_request "GET" "$ollama_url/api/tags" "" "" 10 1); then
        print_error "Failed to list Ollama models"
        return 1
    fi
    
    # Extract models from response
    if command -v jq >/dev/null 2>&1; then
        echo "$response" | head -n -1 | jq -r '.models[]?.name // empty' 2>/dev/null || echo ""
    else
        # Fallback parsing
        echo "$response" | head -n -1 | grep -o '"name":"[^"]*"' | sed 's/"name":"\([^"]*\)"/\1/' || echo ""
    fi
}

# Check if a specific model is available
ollama_has_model() {
    local model_name="$1"
    
    local models
    models=$(ollama_list_models)
    
    if [[ -z "$models" ]]; then
        return 1
    fi
    
    echo "$models" | grep -q "^${model_name}$"
}

# Pull a model if not available
ollama_ensure_model() {
    local model_name="$1"
    local timeout="${2:-300}"  # 5 minutes default for model pulling
    
    print_info "Ensuring Ollama model is available: $model_name"
    
    # Check if model already exists
    if ollama_has_model "$model_name"; then
        print_success "Model already available: $model_name"
        return 0
    fi
    
    print_info "Pulling model: $model_name (timeout: ${timeout}s)"
    
    local ollama_url
    ollama_url=$(get_ollama_url)
    
    if [[ -z "$ollama_url" ]]; then
        print_error "Ollama URL not configured"
        return 1
    fi
    
    # Prepare pull request
    local pull_payload
    pull_payload=$(cat << EOF
{
    "name": "$model_name"
}
EOF
)
    
    # Make pull request
    local response
    if ! response=$(make_http_request "POST" "$ollama_url/api/pull" "Content-Type: application/json" "$pull_payload" "$timeout" 1); then
        print_error "Failed to pull model: $model_name"
        return 1
    fi
    
    # Verify model was pulled
    if ollama_has_model "$model_name"; then
        print_success "Model pulled successfully: $model_name"
        return 0
    else
        print_error "Model pull completed but model not found: $model_name"
        return 1
    fi
}

# Generate text with Ollama
ollama_generate() {
    local model="$1"
    local prompt="$2"
    local options="${3:-}"
    local stream="${4:-false}"
    
    print_debug "Generating with Ollama: $model"
    
    local ollama_url
    ollama_url=$(get_ollama_url)
    
    if [[ -z "$ollama_url" ]]; then
        print_error "Ollama URL not configured"
        return 1
    fi
    
    # Ensure model is available
    if ! ollama_ensure_model "$model" 60; then
        return 1
    fi
    
    # Prepare generation request
    local generate_payload
    generate_payload=$(cat << EOF
{
    "model": "$model",
    "prompt": "$prompt",
    "stream": $stream
EOF
)
    
    # Add options if provided
    if [[ -n "$options" ]]; then
        generate_payload="$generate_payload,\"options\":$options"
    fi
    
    generate_payload="$generate_payload}"
    
    # Make generation request
    local response
    if ! response=$(make_http_request "POST" "$ollama_url/api/generate" "Content-Type: application/json" "$generate_payload" 120 1); then
        print_error "Failed to generate text with Ollama"
        return 1
    fi
    
    # Extract response text
    local response_text=""
    if command -v jq >/dev/null 2>&1; then
        response_text=$(echo "$response" | head -n -1 | jq -r '.response // empty')
    else
        # Fallback parsing
        response_text=$(echo "$response" | head -n -1 | sed -n 's/.*"response":"\([^"]*\)".*/\1/p')
    fi
    
    if [[ -z "$response_text" ]]; then
        print_error "No response from Ollama generation"
        return 1
    fi
    
    echo "$response_text"
    return 0
}

# Generate text and validate response
ollama_generate_and_validate() {
    local model="$1"
    local prompt="$2"
    local expected_contains="${3:-}"
    local min_length="${4:-10}"
    
    print_info "Generating and validating text with Ollama"
    
    local response
    if ! response=$(ollama_generate "$model" "$prompt" "" false); then
        return 1
    fi
    
    # Validate response length
    if [[ ${#response} -lt $min_length ]]; then
        print_error "Response too short: ${#response} chars (min: $min_length)"
        return 1
    fi
    
    # Validate response contains expected text
    if [[ -n "$expected_contains" ]] && [[ "$response" != *"$expected_contains"* ]]; then
        print_error "Response does not contain expected text: '$expected_contains'"
        return 1
    fi
    
    print_success "Generated valid response (${#response} chars)"
    echo "$response"
    return 0
}

# Test Ollama with a simple prompt
test_ollama_basic() {
    local model="${1:-llama2}"
    local test_prompt="Hello, how are you?"
    
    print_info "Testing Ollama basic functionality"
    
    # Check if Ollama is available
    if ! is_ollama_available; then
        print_error "Ollama service not available"
        return 1
    fi
    
    # Test generation
    local response
    if response=$(ollama_generate_and_validate "$model" "$test_prompt" "" 5); then
        print_success "Ollama basic test passed"
        return 0
    else
        print_error "Ollama basic test failed"
        return 1
    fi
}

# Test Ollama performance
test_ollama_performance() {
    local model="${1:-llama2}"
    local max_response_time="${2:-30}"
    
    print_info "Testing Ollama performance"
    
    local start_time
    start_time=$(date +%s.%N)
    
    local response
    response=$(ollama_generate "$model" "Generate a short paragraph about artificial intelligence." "" false)
    
    local end_time
    end_time=$(date +%s.%N)
    local duration
    if duration=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "0"); then
        print_info "Response time: ${duration}s"
        
        if (( $(echo "$duration <= $max_response_time" | bc -l 2>/dev/null || echo "0") )); then
            print_success "Ollama performance test passed"
            return 0
        else
            print_warning "Ollama response time exceeded ${max_response_time}s"
            return 0  # Warning, not failure
        fi
    else
        print_error "Ollama performance test failed"
        return 1
    fi
}

# Get Ollama system information
ollama_system_info() {
    local ollama_url
    ollama_url=$(get_ollama_url)
    
    if [[ -z "$ollama_url" ]]; then
        print_error "Ollama URL not configured"
        return 1
    fi
    
    print_info "Getting Ollama system information"
    
    # Get version info (if available)
    local version_response
    version_response=$(make_http_request "GET" "$ollama_url/api/version" "" "" 10 1 2>/dev/null || echo "")
    
    if [[ -n "$version_response" ]]; then
        local version
        version=$(json_extract "$version_response" ".version")
        if [[ -n "$version" ]]; then
            print_info "Ollama version: $version"
        fi
    fi
    
    # List available models
    local models
    models=$(ollama_list_models)
    
    if [[ -n "$models" ]]; then
        print_info "Available models:"
        echo "$models" | while read -r model; do
            if [[ -n "$model" ]]; then
                echo "  - $model"
            fi
        done
    else
        print_warning "No models available"
    fi
    
    return 0
}

# Export functions
export -f get_ollama_url
export -f is_ollama_available
export -f ollama_list_models
export -f ollama_has_model
export -f ollama_ensure_model
export -f ollama_generate
export -f ollama_generate_and_validate
export -f test_ollama_basic
export -f test_ollama_performance
export -f ollama_system_info