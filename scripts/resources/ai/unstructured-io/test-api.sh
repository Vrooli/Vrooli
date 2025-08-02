#!/usr/bin/env bash

# Unstructured.io API Test Suite (Shell Script)
# This script provides comprehensive testing of the Unstructured.io API functionality

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
BASE_URL="${UNSTRUCTURED_IO_BASE_URL:-http://localhost:11450}"
TIMEOUT=30
PASSED_TESTS=0
TOTAL_TESTS=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

#######################################
# Print test result
#######################################
print_test_result() {
    local test_name="$1"
    local result="$2"
    local details="${3:-}"
    
    ((TOTAL_TESTS++))
    
    if [ "$result" = "PASS" ]; then
        echo -e "  ${GREEN}✓${NC} $test_name"
        ((PASSED_TESTS++))
    else
        echo -e "  ${RED}✗${NC} $test_name"
        [ -n "$details" ] && echo "    $details"
    fi
}

#######################################
# Main test function
#######################################
main() {
    echo -e "${BLUE}🧪 Unstructured.io API Test Suite (Shell Script)${NC}"
    echo "================================================================================"
    echo
    echo "Testing service at ${BASE_URL}..."
    
    # Test 1: Health Check
    echo -e "${YELLOW}=== Test 1: Service Health Check ===${NC}"
    echo
    
    if curl -s -f "${BASE_URL}/healthcheck" >/dev/null 2>&1; then
        print_test_result "Health check endpoint responds" "PASS"
        echo "✅ Service is healthy"
    else
        print_test_result "Health check endpoint responds" "FAIL" "Connection failed"
        echo "❌ Service is not available"
        exit 1
    fi
    echo
    
    # Test 2: Basic Document Processing
    echo -e "${YELLOW}=== Test 2: Basic Document Processing ===${NC}"
    echo
    
    # Create test document
    local test_file="/tmp/unstructured_test_$$.txt"
    cat > "$test_file" << 'EOF'
# Test Document

This is a test document for Unstructured.io processing.

## Features
- Document parsing
- Table extraction  
- OCR capabilities

This demonstrates the API capabilities.
EOF

    # Test processing
    local http_code
    http_code=$(curl -s -w "%{http_code}" -o /dev/null \
        -X POST \
        -F "files=@${test_file}" \
        -F "strategy=fast" \
        -F "languages=eng" \
        --max-time "$TIMEOUT" \
        "${BASE_URL}/general/v0/general" 2>/dev/null)
    
    if [ "$http_code" = "200" ]; then
        print_test_result "Document processing works" "PASS"
        echo "✅ Document processed successfully"
    else
        print_test_result "Document processing works" "FAIL" "HTTP $http_code"
    fi
    
    # Cleanup
    rm -f "$test_file"
    echo
    
    # Test 3: Processing Strategies
    echo -e "${YELLOW}=== Test 3: Processing Strategies ===${NC}"
    echo
    
    local simple_file="/tmp/strategy_test_$$.txt"
    echo "Simple test document for strategy testing." > "$simple_file"
    
    local strategies=("fast" "hi_res" "auto")
    
    for strategy in "${strategies[@]}"; do
        echo "Testing strategy: $strategy"
        
        http_code=$(curl -s -w "%{http_code}" -o /dev/null \
            -X POST \
            -F "files=@${simple_file}" \
            -F "strategy=${strategy}" \
            --max-time "$TIMEOUT" \
            "${BASE_URL}/general/v0/general" 2>/dev/null)
        
        if [ "$http_code" = "200" ]; then
            print_test_result "Strategy '$strategy' works" "PASS"
            echo "  ✓ Processed successfully"
        else
            print_test_result "Strategy '$strategy' works" "FAIL" "HTTP $http_code"
        fi
    done
    
    rm -f "$simple_file"
    echo
    
    # Test 4: Error Handling
    echo -e "${YELLOW}=== Test 4: Error Handling ===${NC}"
    echo
    
    # Test with invalid strategy
    echo "Testing invalid strategy..."
    http_code=$(curl -s -w "%{http_code}" -o /dev/null \
        -X POST \
        -F "files=@/dev/null" \
        -F "strategy=invalid_strategy" \
        "${BASE_URL}/general/v0/general" 2>/dev/null)
    
    if [ "$http_code" = "422" ]; then
        print_test_result "Invalid strategy rejected properly" "PASS"
    else
        print_test_result "Invalid strategy rejected properly" "FAIL" "Expected 422, got $http_code"
    fi
    
    # Test non-existent endpoint
    echo "Testing non-existent endpoint..."
    http_code=$(curl -s -w "%{http_code}" -o /dev/null \
        "${BASE_URL}/nonexistent" 2>/dev/null)
    
    if [ "$http_code" = "404" ]; then
        print_test_result "Non-existent endpoint returns 404" "PASS"
    else
        print_test_result "Non-existent endpoint returns 404" "FAIL" "Expected 404, got $http_code"
    fi
    echo
    
    # Test 5: Integration with manage.sh
    echo -e "${YELLOW}=== Test 5: manage.sh Integration ===${NC}"
    echo
    
    # Test manage.sh test function
    if cd "$SCRIPT_DIR" && ./manage.sh --action test >/dev/null 2>&1; then
        print_test_result "manage.sh test function works" "PASS"
    else
        print_test_result "manage.sh test function works" "FAIL" "Script returned non-zero exit code"
    fi
    
    # Test status check
    if cd "$SCRIPT_DIR" && ./manage.sh --action status >/dev/null 2>&1; then
        print_test_result "manage.sh status check works" "PASS"
    else
        print_test_result "manage.sh status check works" "FAIL" "Script returned non-zero exit code"
    fi
    echo
    
    # Summary
    echo "================================================================================"
    echo -e "${BLUE}Test Summary${NC}"
    echo "================================================================================"
    echo
    
    if [ $PASSED_TESTS -eq $TOTAL_TESTS ]; then
        echo -e "${GREEN}✓ All tests passed!${NC} ($PASSED_TESTS/$TOTAL_TESTS)"
        echo "🎉 Unstructured.io API is fully functional"
    else
        echo -e "${RED}✗ Some tests failed.${NC} ($PASSED_TESTS/$TOTAL_TESTS passed)"
    fi
    
    echo
    echo "================================================================================"
    echo -e "${BLUE}Service Information${NC}"
    echo "================================================================================"
    echo
    echo "Base URL: $BASE_URL"
    echo
    echo "API Endpoints:"
    echo "  - Health Check: GET $BASE_URL/healthcheck"
    echo "  - Process Document: POST $BASE_URL/general/v0/general"
    echo
    echo "Supported Strategies:"
    echo "  - fast: Quick processing for simple documents"
    echo "  - hi_res: High-resolution with advanced features"
    echo "  - auto: Automatically select best strategy"
    echo
    echo "Supported Formats:"
    echo "  - Documents: PDF, DOCX, DOC, TXT, RTF, ODT, MD, RST, HTML, XML, EPUB"
    echo "  - Spreadsheets: XLSX, XLS"
    echo "  - Presentations: PPTX, PPT"
    echo "  - Images (OCR): PNG, JPG, JPEG, TIFF, BMP, HEIC"
    echo "  - Email: EML, MSG"
    echo
    
    # Exit with error code if any tests failed
    if [ $PASSED_TESTS -ne $TOTAL_TESTS ]; then
        exit 1
    fi
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi