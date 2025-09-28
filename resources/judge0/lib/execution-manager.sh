#!/bin/bash

# Execution Manager - Manages different execution methods for Judge0
# Provides automatic fallback between Judge0 API, direct execution, and external API

set -euo pipefail

# Source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="${APP_ROOT:-$(cd "$SCRIPT_DIR/../../.." && pwd)}"
JUDGE0_PORT="${JUDGE0_PORT:-2358}"

# Simple logging if common.sh not available
log() {
    local level="$1"
    shift
    echo "[$(date +'%H:%M:%S')] [$level] $*" >&2
}

# Try to source common utilities
if [[ -f "$SCRIPT_DIR/common.sh" ]]; then
    source "$SCRIPT_DIR/common.sh" 2>/dev/null || true
fi

# Don't source backends by default - call them directly when needed
# This significantly improves startup performance
DIRECT_EXECUTOR="$SCRIPT_DIR/direct-executor.sh"
EXTERNAL_API="$SCRIPT_DIR/external-api.sh"
EXECUTION_POOL="$SCRIPT_DIR/execution-pool.sh"
ANALYTICS="$SCRIPT_DIR/analytics.sh"

# Execution methods (in priority order)
# Changed priority: direct is currently faster than pool
EXECUTION_METHODS=(
    "direct"      # Direct Docker execution (fastest currently)
    "pool"        # Warm container pool (needs optimization)
    "judge0"      # Native Judge0 API (isolate-based)
    "external"    # External Judge0 API (cloud fallback)
    "simple"      # Simple local execution (testing only)
)

# Configuration
EXECUTION_METHOD="${JUDGE0_EXECUTION_METHOD:-auto}"
EXECUTION_TIMEOUT="${JUDGE0_EXECUTION_TIMEOUT:-10}"
EXECUTION_RETRIES="${JUDGE0_EXECUTION_RETRIES:-3}"
EXECUTION_RETRY_DELAY="${JUDGE0_EXECUTION_RETRY_DELAY:-1}"

# Method availability cache
declare -A METHOD_AVAILABLE

# Check if a method is available
check_method_availability() {
    local method="$1"
    
    # Check cache first
    if [[ "${METHOD_AVAILABLE[$method]+isset}" ]]; then
        return "${METHOD_AVAILABLE[$method]}"
    fi
    
    case "$method" in
        pool)
            # Check if execution pool is available
            if [[ -f "$SCRIPT_DIR/execution-pool.sh" ]] && command -v docker &>/dev/null; then
                METHOD_AVAILABLE[$method]=0
                return 0
            else
                METHOD_AVAILABLE[$method]=1
                return 1
            fi
            ;;
            
        judge0)
            # Test Judge0 API with simple code
            local test_result=$(curl -sf -X POST "http://localhost:${JUDGE0_PORT:-2358}/submissions" \
                -H "Content-Type: application/json" \
                -d '{"source_code": "print(1)", "language_id": 92}' 2>/dev/null || echo "failed")
            
            if [[ "$test_result" != "failed" ]] && [[ "$test_result" =~ token ]]; then
                METHOD_AVAILABLE[$method]=0
                return 0
            else
                METHOD_AVAILABLE[$method]=1
                return 1
            fi
            ;;
            
        direct)
            # Check if Docker is available and direct executor exists
            if command -v docker &>/dev/null && [[ -f "$SCRIPT_DIR/direct-executor.sh" ]]; then
                METHOD_AVAILABLE[$method]=0
                return 0
            else
                METHOD_AVAILABLE[$method]=1
                return 1
            fi
            ;;
            
        external)
            # Check if external API is configured
            if [[ -n "${JUDGE0_EXTERNAL_API_URL:-}" ]] && [[ -n "${JUDGE0_EXTERNAL_API_KEY:-}" ]]; then
                METHOD_AVAILABLE[$method]=0
                return 0
            else
                METHOD_AVAILABLE[$method]=1
                return 1
            fi
            ;;
            
        simple)
            # Check if Python is available for simple executor
            if command -v python3 &>/dev/null && [[ -f "$SCRIPT_DIR/simple-exec.sh" ]]; then
                METHOD_AVAILABLE[$method]=0
                return 0
            else
                METHOD_AVAILABLE[$method]=1
                return 1
            fi
            ;;
            
        *)
            METHOD_AVAILABLE[$method]=1
            return 1
            ;;
    esac
}

# Get the best available execution method
get_best_method() {
    # If specific method is configured, try that first
    if [[ "$EXECUTION_METHOD" != "auto" ]]; then
        if check_method_availability "$EXECUTION_METHOD"; then
            echo "$EXECUTION_METHOD"
            return 0
        else
            log warning "Configured method '$EXECUTION_METHOD' not available, falling back to auto"
        fi
    fi
    
    # Auto mode: try each method in priority order
    for method in "${EXECUTION_METHODS[@]}"; do
        if check_method_availability "$method"; then
            echo "$method"
            return 0
        fi
    done
    
    log error "No execution methods available"
    return 1
}

