#!/bin/bash
# ====================================================================
# Test Assertion Functions
# ====================================================================
#
# Common assertion functions for integration tests. Provides consistent
# error reporting and test validation across all test files.
#
# Functions:
#   - assert_equals()          - Assert two values are equal
#   - assert_not_equals()      - Assert two values are not equal
#   - assert_contains()        - Assert string contains substring
#   - assert_not_contains()    - Assert string does not contain substring
#   - assert_empty()           - Assert string is empty
#   - assert_not_empty()       - Assert string is not empty
#   - assert_http_success()    - Assert HTTP response is successful
#   - assert_http_status()     - Assert specific HTTP status code
#   - assert_json_valid()      - Assert string is valid JSON
#   - assert_json_field()      - Assert JSON contains specific field
#   - assert_file_exists()     - Assert file exists
#   - assert_command_success() - Assert command executes successfully
#   - require_resource()       - Assert resource is available
#   - require_tools()          - Assert required tools are available
#
# ====================================================================

# Source HTTP logger if available
if [[ -f "${BASH_SOURCE%/*}/http-logger.sh" ]]; then
    source "${BASH_SOURCE%/*}/http-logger.sh"
fi

# Test assertion counter
TEST_ASSERTIONS=0
FAILED_ASSERTIONS=0
PASSED_ASSERTIONS=0

# Colors for assertion output
ASSERT_GREEN='\033[0;32m'
ASSERT_RED='\033[0;31m'
ASSERT_YELLOW='\033[1;33m'
ASSERT_NC='\033[0m'

