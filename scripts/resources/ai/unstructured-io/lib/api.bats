#!/usr/bin/env bats
# Tests for Unstructured.io api.sh functions

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    setup_standard_mocks
    
    # Set test environment
    export UNSTRUCTURED_IO_CUSTOM_PORT="9999"
    export UNSTRUCTURED_IO_CONTAINER_NAME="unstructured-io-test"
    export UNSTRUCTURED_IO_BASE_URL="http://localhost:9999"
    export UNSTRUCTURED_IO_DEFAULT_STRATEGY="hi_res"
    export UNSTRUCTURED_IO_DEFAULT_LANGUAGES="eng"
    export UNSTRUCTURED_IO_MAX_FILE_SIZE_BYTES=$((50 * 1024 * 1024))
    export UNSTRUCTURED_IO_TIMEOUT_SECONDS=300
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    UNSTRUCTURED_IO_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test files
    export TEST_DIR="/tmp/unstructured_io_test_$$"
    mkdir -p "$TEST_DIR"
    
    # Create test files with different types
    echo "Test content" > "$TEST_DIR/test.txt"
    echo "PDF content" > "$TEST_DIR/test.pdf"
    echo "Word content" > "$TEST_DIR/test.docx"
    echo "Markdown content" > "$TEST_DIR/test.md"
    
    # Create oversized file for testing
    dd if=/dev/zero of="$TEST_DIR/large.pdf" bs=1M count=60 2>/dev/null
    
    # Mock system functions
    
    # Mock file command
    file() {
        case "$1" in
            *".pdf")
                echo "$1: PDF document"
                ;;
            *".docx")
                echo "$1: Microsoft Word document"
                ;;
            *".txt")
                echo "$1: ASCII text"
                ;;
            *".md")
                echo "$1: ASCII text"
                ;;
            *)
                echo "$1: data"
                ;;
        esac
    }
    
    # Mock curl for API calls
    
    # Mock jq for JSON processing
    jq() {
        case "$*" in
            *".elements"*)
                echo '[{"type":"text","text":"Test content"}]'
                ;;
            *".text"*)
                echo "Test content"
                ;;
            *"--raw-output"* | *"-r"*)
                echo "Test content"
                ;;
            *) echo "{}" ;;
        esac
    }
    
    # Mock log functions
    
    
    
    
    # Mock status function
    unstructured_io::status() {
        return 0  # Service available
    }
    
    # Load configuration and messages
    source "${UNSTRUCTURED_IO_DIR}/config/defaults.sh"
    source "${UNSTRUCTURED_IO_DIR}/config/messages.sh"
    unstructured_io::export_config
    unstructured_io::export_messages
    
    # Load the functions to test
    source "${UNSTRUCTURED_IO_DIR}/lib/api.sh"
}

# Cleanup after each test
teardown() {
    rm -rf "$TEST_DIR"
}

# Test document processing - PDF file
@test "unstructured_io::process_document processes PDF file successfully" {
    result=$(unstructured_io::process_document "$TEST_DIR/test.pdf" "hi_res" "json" "eng")
    
    [[ "$result" =~ "Processing" ]]
    [[ "$result" =~ "test.pdf" ]]
    [[ "$result" =~ "Test content from PDF" ]]
}

# Test document processing - Word document
@test "unstructured_io::process_document processes Word document successfully" {
    result=$(unstructured_io::process_document "$TEST_DIR/test.docx" "fast" "json" "eng")
    
    [[ "$result" =~ "Processing" ]]
    [[ "$result" =~ "test.docx" ]]
    [[ "$result" =~ "Test content from Word document" ]]
}

# Test document processing - text file
@test "unstructured_io::process_document processes text file successfully" {
    result=$(unstructured_io::process_document "$TEST_DIR/test.txt" "auto" "text" "eng")
    
    [[ "$result" =~ "Processing" ]]
    [[ "$result" =~ "test.txt" ]]
    [[ "$result" =~ "Test content" ]]
}

# Test document processing with default parameters
@test "unstructured_io::process_document uses default parameters" {
    result=$(unstructured_io::process_document "$TEST_DIR/test.pdf")
    
    [[ "$result" =~ "Processing" ]]
    [[ "$result" =~ "test.pdf" ]]
}

# Test document processing with service unavailable
@test "unstructured_io::process_document fails when service unavailable" {
    # Override status to return unavailable
    unstructured_io::status() {
        return 1
    }
    
    run unstructured_io::process_document "$TEST_DIR/test.pdf"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "service is not available" ]]
}

# Test document processing with invalid file
@test "unstructured_io::process_document fails with invalid file" {
    run unstructured_io::process_document "/nonexistent/file.pdf"
    [ "$status" -eq 1 ]
}

# Test file validation - valid file
@test "unstructured_io::validate_file accepts valid files" {
    unstructured_io::validate_file "$TEST_DIR/test.pdf"
    [ "$?" -eq 0 ]
    
    unstructured_io::validate_file "$TEST_DIR/test.docx"
    [ "$?" -eq 0 ]
    
    unstructured_io::validate_file "$TEST_DIR/test.txt"
    [ "$?" -eq 0 ]
}

# Test file validation - nonexistent file
@test "unstructured_io::validate_file rejects nonexistent file" {
    run unstructured_io::validate_file "/nonexistent/file.pdf"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "does not exist" ]]
}

# Test file validation - oversized file
@test "unstructured_io::validate_file rejects oversized file" {
    run unstructured_io::validate_file "$TEST_DIR/large.pdf"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "too large" ]]
}

