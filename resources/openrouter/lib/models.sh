#!/bin/bash
# OpenRouter model selection and fallback library

# Define directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENROUTER_LIB_DIR="${APP_ROOT}/resources/openrouter/lib"
OPENROUTER_RESOURCE_DIR="${APP_ROOT}/resources/openrouter"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${OPENROUTER_LIB_DIR}/core.sh"
source "${OPENROUTER_RESOURCE_DIR}/config/defaults.sh"

# Model selection strategies
openrouter::models::select_auto() {
    local task_type="${1:-general}"
    local budget="${2:-0.10}"
    
    # Auto-select based on task type
    case "$task_type" in
        code|programming)
            echo "anthropic/claude-3-sonnet"
            ;;
        creative|writing)
            echo "anthropic/claude-3-opus"
            ;;
        simple|quick)
            echo "openai/gpt-3.5-turbo"
            ;;
        analysis|reasoning)
            echo "openai/gpt-4-turbo"
            ;;
        budget|cheap)
            echo "mistralai/mistral-7b"
            ;;
        *)
            echo "${OPENROUTER_DEFAULT_MODEL:-openai/gpt-3.5-turbo}"
            ;;
    esac
}

# Select cheapest model that meets requirements
openrouter::models::select_cheapest() {
    local min_capability="${1:-basic}"
    
    # Ordered from cheapest to most expensive
    local cheap_models=(
        "mistralai/mistral-7b"
        "openai/gpt-3.5-turbo"
        "anthropic/claude-3-haiku"
        "openai/gpt-4-turbo"
        "anthropic/claude-3-sonnet"
        "anthropic/claude-3-opus"
    )
    
    # For now, return the cheapest that's available
    # In a full implementation, we'd check actual pricing via API
    echo "${cheap_models[0]}"
}

# Select fastest model
openrouter::models::select_fastest() {
    # Models ordered by typical response speed
    local fast_models=(
        "openai/gpt-3.5-turbo"
        "mistralai/mistral-7b"
        "anthropic/claude-3-haiku"
    )
    
    echo "${fast_models[0]}"
}

# Select highest quality model
openrouter::models::select_quality() {
    # Models ordered by quality/capability
    local quality_models=(
        "anthropic/claude-3-opus"
        "openai/gpt-4-turbo"
        "anthropic/claude-3-sonnet"
    )
    
    echo "${quality_models[0]}"
}

# Execute request with fallback chain
openrouter::models::execute_with_fallback() {
    local primary_model="${1}"
    local prompt="${2}"
    local max_tokens="${3:-1000}"
    shift 3
    local fallback_models=("$@")
    
    # Add default fallbacks if none provided
    if [[ ${#fallback_models[@]} -eq 0 ]]; then
        fallback_models=(
            "openai/gpt-3.5-turbo"
            "mistralai/mistral-7b"
        )
    fi
    
    # Initialize API key if needed
    if [[ -z "${OPENROUTER_API_KEY:-}" ]]; then
        if ! openrouter::init; then
            log::error "Failed to initialize OpenRouter API key"
            return 1
        fi
    fi
    
    # Try primary model first
    local models_to_try=("$primary_model" "${fallback_models[@]}")
    local timeout="${OPENROUTER_TIMEOUT:-30}"
    
    for model in "${models_to_try[@]}"; do
        log::info "Trying model: $model"
        
        local response
        response=$(timeout "$timeout" curl -s -X POST \
            -H "Authorization: Bearer ${OPENROUTER_API_KEY}" \
            -H "Content-Type: application/json" \
            -d "$(jq -n \
                --arg model "$model" \
                --arg content "$prompt" \
                --argjson max_tokens "$max_tokens" \
                '{
                    model: $model,
                    messages: [{role: "user", content: $content}],
                    max_tokens: $max_tokens
                }')" \
            "${OPENROUTER_API_BASE}/chat/completions" 2>/dev/null)
        
        # Check if successful
        if [[ -n "$response" ]] && ! echo "$response" | jq -e '.error' >/dev/null 2>&1; then
            # Success - return the response with model info
            echo "$response" | jq --arg model "$model" '. + {actual_model: $model}'
            return 0
        fi
        
        # Log the error and try next model
        local error_msg=$(echo "$response" | jq -r '.error.message // .error // "Unknown error"' 2>/dev/null || echo "Request failed")
        log::warn "Model $model failed: $error_msg"
    done
    
    log::error "All models failed. Tried: ${models_to_try[*]}"
    return 1
}

# Get model pricing information
openrouter::models::get_pricing() {
    local model="${1}"
    local timeout="${OPENROUTER_TIMEOUT:-30}"
    
    if [[ -z "${OPENROUTER_API_KEY:-}" ]]; then
        if ! openrouter::init; then
            return 1
        fi
    fi
    
    # Fetch model details including pricing
    local response
    response=$(timeout "$timeout" curl -s \
        -H "Authorization: Bearer ${OPENROUTER_API_KEY}" \
        "${OPENROUTER_API_BASE}/models" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        return 1
    fi
    
    # Extract pricing for specific model
    echo "$response" | jq -r --arg model "$model" '
        .data[] | 
        select(.id == $model) | 
        {
            id: .id,
            name: .name,
            pricing: .pricing,
            context_length: .context_length,
            per_request_limits: .per_request_limits
        }'
}

# List models by category
openrouter::models::list_by_category() {
    local category="${1:-all}"
    local timeout="${OPENROUTER_TIMEOUT:-30}"
    
    if [[ -z "${OPENROUTER_API_KEY:-}" ]]; then
        if ! openrouter::init; then
            return 1
        fi
    fi
    
    local response
    response=$(timeout "$timeout" curl -s \
        -H "Authorization: Bearer ${OPENROUTER_API_KEY}" \
        "${OPENROUTER_API_BASE}/models" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        return 1
    fi
    
    case "$category" in
        code|coding|programming)
            echo "$response" | jq -r '.data[] | select(.id | contains("code") or contains("codestral") or contains("deepseek")) | .id'
            ;;
        chat|conversation)
            echo "$response" | jq -r '.data[] | select(.id | contains("chat") or contains("turbo") or contains("claude")) | .id'
            ;;
        vision|image)
            echo "$response" | jq -r '.data[] | select(.id | contains("vision") or contains("gpt-4o")) | .id'
            ;;
        cheap|budget)
            echo "$response" | jq -r '.data[] | select(.pricing.prompt < 0.0001) | .id'
            ;;
        fast|speed)
            echo "$response" | jq -r '.data[] | select(.id | contains("turbo") or contains("haiku") or contains("mistral-7b")) | .id'
            ;;
        all|*)
            echo "$response" | jq -r '.data[].id'
            ;;
    esac
}

# Test model with sample prompt
openrouter::models::test() {
    local model="${1}"
    local test_prompt="${2:-Hello, please respond with 'OK' if you receive this message.}"
    
    log::info "Testing model: $model"
    
    local response
    response=$(openrouter::models::execute_with_fallback "$model" "$test_prompt" 10)
    
    if [[ $? -eq 0 ]]; then
        local content=$(echo "$response" | jq -r '.choices[0].message.content // "No response"')
        log::success "Model test successful: $content"
        return 0
    else
        log::error "Model test failed"
        return 1
    fi
}