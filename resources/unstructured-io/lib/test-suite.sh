#!/usr/bin/env bash
# Comprehensive test suite for Unstructured.io resource
# Tests all major functionality and edge cases

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../.." && builtin pwd)}"
SCRIPT_DIR="$APP_ROOT/resources/unstructured-io/lib"
MANAGE_SCRIPT="$APP_ROOT/resources/unstructured-io/manage.sh"
TEST_FIXTURES="$SCRIPT_DIR/test-fixtures"
RESULTS_DIR="$SCRIPT_DIR/test-results"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Create results directory
mkdir -p "$RESULTS_DIR"

#######################################
# Test utilities
#######################################
test_start() {
    local test_name="$1"
    echo -e "\n${YELLOW}TEST:${NC} $test_name"
    ((TESTS_RUN++))
}

test_pass() {
    echo -e "${GREEN}✓ PASSED${NC}"
    ((TESTS_PASSED++))
}

test_fail() {
    local reason="${1:-Unknown reason}"
    echo -e "${RED}✗ FAILED${NC}: $reason"
    ((TESTS_FAILED++))
}

#######################################
# Service tests
#######################################
test_service_health() {
    test_start "Service health check"
    
    if "$MANAGE_SCRIPT" --action status --quiet yes; then
        test_pass
    else
        test_fail "Service is not healthy"
        echo "Starting service..."
        "$MANAGE_SCRIPT" --action start
        sleep 5
    fi
}

#######################################
# Basic processing tests
#######################################
test_text_processing() {
    test_start "Basic text file processing"
    
    echo "Test content" > "$RESULTS_DIR/test.txt"
    
    if "$MANAGE_SCRIPT" --action process --file "$RESULTS_DIR/test.txt" --output text --quiet yes > "$RESULTS_DIR/text_output.txt"; then
        if grep -q "Test content" "$RESULTS_DIR/text_output.txt"; then
            test_pass
        else
            test_fail "Output doesn't contain expected content"
        fi
    else
        test_fail "Processing failed"
    fi
}

test_markdown_processing() {
    test_start "Markdown file processing"
    
    cat > "$RESULTS_DIR/test.md" << 'EOF'
# Test Document
## Section 1
This is a **test** document.

### Code Example
```python
def hello():
    print("Hello, World!")
```
EOF
    
    if "$MANAGE_SCRIPT" --action process --file "$RESULTS_DIR/test.md" --output markdown --quiet yes > "$RESULTS_DIR/md_output.txt"; then
        if grep -q "Test Document" "$RESULTS_DIR/md_output.txt" && grep -q "hello()" "$RESULTS_DIR/md_output.txt"; then
            test_pass
        else
            test_fail "Markdown structure not preserved"
        fi
    else
        test_fail "Processing failed"
    fi
}

test_html_processing() {
    test_start "HTML file processing"
    
    cat > "$RESULTS_DIR/test.html" << 'EOF'
<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
<h1>Test Heading</h1>
<p>Test paragraph</p>
<table>
<tr><th>Header 1</th><th>Header 2</th></tr>
<tr><td>Data 1</td><td>Data 2</td></tr>
</table>
</body>
</html>
EOF
    
    if "$MANAGE_SCRIPT" --action process --file "$RESULTS_DIR/test.html" --output markdown --quiet yes > "$RESULTS_DIR/html_output.txt"; then
        if grep -q "Test Heading" "$RESULTS_DIR/html_output.txt" && grep -q "Header 1" "$RESULTS_DIR/html_output.txt"; then
            test_pass
        else
            test_fail "HTML not properly converted"
        fi
    else
        test_fail "Processing failed"
    fi
}

#######################################
# PDF tests
#######################################
test_pdf_processing() {
    test_start "PDF file processing"
    
    local pdf_file="$TEST_FIXTURES/pdfs/simple_text.pdf"
    
    if [[ -f "$pdf_file" ]]; then
        if "$MANAGE_SCRIPT" --action process --file "$pdf_file" --output text --quiet yes > "$RESULTS_DIR/pdf_output.txt"; then
            if grep -q "Simple Text Document" "$RESULTS_DIR/pdf_output.txt"; then
                test_pass
            else
                test_fail "PDF content not extracted correctly"
            fi
        else
            test_fail "PDF processing failed"
        fi
    else
        test_fail "Test PDF not found"
    fi
}

test_pdf_with_tables() {
    test_start "PDF with tables processing"
    
    local pdf_file="$TEST_FIXTURES/pdfs/table_document.pdf"
    
    if [[ -f "$pdf_file" ]]; then
        if "$MANAGE_SCRIPT" --action extract-tables --file "$pdf_file" > "$RESULTS_DIR/table_output.txt" 2>&1; then
            if grep -q "table" "$RESULTS_DIR/table_output.txt"; then
                test_pass
            else
                test_fail "Tables not extracted from PDF"
            fi
        else
            test_fail "Table extraction failed"
        fi
    else
        test_fail "Table PDF not found"
    fi
}

