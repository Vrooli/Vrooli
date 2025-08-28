#!/usr/bin/env bash
# Integration Test Library
# Shared utilities and patterns for resource integration tests
# This library eliminates code duplication and standardizes test behavior

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"

#######################################
# CORE CONFIGURATION FRAMEWORK
#######################################

# Configuration will be set by service-specific tests
# These provide fallback defaults if not set
init_config() {
    # Set defaults if not already configured by service
    SERVICE_NAME="${SERVICE_NAME:-${RESOURCE_NAME:-unknown-service}}"
    BASE_URL="${BASE_URL:-${RESOURCE_BASE_URL:-${SERVICE_BASE_URL:-http://localhost:8080}}}"
    TIMEOUT="${TIMEOUT:-${RESOURCE_TIMEOUT:-${SERVICE_TIMEOUT:-30}}}"
    VERBOSE="${VERBOSE:-${TEST_VERBOSE:-${SERVICE_VERBOSE:-false}}}"
    
    # Service-specific configuration with defaults
    HEALTH_ENDPOINT="${HEALTH_ENDPOINT:-/health}"
    
    # Ensure arrays are initialized
    if [[ ! -v REQUIRED_TOOLS ]]; then
        REQUIRED_TOOLS=("curl")
    fi
    if [[ ! -v SERVICE_METADATA ]]; then
        SERVICE_METADATA=()
    fi
}

#######################################
# STANDARD COLORS AND FORMATTING  
#######################################

readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

#######################################
# TEST TRACKING STATE
#######################################

TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0
declare -a FAILED_TESTS=()
declare -a REGISTERED_TESTS=()

#######################################
# CORE UTILITY FUNCTIONS
#######################################

log_test_result() {
    local test_name="$1"
    local status="$2"
    local message="${3:-}"
    
    case "$status" in
        "PASS")
            echo -e "Testing $test_name... ${GREEN}PASS${NC}${message:+ ($message)}"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            ;;
        "FAIL")
            echo -e "Testing $test_name... ${RED}FAIL${NC}${message:+ ($message)}"
            TESTS_FAILED=$((TESTS_FAILED + 1))
            FAILED_TESTS+=("$test_name")
            ;;
        "SKIP")
            echo -e "Testing $test_name... ${YELLOW}SKIP${NC}${message:+ ($message)}"
            TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
            ;;
    esac
}

check_service_available() {
    local url="$1"
    local timeout="${2:-5}"
    local endpoint="${3:-$HEALTH_ENDPOINT}"
    
    if curl -sf --max-time "$timeout" "$url$endpoint" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

integration_test_lib::make_api_request() {
    local endpoint="$1"
    local method="${2:-GET}"
    local timeout="${3:-$TIMEOUT}"
    local additional_args="${4:-}"
    
    # Build curl command without eval for safety
    if [[ -n "$additional_args" ]]; then
        # Use array expansion for safer handling
        # shellcheck disable=SC2086
        curl -sf --max-time "$timeout" -X "$method" $additional_args "$BASE_URL$endpoint" 2>/dev/null
    else
        curl -sf --max-time "$timeout" -X "$method" "$BASE_URL$endpoint" 2>/dev/null
    fi
}

check_required_tools() {
    local missing_tools=()
    
    for tool in "${REQUIRED_TOOLS[@]}"; do
        if ! command -v "$tool" >/dev/null 2>&1; then
            missing_tools+=("$tool")
        fi
    done
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        echo -e "${RED}Required tools missing: ${missing_tools[*]}${NC}"
        return 1
    fi
    
    return 0
}

#######################################
# ENHANCED COMMON PATTERNS
#######################################

# File upload helper for multipart/form-data requests
# Usage: test_file_upload "/endpoint" "/path/to/file" ["field_name"] ["additional_curl_args"]
test_file_upload() {
    local endpoint="$1"
    local file_path="$2"
    local field_name="${3:-file}"
    local additional_args="${4:-}"
    
    # Check if file exists
    if [[ ! -f "$file_path" ]]; then
        return 1
    fi
    
    # Make the upload request
    local response
    local http_code
    http_code=$(curl -sf -w "%{http_code}" -o /dev/null \
        --max-time "$TIMEOUT" \
        -X POST \
        -F "${field_name}=@${file_path}" \
        $additional_args \
        "$BASE_URL$endpoint" 2>/dev/null)
    
    if [[ "$http_code" == "200" ]] || [[ "$http_code" == "201" ]]; then
        return 0
    else
        return 1
    fi
}

# Async job polling with configurable timeout and status checking
# Usage: poll_async_job "/endpoint" "job_id" [max_attempts] [sleep_time]
poll_async_job() {
    local endpoint="$1"
    local job_id="$2"
    local max_attempts="${3:-20}"
    local sleep_time="${4:-3}"
    
    for i in $(seq 1 "$max_attempts"); do
        local response
        if response=$(integration_test_lib::make_api_request "$endpoint/$job_id" "GET" 10); then
            # Try to extract status field (adjust based on actual response format)
            local status
            status=$(echo "$response" | jq -r '.status // .state // empty' 2>/dev/null)
            
            case "$status" in
                completed|success|done|finished)
                    return 0
                    ;;
                failed|error|cancelled)
                    return 1
                    ;;
                *)
                    # Status is pending or unknown, continue polling
                    sleep "$sleep_time"
                    ;;
            esac
        else
            # Request failed, wait and retry
            sleep "$sleep_time"
        fi
    done
    
    # Timeout reached
    return 2
}

