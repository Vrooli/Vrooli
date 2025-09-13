#!/bin/bash
# OpenRouter model benchmarking library

# Define directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENROUTER_LIB_DIR="${APP_ROOT}/resources/openrouter/lib"
OPENROUTER_RESOURCE_DIR="${APP_ROOT}/resources/openrouter"
OPENROUTER_DATA_DIR="${VROOLI_ROOT:-$HOME/Vrooli}/data/openrouter"
BENCHMARK_DIR="${OPENROUTER_DATA_DIR}/benchmarks"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${OPENROUTER_LIB_DIR}/core.sh"
source "${OPENROUTER_LIB_DIR}/models.sh"
source "${OPENROUTER_RESOURCE_DIR}/config/defaults.sh"

# Initialize benchmark directory
openrouter::benchmark::init() {
    mkdir -p "${BENCHMARK_DIR}"
}

# Run a simple benchmark test on a model
openrouter::benchmark::test_model() {
    local model="${1}"
    local test_prompt="${2:-Generate a haiku about computers.}"
    
    if [[ -z "$model" ]]; then
        log::error "Model name required"
        return 1
    fi
    
    # Check for placeholder API key
    local api_key="${OPENROUTER_API_KEY:-}"
    if [[ -z "$api_key" ]] || [[ "$api_key" == "PLACEHOLDER_KEY_INSTALL_SUCCESS" ]]; then
        log::warning "Real API key required for benchmarking" >&2
        log::info "Using simulated benchmark data for demonstration" >&2
        
        # Generate simulated benchmark results
        local response_time=$((RANDOM % 2000 + 500))  # 500-2500ms
        local tokens_per_sec=$((RANDOM % 50 + 10))    # 10-60 tokens/sec
        
        cat <<EOF
{
  "model": "$model",
  "test_type": "simulated",
  "metrics": {
    "response_time_ms": $response_time,
    "tokens_per_second": $tokens_per_sec,
    "success": true
  },
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
        return 0
    fi
    
    # Measure actual performance
    local start_time=$(date +%s%N)
    
    local response=$(timeout "${OPENROUTER_TIMEOUT:-30}" curl -sf \
        -X POST "${OPENROUTER_API_BASE}/chat/completions" \
        -H "Authorization: Bearer ${api_key}" \
        -H "Content-Type: application/json" \
        -d "{
            \"model\": \"$model\",
            \"messages\": [{\"role\": \"user\", \"content\": \"$test_prompt\"}],
            \"max_tokens\": 100,
            \"temperature\": 0.7
        }" 2>&1)
    
    local end_time=$(date +%s%N)
    local elapsed_ms=$(( (end_time - start_time) / 1000000 ))
    
    if [[ $? -eq 0 ]] && [[ -n "$response" ]]; then
        # Extract token usage if available
        local total_tokens=$(echo "$response" | jq -r '.usage.total_tokens // 0')
        local tokens_per_sec=0
        if [[ $total_tokens -gt 0 ]] && [[ $elapsed_ms -gt 0 ]]; then
            tokens_per_sec=$(( total_tokens * 1000 / elapsed_ms ))
        fi
        
        cat <<EOF
{
  "model": "$model",
  "test_type": "actual",
  "metrics": {
    "response_time_ms": $elapsed_ms,
    "tokens_per_second": $tokens_per_sec,
    "total_tokens": $total_tokens,
    "success": true
  },
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
    else
        cat <<EOF
{
  "model": "$model",
  "test_type": "actual",
  "metrics": {
    "response_time_ms": $elapsed_ms,
    "success": false,
    "error": "Request failed or timed out"
  },
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
        return 1
    fi
}

# Compare multiple models
openrouter::benchmark::compare() {
    local models=("$@")
    
    if [[ ${#models[@]} -eq 0 ]]; then
        # Default comparison set
        models=(
            "openai/gpt-3.5-turbo"
            "anthropic/claude-3-haiku"
            "mistralai/mistral-7b"
        )
    fi
    
    openrouter::benchmark::init
    
    log::info "Benchmarking ${#models[@]} models..."
    
    local results_file="${BENCHMARK_DIR}/comparison_$(date +%Y%m%d_%H%M%S).json"
    echo "[" > "$results_file"
    
    local first=true
    for model in "${models[@]}"; do
        log::info "Testing $model..."
        
        if [[ "$first" != true ]]; then
            echo "," >> "$results_file"
        fi
        first=false
        
        openrouter::benchmark::test_model "$model" >> "$results_file"
    done
    
    echo "]" >> "$results_file"
    
    log::success "Benchmark results saved to: $results_file"
    
    # Display summary
    openrouter::benchmark::summarize "$results_file"
}

# Summarize benchmark results
openrouter::benchmark::summarize() {
    local results_file="${1}"
    
    if [[ ! -f "$results_file" ]]; then
        log::error "Results file not found: $results_file"
        return 1
    fi
    
    echo ""
    echo "=== Benchmark Summary ==="
    echo ""
    
    # Parse and display results
    jq -r '.[] | 
        "\(.model): " +
        if .metrics.success then
            "\(.metrics.response_time_ms)ms | \(.metrics.tokens_per_second) tok/s"
        else
            "Failed - \(.metrics.error)"
        end
    ' "$results_file" 2>/dev/null || {
        log::error "Failed to parse benchmark results"
        return 1
    }
    
    echo ""
    
    # Find fastest model
    local fastest=$(jq -r '
        map(select(.metrics.success == true)) |
        min_by(.metrics.response_time_ms) |
        .model
    ' "$results_file" 2>/dev/null)
    
    if [[ -n "$fastest" ]]; then
        log::success "Fastest model: $fastest"
    fi
    
    # Find highest throughput
    local highest_throughput=$(jq -r '
        map(select(.metrics.success == true)) |
        max_by(.metrics.tokens_per_second) |
        .model
    ' "$results_file" 2>/dev/null)
    
    if [[ -n "$highest_throughput" ]]; then
        log::success "Highest throughput: $highest_throughput"
    fi
}

# List all benchmark results
openrouter::benchmark::list() {
    openrouter::benchmark::init
    
    if [[ ! -d "$BENCHMARK_DIR" ]] || [[ -z "$(ls -A "$BENCHMARK_DIR" 2>/dev/null)" ]]; then
        log::info "No benchmark results found"
        return 0
    fi
    
    echo "Available benchmark results:"
    echo ""
    
    for file in "$BENCHMARK_DIR"/*.json; do
        [[ -f "$file" ]] || continue
        local basename=$(basename "$file")
        local timestamp=$(echo "$basename" | sed 's/comparison_\(.*\)\.json/\1/')
        echo "  - $timestamp: $file"
    done
}

# Clean old benchmark results
openrouter::benchmark::clean() {
    local days="${1:-30}"
    
    if [[ ! -d "$BENCHMARK_DIR" ]]; then
        log::info "No benchmarks to clean"
        return 0
    fi
    
    log::info "Cleaning benchmarks older than $days days..."
    
    find "$BENCHMARK_DIR" -name "*.json" -type f -mtime +$days -delete
    
    log::success "Benchmark cleanup complete"
}

# Main benchmark command handler
openrouter::benchmark::main() {
    local action="${1:-compare}"
    shift
    
    case "$action" in
        test)
            openrouter::benchmark::test_model "$@"
            ;;
        compare)
            openrouter::benchmark::compare "$@"
            ;;
        list)
            openrouter::benchmark::list
            ;;
        summarize)
            openrouter::benchmark::summarize "$@"
            ;;
        clean)
            openrouter::benchmark::clean "$@"
            ;;
        *)
            log::error "Unknown benchmark action: $action"
            echo "Usage: benchmark [test|compare|list|summarize|clean]"
            return 1
            ;;
    esac
}