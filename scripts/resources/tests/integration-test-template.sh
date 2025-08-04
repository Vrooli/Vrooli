#!/usr/bin/env bash
# Resource Integration Test Template
# This template shows how to create a standardized integration test using the shared library
# 
# STANDARD INTERFACE:
# Exit Codes:
#   0 - All tests passed
#   1 - One or more tests failed  
#   2 - Tests skipped (resource not available/configured)
#
# Environment Variables:
#   RESOURCE_NAME        - Name of the resource being tested
#   RESOURCE_BASE_URL    - Base URL for the resource API
#   RESOURCE_TIMEOUT     - Test timeout in seconds (default: 30)
#   TEST_VERBOSE         - Enable verbose output (default: false)
#
# Usage:
#   Copy this template to: scripts/resources/{category}/{resource}/test/integration-test.sh
#   Customize the service-specific configuration and test functions

set -euo pipefail

# Source shared integration test library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../../../tests/lib/integration-test-lib.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# CUSTOMIZE: Set these values for your resource
SERVICE_NAME="template-resource"
BASE_URL="${TEMPLATE_RESOURCE_BASE_URL:-http://localhost:8080}"
TIMEOUT="${RESOURCE_TIMEOUT:-30}"

# CUSTOMIZE: Set health endpoint and required tools
HEALTH_ENDPOINT="/health"                    # Common alternatives: /healthcheck, /api/health, /status
REQUIRED_TOOLS=("curl")                      # Common tools: curl, jq, python3, etc.
SERVICE_METADATA=()                          # Optional: ("Version: 1.0", "Mode: production")

#######################################
# SERVICE-SPECIFIC TEST FUNCTIONS
#######################################

# CUSTOMIZE: Add your resource-specific test functions here
# Each function should:
# - Have a descriptive name starting with "test_"
# - Use log_test_result() to report results
# - Return 0 (pass), 1 (fail), or 2 (skip)

test_version_endpoint() {
    local test_name="version endpoint"
    
    # EXAMPLE: Test a version endpoint
    local response
    if response=$(make_api_request "/version" "GET" 10); then
        # CUSTOMIZE: Adjust JSON parsing for your API response format
        if [[ -n "$response" ]] && echo "$response" | jq -e '.version' >/dev/null 2>&1; then
            local version
            version=$(echo "$response" | jq -r '.version')
            log_test_result "$test_name" "PASS" "version: $version"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "version endpoint not available"
    return 2
}

test_basic_functionality() {
    local test_name="basic functionality"
    
    # CUSTOMIZE: Add tests for core functionality
    # Example patterns:
    
    # 1. Simple GET request
    # if make_api_request "/api/status" "GET" 10 >/dev/null; then
    #     log_test_result "$test_name" "PASS"
    #     return 0
    # fi
    
    # 2. POST request with data
    # local data='{"test": "value"}'
    # if make_api_request "/api/test" "POST" 15 "-H 'Content-Type: application/json' -d '$data'" >/dev/null; then
    #     log_test_result "$test_name" "PASS"
    #     return 0
    # fi
    
    # 3. File-based testing
    # local test_file="/tmp/test_file_$$.txt"
    # echo "test content" > "$test_file"
    # # ... perform test with file ...
    # rm -f "$test_file"
    
    log_test_result "$test_name" "SKIP" "not implemented for this resource"
    return 2
}

test_error_handling() {
    local test_name="error handling"
    
    # CUSTOMIZE: Test error conditions
    # Example: Test invalid endpoint returns 404
    local http_code
    http_code=$(curl -s -w "%{http_code}" -o /dev/null "$BASE_URL/nonexistent" 2>/dev/null)
    
    if [[ "$http_code" == "404" ]]; then
        log_test_result "$test_name" "PASS" "404 properly returned"
        return 0
    else
        log_test_result "$test_name" "FAIL" "expected 404, got $http_code"
        return 1
    fi
}

#######################################
# SERVICE-SPECIFIC VERBOSE INFO (OPTIONAL)
#######################################

# CUSTOMIZE: Override this function to show service-specific information in verbose mode
show_verbose_info() {
    echo
    echo "Template Resource Information:"
    echo "  API Endpoints:"
    echo "    - Health Check: GET $BASE_URL$HEALTH_ENDPOINT"
    echo "    - Version: GET $BASE_URL/version"
    echo "  Documentation: https://example.com/docs"
    echo "  Required Tools: ${REQUIRED_TOOLS[*]}"
}

#######################################
# TEST REGISTRATION AND EXECUTION
#######################################

# CUSTOMIZE: Register your test functions here
# The shared library will run service_availability first, then these tests
register_tests \
    "test_version_endpoint" \
    "test_basic_functionality" \
    "test_error_handling"

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi

#######################################
# CUSTOMIZATION NOTES
#######################################

# Common Health Endpoints:
# - REST APIs: /health, /healthcheck, /api/health
# - Specific services: /system_stats (ComfyUI), /api/tags (Ollama)

# Common Required Tools:
# - Basic HTTP: ("curl")
# - JSON parsing: ("curl" "jq") 
# - Python services: ("curl" "python3")
# - File processing: ("curl" "jq" "python3")

# Test Function Patterns:
# - Always return 0 (pass), 1 (fail), or 2 (skip)
# - Use make_api_request() for HTTP requests
# - Use log_test_result() to report outcomes
# - Handle cleanup in error conditions
# - Use descriptive test names

# Advanced Patterns:
# - For complex tests like ComfyUI workflows, create custom test runners
# - For file cleanup, use trap functions
# - For dependency checking, test prerequisites before main functionality
# - For multi-step workflows, chain tests and skip later ones if earlier fail