test_multipage_pdf() {
    test_start "Multi-page PDF processing"
    
    local pdf_file="$TEST_FIXTURES/pdfs/multipage.pdf"
    
    if [[ -f "$pdf_file" ]]; then
        if "$MANAGE_SCRIPT" --action process --file "$pdf_file" --output json --quiet yes > "$RESULTS_DIR/multipage_output.json"; then
            local page_count=$(jq '[.[] | select(.metadata.page_number != null) | .metadata.page_number] | max' "$RESULTS_DIR/multipage_output.json" 2>/dev/null)
            if [[ "$page_count" -ge 3 ]]; then
                test_pass
            else
                test_fail "Not all pages processed (found $page_count pages)"
            fi
        else
            test_fail "Multi-page PDF processing failed"
        fi
    else
        test_fail "Multi-page PDF not found"
    fi
}

#######################################
# Strategy tests
#######################################
test_fast_strategy() {
    test_start "Fast processing strategy"
    
    local pdf_file="$TEST_FIXTURES/pdfs/simple_text.pdf"
    
    if [[ -f "$pdf_file" ]]; then
        local start_time=$(date +%s)
        if "$MANAGE_SCRIPT" --action process --file "$pdf_file" --strategy fast --output text --quiet yes > /dev/null; then
            local end_time=$(date +%s)
            local duration=$((end_time - start_time))
            if [[ $duration -lt 10 ]]; then
                test_pass
            else
                test_fail "Fast strategy took too long: ${duration}s"
            fi
        else
            test_fail "Fast strategy processing failed"
        fi
    else
        test_fail "Test file not found"
    fi
}

test_hi_res_strategy() {
    test_start "Hi-res processing strategy"
    
    local pdf_file="$TEST_FIXTURES/pdfs/table_document.pdf"
    
    if [[ -f "$pdf_file" ]]; then
        if "$MANAGE_SCRIPT" --action process --file "$pdf_file" --strategy hi_res --output json --quiet yes > "$RESULTS_DIR/hires_output.json"; then
            # Check if tables are properly detected
            if jq -e '.[] | select(.type == "Table")' "$RESULTS_DIR/hires_output.json" > /dev/null 2>&1; then
                test_pass
            else
                test_fail "Hi-res strategy didn't detect tables"
            fi
        else
            test_fail "Hi-res strategy processing failed"
        fi
    else
        test_fail "Test file not found"
    fi
}

#######################################
# Cache tests
#######################################
test_cache_functionality() {
    test_start "Cache functionality"
    
    # Clear cache first
    "$MANAGE_SCRIPT" --action clear-cache > /dev/null 2>&1
    
    # Process file first time
    echo "Cache test content" > "$RESULTS_DIR/cache_test.txt"
    local start1=$(date +%s%N)
    "$MANAGE_SCRIPT" --action process --file "$RESULTS_DIR/cache_test.txt" --output text --quiet yes > /dev/null
    local end1=$(date +%s%N)
    local duration1=$(( (end1 - start1) / 1000000 )) # Convert to milliseconds
    
    # Process same file again (should use cache)
    local start2=$(date +%s%N)
    local output=$("$MANAGE_SCRIPT" --action process --file "$RESULTS_DIR/cache_test.txt" --output text 2>&1)
    local end2=$(date +%s%N)
    local duration2=$(( (end2 - start2) / 1000000 ))
    
    if echo "$output" | grep -q "Using cached result"; then
        if [[ $duration2 -lt $((duration1 / 2)) ]]; then
            test_pass
        else
            test_fail "Cached processing not significantly faster"
        fi
    else
        test_fail "Cache not being used"
    fi
}

#######################################
# Error handling tests
#######################################
test_unsupported_format() {
    test_start "Unsupported format error handling"
    
    echo "Test" > "$RESULTS_DIR/test.xyz"
    
    if ! "$MANAGE_SCRIPT" --action process --file "$RESULTS_DIR/test.xyz" --output text 2>&1 | grep -q "ERROR.*UNSUPPORTED.*xyz"; then
        test_fail "Unsupported format error not properly reported"
    else
        test_pass
    fi
}

test_invalid_strategy() {
    test_start "Invalid strategy error handling"
    
    echo "Test" > "$RESULTS_DIR/test.txt"
    
    if ! "$MANAGE_SCRIPT" --action process --file "$RESULTS_DIR/test.txt" --strategy invalid --output text 2>&1 | grep -q "ERROR.*INVALID_PARAMS"; then
        test_fail "Invalid strategy error not properly reported"
    else
        test_pass
    fi
}

test_nonexistent_file() {
    test_start "Non-existent file error handling"
    
    if ! "$MANAGE_SCRIPT" --action process --file "/tmp/nonexistent_file_12345.pdf" --output text 2>&1 | grep -q "File not found"; then
        test_fail "Non-existent file error not properly reported"
    else
        test_pass
    fi
}

#######################################
# Batch processing tests
#######################################
test_batch_processing() {
    test_start "Batch processing"
    
    # Create multiple test files
    echo "File 1" > "$RESULTS_DIR/batch1.txt"
    echo "File 2" > "$RESULTS_DIR/batch2.txt"
    echo "File 3" > "$RESULTS_DIR/batch3.txt"
    
    if "$MANAGE_SCRIPT" --action process --file "$RESULTS_DIR/batch1.txt,$RESULTS_DIR/batch2.txt,$RESULTS_DIR/batch3.txt" --batch yes --output text --quiet yes > "$RESULTS_DIR/batch_output.txt"; then
        if grep -q "File 1" "$RESULTS_DIR/batch_output.txt" && \
           grep -q "File 2" "$RESULTS_DIR/batch_output.txt" && \
           grep -q "File 3" "$RESULTS_DIR/batch_output.txt"; then
            test_pass
        else
            test_fail "Not all files processed in batch"
        fi
    else
        test_fail "Batch processing failed"
    fi
}

#######################################
# Output format tests
#######################################
test_json_output() {
    test_start "JSON output format"
    
    echo "JSON test" > "$RESULTS_DIR/json_test.txt"
    
    if "$MANAGE_SCRIPT" --action process --file "$RESULTS_DIR/json_test.txt" --output json --quiet yes > "$RESULTS_DIR/json_output.json"; then
        if jq -e '.[0].text' "$RESULTS_DIR/json_output.json" > /dev/null 2>&1; then
            test_pass
        else
            test_fail "Invalid JSON output"
        fi
    else
        test_fail "JSON processing failed"
    fi
}

test_elements_output() {
    test_start "Elements output format"
    
    echo "Elements test" > "$RESULTS_DIR/elements_test.txt"
    
    if "$MANAGE_SCRIPT" --action process --file "$RESULTS_DIR/elements_test.txt" --output elements --quiet yes > "$RESULTS_DIR/elements_output.txt"; then
        if grep -q "\[.*\]" "$RESULTS_DIR/elements_output.txt"; then
            test_pass
        else
            test_fail "Elements format not correct"
        fi
    else
        test_fail "Elements processing failed"
    fi
}

#######################################
# Performance tests
#######################################
test_large_file_handling() {
    test_start "Large file handling"
    
    # Use the pre-created large PDF
    local large_pdf="$TEST_FIXTURES/pdfs/large_document.pdf"
    
    if [[ -f "$large_pdf" ]]; then
        if timeout 60 "$MANAGE_SCRIPT" --action process --file "$large_pdf" --strategy fast --output text --quiet yes > /dev/null; then
            test_pass
        else
            test_fail "Large file processing timed out or failed"
        fi
    else
        test_fail "Large test file not found"
    fi
}

#######################################
# Integration tests
#######################################
test_doc_qa_integration() {
    test_start "Document Q&A integration"
    
    if command -v ollama &> /dev/null; then
        if "$TEST_FIXTURES/../integrations/doc-qa.sh" "$TEST_FIXTURES/pdfs/simple_text.pdf" "What is this document about?" > "$RESULTS_DIR/qa_output.txt" 2>&1; then
            if grep -qi "simple\|text" "$RESULTS_DIR/qa_output.txt"; then
                test_pass
            else
                test_fail "Q&A didn't produce relevant answer"
            fi
        else
            test_fail "Document Q&A integration failed"
        fi
    else
        test_fail "Ollama not available - skipping"
    fi
}

#######################################
# Run all tests
#######################################
main() {
    echo "========================================="
    echo "Unstructured.io Comprehensive Test Suite"
    echo "========================================="
    
    # Service tests
    test_service_health
    
    # Basic processing tests
    test_text_processing
    test_markdown_processing
    test_html_processing
    
    # PDF tests
    test_pdf_processing
    test_pdf_with_tables
    test_multipage_pdf
    
    # Strategy tests
    test_fast_strategy
    test_hi_res_strategy
    
    # Cache tests
    test_cache_functionality
    
    # Error handling tests
    test_unsupported_format
    test_invalid_strategy
    test_nonexistent_file
    
    # Batch processing tests
    test_batch_processing
    
    # Output format tests
    test_json_output
    test_elements_output
    
    # Performance tests
    test_large_file_handling
    
    # Integration tests
    test_doc_qa_integration
    
    # Summary
    echo
    echo "========================================="
    echo "Test Results Summary"
    echo "========================================="
    echo -e "Total tests run: $TESTS_RUN"
    echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
    echo -e "${RED}Failed: $TESTS_FAILED${NC}"
    echo
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}✅ All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}❌ Some tests failed${NC}"
        exit 1
    fi
}

# Run tests
main "$@"