#!/usr/bin/env bash
# Judge0 API Module
# Handles all API interactions with Judge0

#######################################
# Make authenticated API request
# Arguments:
#   $1 - HTTP method
#   $2 - Endpoint path
#   $3 - Request body (optional)
# Returns:
#   Response body
#######################################
judge0::api::request() {
    local method="$1"
    local endpoint="$2"
    local body="${3:-}"
    
    local url="${JUDGE0_BASE_URL}${endpoint}"
    local api_key=$(judge0::get_api_key)
    
    local curl_args=(
        -s
        -X "$method"
        -H "Content-Type: application/json"
        -H "Accept: application/json"
    )
    
    # Add authentication if enabled
    if [[ "$JUDGE0_ENABLE_AUTHENTICATION" == "true" ]] && [[ -n "$api_key" ]]; then
        curl_args+=(-H "X-Auth-Token: $api_key")
    fi
    
    # Add body if provided
    if [[ -n "$body" ]]; then
        curl_args+=(-d "$body")
    fi
    
    curl "${curl_args[@]}" "$url"
}

#######################################
# Health check
# Returns:
#   0 if healthy, 1 if not
#######################################
judge0::api::health_check() {
    local response=$(judge0::api::request "GET" "$JUDGE0_HEALTH_ENDPOINT" 2>/dev/null)
    
    # Check if we got a non-empty response (version endpoint returns plain text)
    if [[ -n "$response" ]] && [[ "$response" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Test API connectivity
#######################################
judge0::api::test() {
    log::info "$JUDGE0_MSG_API_TEST"
    
    if judge0::api::health_check; then
        log::success "$JUDGE0_MSG_API_SUCCESS"
        return 0
    else
        log::error "$JUDGE0_MSG_API_FAILED"
        return 1
    fi
}

#######################################
# Get system hardware information
# Returns:
#   JSON object with system hardware info
#######################################
judge0::api::system_info() {
    judge0::api::request "GET" "/system_info"
}

#######################################
# Get Judge0 about information (version, etc)
# Returns:
#   JSON object with about info including version
#######################################
judge0::api::about_info() {
    judge0::api::request "GET" "/about"
}

#######################################
# Get available languages
# Returns:
#   JSON array of languages
#######################################
judge0::api::get_languages() {
    judge0::api::request "GET" "/languages"
}

#######################################
# Get language by ID
# Arguments:
#   $1 - Language ID
# Returns:
#   JSON object with language info
#######################################
judge0::api::get_language() {
    local language_id="$1"
    judge0::api::request "GET" "/languages/$language_id"
}

#######################################
# Submit code for execution
# Arguments:
#   $1 - Source code
#   $2 - Language name
#   $3 - Standard input (optional)
#   $4 - Expected output (optional)
# Returns:
#   Execution result or error
#######################################
judge0::api::submit() {
    local source_code="$1"
    local language="$2"
    local stdin="${3:-}"
    local expected_output="${4:-}"
    
    # Get language ID
    local language_id=$(judge0::get_language_id "$language")
    if [[ -z "$language_id" ]]; then
        # Try to find language ID from API
        local languages=$(judge0::api::get_languages 2>/dev/null || echo "[]")
        language_id=$(echo "$languages" | jq -r ".[] | select(.name | ascii_downcase == \"$language\" | ascii_downcase) | .id" | head -1)
        
        if [[ -z "$language_id" ]]; then
            log::error "$(printf "$JUDGE0_MSG_LANG_NOT_FOUND" "$language")"
            return 1
        fi
    fi
    
    log::info "$JUDGE0_MSG_API_SUBMISSION"
    
    # Prepare submission JSON
    local submission_json=$(jq -n \
        --arg code "$source_code" \
        --arg lang "$language_id" \
        --arg stdin "$stdin" \
        --arg expected "$expected_output" \
        '{
            source_code: $code,
            language_id: ($lang | tonumber),
            stdin: $stdin,
            expected_output: (if $expected != "" then $expected else null end),
            cpu_time_limit: env.JUDGE0_CPU_TIME_LIMIT | tonumber,
            wall_time_limit: env.JUDGE0_WALL_TIME_LIMIT | tonumber,
            memory_limit: env.JUDGE0_MEMORY_LIMIT | tonumber,
            stack_limit: env.JUDGE0_STACK_LIMIT | tonumber,
            max_processes_and_or_threads: env.JUDGE0_MAX_PROCESSES | tonumber,
            max_file_size: env.JUDGE0_MAX_FILE_SIZE | tonumber,
            enable_network: (env.JUDGE0_ENABLE_NETWORK == "true")
        }')
    
    # Submit code
    local response=$(judge0::api::request "POST" "/submissions?wait=true" "$submission_json")
    
    if [[ -z "$response" ]]; then
        log::error "$JUDGE0_MSG_ERR_SUBMISSION"
        return 1
    fi
    
    # Check for errors
    local error=$(echo "$response" | jq -r '.error // empty')
    if [[ -n "$error" ]]; then
        log::error "Submission error: $error"
        return 1
    fi
    
    log::info "$JUDGE0_MSG_API_RESULT"
    
    # Display result
    judge0::api::display_result "$response"
}

#######################################
# Submit batch of code executions
# Arguments:
#   $1 - JSON array of submissions
# Returns:
#   JSON array of results
#######################################
judge0::api::submit_batch() {
    local submissions="$1"
    
    log::info "Submitting batch of $(echo "$submissions" | jq 'length') executions..."
    
    local response=$(judge0::api::request "POST" "/submissions/batch?wait=true" "$submissions")
    
    if [[ -z "$response" ]]; then
        log::error "Batch submission failed"
        return 1
    fi
    
    echo "$response"
}

#######################################
# Get submission by token
# Arguments:
#   $1 - Submission token
# Returns:
#   JSON object with submission details
#######################################
judge0::api::get_submission() {
    local token="$1"
    judge0::api::request "GET" "/submissions/$token"
}

#######################################
# Delete submission by token
# Arguments:
#   $1 - Submission token
#######################################
judge0::api::delete_submission() {
    local token="$1"
    
    if [[ "$JUDGE0_ENABLE_SUBMISSION_DELETE" != "true" ]]; then
        log::error "Submission deletion is disabled"
        return 1
    fi
    
    judge0::api::request "DELETE" "/submissions/$token"
}

#######################################
# Get queue status
# Returns:
#   JSON object with queue info
#######################################
judge0::api::get_queue_status() {
    # This is a simplified version - actual endpoint may vary
    judge0::api::request "GET" "/workers"
}

#######################################
# Display execution result
# Arguments:
#   $1 - Result JSON
#######################################
judge0::api::display_result() {
    local result="$1"
    
    # Parse result
    local status_id=$(echo "$result" | jq -r '.status.id // 0')
    local status_desc=$(echo "$result" | jq -r '.status.description // "Unknown"')
    local stdout=$(echo "$result" | jq -r '.stdout // empty' | base64 -d 2>/dev/null || echo "")
    local stderr=$(echo "$result" | jq -r '.stderr // empty' | base64 -d 2>/dev/null || echo "")
    local compile_output=$(echo "$result" | jq -r '.compile_output // empty' | base64 -d 2>/dev/null || echo "")
    local time=$(echo "$result" | jq -r '.time // "N/A"')
    local memory=$(echo "$result" | jq -r '.memory // "N/A"')
    
    echo
    echo "📊 Execution Result:"
    echo "  Status: $status_desc"
    echo "  Time: ${time}s"
    echo "  Memory: ${memory}KB"
    
    # Handle different status codes
    case $status_id in
        3) # Accepted
            log::success "✅ Execution successful"
            if [[ -n "$stdout" ]]; then
                echo
                echo "📤 Output:"
                echo "$stdout"
            fi
            ;;
        4) # Wrong Answer
            log::error "❌ Wrong answer"
            if [[ -n "$stdout" ]]; then
                echo
                echo "📤 Output:"
                echo "$stdout"
            fi
            ;;
        5) # Time Limit Exceeded
            log::error "$JUDGE0_MSG_ERR_TIME_LIMIT"
            ;;
        6) # Compilation Error
            log::error "$JUDGE0_MSG_ERR_COMPILE"
            if [[ -n "$compile_output" ]]; then
                echo
                echo "🔧 Compilation output:"
                echo "$compile_output"
            fi
            ;;
        7|8|9|10|11) # Runtime Error
            log::error "$JUDGE0_MSG_ERR_RUNTIME"
            if [[ -n "$stderr" ]]; then
                echo
                echo "❌ Error output:"
                echo "$stderr"
            fi
            ;;
        12) # Runtime Error - Memory Limit
            log::error "$JUDGE0_MSG_ERR_MEMORY_LIMIT"
            ;;
        *)
            log::warning "Status: $status_desc (ID: $status_id)"
            ;;
    esac
    
    echo
}