#!/usr/bin/env bash
# Unstructured.io Integration Test
# Comprehensive testing of Unstructured.io API functionality
# Enhanced with fixture-based testing for document processing validation

set -euo pipefail

# Source shared integration test library
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/unstructured-io/test"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/tests/integration-test-lib.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/tests/lib/fixture-helpers.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Override library defaults with Unstructured.io-specific settings
SERVICE_NAME="unstructured-io"
BASE_URL="${UNSTRUCTURED_IO_BASE_URL}"
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

test_fixture_pdf_processing() {
    local test_name="fixture PDF processing"
    
    # Get a PDF fixture
    local test_file
    test_file=$(get_document_fixture "pdf" "simple")
    
    if ! validate_fixture_file "$test_file"; then
        log_test_result "$test_name" "SKIP" "PDF fixture not available"
        return 2
    fi
    
    # Process PDF document
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "$BASE_URL/general/v0/general" \
        -F "files=@$test_file" \
        -F "strategy=fast" \
        --max-time 45 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        response_body=$(echo "$response" | sed 's/HTTPSTATUS:[0-9]*$//')
        
        if [[ "$status_code" == "200" ]]; then
            # Check if we got structured output
            if echo "$response_body" | jq -e '.[0].text' >/dev/null 2>&1; then
                local extracted_text
                extracted_text=$(echo "$response_body" | jq -r '.[0].text' | head -c 100)
                log_test_result "$test_name" "PASS" "extracted: ${extracted_text}..."
                return 0
            fi
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "PDF processing failed"
    return 1
}

test_fixture_word_processing() {
    local test_name="fixture Word document processing"
    
    # Get a Word document fixture
    local test_file
    test_file=$(get_document_fixture "word")
    
    if ! validate_fixture_file "$test_file"; then
        log_test_result "$test_name" "SKIP" "Word fixture not available"
        return 2
    fi
    
    # Process Word document
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "$BASE_URL/general/v0/general" \
        -F "files=@$test_file" \
        -F "strategy=auto" \
        --max-time 45 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [[ "$status_code" == "200" ]]; then
            log_test_result "$test_name" "PASS" "Word document processed"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "Word processing failed"
    return 1
}

test_fixture_excel_processing() {
    local test_name="fixture Excel processing"
    
    # Get an Excel fixture
    local test_file
    test_file=$(get_document_fixture "excel")
    
    if ! validate_fixture_file "$test_file"; then
        log_test_result "$test_name" "SKIP" "Excel fixture not available"
        return 2
    fi
    
    # Process Excel file with table extraction
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "$BASE_URL/general/v0/general" \
        -F "files=@$test_file" \
        -F "strategy=hi_res" \
        -F "pdf_infer_table_structure=true" \
        --max-time 60 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        response_body=$(echo "$response" | sed 's/HTTPSTATUS:[0-9]*$//')
        
        if [[ "$status_code" == "200" ]]; then
            # Check for table elements in response
            if echo "$response_body" | jq -e '.[].type' | grep -q "Table\|table"; then
                log_test_result "$test_name" "PASS" "Excel tables extracted"
                return 0
            else
                log_test_result "$test_name" "PASS" "Excel processed (no tables detected)"
                return 0
            fi
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "Excel processing failed"
    return 1
}

test_fixture_html_processing() {
    local test_name="fixture HTML processing"
    
    # Get an HTML fixture
    local test_file
    test_file=$(get_document_fixture "html")
    
    if ! validate_fixture_file "$test_file"; then
        log_test_result "$test_name" "SKIP" "HTML fixture not available"
        return 2
    fi
    
    # Process HTML document
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "$BASE_URL/general/v0/general" \
        -F "files=@$test_file" \
        -F "strategy=fast" \
        --max-time 30 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [[ "$status_code" == "200" ]]; then
            log_test_result "$test_name" "PASS" "HTML processed successfully"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "HTML processing failed"
    return 1
}

test_fixture_corrupted_document() {
    local test_name="fixture corrupted document handling"
    
    # Get a corrupted document fixture
    local test_file
    test_file=$(get_document_fixture "corrupted")
    
    if ! validate_fixture_file "$test_file"; then
        log_test_result "$test_name" "SKIP" "corrupted fixture not available"
        return 2
    fi
    
    # Try to process corrupted file - should handle gracefully
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "$BASE_URL/general/v0/general" \
        -F "files=@$test_file" \
        -F "strategy=auto" \
        --max-time 30 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        # Should either reject (400/422) or process with limited results
        if [[ "$status_code" =~ ^(400|422|415)$ ]]; then
            log_test_result "$test_name" "PASS" "corrupted file rejected (HTTP: $status_code)"
            return 0
        elif [[ "$status_code" == "200" ]]; then
            log_test_result "$test_name" "PASS" "corrupted file handled gracefully"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "corrupted file not handled properly"
    return 1
}

test_fixture_empty_document() {
    local test_name="fixture empty document handling"
    
    # Get an empty document fixture
    local test_file
    test_file=$(get_document_fixture "empty")
    
    if ! validate_fixture_file "$test_file"; then
        log_test_result "$test_name" "SKIP" "empty fixture not available"
        return 2
    fi
    
    # Process empty file
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "$BASE_URL/general/v0/general" \
        -F "files=@$test_file" \
        -F "strategy=fast" \
        --max-time 30 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        response_body=$(echo "$response" | sed 's/HTTPSTATUS:[0-9]*$//')
        
        if [[ "$status_code" == "200" ]]; then
            # Empty file should return minimal or empty results
            if echo "$response_body" | jq -e 'length == 0 or .[0].text == ""' >/dev/null 2>&1; then
                log_test_result "$test_name" "PASS" "empty file handled correctly"
                return 0
            else
                log_test_result "$test_name" "PASS" "empty file processed"
                return 0
            fi
        elif [[ "$status_code" =~ ^(400|422)$ ]]; then
            log_test_result "$test_name" "PASS" "empty file properly rejected"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "empty file handling failed"
    return 1
}

test_fixture_large_document() {
    local test_name="fixture large document processing"
    
    # Get a large document fixture
    local test_file
    test_file=$(get_document_fixture "pdf" "large")
    
    if ! validate_fixture_file "$test_file"; then
        # Try multipage as alternative
        test_file=$(get_document_fixture "pdf" "multipage")
        if ! validate_fixture_file "$test_file"; then
            log_test_result "$test_name" "SKIP" "large document fixture not available"
            return 2
        fi
    fi
    
    # Process large document with extended timeout
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "$BASE_URL/general/v0/general" \
        -F "files=@$test_file" \
        -F "strategy=fast" \
        -F "max_characters=10000" \
        --max-time 90 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [[ "$status_code" == "200" ]]; then
            log_test_result "$test_name" "PASS" "large document processed"
            return 0
        elif [[ "$status_code" == "413" ]]; then
            log_test_result "$test_name" "PASS" "large document size limit enforced"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "large document processing failed"
    return 1
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
    "test_non_existent_endpoint" \
    "test_fixture_pdf_processing" \
    "test_fixture_word_processing" \
    "test_fixture_excel_processing" \
    "test_fixture_html_processing" \
    "test_fixture_corrupted_document" \
    "test_fixture_empty_document" \
    "test_fixture_large_document"

# Register standard interface tests
register_standard_interface_tests

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi