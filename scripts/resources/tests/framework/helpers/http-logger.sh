#!/bin/bash
# ====================================================================
# HTTP Request/Response Logging for Tests
# ====================================================================
#
# Provides enhanced HTTP logging capabilities for debugging test
# failures and understanding API interactions.
#
# Functions:
#   - http_get()     - GET request with logging
#   - http_post()    - POST request with logging
#   - http_put()     - PUT request with logging
#   - http_delete()  - DELETE request with logging
#   - http_request() - Generic request with logging
#   - log_http()     - Log HTTP request/response details
#
# ====================================================================

# HTTP logging configuration
HTTP_LOG_ENABLED="${HTTP_LOG_ENABLED:-false}"
HTTP_LOG_FILE="${HTTP_LOG_FILE:-/tmp/vrooli_http_${TEST_ID:-test}.log}"
HTTP_LOG_VERBOSE="${HTTP_LOG_VERBOSE:-false}"
HTTP_MAX_RESPONSE_LOG="${HTTP_MAX_RESPONSE_LOG:-1000}"  # Max response size to log

# Initialize HTTP logging
init_http_logging() {
    if [[ "$HTTP_LOG_ENABLED" == "true" ]] || [[ "${TEST_VERBOSE:-false}" == "true" ]]; then
        HTTP_LOG_ENABLED="true"
        echo "📝 HTTP logging enabled: $HTTP_LOG_FILE" >&2
    fi
}