# Execute code using the selected method
execute_with_method() {
    local method="$1"
    local language="$2"
    local source_code="$3"
    local stdin="${4:-}"
    local expected_output="${5:-}"
    local time_limit="${6:-$EXECUTION_TIMEOUT}"
    local memory_limit="${7:-128}"
    
    log info "Executing with method: $method"
    
    # Track execution start time for metrics
    local start_time=$(date +%s%N)
    local result=""
    local exec_status=0
    
    case "$method" in
        pool)
            # Use warm container pool for fastest execution
            # Map language to Docker image (use slim versions for speed)
            local docker_image
            case "$language" in
                python|python3|py)
                    docker_image="python:3.11-slim"
                    ;;
                javascript|js|node)
                    docker_image="node:18-slim"
                    ;;
                java)
                    docker_image="openjdk:17-slim"
                    ;;
                ruby|rb)
                    docker_image="ruby:3.2-slim"
                    ;;
                go|golang)
                    docker_image="golang:1.20-alpine"
                    ;;
                *)
                    # Fallback to direct method for unsupported languages (normalize language name)
                    local unsupported_language="$language"
                    if [[ "$language" == "python" ]]; then
                        unsupported_language="python3"
                    fi
                    "$SCRIPT_DIR/direct-executor.sh" execute "$unsupported_language" "$source_code" "$stdin" "$time_limit" "$memory_limit"
                    return $?
                    ;;
            esac
            
            # Execute using pool
            if [[ -f "$SCRIPT_DIR/execution-pool.sh" ]]; then
                "$SCRIPT_DIR/execution-pool.sh" execute "$docker_image" "$source_code" "$stdin"
            else
                # Fallback to direct if pool not available (normalize language name)
                local fallback_language="$language"
                if [[ "$language" == "python" ]]; then
                    fallback_language="python3"
                fi
                "$SCRIPT_DIR/direct-executor.sh" execute "$fallback_language" "$source_code" "$stdin" "$time_limit" "$memory_limit"
            fi
            ;;
            
        judge0)
            # Use native Judge0 API
            execute_via_judge0_api "$language" "$source_code" "$stdin" "$expected_output" "$time_limit" "$memory_limit"
            ;;
            
        direct)
            # Use direct Docker execution (normalize language name)
            local direct_language="$language"
            if [[ "$language" == "python" ]]; then
                direct_language="python3"
            fi
            "$SCRIPT_DIR/direct-executor.sh" execute "$direct_language" "$source_code" "$stdin" "$time_limit" "$memory_limit"
            ;;
            
        external)
            # Use external Judge0 API
            "$SCRIPT_DIR/external-api.sh" execute "$language" "$source_code" "$stdin" "$time_limit" "$memory_limit"
            ;;
            
        simple)
            # Use simple executor (Python only)
            if [[ "$language" == "python3" ]] || [[ "$language" == "python" ]]; then
                "$SCRIPT_DIR/simple-exec.sh" "$source_code" "$stdin"
            else
                echo '{"status": "error", "message": "Simple executor only supports Python"}'
                return 1
            fi
            ;;
            
        *)
            echo '{"status": "error", "message": "Unknown execution method"}'
            return 1
            ;;
    esac
}

# Execute via Judge0 API with proper error handling
execute_via_judge0_api() {
    local language="$1"
    local source_code="$2"
    local stdin="$3"
    local expected_output="$4"
    local time_limit="$5"
    local memory_limit="$6"
    
    # Map language name to Judge0 language ID
    local language_id=$(get_language_id "$language")
    if [[ -z "$language_id" ]]; then
        echo '{"status": "error", "message": "Unknown language"}'
        return 1
    fi
    
    # Prepare submission data
    local submission_data=$(jq -n \
        --arg code "$source_code" \
        --arg lang "$language_id" \
        --arg stdin "$stdin" \
        --arg expected "$expected_output" \
        --arg cpu "$time_limit" \
        --arg mem "$((memory_limit * 1024))" \
        '{
            source_code: $code,
            language_id: ($lang | tonumber),
            stdin: $stdin,
            expected_output: $expected,
            cpu_time_limit: ($cpu | tonumber),
            memory_limit: ($mem | tonumber),
            wall_time_limit: (($cpu | tonumber) * 2),
            enable_network: false
        }')
    
    # Submit code
    local response=$(curl -sf -X POST "http://localhost:${JUDGE0_PORT:-2358}/submissions?wait=true" \
        -H "Content-Type: application/json" \
        -d "$submission_data" 2>/dev/null)
    
    if [[ $? -ne 0 ]] || [[ -z "$response" ]]; then
        echo '{"status": "error", "message": "Failed to submit to Judge0 API"}'
        return 1
    fi
    
    echo "$response"
}

