#!/bin/bash
# ====================================================================
# HTTP Assertion Functions
# ====================================================================
#
# HTTP-specific assertion functions for testing web services and APIs.
#
# Functions:
#   - assert_http_success()     - Assert HTTP response is successful
#   - assert_http_status()      - Assert specific HTTP status code (legacy)
#   - curl_and_assert_status()  - Make request and assert status code
#   - curl_and_assert_success() - Make request and assert 2xx success
#   - http_request_with_logging() - Make HTTP request with full logging
#   - log_http_request()        - Log HTTP request details
#   - log_http_response()       - Log HTTP response details
#
# ====================================================================

# Source HTTP logger if available
if [[ -f "${BASH_SOURCE%/*}/../http-logger.sh" ]]; then
    source "${BASH_SOURCE%/*}/../http-logger.sh"
fi

# Assert HTTP response is successful (2xx status)
assert_http_success() {
    local response="$1"
    local message="${2:-HTTP success assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    # Use enhanced logging if available
    if [[ "$(type -t assert_http_success_logged)" == "function" ]]; then
        if assert_http_success_logged "$response" "$message"; then
            return 0
        else
            FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
            return 1
        fi
    fi
    
    # Fallback to original implementation
    # Check if response is empty (likely connection failure)
    if [[ -z "$response" ]]; then
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  No response received (connection failed)"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
    
    # Check for obvious error indicators in response
    if echo "$response" | grep -qi "error\|failed\|not found\|connection"; then
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Response contains error: '${response:0:200}...'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
    
    echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
    PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
    return 0
}

# Assert specific HTTP status code (legacy function - use curl_and_assert_status for new tests)
assert_http_status() {
    local response="$1"
    local expected_status="$2"
    local message="${3:-HTTP status assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    # This function is kept for backward compatibility
    # For new tests, use curl_and_assert_status instead
    echo -e "${ASSERT_YELLOW}⚠${ASSERT_NC} $message (using legacy HTTP status check)"
    echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
    PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
    return 0
}

# Make HTTP request and assert status code (NEW - recommended for new tests)
curl_and_assert_status() {
    local url="$1"
    local expected_status="${2:-200}"
    local message="${3:-HTTP request assertion}"
    local curl_args="${4:--s --max-time 10}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    local response_file="/tmp/curl_response_$$"
    local status_code
    
    # Make request with status code capture
    status_code=$(curl $curl_args -w "%{http_code}" -o "$response_file" "$url" 2>/dev/null)
    local curl_exit_code=$?
    
    # Check if curl command succeeded
    if [[ $curl_exit_code -ne 0 ]]; then
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Connection failed to: $url"
        rm -f "$response_file"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
    
    # Check status code
    if [[ "$status_code" == "$expected_status" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message (HTTP $status_code)"
        cat "$response_file"  # Return response body
        rm -f "$response_file"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Expected HTTP status: $expected_status"
        echo "  Actual HTTP status: $status_code"
        echo "  URL: $url"
        rm -f "$response_file"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Make HTTP request and assert 2xx success (NEW - recommended for new tests)
curl_and_assert_success() {
    local url="$1"
    local message="${2:-HTTP success assertion}"
    local curl_args="${3:--s --max-time 10}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    local response_file="/tmp/curl_response_$$"
    local status_code
    
    # Make request with status code capture
    status_code=$(curl $curl_args -w "%{http_code}" -o "$response_file" "$url" 2>/dev/null)
    local curl_exit_code=$?
    
    # Check if curl command succeeded
    if [[ $curl_exit_code -ne 0 ]]; then
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Connection failed to: $url"
        rm -f "$response_file"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
    
    # Check if status code is 2xx
    if [[ "$status_code" =~ ^2[0-9][0-9]$ ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message (HTTP $status_code)"
        cat "$response_file"  # Return response body
        rm -f "$response_file"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Expected 2xx status code, got: $status_code"
        echo "  URL: $url"
        if [[ -f "$response_file" ]]; then
            echo "  Response: $(head -c 200 "$response_file")..."
        fi
        rm -f "$response_file"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Log HTTP request details for debugging
log_http_request() {
    local method="${1:-GET}"
    local url="$2"
    local headers="${3:-}"
    local body="${4:-}"
    
    if [[ "${TEST_VERBOSE:-false}" == "true" ]]; then
        echo "🔍 HTTP Request Debug:"
        echo "  Method: $method"
        echo "  URL: $url"
        if [[ -n "$headers" ]]; then
            echo "  Headers: $headers"
        fi
        if [[ -n "$body" ]]; then
            echo "  Body: ${body:0:500}$([ ${#body} -gt 500 ] && echo "..." || echo "")"
        fi
        echo
    fi
}

# Log HTTP response details for debugging  
log_http_response() {
    local status_code="$1"
    local response_body="$2"
    local response_headers="${3:-}"
    
    if [[ "${TEST_VERBOSE:-false}" == "true" ]]; then
        echo "🔍 HTTP Response Debug:"
        echo "  Status: $status_code"
        if [[ -n "$response_headers" ]]; then
            echo "  Headers: $response_headers"
        fi
        echo "  Body: ${response_body:0:1000}$([ ${#response_body} -gt 1000 ] && echo "..." || echo "")"
        echo
    fi
}

# Make HTTP request with full logging (enhanced curl wrapper)
http_request_with_logging() {
    local method="${1:-GET}"
    local url="$2"
    local message="${3:-HTTP request}"
    local curl_args="${4:--s --max-time 10}"
    local request_body="${5:-}"
    
    log_http_request "$method" "$url" "" "$request_body"
    
    local response_file="/tmp/http_response_$$"
    local headers_file="/tmp/http_headers_$$"
    local status_code
    
    # Build curl command
    local curl_cmd="curl $curl_args -w '%{http_code}' -o '$response_file' -D '$headers_file'"
    
    if [[ "$method" != "GET" ]]; then
        curl_cmd="$curl_cmd -X $method"
    fi
    
    if [[ -n "$request_body" ]]; then
        curl_cmd="$curl_cmd -d '$request_body' -H 'Content-Type: application/json'"
    fi
    
    curl_cmd="$curl_cmd '$url'"
    
    # Execute request
    status_code=$(eval "$curl_cmd" 2>/dev/null)
    local curl_exit_code=$?
    
    # Read response
    local response_body=""
    local response_headers=""
    if [[ -f "$response_file" ]]; then
        response_body=$(cat "$response_file")
    fi
    if [[ -f "$headers_file" ]]; then
        response_headers=$(cat "$headers_file")
    fi
    
    log_http_response "$status_code" "$response_body" "$response_headers"
    
    # Cleanup
    rm -f "$response_file" "$headers_file"
    
    # Return results
    if [[ $curl_exit_code -eq 0 ]]; then
        echo "$response_body"
        return 0
    else
        echo "HTTP request failed: $message" >&2
        return 1
    fi
}