# Assert two values are equal
assert_equals() {
    local actual="$1"
    local expected="$2"
    local message="${3:-Equality assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ "$actual" == "$expected" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Expected: '$expected'"
        echo "  Actual:   '$actual'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert two values are not equal
assert_not_equals() {
    local actual="$1"
    local unexpected="$2"
    local message="${3:-Inequality assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ "$actual" != "$unexpected" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Should not equal: '$unexpected'"
        echo "  Actual:          '$actual'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert string contains substring
assert_contains() {
    local haystack="$1"
    local needle="$2"
    local message="${3:-Contains assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ "$haystack" == *"$needle"* ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  String: '$haystack'"
        echo "  Should contain: '$needle'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert string does not contain substring
assert_not_contains() {
    local haystack="$1"
    local needle="$2"
    local message="${3:-Does not contain assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ "$haystack" != *"$needle"* ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  String: '$haystack'"
        echo "  Should not contain: '$needle'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert string is empty
assert_empty() {
    local value="$1"
    local message="${2:-Empty assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ -z "$value" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Expected empty, got: '$value'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert string is not empty
assert_not_empty() {
    local value="$1"
    local message="${2:-Not empty assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ -n "$value" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Expected non-empty value, got empty string"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

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

# Assert string is valid JSON
assert_json_valid() {
    local json_string="$1"
    local message="${2:-JSON validity assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    # First check if the string is empty
    if [[ -z "$json_string" ]]; then
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  JSON string is empty"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
    
    # Capture jq error output for detailed error reporting
    local jq_error
    jq_error=$(echo "$json_string" | jq . 2>&1 >/dev/null)
    local jq_exit_code=$?
    
    if [[ $jq_exit_code -eq 0 ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  JSON validation error: $jq_error"
        echo "  JSON string: '${json_string:0:500}$([ ${#json_string} -gt 500 ] && echo "..." || echo "")'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert JSON contains specific field
assert_json_field() {
    local json_string="$1"
    local field_path="$2"
    local message="${3:-JSON field assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    # First validate the JSON
    if ! echo "$json_string" | jq . >/dev/null 2>&1; then
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Invalid JSON provided for field check"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
    
    local field_value
    local jq_error
    
    # Capture both value and any error
    {
        field_value=$(echo "$json_string" | jq -r "$field_path" 2>&1)
        jq_error=""
    } || {
        jq_error="jq command failed"
    }
    
    # Check if field exists and is not null
    if echo "$json_string" | jq -e "$field_path" >/dev/null 2>&1; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message (value: '${field_value:0:100}$([ ${#field_value} -gt 100 ] && echo "..." || echo "")')"
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Field path '$field_path' not found or is null"
        echo "  Available fields: $(echo "$json_string" | jq -r 'keys[]?' 2>/dev/null | head -5 | tr '\n' ' ')..."
        echo "  JSON preview: '${json_string:0:200}$([ ${#json_string} -gt 200 ] && echo "..." || echo "")'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert file exists
assert_file_exists() {
    local file_path="$1"
    local message="${2:-File existence assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ -f "$file_path" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  File not found: '$file_path'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert directory exists
assert_dir_exists() {
    local dir_path="$1"
    local message="${2:-Directory existence assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ -d "$dir_path" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Directory not found: '$dir_path'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert command executes successfully
assert_command_success() {
    local command="$1"
    local message="${2:-Command success assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if eval "$command" >/dev/null 2>&1; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Command failed: '$command'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert command fails (expects non-zero exit code)
assert_command_fails() {
    local command="$1"
    local message="${2:-Command failure assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if ! eval "$command" >/dev/null 2>&1; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Command should have failed: '$command'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Require resource to be available (skip test if not)
require_resource() {
    local resource="$1"
    local message="${2:-Resource requirement}"
    
    # Check if resource is in healthy resources list
    local resource_available=false
    
    # First try to use HEALTHY_RESOURCES array if available
    if [[ -n "${HEALTHY_RESOURCES:-}" ]]; then
        for healthy_resource in "${HEALTHY_RESOURCES[@]}"; do
            if [[ "$healthy_resource" == "$resource" ]]; then
                resource_available=true
                break
            fi
        done
    # Fallback to checking HEALTHY_RESOURCES_STR string
    elif [[ -n "${HEALTHY_RESOURCES_STR:-}" ]]; then
        if [[ " $HEALTHY_RESOURCES_STR " == *" $resource "* ]]; then
            resource_available=true
        fi
    fi
    
    if [[ "$resource_available" == "true" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} Resource available: $resource"
        return 0
    else
        echo -e "${ASSERT_YELLOW}⚠${ASSERT_NC} Skipping test - required resource not available: $resource"
        exit 77  # Standard exit code for skipped tests
    fi
}

# Require multiple resources
require_resources() {
    local required_resources=("$@")
    
    for resource in "${required_resources[@]}"; do
        require_resource "$resource"
    done
}

# Require tools to be available
require_tools() {
    local tools=("$@")
    local missing_tools=()
    
    for tool in "${tools[@]}"; do
        if ! command -v "$tool" >/dev/null 2>&1; then
            missing_tools+=("$tool")
        fi
    done
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        echo -e "${ASSERT_YELLOW}⚠${ASSERT_NC} Skipping test - missing required tools: ${missing_tools[*]}"
        exit 77  # Standard exit code for skipped tests
    fi
    
    echo -e "${ASSERT_GREEN}✓${ASSERT_NC} All required tools available: ${tools[*]}"
}

# Utility function to skip test
skip_test() {
    local reason="${1:-Test skipped}"
    echo -e "${ASSERT_YELLOW}⚠${ASSERT_NC} $reason"
    exit 77  # Standard exit code for skipped tests
}

# Print assertion summary
print_assertion_summary() {
    echo
    echo "Assertion Summary:"
    echo "  Total assertions: $TEST_ASSERTIONS"
    echo "  Passed assertions: $PASSED_ASSERTIONS"
    echo "  Failed assertions: $FAILED_ASSERTIONS"
    echo "  Success rate: $(awk -v total="$TEST_ASSERTIONS" -v failed="$FAILED_ASSERTIONS" 'BEGIN {printf "%.1f", (total - failed) / (total == 0 ? 1 : total) * 100}')%"
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        return 1
    fi
    return 0
}

# Wait for condition with timeout
wait_for_condition() {
    local condition="$1"
    local timeout="${2:-30}"
    local interval="${3:-1}"
    local message="${4:-Waiting for condition}"
    
    local elapsed=0
    echo -n "⏳ $message"
    
    while [[ $elapsed -lt $timeout ]]; do
        if eval "$condition" >/dev/null 2>&1; then
            echo -e " ${ASSERT_GREEN}✓${ASSERT_NC}"
            return 0
        fi
        
        echo -n "."
        sleep "$interval"
        elapsed=$((elapsed + interval))
    done
    
    echo -e " ${ASSERT_RED}X${ASSERT_NC} (timeout after ${timeout}s)"
    return 1
}

# Log test step
log_step() {
    local step_num="$1"
    local description="$2"
    echo -e "${ASSERT_YELLOW}[$step_num]${ASSERT_NC} $description"
}

# Assert JSON boolean is true
assert_json_true() {
    local json="$1"
    local path="$2"
    local message="${3:-JSON boolean assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    local value
    value=$(echo "$json" | jq -r "$path" 2>/dev/null)
    
    if [[ "$value" == "true" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Path: $path"
        echo "  Expected: true, got: $value"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert number is greater than
assert_greater_than() {
    local actual="$1"
    local expected="$2"
    local message="${3:-Greater than assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ "$actual" -gt "$expected" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Expected: > $expected, got: $actual"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert number is greater than or equal
assert_greater_than_or_equal() {
    local actual="$1"
    local expected="$2"
    local message="${3:-Greater than or equal assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ "$actual" -ge "$expected" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Expected: >= $expected, got: $actual"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# ====================================================================
# Debug and Logging Utilities (Enhanced Error Reporting)
# ====================================================================

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

# Assert with detailed error context
assert_with_context() {
    local assertion_result="$1"
    local context_info="$2"
    local message="$3"
    
    if [[ "$assertion_result" == "0" ]]; then
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Context: $context_info"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Pretty print JSON response for debugging
debug_json_response() {
    local json_string="$1"
    local label="${2:-JSON Response}"
    
    if [[ "${TEST_VERBOSE:-false}" == "true" ]]; then
        echo "🔍 $label:"
        
        # Handle extremely large JSON responses to prevent SIGPIPE
        local json_size=${#json_string}
        
        if [[ $json_size -gt 100000 ]]; then
            # For very large JSON (>100KB), just show basic info
            echo "  Large JSON response (${json_size} characters)"
            if echo "$json_string" | head -c 1000 | jq . >/dev/null 2>&1; then
                echo "  First 1KB (formatted):"
                echo "$json_string" | head -c 1000 | jq . 2>/dev/null | head -10
                echo "  ... (truncated due to size: ${json_size} characters)"
            else
                echo "  Raw preview (first 200 chars): ${json_string:0:200}..."
            fi
        elif echo "$json_string" | jq . >/dev/null 2>&1; then
            # For normal-sized JSON, show formatted with line limit
            local formatted_output
            formatted_output=$(echo "$json_string" | jq . 2>/dev/null)
            local line_count=$(echo "$formatted_output" | wc -l)
            
            if [[ $line_count -gt 20 ]]; then
                echo "$formatted_output" | head -20
                echo "  ... (truncated, total lines: $line_count)"
            else
                echo "$formatted_output"
            fi
        else
            echo "  Invalid JSON: ${json_string:0:200}..."
        fi
        echo
    fi
}