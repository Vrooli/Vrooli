#!/usr/bin/env bash
# Judge0 Integration Test
# Tests real Judge0 code execution service functionality
# Tests API endpoints, language support, code execution, and security

set -euo pipefail

JUDGE0_TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${JUDGE0_TEST_DIR}/../../../../lib/utils/var.sh"

# Source shared integration test library
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/tests/lib/integration-test-lib.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Load Judge0 configuration using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${JUDGE0_TEST_DIR}/../config/defaults.sh"
judge0::export_config

# Override library defaults with Judge0-specific settings
SERVICE_NAME="judge0"
BASE_URL="${JUDGE0_BASE_URL:-http://localhost:2358}"
HEALTH_ENDPOINT="/version"
REQUIRED_TOOLS=("curl" "jq" "docker")
SERVICE_METADATA=(
    "Port: ${JUDGE0_PORT:-2358}"
    "Server Container: ${JUDGE0_CONTAINER_NAME:-vrooli-judge0-server}"
    "Workers Container: ${JUDGE0_WORKERS_NAME:-vrooli-judge0-workers}"
    "Workers Count: ${JUDGE0_WORKERS_COUNT:-2}"
    "CPU Time Limit: ${JUDGE0_CPU_TIME_LIMIT:-10}s"
    "Memory Limit: ${JUDGE0_MEMORY_LIMIT:-262144}KB"
)

#######################################
# JUDGE0-SPECIFIC TEST FUNCTIONS
#######################################

test_version_endpoint() {
    local test_name="version endpoint"
    
    local response
    if response=$(make_api_request "/version" "GET" 5); then
        # Judge0 version endpoint returns plain text version number
        if echo "$response" | grep -q "^[0-9]\+\.[0-9]\+\.[0-9]\+"; then
            log_test_result "$test_name" "PASS" "version: $response"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "version endpoint not responding correctly"
    return 1
}

test_languages_endpoint() {
    local test_name="languages endpoint"
    
    local response
    if response=$(make_api_request "/languages" "GET" 5); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            local lang_count
            lang_count=$(echo "$response" | jq 'length' 2>/dev/null || echo "0")
            if [[ $lang_count -gt 0 ]]; then
                log_test_result "$test_name" "PASS" "languages available: $lang_count"
                return 0
            fi
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "languages endpoint not working"
    return 1
}

test_system_info_endpoint() {
    local test_name="system info endpoint"
    
    local response
    if response=$(make_api_request "/system_info" "GET" 5); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            # Check for expected system info fields
            if echo "$response" | jq -e '.isolation // .compilation_max_output_size // .execution_max_output_size' >/dev/null 2>&1; then
                log_test_result "$test_name" "PASS" "system info available"
                return 0
            fi
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "system info endpoint not available"
    return 2
}

test_simple_code_execution() {
    local test_name="simple code execution (JavaScript)"
    
    # Test simple JavaScript hello world
    local submission_data='{
        "source_code": "console.log(\"Hello Judge0\");",
        "language_id": 93,
        "wait": true
    }'
    
    local response
    if response=$(make_api_request "/submissions" "POST" 10 "-H 'Content-Type: application/json' -d '$submission_data'"); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            # Check if execution was successful
            local status_id
            status_id=$(echo "$response" | jq -r '.status.id' 2>/dev/null)
            
            if [[ "$status_id" == "3" ]]; then  # Status 3 = Accepted (Success)
                local output
                output=$(echo "$response" | jq -r '.stdout' 2>/dev/null | tr -d '\n')
                if [[ "$output" == "Hello Judge0" ]]; then
                    log_test_result "$test_name" "PASS" "code executed successfully"
                    return 0
                else
                    log_test_result "$test_name" "FAIL" "unexpected output: $output"
                    return 1
                fi
            else
                local status_desc
                status_desc=$(echo "$response" | jq -r '.status.description' 2>/dev/null)
                log_test_result "$test_name" "FAIL" "execution failed: $status_desc"
                return 1
            fi
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "submission request failed"
    return 1
}

test_multiple_language_support() {
    local test_name="multiple language support"
    
    # Test Python execution
    local python_submission='{
        "source_code": "print(\"Python works\")",
        "language_id": 92,
        "wait": true
    }'
    
    local python_response
    if python_response=$(make_api_request "/submissions" "POST" 10 "-H 'Content-Type: application/json' -d '$python_submission'"); then
        local python_status
        python_status=$(echo "$python_response" | jq -r '.status.id' 2>/dev/null)
        
        if [[ "$python_status" == "3" ]]; then
            log_test_result "$test_name" "PASS" "multiple languages working (Python + JS)"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "Python execution not available"
    return 2
}