# Enhanced health check with retries and optional custom endpoint
# Usage: wait_for_health [max_attempts] [sleep_time] [custom_endpoint]
wait_for_health() {
    local max_attempts="${1:-10}"
    local sleep_time="${2:-3}"
    local endpoint="${3:-$HEALTH_ENDPOINT}"
    
    for i in $(seq 1 "$max_attempts"); do
        if check_service_available "$BASE_URL" 5 "$endpoint"; then
            return 0
        fi
        
        # Don't sleep on last attempt
        if [[ $i -lt $max_attempts ]]; then
            [[ "$VERBOSE" == "true" ]] && echo "  Waiting for service to be ready... (attempt $i/$max_attempts)"
            sleep "$sleep_time"
        fi
    done
    
    return 1
}

# HTTP status code validation helper
# Usage: check_http_status "/endpoint" "expected_code" ["method"] ["additional_args"]
check_http_status() {
    local endpoint="$1"
    local expected_code="$2"
    local method="${3:-GET}"
    local additional_args="${4:-}"
    
    local http_code
    http_code=$(curl -sf -w "%{http_code}" -o /dev/null \
        --max-time "${TIMEOUT:-30}" \
        -X "$method" \
        $additional_args \
        "$BASE_URL$endpoint" 2>/dev/null)
    
    if [[ "$http_code" == "$expected_code" ]]; then
        return 0
    else
        # Return the actual code for error reporting
        echo "$http_code"
        return 1
    fi
}

# Create temporary test file with automatic cleanup
# Usage: test_file=$(create_test_file "test_content" [".txt"])
# Note: Files are automatically cleaned up on script exit
create_test_file() {
    local content="$1"
    local extension="${2:-.txt}"
    
    local temp_file="${TMPDIR:-/tmp}/$(basename "$SERVICE_NAME")_test_$$_$(date +%s)${extension}"
    echo "$content" > "$temp_file"
    
    # Add to cleanup list
    TEST_TEMP_FILES+=("$temp_file")
    
    echo "$temp_file"
}

