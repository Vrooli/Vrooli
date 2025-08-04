#!/usr/bin/env bash
# Unstructured.io Integration Test
# Comprehensive testing of Unstructured.io API functionality
# Refactored to use shared integration test library

set -euo pipefail

# Source shared integration test library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../../../tests/lib/integration-test-lib.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Override library defaults with Unstructured.io-specific settings
SERVICE_NAME="unstructured-io"
BASE_URL="${UNSTRUCTURED_IO_BASE_URL:-http://localhost:11450}"
HEALTH_ENDPOINT="/healthcheck"
REQUIRED_TOOLS=("curl")
SERVICE_METADATA=()

#######################################
# UNSTRUCTURED.IO-SPECIFIC TEST FUNCTIONS
#######################################

test_health_endpoint() {
    local test_name="health endpoint"
    
    if make_api_request "/healthcheck" "GET" 10 >/dev/null 2>&1; then
        log_test_result "$test_name" "PASS"
        return 0
    else
        log_test_result "$test_name" "FAIL" "health check failed"
        return 1
    fi
}

test_document_processing() {
    local test_name="document processing"
    
    # Create test document using framework helper
    local test_content='# Test Document

This is a test document for Unstructured.io processing.

## Features
- Document parsing
- Table extraction  
- OCR capabilities

This demonstrates the API capabilities.'
    
    local test_file
    test_file=$(create_test_file "$test_content" ".txt")
    
    # Test processing using framework file upload helper
    if test_file_upload "/general/v0/general" "$test_file" "files" "-F 'strategy=fast' -F 'languages=eng'"; then
        log_test_result "$test_name" "PASS"
        return 0
    else
        log_test_result "$test_name" "FAIL" "document processing failed"
        return 1
    fi
}

test_processing_strategies() {
    local test_name="processing strategies"
    
    # Create simple test file using framework helper
    local simple_file
    simple_file=$(create_test_file "Simple test document for strategy testing." ".txt")
    
    local strategies=("fast" "hi_res" "auto")
    local strategy_failures=0
    
    for strategy in "${strategies[@]}"; do
        if ! test_file_upload "/general/v0/general" "$simple_file" "files" "-F 'strategy=${strategy}'"; then
            strategy_failures=$((strategy_failures + 1))
            [[ "$VERBOSE" == "true" ]] && echo "  Strategy '$strategy' failed"
        fi
    done
    
    if [[ $strategy_failures -eq 0 ]]; then
        log_test_result "$test_name" "PASS" "all strategies work"
        return 0
    else
        log_test_result "$test_name" "FAIL" "$strategy_failures strategies failed"
        return 1
    fi
}

test_error_handling() {
    local test_name="error handling"
    
    # Test with invalid strategy using framework helper
    if check_http_status "/general/v0/general" "422" "POST" "-F 'files=@/dev/null' -F 'strategy=invalid_strategy'"; then
        log_test_result "$test_name" "PASS" "invalid strategy properly rejected"
        return 0
    else
        local actual_code=$?
        log_test_result "$test_name" "FAIL" "expected 422 for invalid strategy"
        return 1
    fi
}

test_non_existent_endpoint() {
    local test_name="404 error handling"
    
    # Use framework helper for error response testing
    if test_error_response "/nonexistent" "404" "non-existent endpoint"; then
        log_test_result "$test_name" "PASS"
        return 0
    else
        log_test_result "$test_name" "FAIL" "404 handling failed"
        return 1
    fi
}

#######################################
# SERVICE-SPECIFIC VERBOSE INFO
#######################################

show_verbose_info() {
    echo
    echo "Unstructured.io Information:"
    echo "  API Endpoints:"
    echo "    - Health Check: GET $BASE_URL/healthcheck"
    echo "    - Process Document: POST $BASE_URL/general/v0/general"
    echo "  Supported Strategies: fast, hi_res, auto"
    echo "  Supported Formats: PDF, DOCX, TXT, HTML, Images (OCR), and more"
}

#######################################
# TEST REGISTRATION AND EXECUTION
#######################################

# Register Unstructured.io-specific tests
register_tests \
    "test_health_endpoint" \
    "test_document_processing" \
    "test_processing_strategies" \
    "test_error_handling" \
    "test_non_existent_endpoint"

# Register standard interface tests
register_standard_interface_tests

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi