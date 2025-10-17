#!/bin/bash
# Gemini token counting and usage tracking functionality

# Define directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
GEMINI_RESOURCE_DIR="${APP_ROOT}/resources/gemini"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${GEMINI_RESOURCE_DIR}/config/defaults.sh"

# Token counting configuration
GEMINI_TOKEN_TRACKING_ENABLED="${GEMINI_TOKEN_TRACKING_ENABLED:-true}"
GEMINI_TOKEN_LOG_FILE="${GEMINI_TOKEN_LOG_FILE:-${GEMINI_RESOURCE_DIR}/logs/token_usage.log}"

# Model token limits (as of 2025)
declare -A GEMINI_MODEL_LIMITS=(
    ["gemini-pro"]="32768"
    ["gemini-pro-vision"]="16384"
    ["gemini-1.5-pro"]="2097152"  # 2M context
    ["gemini-1.5-flash"]="1048576" # 1M context
)

# Approximate token counting (rough estimate)
# Real token counting requires the API, this is for local estimation
gemini::tokens::estimate_count() {
    local text="$1"
    
    # Rough approximation: ~1.3 characters per token for English text
    # This is a simplified heuristic
    local char_count=${#text}
    local estimated_tokens=$((char_count * 100 / 130))  # Avoid floating point
    
    echo "$estimated_tokens"
}

# Check if prompt exceeds model limit
gemini::tokens::check_limit() {
    local prompt="$1"
    local model="${2:-$GEMINI_DEFAULT_MODEL}"
    
    local limit="${GEMINI_MODEL_LIMITS[$model]:-32768}"
    local estimated=$(gemini::tokens::estimate_count "$prompt")
    
    if [[ $estimated -gt $limit ]]; then
        log::warn "Prompt may exceed model limit (estimated: $estimated, limit: $limit)"
        return 1
    fi
    
    return 0
}

# Log token usage
gemini::tokens::log_usage() {
    local prompt="$1"
    local response="$2"
    local model="${3:-$GEMINI_DEFAULT_MODEL}"
    local timestamp=$(date -Iseconds)
    
    if [[ "$GEMINI_TOKEN_TRACKING_ENABLED" != "true" ]]; then
        return 0
    fi
    
    local prompt_tokens=$(gemini::tokens::estimate_count "$prompt")
    local response_tokens=$(gemini::tokens::estimate_count "$response")
    local total_tokens=$((prompt_tokens + response_tokens))
    
    # Ensure log directory exists
    local log_dir=$(dirname "$GEMINI_TOKEN_LOG_FILE")
    mkdir -p "$log_dir" 2>/dev/null || true
    
    # Log in JSON format for easy parsing
    cat >> "$GEMINI_TOKEN_LOG_FILE" <<EOF
{"timestamp":"$timestamp","model":"$model","prompt_tokens":$prompt_tokens,"response_tokens":$response_tokens,"total_tokens":$total_tokens}
EOF
    
    return 0
}

# Get token usage statistics
gemini::tokens::get_stats() {
    if [[ ! -f "$GEMINI_TOKEN_LOG_FILE" ]]; then
        echo "No token usage data available"
        return 0
    fi
    
    # Calculate totals using jq if available
    if command -v jq >/dev/null 2>&1; then
        local total_tokens=$(jq -s 'map(.total_tokens) | add' "$GEMINI_TOKEN_LOG_FILE" 2>/dev/null || echo "0")
        local total_requests=$(wc -l < "$GEMINI_TOKEN_LOG_FILE")
        local avg_tokens=0
        
        if [[ $total_requests -gt 0 ]]; then
            avg_tokens=$((total_tokens / total_requests))
        fi
        
        echo "Token Usage Statistics:"
        echo "----------------------"
        echo "Total Requests: $total_requests"
        echo "Total Tokens: $total_tokens"
        echo "Average Tokens/Request: $avg_tokens"
        
        # Model breakdown
        echo ""
        echo "By Model:"
        jq -r 'group_by(.model) | map({model: .[0].model, count: length, total: map(.total_tokens) | add}) | .[] | "\(.model): \(.count) requests, \(.total) tokens"' "$GEMINI_TOKEN_LOG_FILE" 2>/dev/null || true
    else
        # Fallback without jq
        local line_count=$(wc -l < "$GEMINI_TOKEN_LOG_FILE")
        echo "Token Usage Statistics:"
        echo "----------------------"
        echo "Total Requests: $line_count"
        echo "(Install jq for detailed statistics)"
    fi
    
    return 0
}

# Clear token usage history
gemini::tokens::clear_history() {
    if [[ -f "$GEMINI_TOKEN_LOG_FILE" ]]; then
        > "$GEMINI_TOKEN_LOG_FILE"
        log::info "Token usage history cleared"
    else
        log::info "No token usage history to clear"
    fi
    
    return 0
}

# Estimate cost based on token usage (using rough pricing estimates)
gemini::tokens::estimate_cost() {
    if [[ ! -f "$GEMINI_TOKEN_LOG_FILE" ]]; then
        echo "No token usage data available for cost estimation"
        return 0
    fi
    
    # Rough pricing estimates (subject to change, check Google's pricing)
    # Prices per 1M tokens (as of 2025)
    local pro_input_cost="0.50"   # $0.50 per 1M input tokens
    local pro_output_cost="1.50"  # $1.50 per 1M output tokens
    
    if command -v jq >/dev/null 2>&1; then
        local total_prompt=$(jq -s 'map(.prompt_tokens) | add' "$GEMINI_TOKEN_LOG_FILE" 2>/dev/null || echo "0")
        local total_response=$(jq -s 'map(.response_tokens) | add' "$GEMINI_TOKEN_LOG_FILE" 2>/dev/null || echo "0")
        
        # Calculate costs (using bc for floating point if available)
        if command -v bc >/dev/null 2>&1; then
            local input_cost=$(echo "scale=4; $total_prompt * $pro_input_cost / 1000000" | bc)
            local output_cost=$(echo "scale=4; $total_response * $pro_output_cost / 1000000" | bc)
            local total_cost=$(echo "scale=4; $input_cost + $output_cost" | bc)
            
            echo "Estimated Cost (USD):"
            echo "--------------------"
            echo "Input tokens: $total_prompt (~\$$input_cost)"
            echo "Output tokens: $total_response (~\$$output_cost)"
            echo "Total estimated cost: ~\$$total_cost"
            echo ""
            echo "Note: These are rough estimates. Check Google's current pricing."
        else
            echo "Total tokens used: $((total_prompt + total_response))"
            echo "(Install bc for cost calculations)"
        fi
    else
        echo "Install jq for cost estimation"
    fi
    
    return 0
}

# Export functions
export -f gemini::tokens::estimate_count
export -f gemini::tokens::check_limit
export -f gemini::tokens::log_usage
export -f gemini::tokens::get_stats
export -f gemini::tokens::clear_history
export -f gemini::tokens::estimate_cost