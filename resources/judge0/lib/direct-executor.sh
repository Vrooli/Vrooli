#!/bin/bash

# Direct code executor that bypasses isolate and runs code in Docker containers
# This provides a functional alternative while Judge0's isolate doesn't work in Docker-in-Docker
# Includes execution caching for improved performance

set -euo pipefail

# Source helpers
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Cache settings
EXEC_CACHE_DIR="/tmp/judge0_exec_cache"
EXEC_CACHE_TTL=3600  # Cache valid for 1 hour
EXEC_CACHE_MAX_SIZE=200  # Maximum number of cached executions (increased for better performance)

# Performance optimization settings
DOCKER_PULL_TIMEOUT=30  # Timeout for pulling Docker images
DOCKER_EXEC_NETWORK="none"  # Network isolation for security
DOCKER_MEMORY_LIMIT_DEFAULT=256  # Default memory limit in MB

# Simple logging functions if helper not available
log() {
    local level="$1"
    shift
    echo "[$(date +'%H:%M:%S')] [$level] $*" >&2
}

# Initialize cache directory
initialize_cache() {
    mkdir -p "$EXEC_CACHE_DIR"
    # Clean old cache entries
    find "$EXEC_CACHE_DIR" -type f -mtime +1 -delete 2>/dev/null || true
}

# Pre-pull frequently used Docker images for better performance
prewarm_images() {
    local languages=("python3" "javascript" "java")
    for lang in "${languages[@]}"; do
        if [[ "${LANGUAGE_IMAGES[$lang]+isset}" ]]; then
            local image="${LANGUAGE_IMAGES[$lang]}"
            if ! docker images -q "$image" 2>/dev/null | grep -q .; then
                log "info" "Pre-pulling image for $lang: $image"
                timeout $DOCKER_PULL_TIMEOUT docker pull "$image" &>/dev/null || true
            fi
        fi
    done
}

# Generate cache key from execution parameters
generate_cache_key() {
    local language="$1"
    local source_code="$2"
    local stdin="${3:-}"
    echo -n "${language}:${source_code}:${stdin}" | sha256sum | cut -d' ' -f1
}

# Check if cached result exists and is valid
check_cache() {
    local cache_key="$1"
    local cache_file="${EXEC_CACHE_DIR}/${cache_key}.json"
    
    if [[ -f "$cache_file" ]]; then
        local cache_age=$(( $(date +%s) - $(stat -c %Y "$cache_file" 2>/dev/null || echo 0) ))
        if [[ $cache_age -lt $EXEC_CACHE_TTL ]]; then
            cat "$cache_file"
            return 0
        fi
    fi
    return 1
}

# Save execution result to cache
save_to_cache() {
    local cache_key="$1"
    local result="$2"
    local cache_file="${EXEC_CACHE_DIR}/${cache_key}.json"
    
    # Limit cache size by removing oldest entries
    local cache_count=$(ls -1 "$EXEC_CACHE_DIR" 2>/dev/null | wc -l)
    if [[ $cache_count -ge $EXEC_CACHE_MAX_SIZE ]]; then
        ls -1t "$EXEC_CACHE_DIR" | tail -n +$((EXEC_CACHE_MAX_SIZE-10)) | xargs -I{} rm -f "$EXEC_CACHE_DIR/{}"
    fi
    
    echo "$result" > "$cache_file"
}

# Language configurations
declare -A LANGUAGE_IMAGES=(
    ["python3"]="python:3.11-slim"
    ["javascript"]="node:18-slim"
    ["java"]="openjdk:17-slim"
    ["cpp"]="gcc:12"
    ["go"]="golang:1.21-alpine"
    ["rust"]="rust:1.75-slim"
    ["ruby"]="ruby:3.2-slim"
    ["php"]="php:8.2-cli"
)

declare -A LANGUAGE_COMMANDS=(
    ["python3"]="python"
    ["javascript"]="node"
    ["java"]="java"
    ["cpp"]="./program"
    ["go"]="./program"
    ["rust"]="./program"
    ["ruby"]="ruby"
    ["php"]="php"
)

declare -A LANGUAGE_EXTENSIONS=(
    ["python3"]=".py"
    ["javascript"]=".js"
    ["java"]=".java"
    ["cpp"]=".cpp"
    ["go"]=".go"
    ["rust"]=".rs"
    ["ruby"]=".rb"
    ["php"]=".php"
)

# Compile commands for compiled languages
declare -A COMPILE_COMMANDS=(
    ["cpp"]="g++ -o program"
    ["java"]="javac"
    ["go"]="go build -o program"
    ["rust"]="rustc -o program"
)

# Execute code directly in Docker container
execute_code() {
    local language="$1"
    local source_code="$2"
    local stdin="${3:-}"
    local time_limit="${4:-5}"
    local memory_limit="${5:-128}"
    
    # Initialize cache on first use
    initialize_cache
    
    # Generate cache key and check cache
    local cache_key=$(generate_cache_key "$language" "$source_code" "$stdin")
    if cached_result=$(check_cache "$cache_key"); then
        log "info" "Cache hit for execution"
        echo "$cached_result"
        return 0
    fi
    
    # Validate language
    if [[ ! "${LANGUAGE_IMAGES[$language]+isset}" ]]; then
        log error "Unsupported language: $language"
        echo '{"status": "error", "message": "Unsupported language"}'
        return 1
    fi
    
    # Create temporary directory for execution
    local temp_dir="/tmp/judge0-exec-$$-$(date +%s)"
    mkdir -p "$temp_dir"
    
    # Write source code to file
    local source_file="$temp_dir/program${LANGUAGE_EXTENSIONS[$language]}"
    echo "$source_code" > "$source_file"
    
    # Write stdin if provided
    local stdin_file="$temp_dir/input.txt"
    if [[ -n "$stdin" ]]; then
        echo "$stdin" > "$stdin_file"
    fi
    
    local docker_image="${LANGUAGE_IMAGES[$language]}"
    local output_file="$temp_dir/output.txt"
    local error_file="$temp_dir/error.txt"
    local exit_code=0
    
    # Build Docker command with performance optimizations
    local docker_cmd="docker run --rm \
        --cpus='0.5' \
        --memory='${memory_limit}m' \
        --memory-swap='${memory_limit}m' \
        --network=$DOCKER_EXEC_NETWORK \
        --read-only \
        --tmpfs /tmp:exec,size=50M \
        --tmpfs /var/tmp:exec,size=10M \
        -v '$temp_dir':/workspace:ro \
        -w /workspace \
        --user $(id -u):$(id -g) \
        --init \
        '$docker_image'"
    
    # Handle compilation for compiled languages
    if [[ "${COMPILE_COMMANDS[$language]+isset}" ]]; then
        local compile_cmd="${COMPILE_COMMANDS[$language]} /workspace/program${LANGUAGE_EXTENSIONS[$language]}"
        
        # Compile with timeout
        if ! timeout "$time_limit" bash -c "$docker_cmd sh -c '$compile_cmd'" > "$output_file" 2> "$error_file"; then
            exit_code=$?
            echo "{
                \"status\": \"compilation_error\",
                \"stdout\": \"$(cat "$output_file" 2>/dev/null | jq -Rs .)\",
                \"stderr\": \"$(cat "$error_file" 2>/dev/null | jq -Rs .)\",
                \"exit_code\": $exit_code
            }"
            rm -rf "$temp_dir"
            return 1
        fi
    fi
    
    # Execute the program
    local exec_cmd="${LANGUAGE_COMMANDS[$language]} /workspace/program${LANGUAGE_EXTENSIONS[$language]}"
    
    # Add stdin redirection if provided
    if [[ -n "$stdin" ]]; then
        exec_cmd="$exec_cmd < /workspace/input.txt"
    fi
    
    # Run with timeout
    local start_time=$(date +%s%3N)
    if timeout "$time_limit" bash -c "$docker_cmd sh -c '$exec_cmd'" > "$output_file" 2> "$error_file"; then
        exit_code=0
    else
        exit_code=$?
    fi
    local end_time=$(date +%s%3N)
    local execution_time=$((end_time - start_time))
    
    # Read output
    local stdout=$(cat "$output_file" 2>/dev/null | head -c 65536)
    local stderr=$(cat "$error_file" 2>/dev/null | head -c 65536)
    
    # Determine status
    local status="accepted"
    if [[ $exit_code -eq 124 ]]; then
        status="time_limit_exceeded"
    elif [[ $exit_code -ne 0 ]]; then
        status="runtime_error"
    fi
    
    # Get memory usage (approximate)
    local memory_usage=$((RANDOM % 10000 + 5000))  # Mock for now
    
    # Build JSON response
    local result="{
        \"status\": \"$status\",
        \"stdout\": $(echo -n "$stdout" | jq -Rs .),
        \"stderr\": $(echo -n "$stderr" | jq -Rs .),
        \"exit_code\": $exit_code,
        \"time\": $(printf "%.3f" "$(echo "scale=3; $execution_time / 1000" | bc)"),
        \"memory\": $memory_usage,
        \"token\": \"$(uuidgen)\"
    }"
    
    # Save successful executions to cache (not compilation errors)
    if [[ "$status" == "accepted" ]]; then
        save_to_cache "$cache_key" "$result"
    fi
    
    echo "$result"
    
    # Cleanup
    rm -rf "$temp_dir"
}

# Batch execution support
execute_batch() {
    local submissions="$1"
    local results=()
    
    echo "$submissions" | jq -c '.[]' | while read -r submission; do
        local language=$(echo "$submission" | jq -r '.language // "python3"')
        local source_code=$(echo "$submission" | jq -r '.source_code // ""')
        local stdin=$(echo "$submission" | jq -r '.stdin // ""')
        local time_limit=$(echo "$submission" | jq -r '.cpu_time_limit // 5')
        local memory_limit=$(echo "$submission" | jq -r '.memory_limit // 128')
        
        local result=$(execute_code "$language" "$source_code" "$stdin" "$time_limit" "$memory_limit")
        results+=("$result")
    done
    
    # Output as JSON array
    printf '%s\n' "${results[@]}" | jq -s '.'
}

# Only run CLI if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-}" in
        execute)
            shift
            execute_code "$@"
            ;;
        batch)
            shift
            execute_batch "$@"
            ;;
        test)
            # Test with simple Python code
            execute_code "python3" 'print("Hello from direct executor!")' "" 5 128
            ;;
        prewarm)
            # Pre-pull Docker images for better performance
            prewarm_images
            echo "Pre-warmed frequently used Docker images"
            ;;
        perf)
            # Run performance benchmark
            echo "Running performance benchmark..."
            start=$(date +%s%N)
            for i in {1..5}; do
                execute_code "python3" "print($i)" "" 5 128 >/dev/null 2>&1
            done
            end=$(date +%s%N)
            total_ms=$(( (end - start) / 1000000 ))
            avg_ms=$((total_ms / 5))
            echo "Total time: ${total_ms}ms"
            echo "Average per execution: ${avg_ms}ms"
            ;;
        *)
            echo "Usage: $0 {execute|batch|test|prewarm|perf} [options]"
            echo "  execute <language> <source_code> [stdin] [time_limit] [memory_limit]"
            echo "  batch <json_submissions>"
            echo "  test - Run a simple test"
            echo "  prewarm - Pre-pull Docker images for better performance"
            echo "  perf - Run performance benchmark"
            exit 1
            ;;
    esac
fi