# Log HTTP request/response details
log_http() {
    local method="$1"
    local url="$2"
    local status="$3"
    local request_body="$4"
    local response_body="$5"
    local headers="$6"
    local duration="$7"
    
    if [[ "$HTTP_LOG_ENABLED" != "true" ]]; then
        return
    fi
    
    # Create log entry
    {
        echo "=========================================="
        echo "Timestamp: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "Method: $method"
        echo "URL: $url"
        echo "Status: $status"
        echo "Duration: ${duration}ms"
        
        if [[ -n "$headers" ]]; then
            echo "Headers: $headers"
        fi
        
        if [[ -n "$request_body" ]] && [[ "$request_body" != "null" ]]; then
            echo "Request Body:"
            echo "$request_body" | head -c "$HTTP_MAX_RESPONSE_LOG"
            if [[ ${#request_body} -gt $HTTP_MAX_RESPONSE_LOG ]]; then
                echo "... (truncated, ${#request_body} bytes total)"
            fi
        fi
        
        if [[ -n "$response_body" ]]; then
            echo "Response Body:"
            echo "$response_body" | head -c "$HTTP_MAX_RESPONSE_LOG"
            if [[ ${#response_body} -gt $HTTP_MAX_RESPONSE_LOG ]]; then
                echo "... (truncated, ${#response_body} bytes total)"
            fi
        fi
        echo ""
    } >> "$HTTP_LOG_FILE"
    
    # Also log to stderr if verbose
    if [[ "$HTTP_LOG_VERBOSE" == "true" ]] || [[ "$status" -ge 400 ]]; then
        {
            echo "🌐 HTTP $method $url → $status (${duration}ms)"
            if [[ "$status" -ge 400 ]]; then
                echo "   Error Response: ${response_body:0:200}..."
            fi
        } >&2
    fi
}

# Make HTTP request with logging
http_request() {
    local method="$1"
    local url="$2"
    shift 2
    local curl_args=("$@")
    
    local start_time=$(date +%s%3N || echo $(($(date +%s) * 1000)))
    local temp_file="/tmp/http_response_$$"
    local headers_file="/tmp/http_headers_$$"
    
    # Extract request body if present
    local request_body=""
    local i=0
    for arg in "${curl_args[@]}"; do
        if [[ "$arg" == "-d" ]] || [[ "$arg" == "--data" ]]; then
            request_body="${curl_args[$((i+1))]}"
            break
        fi
        ((i++))
    done
    
    # Add standard curl options for logging
    local curl_cmd=(
        curl
        -s                              # Silent
        -w "\n%{http_code}"            # Write status code
        -D "$headers_file"             # Dump headers
        -o "$temp_file"                # Output to file
        --max-time "${HTTP_TIMEOUT:-30}"
        "$url"
        "${curl_args[@]}"
    )
    
    # Execute request
    local output
    output=$("${curl_cmd[@]}" 2>&1)
    local exit_code=$?
    
    # Extract status code (last line)
    local status_code=""
    local response_body=""
    
    if [[ $exit_code -eq 0 ]]; then
        status_code=$(echo "$output" | tail -1)
        response_body=$(cat "$temp_file" 2>/dev/null || echo "")
    else
        status_code="000"
        response_body="Curl error: $output (exit code: $exit_code)"
    fi
    
    # Calculate duration
    local end_time=$(date +%s%3N || echo $(($(date +%s) * 1000)))
    local duration=$((end_time - start_time))
    
    # Read headers
    local response_headers=""
    if [[ -f "$headers_file" ]]; then
        response_headers=$(cat "$headers_file" | grep -v "^HTTP/" | grep -v "^$" | head -10)
    fi
    
    # Log the request/response
    log_http "$method" "$url" "$status_code" "$request_body" "$response_body" "$response_headers" "$duration"
    
    # Cleanup
    rm -f "$temp_file" "$headers_file"
    
    # Output response for caller
    echo "$response_body"
    
    # Return based on status code
    if [[ "$status_code" =~ ^[23][0-9][0-9]$ ]]; then
        return 0
    else
        return 1
    fi
}

# Convenience wrappers for common HTTP methods
http_get() {
    local url="$1"
    shift
    http_request "GET" "$url" "$@"
}

http_post() {
    local url="$1"
    shift
    http_request "POST" "$url" "$@"
}

http_put() {
    local url="$1"
    shift
    http_request "PUT" "$url" "$@"
}

http_delete() {
    local url="$1"
    shift
    http_request "DELETE" "$url" "$@"
}

# Enhanced assert_http_success with logging
assert_http_success_logged() {
    local response="$1"
    local message="${2:-HTTP request}"
    
    # Try to extract status from response if it's a curl write-out format
    local status_code=""
    if [[ "$response" =~ HTTP/[0-9.]+[[:space:]]+([0-9]+) ]]; then
        status_code="${BASH_REMATCH[1]}"
    elif [[ "$response" =~ \{\"http_code\":([0-9]+)\} ]]; then
        status_code="${BASH_REMATCH[1]}"
    fi
    
    if [[ -n "$status_code" ]]; then
        if [[ "$status_code" =~ ^[23][0-9][0-9]$ ]]; then
            echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message succeeded (HTTP $status_code)"
            return 0
        else
            echo -e "${ASSERT_RED}✗${ASSERT_NC} $message failed (HTTP $status_code)"
            if [[ "$HTTP_LOG_ENABLED" == "true" ]]; then
                echo "   See HTTP log: $HTTP_LOG_FILE" >&2
            fi
            return 1
        fi
    else
        # Fallback to checking if response is not empty
        if [[ -n "$response" ]]; then
            echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message succeeded"
            return 0
        else
            echo -e "${ASSERT_RED}✗${ASSERT_NC} $message failed (empty response)"
            return 1
        fi
    fi
}

# Clean up HTTP logs
cleanup_http_logs() {
    if [[ -f "$HTTP_LOG_FILE" ]]; then
        rm -f "$HTTP_LOG_FILE"
    fi
}

# Show HTTP log summary
show_http_summary() {
    if [[ ! -f "$HTTP_LOG_FILE" ]]; then
        return
    fi
    
    echo "📊 HTTP Request Summary:"
    echo "========================"
    
    # Count requests by status
    echo "Status Codes:"
    grep "^Status: " "$HTTP_LOG_FILE" | sort | uniq -c | sort -rn
    
    # Show failed requests
    local failed_count=$(grep -c "^Status: [45]" "$HTTP_LOG_FILE" || echo "0")
    if [[ $failed_count -gt 0 ]]; then
        echo -e "\n❌ Failed Requests ($failed_count):"
        grep -B3 "^Status: [45]" "$HTTP_LOG_FILE" | grep -E "^(URL:|Status:)" | paste - - | head -10
    fi
    
    # Show slow requests
    echo -e "\n⏱️  Slowest Requests:"
    grep "^Duration: " "$HTTP_LOG_FILE" | sed 's/Duration: //; s/ms//' | sort -rn | head -5 | \
        while read duration; do
            grep -B4 "^Duration: ${duration}ms" "$HTTP_LOG_FILE" | grep "^URL:" | head -1 | \
                sed "s/^URL: /  ${duration}ms - /"
        done
}

# Initialize on source
init_http_logging