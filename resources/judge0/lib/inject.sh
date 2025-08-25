#!/usr/bin/env bash
# Judge0 Injection Module
# Handles code submission and execution

set -eo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
JUDGE0_INJECT_LIB_DIR="${APP_ROOT}/resources/judge0/lib"

# Source common functions
source "${JUDGE0_INJECT_LIB_DIR}/common.sh"
source "${JUDGE0_INJECT_LIB_DIR}/api.sh"

# Source utilities
SCRIPTS_DIR="${APP_ROOT}/scripts"
source "${SCRIPTS_DIR}/lib/utils/format.sh"
source "${SCRIPTS_DIR}/lib/utils/log.sh"

#######################################
# Submit code for execution
# Arguments:
#   $1 - Language ID (e.g., 71 for Python, 63 for JavaScript)
#   $2 - Source code
#   $3 - Standard input (optional)
#   $4 - Expected output (optional)
# Returns:
#   Submission token
#######################################
judge0::inject::submit_code() {
    local language_id="$1"
    local source_code="$2"
    local stdin="${3:-}"
    local expected_output="${4:-}"
    
    local submission_json=$(jq -n \
        --arg lang "$language_id" \
        --arg code "$source_code" \
        --arg input "$stdin" \
        --arg output "$expected_output" \
        '{
            language_id: ($lang | tonumber),
            source_code: $code,
            stdin: $input,
            expected_output: $output,
            wall_time_limit: "10",
            cpu_time_limit: "10",
            memory_limit: 262144,
            stack_limit: 128000,
            enable_per_process_and_thread_time_limit: false,
            enable_per_process_and_thread_memory_limit: false,
            max_file_size: 4096
        }')
    
    local response=$(judge0::api::request "POST" "/submissions?wait=false" "$submission_json")
    echo "$response" | jq -r '.token // empty'
}

#######################################
# Get submission result
# Arguments:
#   $1 - Submission token
# Returns:
#   Submission result JSON
#######################################
judge0::inject::get_result() {
    local token="$1"
    judge0::api::request "GET" "/submissions/${token}?fields=*"
}

#######################################
# Wait for submission to complete
# Arguments:
#   $1 - Submission token
#   $2 - Max wait time in seconds (default: 30)
# Returns:
#   Final submission result
#######################################
judge0::inject::wait_for_result() {
    local token="$1"
    local max_wait="${2:-30}"
    local elapsed=0
    
    while [[ $elapsed -lt $max_wait ]]; do
        local result=$(judge0::inject::get_result "$token")
        local status_id=$(echo "$result" | jq -r '.status.id // 0')
        
        # Status IDs: 1-2 = queued/processing, 3+ = finished
        if [[ $status_id -ge 3 ]]; then
            echo "$result"
            return 0
        fi
        
        sleep 1
        ((elapsed++))
    done
    
    log::error "Submission timeout after ${max_wait} seconds"
    return 1
}

#######################################
# Submit and execute code from file
# Arguments:
#   $1 - Code file path
#   $2 - Language (python, javascript, go, etc.)
#   $3 - Input file (optional)
#   $4 - Expected output file (optional)
# Returns:
#   Execution result
#######################################
judge0::inject::execute_file() {
    local code_file="$1"
    local language="$2"
    local input_file="${3:-}"
    local expected_file="${4:-}"
    
    if [[ ! -f "$code_file" ]]; then
        log::error "Code file not found: $code_file"
        return 1
    fi
    
    local source_code=$(cat "$code_file")
    local stdin=""
    local expected_output=""
    
    [[ -f "$input_file" ]] && stdin=$(cat "$input_file")
    [[ -f "$expected_file" ]] && expected_output=$(cat "$expected_file")
    
    # Map language names to IDs
    local language_id
    case "$language" in
        python|python3)     language_id=71 ;;
        javascript|js|node) language_id=63 ;;
        typescript|ts)      language_id=74 ;;
        go|golang)          language_id=60 ;;
        java)               language_id=62 ;;
        c)                  language_id=50 ;;
        cpp|c++)            language_id=54 ;;
        rust)               language_id=73 ;;
        ruby)               language_id=72 ;;
        bash|shell)         language_id=46 ;;
        *)
            log::error "Unsupported language: $language"
            return 1
            ;;
    esac
    
    log::info "Submitting $language code for execution..."
    
    local token=$(judge0::inject::submit_code "$language_id" "$source_code" "$stdin" "$expected_output")
    if [[ -z "$token" ]]; then
        log::error "Failed to submit code"
        return 1
    fi
    
    log::info "Submission token: $token"
    
    local result=$(judge0::inject::wait_for_result "$token")
    local status=$(echo "$result" | jq -r '.status.description // "unknown"')
    local stdout=$(echo "$result" | jq -r '.stdout // ""')
    local stderr=$(echo "$result" | jq -r '.stderr // ""')
    local compile_output=$(echo "$result" | jq -r '.compile_output // ""')
    
    log::info "Execution status: $status"
    
    if [[ -n "$stdout" ]]; then
        log::success "Output:"
        echo "$stdout"
    fi
    
    if [[ -n "$stderr" ]]; then
        log::warning "Errors:"
        echo "$stderr"
    fi
    
    if [[ -n "$compile_output" ]]; then
        log::warning "Compile output:"
        echo "$compile_output"
    fi
    
    echo "$result"
}

#######################################
# List available languages
# Returns:
#   List of supported languages with IDs
#######################################
judge0::inject::list_languages() {
    local response=$(judge0::api::request "GET" "/languages")
    echo "$response" | jq -r '.[] | "\(.id): \(.name)"' | sort -n
}

#######################################
# Main injection handler
# Arguments:
#   $1 - Injection type (file, code, list-languages)
#   $@ - Additional arguments
#######################################
judge0::inject::main() {
    local action="${1:-file}"
    shift || true
    
    case "$action" in
        file)
            judge0::inject::execute_file "$@"
            ;;
        code)
            local language="$1"
            local code="$2"
            shift 2 || true
            
            # Create temp file for code
            local temp_file=$(mktemp /tmp/judge0_code_XXXXX)
            echo "$code" > "$temp_file"
            
            judge0::inject::execute_file "$temp_file" "$language" "$@"
            local result=$?
            
            rm -f "$temp_file"
            return $result
            ;;
        list-languages)
            judge0::inject::list_languages
            ;;
        *)
            log::error "Unknown action: $action"
            echo "Usage: inject [file|code|list-languages] <args>"
            return 1
            ;;
    esac
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    judge0::inject::main "$@"
fi