# Test file validation - unsupported type
@test "unstructured_io::validate_file rejects unsupported file type" {
    echo "binary" > "$TEST_DIR/test.bin"
    
    run unstructured_io::validate_file "$TEST_DIR/test.bin"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not supported" ]]
}

# Test file type detection
@test "unstructured_io::get_file_type detects file types correctly" {
    result=$(unstructured_io::get_file_type "$TEST_DIR/test.pdf")
    [[ "$result" =~ "pdf" ]]
    
    result=$(unstructured_io::get_file_type "$TEST_DIR/test.docx")
    [[ "$result" =~ "docx" ]]
    
    result=$(unstructured_io::get_file_type "$TEST_DIR/test.txt")
    [[ "$result" =~ "txt" ]]
}

# Test file size check
@test "unstructured_io::check_file_size validates file size correctly" {
    # Small file should pass
    unstructured_io::check_file_size "$TEST_DIR/test.txt"
    [ "$?" -eq 0 ]
    
    # Large file should fail
    run unstructured_io::check_file_size "$TEST_DIR/large.pdf"
    [ "$status" -eq 1 ]
}

# Test output format conversion - JSON to text
@test "unstructured_io::convert_output converts JSON to text format" {
    local json_input='{"elements":[{"type":"text","text":"Hello world"},{"type":"text","text":"Second paragraph"}]}'
    
    result=$(echo "$json_input" | unstructured_io::convert_output "text")
    
    [[ "$result" =~ "Hello world" ]]
    [[ "$result" =~ "Second paragraph" ]]
}

# Test output format conversion - JSON to markdown
@test "unstructured_io::convert_output converts JSON to markdown format" {
    local json_input='{"elements":[{"type":"title","text":"Heading"},{"type":"text","text":"Paragraph"}]}'
    
    result=$(echo "$json_input" | unstructured_io::convert_output "markdown")
    
    [[ "$result" =~ "Heading" ]]
    [[ "$result" =~ "Paragraph" ]]
}

# Test API request preparation
@test "unstructured_io::prepare_api_request builds correct API request" {
    result=$(unstructured_io::prepare_api_request "$TEST_DIR/test.pdf" "hi_res" "json" "eng")
    
    [[ "$result" =~ "files=@" ]]
    [[ "$result" =~ "test.pdf" ]]
    [[ "$result" =~ "strategy=hi_res" ]]
    [[ "$result" =~ "output_format=json" ]]
    [[ "$result" =~ "languages=eng" ]]
}

# Test API response processing
@test "unstructured_io::process_api_response handles API response correctly" {
    local response='{"elements":[{"type":"text","text":"Processed content"}]}'
    
    result=$(echo "$response" | unstructured_io::process_api_response "json")
    
    [[ "$result" =~ "Processed content" ]]
}

# Test API response processing with error
@test "unstructured_io::process_api_response handles API error response" {
    local error_response='{"error":"Processing failed","details":"Invalid file format"}'
    
    run echo "$error_response" | unstructured_io::process_api_response "json"
    [ "$status" -eq 1 ]
}

# Test batch processing setup
@test "unstructured_io::setup_batch_processing prepares batch environment" {
    result=$(unstructured_io::setup_batch_processing "$TEST_DIR")
    
    [[ "$result" =~ "batch" ]]
    [ "$?" -eq 0 ]
}

# Test supported file types check
@test "unstructured_io::is_supported_file_type checks file type support" {
    unstructured_io::is_supported_file_type "pdf"
    [ "$?" -eq 0 ]
    
    unstructured_io::is_supported_file_type "docx"
    [ "$?" -eq 0 ]
    
    unstructured_io::is_supported_file_type "txt"
    [ "$?" -eq 0 ]
    
    run unstructured_io::is_supported_file_type "xyz"
    [ "$status" -eq 1 ]
}

# Test processing strategy validation
@test "unstructured_io::validate_strategy checks strategy validity" {
    unstructured_io::validate_strategy "hi_res"
    [ "$?" -eq 0 ]
    
    unstructured_io::validate_strategy "fast"
    [ "$?" -eq 0 ]
    
    unstructured_io::validate_strategy "auto"
    [ "$?" -eq 0 ]
    
    run unstructured_io::validate_strategy "invalid"
    [ "$status" -eq 1 ]
}

# Test language parameter validation
@test "unstructured_io::validate_languages checks language parameter" {
    unstructured_io::validate_languages "eng"
    [ "$?" -eq 0 ]
    
    unstructured_io::validate_languages "eng,spa"
    [ "$?" -eq 0 ]
    
    unstructured_io::validate_languages "eng,spa,fra"
    [ "$?" -eq 0 ]
}

# Test API endpoint availability
@test "unstructured_io::check_api_endpoint verifies API endpoint accessibility" {
    unstructured_io::check_api_endpoint "/general/v0/general"
    [ "$?" -eq 0 ]
}

# Test API endpoint with unavailable service
@test "unstructured_io::check_api_endpoint fails with unavailable endpoint" {
    # Override curl to fail for specific endpoint
    curl() {
        case "$*" in
            *"/unavailable"*)
                return 1
                ;;
            *)
                echo '{"status":"ok"}'
                return 0
                ;;
        esac
    }
    
    run unstructured_io::check_api_endpoint "/unavailable"
    [ "$status" -eq 1 ]
}

# Test processing timeout handling
@test "unstructured_io::process_with_timeout handles processing timeout" {
    # Mock curl to simulate timeout
    curl() {
        case "$*" in
            *"--max-time"*)
                echo "curl: (28) Operation timed out"
                return 28
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    run unstructured_io::process_with_timeout "$TEST_DIR/test.pdf" "hi_res" "json" "eng"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "timeout" ]]
}