# Cleanup function for temporary test files
cleanup_test_files() {
    if [[ ${#TEST_TEMP_FILES[@]} -gt 0 ]]; then
        for file in "${TEST_TEMP_FILES[@]}"; do
            [[ -f "$file" ]] && trash::safe_remove "$file" --test-cleanup
        done
    fi
}

# Standard error response testing
# Usage: test_error_response "/endpoint" "404" "non-existent endpoint"
test_error_response() {
    local endpoint="$1"
    local expected_code="$2"
    local test_description="$3"
    
    local actual_code
    if actual_code=$(check_http_status "$endpoint" "$expected_code"); then
        return 0
    else
        [[ "$VERBOSE" == "true" ]] && echo "  Expected $expected_code, got ${actual_code:-unknown}"
        return 1
    fi
}

# JSON response validation helper
# Usage: validate_json_field "$response" ".field.name" "expected_value"
validate_json_field() {
    local json="$1"
    local field_path="$2"
    local expected_value="${3:-}"
    
    # Check if jq is available
    if ! command -v jq >/dev/null 2>&1; then
        return 1
    fi
    
    local actual_value
    actual_value=$(echo "$json" | jq -r "$field_path" 2>/dev/null)
    
    if [[ -z "$expected_value" ]]; then
        # Just check field exists and is not null
        if [[ -n "$actual_value" ]] && [[ "$actual_value" != "null" ]]; then
            return 0
        fi
    else
        # Check field matches expected value
        if [[ "$actual_value" == "$expected_value" ]]; then
            return 0
        fi
    fi
    
    return 1
}

# Multi-file upload helper
# Usage: test_multi_file_upload "/endpoint" file1 file2 ... -- -F "key=value"
test_multi_file_upload() {
    local endpoint="$1"
    shift
    
    local curl_args=""
    local files=()
    local additional_args=""
    local parsing_files=true
    
    # Parse arguments
    for arg in "$@"; do
        if [[ "$arg" == "--" ]]; then
            parsing_files=false
            continue
        fi
        
        if [[ "$parsing_files" == "true" ]]; then
            files+=("$arg")
        else
            additional_args="$additional_args $arg"
        fi
    done
    
    # Build curl file arguments as an array for safety
    local curl_cmd_args=()
    for file in "${files[@]}"; do
        if [[ ! -f "$file" ]]; then
            return 1
        fi
        curl_cmd_args+=(-F "files=@$file")
    done
    
    # Make the request without eval
    local http_code
    # shellcheck disable=SC2086
    http_code=$(curl -sf -w '%{http_code}' -o /dev/null \
        --max-time "$TIMEOUT" \
        -X POST \
        "${curl_cmd_args[@]}" \
        $additional_args \
        "$BASE_URL$endpoint" 2>/dev/null)
    
    if [[ "$http_code" == "200" ]] || [[ "$http_code" == "201" ]]; then
        return 0
    else
        return 1
    fi
}

# Initialize temp files array for cleanup tracking
declare -a TEST_TEMP_FILES=()

#######################################
# TEST REGISTRATION SYSTEM
#######################################

register_tests() {
    REGISTERED_TESTS=("$@")
}

run_test_with_error_handling() {
    local test_function="$1"
    
    if declare -f "$test_function" >/dev/null 2>&1; then
        # Temporarily disable exit-on-error for test execution
        set +e
        "$test_function"
        local result=$?
        set -e
        # Don't return individual test result codes - let tests update global counters
        return 0
    else
        log_test_result "$test_function" "FAIL" "test function not found"
        return 1
    fi
}

#######################################
# STANDARD TEST IMPLEMENTATIONS
#######################################

test_service_availability() {
    local test_name="service availability"
    
    if check_service_available "$BASE_URL" 5; then
        log_test_result "$test_name" "PASS"
        return 0
    else
        log_test_result "$test_name" "FAIL" "service not responding at $BASE_URL"
        return 1
    fi
}

#######################################
# MAIN TEST EXECUTION FRAMEWORK
#######################################

show_test_header() {
    echo "======================================="
    echo "    $SERVICE_NAME Integration Test Suite"
    echo "======================================="
    echo "Base URL: $BASE_URL"
    echo "Timeout: ${TIMEOUT}s"
    
    # Show service-specific metadata
    for metadata in "${SERVICE_METADATA[@]}"; do
        [[ -n "$metadata" ]] && echo "$metadata"
    done
    
    echo
}

run_all_tests() {
    # Initialize configuration with service-specific overrides
    init_config
    
    show_test_header
    
    # Check required tools
    if ! check_required_tools; then
        return 1
    fi
    
    # Check if service is available before running detailed tests
    if ! check_service_available "$BASE_URL" 5; then
        echo -e "${YELLOW}Service not available at $BASE_URL${NC}"
        echo -e "${YELLOW}Skipping all tests${NC}"
        return 2
    fi
    
    # Run registered tests with error handling
    set +e  # Temporarily disable exit-on-error for test execution
    
    # Always run service availability test first
    test_service_availability
    
    # Run all registered service-specific tests
    for test_function in "${REGISTERED_TESTS[@]}"; do
        run_test_with_error_handling "$test_function"
    done
    
    set -e  # Re-enable exit-on-error
    
    return 0
}

show_summary() {
    local total_tests=$((TESTS_PASSED + TESTS_FAILED + TESTS_SKIPPED))
    
    echo
    echo "======================================="
    echo "Results: $TESTS_PASSED passed, $TESTS_FAILED failed, $TESTS_SKIPPED skipped"
    
    if [[ ${#FAILED_TESTS[@]} -gt 0 ]]; then
        echo
        echo -e "${RED}Failed tests:${NC}"
        for test in "${FAILED_TESTS[@]}"; do
            echo "  - $test"
        done
    fi
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        if [[ $TESTS_PASSED -gt 0 ]]; then
            echo -e "${GREEN}All tests passed!${NC}"
        else
            echo -e "${YELLOW}All tests were skipped${NC}"
        fi
    else
        echo -e "${RED}$TESTS_FAILED test(s) failed${NC}"
    fi
    echo "======================================="
    
    # Show service-specific information when verbose
    if [[ "$VERBOSE" == "true" ]]; then
        show_verbose_info
    fi
}

# Override this function in service-specific tests for custom verbose output
show_verbose_info() {
    echo
    echo "Service Information:"
    echo "  Base URL: $BASE_URL"
    echo "  Health Endpoint: $HEALTH_ENDPOINT"
    echo "  Required Tools: ${REQUIRED_TOOLS[*]}"
}

#######################################
# STANDARD RESOURCE INTERFACE TESTS
#######################################

# Test that manage.sh script exists and is executable
test_manage_script_exists() {
    local test_name="manage.sh exists"
    # SCRIPT_DIR is set by the calling test script (points to test/ directory)
    # Go up one level to get to the resource directory
    local resource_dir="${RESOURCE_DIR:-$(dirname "$SCRIPT_DIR")}"
    local manage_script="$resource_dir/manage.sh"
    
    if [[ -f "$manage_script" ]] && [[ -x "$manage_script" ]]; then
        log_test_result "$test_name" "PASS"
        return 0
    elif [[ -f "$manage_script" ]]; then
        log_test_result "$test_name" "FAIL" "exists but not executable"
        return 1
    else
        log_test_result "$test_name" "FAIL" "script not found"
        return 1
    fi
}

# Test that manage.sh supports required actions
test_manage_required_actions() {
    local test_name="manage.sh required actions"
    local resource_dir="${RESOURCE_DIR:-$(dirname "$SCRIPT_DIR")}"
    local manage_script="$resource_dir/manage.sh"
    
    # Required actions for all resources
    local required_actions=("status" "start" "stop" "install")
    local missing_actions=()
    
    for action in "${required_actions[@]}"; do
        # Test if action is supported by checking help output
        if ! bash "$manage_script" --help 2>&1 | grep -q "$action"; then
            missing_actions+=("$action")
        fi
    done
    
    if [[ ${#missing_actions[@]} -eq 0 ]]; then
        log_test_result "$test_name" "PASS"
        return 0
    else
        log_test_result "$test_name" "FAIL" "missing actions: ${missing_actions[*]}"
        return 1
    fi
}

# Test that config/defaults.sh exists
test_config_defaults_exists() {
    local test_name="config/defaults.sh exists"
    local resource_dir="${RESOURCE_DIR:-$(dirname "$SCRIPT_DIR")}"
    local defaults_file="$resource_dir/config/defaults.sh"
    
    if [[ -f "$defaults_file" ]]; then
        log_test_result "$test_name" "PASS"
        return 0
    else
        log_test_result "$test_name" "FAIL" "file not found"
        return 1
    fi
}

# Test that config/messages.sh exists
test_config_messages_exists() {
    local test_name="config/messages.sh exists"
    local resource_dir="${RESOURCE_DIR:-$(dirname "$SCRIPT_DIR")}"
    local messages_file="$resource_dir/config/messages.sh"
    
    if [[ -f "$messages_file" ]]; then
        log_test_result "$test_name" "PASS"
        return 0
    else
        log_test_result "$test_name" "FAIL" "file not found"
        return 1
    fi
}

# Test that README.md documentation exists
test_readme_exists() {
    local test_name="README.md exists"
    local resource_dir="${RESOURCE_DIR:-$(dirname "$SCRIPT_DIR")}"
    local readme_file="$resource_dir/README.md"
    
    if [[ -f "$readme_file" ]]; then
        # Check if README has minimum content
        local line_count
        line_count=$(wc -l < "$readme_file")
        if [[ $line_count -gt 10 ]]; then
            log_test_result "$test_name" "PASS"
            return 0
        else
            log_test_result "$test_name" "FAIL" "README too short (${line_count} lines)"
            return 1
        fi
    else
        log_test_result "$test_name" "SKIP" "README not found (optional)"
        return 2
    fi
}

# Test that manage.sh responds to --version
test_manage_version_flag() {
    local test_name="manage.sh --version"
    local resource_dir="${RESOURCE_DIR:-$(dirname "$SCRIPT_DIR")}"
    local manage_script="$resource_dir/manage.sh"
    
    if output=$(bash "$manage_script" --version 2>&1); then
        if [[ -n "$output" ]]; then
            log_test_result "$test_name" "PASS" "version: $output"
            return 0
        else
            log_test_result "$test_name" "SKIP" "no version output"
            return 2
        fi
    else
        log_test_result "$test_name" "SKIP" "--version not supported"
        return 2
    fi
}

# Test that manage.sh responds to --help
test_manage_help_flag() {
    local test_name="manage.sh --help"
    local resource_dir="${RESOURCE_DIR:-$(dirname "$SCRIPT_DIR")}"
    local manage_script="$resource_dir/manage.sh"
    
    if output=$(bash "$manage_script" --help 2>&1); then
        if [[ "$output" == *"Usage"* ]] || [[ "$output" == *"usage"* ]]; then
            log_test_result "$test_name" "PASS"
            return 0
        else
            log_test_result "$test_name" "FAIL" "help output doesn't contain usage info"
            return 1
        fi
    else
        log_test_result "$test_name" "FAIL" "--help failed"
        return 1
    fi
}

# Test that port is properly registered
test_port_registration() {
    local test_name="port registration"
    local resource_dir="${RESOURCE_DIR:-$(dirname "$SCRIPT_DIR")}"
    
    # Get resource name from path
    local resource_name
    resource_name=$(basename "$resource_dir")
    
    # Check if resource has a port in port_registry.sh
    local port_registry="$var_PORT_REGISTRY_FILE"
    if [[ -f "$port_registry" ]]; then
        if grep -q "$resource_name" "$port_registry" 2>/dev/null; then
            log_test_result "$test_name" "PASS"
            return 0
        else
            log_test_result "$test_name" "SKIP" "no port registration found"
            return 2
        fi
    else
        log_test_result "$test_name" "SKIP" "port registry not found"
        return 2
    fi
}

# Register standard interface tests
# These can be called from service-specific tests
register_standard_interface_tests() {
    # Add standard tests to the registered tests array
    local standard_tests=(
        "test_manage_script_exists"
        "test_manage_required_actions"
        "test_config_defaults_exists"
        "test_config_messages_exists"
        "test_manage_help_flag"
    )
    
    # Optional tests that won't fail the suite
    local optional_tests=(
        "test_readme_exists"
        "test_manage_version_flag"
        "test_port_registration"
    )
    
    # Combine with existing registered tests
    REGISTERED_TESTS=("${standard_tests[@]}" "${optional_tests[@]}" "${REGISTERED_TESTS[@]}")
}

#######################################
# MAIN EXECUTION FRAMEWORK
#######################################

integration_test_main() {
    # Set up cleanup trap for temp files
    trap cleanup_test_files EXIT
    
    # Run tests
    run_all_tests
    local test_result=$?
    
    # Show summary
    show_summary
    
    # Cleanup temp files (trap will also handle this on exit)
    cleanup_test_files
    
    # Determine exit code based on results
    if [[ $test_result -eq 2 ]]; then
        # Service not available - skip
        exit 2
    elif [[ $TESTS_FAILED -gt 0 ]]; then
        # Some tests failed
        exit 1
    else
        # All tests passed or skipped
        exit 0
    fi
}

#######################################
# LIBRARY INITIALIZATION
#######################################

# Provide easy way for services to use standard main execution
# Usage in service test: integration_test_main "$@"
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    # Library is being sourced, not executed directly
    : # Do nothing, just provide functions
else
    # Library is being executed directly (shouldn't happen, but handle gracefully)
    echo "Integration test library should be sourced, not executed directly"
    exit 1
fi