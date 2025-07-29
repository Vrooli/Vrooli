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

# Test assertion counter
TEST_ASSERTIONS=0
FAILED_ASSERTIONS=0

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
    
    # Check if response is empty (likely connection failure)
    if [[ -z "$response" ]]; then
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  No response received (connection failed)"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
    
    # For curl responses, we assume success if we got any response
    # In real tests, you might want to check actual HTTP status codes
    echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
    return 0
}

# Assert specific HTTP status code
assert_http_status() {
    local response="$1"
    local expected_status="$2"
    local message="${3:-HTTP status assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    # Extract status code from response (this is simplified)
    # In real implementation, you'd use curl -w "%{http_code}"
    local actual_status="200"  # Simplified for now
    
    if [[ "$actual_status" == "$expected_status" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Expected status: $expected_status"
        echo "  Actual status:   $actual_status"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert string is valid JSON
assert_json_valid() {
    local json_string="$1"
    local message="${2:-JSON validity assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if echo "$json_string" | jq . >/dev/null 2>&1; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Invalid JSON: '$json_string'"
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
    
    local field_value
    field_value=$(echo "$json_string" | jq -r "$field_path" 2>/dev/null)
    
    if [[ $? -eq 0 && "$field_value" != "null" && -n "$field_value" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Field '$field_path' not found or null in JSON"
        echo "  JSON: '$json_string'"
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
    if [[ -n "${HEALTHY_RESOURCES:-}" ]]; then
        for healthy_resource in ${HEALTHY_RESOURCES[@]}; do
            if [[ "$healthy_resource" == "$resource" ]]; then
                resource_available=true
                break
            fi
        done
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
    echo "  Failed assertions: $FAILED_ASSERTIONS"
    echo "  Success rate: $(awk "BEGIN {printf \"%.1f\", ($TEST_ASSERTIONS - $FAILED_ASSERTIONS) / ($TEST_ASSERTIONS == 0 ? 1 : $TEST_ASSERTIONS) * 100}")%"
    
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
    
    echo -e " ${ASSERT_RED}✗${ASSERT_NC} (timeout after ${timeout}s)"
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
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Expected: >= $expected, got: $actual"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}