# Get Judge0 language ID from language name
get_language_id() {
    local language="$1"
    
    case "$language" in
        python|python3) echo "92" ;;
        javascript|js|node) echo "93" ;;
        typescript|ts) echo "94" ;;
        java) echo "91" ;;
        c) echo "104" ;;
        cpp|c++) echo "105" ;;
        go) echo "95" ;;
        rust) echo "73" ;;
        ruby) echo "72" ;;
        php) echo "68" ;;
        *) echo "" ;;
    esac
}

# Main execution function with automatic fallback
execute_code() {
    local language="$1"
    local source_code="$2"
    local stdin="${3:-}"
    local expected_output="${4:-}"
    local time_limit="${5:-$EXECUTION_TIMEOUT}"
    local memory_limit="${6:-128}"
    
    # Get the best available method
    local method=$(get_best_method)
    if [[ $? -ne 0 ]]; then
        echo '{"status": "error", "message": "No execution methods available"}'
        return 1
    fi
    
    # Try execution with retries
    local attempt=1
    local result=""
    local tried_methods=()
    
    while [[ $attempt -le $EXECUTION_RETRIES ]]; do
        log info "Execution attempt $attempt/$EXECUTION_RETRIES with method: $method"
        
        # Execute (individual methods have their own timeouts)
        result=$(execute_with_method "$method" "$language" "$source_code" "$stdin" "$expected_output" "$time_limit" "$memory_limit" 2>&1)
        local exec_status=$?
        
        # Check if execution was successful
        if [[ "$result" =~ \"status\":\ *\"(accepted|wrong_answer)\" ]]; then
            # Record analytics if available
            if command -v judge0::analytics::record_submission &>/dev/null; then
                local exec_time=$(echo "$result" | jq -r '.time // "0"' 2>/dev/null)
                local memory=$(echo "$result" | jq -r '.memory // "0"' 2>/dev/null)
                local status=$(echo "$result" | jq -r '.status // "unknown"' 2>/dev/null)
                local token=$(echo "$result" | jq -r '.token // "'$(uuidgen 2>/dev/null || echo "exec-$$-$RANDOM")'"' 2>/dev/null)
                local lang_id=$(get_language_id "$language")
                judge0::analytics::record_submission "$token" "$lang_id" "$status" "$exec_time" "$memory" 2>/dev/null || true
            fi
            echo "$result"
            return 0
        fi
        
        # If execution timed out or failed
        tried_methods+=("$method")
        
        if [[ $exec_status -eq 124 ]]; then
            log warning "Method $method timed out after $EXECUTION_TIMEOUT seconds"
            result='{"status": "error", "message": "Execution timed out"}'
        fi
        
        # If this method failed, try the next available one
        if [[ $attempt -lt $EXECUTION_RETRIES ]]; then
            log warning "Method $method failed, waiting ${EXECUTION_RETRY_DELAY}s before retry..."
            sleep "$EXECUTION_RETRY_DELAY"
            
            # Mark current method as temporarily unavailable
            METHOD_AVAILABLE[$method]=1
            
            # Find next best method not yet tried
            local next_method=""
            for candidate in "${EXECUTION_METHODS[@]}"; do
                if ! printf '%s\n' "${tried_methods[@]}" | grep -q "^$candidate$" 2>/dev/null; then
                    if check_method_availability "$candidate"; then
                        next_method="$candidate"
                        break
                    fi
                fi
            done
            
            if [[ -n "$next_method" ]]; then
                method="$next_method"
                log info "Switching to fallback method: $method"
            else
                # No new method, retry same method
                unset METHOD_AVAILABLE[$method]
                log info "Retrying with same method: $method"
            fi
        fi
        
        ((attempt++))
    done
    
    echo "$result"
}

# Test execution manager
test_execution_manager() {
    echo "Testing execution manager..."
    
    # Test method availability
    echo "Available methods:"
    for method in "${EXECUTION_METHODS[@]}"; do
        if check_method_availability "$method"; then
            echo "  ✅ $method"
        else
            echo "  ❌ $method"
        fi
    done
    
    # Test Python execution
    echo ""
    echo "Testing Python execution:"
    local result=$(execute_code "python3" 'print("Hello from execution manager!")' "" "" 5 128)
    
    # Only use jq if result is valid JSON
    if echo "$result" | jq . &>/dev/null; then
        echo "$result" | jq .
    else
        echo "$result"
    fi
    
    if [[ "$result" =~ \"status\":\ *\"accepted\" ]]; then
        echo "✅ Execution successful"
    else
        echo "❌ Execution failed"
    fi
}

# Only run CLI if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-}" in
        execute)
            shift
            execute_code "$@"
            ;;
        test)
            test_execution_manager
            ;;
        methods)
            get_best_method
            ;;
        check)
            check_method_availability "${2:-judge0}"
            exit $?
            ;;
        *)
            echo "Usage: $0 {execute|test|methods|check} [args...]"
            echo ""
            echo "Commands:"
            echo "  execute <language> <code> [stdin] [expected] [time_limit] [memory_limit]"
            echo "  test                     - Run self-test"
            echo "  methods                  - Show best available method"
            echo "  check <method>          - Check if method is available"
            exit 1
            ;;
    esac
fi