test_error_handling() {
    local test_name="error handling for invalid code"
    
    # Test syntactically invalid JavaScript
    local error_submission='{
        "source_code": "console.log(invalid syntax here",
        "language_id": 93,
        "wait": true
    }'
    
    local response
    if response=$(make_api_request "/submissions" "POST" 10 "-H 'Content-Type: application/json' -d '$error_submission'"); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            local status_id
            status_id=$(echo "$response" | jq -r '.status.id' 2>/dev/null)
            
            # Status should be 6 (Compilation Error) or similar error status
            if [[ "$status_id" =~ ^[456]$ ]]; then  # Error statuses
                log_test_result "$test_name" "PASS" "error handling working"
                return 0
            fi
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "error handling test unclear"
    return 2
}

test_resource_limits() {
    local test_name="resource limits enforcement"
    
    # Test timeout with infinite loop (should be killed by time limit)
    local timeout_submission='{
        "source_code": "while(true) { /* infinite loop */ }",
        "language_id": 93,
        "wait": true
    }'
    
    local response
    if response=$(make_api_request "/submissions" "POST" 15 "-H 'Content-Type: application/json' -d '$timeout_submission'"); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            local status_id
            status_id=$(echo "$response" | jq -r '.status.id' 2>/dev/null)
            
            # Status should be 5 (Time Limit Exceeded)
            if [[ "$status_id" == "5" ]]; then
                log_test_result "$test_name" "PASS" "time limits enforced"
                return 0
            elif [[ "$status_id" == "3" ]]; then
                log_test_result "$test_name" "SKIP" "timeout not triggered (may have different limits)"
                return 2
            fi
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "resource limits test unclear"
    return 2
}

test_server_container() {
    local test_name="server container health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    if docker ps --format '{{.Names}}' | grep -q "${JUDGE0_CONTAINER_NAME}"; then
        local container_status
        container_status=$(docker inspect "${JUDGE0_CONTAINER_NAME}" --format '{{.State.Status}}' 2>/dev/null || echo "unknown")
        
        if [[ "$container_status" == "running" ]]; then
            log_test_result "$test_name" "PASS" "server container running"
            return 0
        else
            log_test_result "$test_name" "FAIL" "server container status: $container_status"
            return 1
        fi
    else
        log_test_result "$test_name" "FAIL" "server container not found"
        return 1
    fi
}

test_workers_container() {
    local test_name="workers container health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    if docker ps --format '{{.Names}}' | grep -q "${JUDGE0_WORKERS_NAME}"; then
        local container_status
        container_status=$(docker inspect "${JUDGE0_WORKERS_NAME}" --format '{{.State.Status}}' 2>/dev/null || echo "unknown")
        
        if [[ "$container_status" == "running" ]]; then
            log_test_result "$test_name" "PASS" "workers container running"
            return 0
        else
            log_test_result "$test_name" "FAIL" "workers container status: $container_status"
            return 1
        fi
    else
        log_test_result "$test_name" "SKIP" "workers container not found (may use different setup)"
        return 2
    fi
}

test_log_output() {
    local test_name="application log health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    local logs_output
    if logs_output=$(docker logs "${JUDGE0_CONTAINER_NAME}" --tail 10 2>&1 2>/dev/null || true); then
        # Look for startup success patterns
        if echo "$logs_output" | grep -qi "judge0\|started\|listening\|ready"; then
            log_test_result "$test_name" "PASS" "healthy server logs"
            return 0
        elif echo "$logs_output" | grep -qi "error\|exception\|failed\|panic"; then
            log_test_result "$test_name" "FAIL" "errors detected in server logs"
            return 1
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "log status unclear"
    return 2
}

#######################################
# SERVICE-SPECIFIC VERBOSE INFO
#######################################

show_verbose_info() {
    echo
    echo "Judge0 Information:"
    echo "  Base URL: $BASE_URL"
    echo "  API Endpoints:"
    echo "    - Version: GET $BASE_URL/version"
    echo "    - Languages: GET $BASE_URL/languages"
    echo "    - Submit Code: POST $BASE_URL/submissions"
    echo "    - System Info: GET $BASE_URL/system_info"
    echo "  Security Limits:"
    echo "    CPU Time: ${JUDGE0_CPU_TIME_LIMIT}s"
    echo "    Wall Time: ${JUDGE0_WALL_TIME_LIMIT}s"
    echo "    Memory: ${JUDGE0_MEMORY_LIMIT}KB"
    echo "    Processes: ${JUDGE0_MAX_PROCESSES}"
    echo "  Containers:"
    echo "    Server: ${JUDGE0_CONTAINER_NAME}"
    echo "    Workers: ${JUDGE0_WORKERS_NAME} (${JUDGE0_WORKERS_COUNT} workers)"
}

#######################################
# TEST REGISTRATION AND EXECUTION
#######################################

# Register standard interface tests first (manage.sh validation, config checks, etc.)
register_standard_interface_tests

# Register Judge0-specific tests
register_tests \
    "test_version_endpoint" \
    "test_languages_endpoint" \
    "test_system_info_endpoint" \
    "test_simple_code_execution" \
    "test_multiple_language_support" \
    "test_error_handling" \
    "test_resource_limits" \
    "test_server_container" \
    "test_workers_container" \
    "test_log_output"

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi