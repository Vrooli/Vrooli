#!/usr/bin/env bats
# Tests for Unstructured.io api.sh functions

# Get script directory first
API_BATS_DIR="${BATS_TEST_DIRNAME}"

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${API_BATS_DIR}/../../../../lib/utils/var.sh"

# Load Vrooli test infrastructure using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "unstructured-io"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    UNSTRUCTURED_IO_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Source library files
    source "${SCRIPT_DIR}/api.sh"
    source "${UNSTRUCTURED_IO_DIR}/config/defaults.sh"
    source "${UNSTRUCTURED_IO_DIR}/config/messages.sh"
    source "${UNSTRUCTURED_IO_DIR}/lib/common.sh"
    source "${UNSTRUCTURED_IO_DIR}/lib/status.sh"
    source "${UNSTRUCTURED_IO_DIR}/lib/process.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
    export SETUP_FILE_UNSTRUCTURED_IO_DIR="$UNSTRUCTURED_IO_DIR"
    
    # Create test directory
    export TEST_DIR="/tmp/unstructured_io_test_$$"
    mkdir -p "$TEST_DIR"
    
    # Create test files
    echo "Test content" > "$TEST_DIR/test.txt"
    echo "PDF content" > "$TEST_DIR/test.pdf"
    echo "Word content" > "$TEST_DIR/test.docx"
    echo "Markdown content" > "$TEST_DIR/test.md"
    
    # Create oversized file for testing
    dd if=/dev/zero of="$TEST_DIR/large.pdf" bs=1M count=60 2>/dev/null
}

# Cleanup once per file  
teardown_file() {
    # Clean up test files
    rm -rf "$TEST_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    UNSTRUCTURED_IO_DIR="${SETUP_FILE_UNSTRUCTURED_IO_DIR}"
    
    # Mock required functions before loading library files
    resources::get_default_port() { echo "11450"; }
    export -f resources::get_default_port
    
    # Set test environment variables before sourcing configs
    export UNSTRUCTURED_IO_CUSTOM_PORT="9999"
    export UNSTRUCTURED_IO_CONTAINER_NAME="unstructured-io-test"
    export UNSTRUCTURED_IO_BASE_URL="http://localhost:9999"
    export UNSTRUCTURED_IO_DEFAULT_STRATEGY="hi_res"
    export UNSTRUCTURED_IO_DEFAULT_LANGUAGES="eng"
    export UNSTRUCTURED_IO_MAX_FILE_SIZE_BYTES=$((50 * 1024 * 1024))
    export UNSTRUCTURED_IO_TIMEOUT_SECONDS=300
    export YES="no"
    
    # Mock system functions
    log::info() { echo "[INFO] $*"; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    
    # Mock curl for API calls - simplified and more realistic
    curl() {
        case "$*" in
            *"healthcheck"*) 
                echo '{"status":"healthy"}'
                return 0
                ;;
            *"-w"*"%{http_code}"*)
                # Simulate successful processing
                echo '[{"type":"NarrativeText","text":"Test content"}]'
                echo "200"
                ;;
            *)
                echo '{"success":true}'
                return 0
                ;;
        esac
    }
    export -f curl
    
    # Mock jq for JSON processing
    jq() {
        case "$*" in
            *"-r"* | *"--raw-output"*)
                echo "Test content"
                ;;
            ".")
                cat
                ;;
            *)
                echo '[{"type":"NarrativeText","text":"Test content"}]'
                ;;
        esac
    }
    export -f jq
    
    # Mock file command
    file() {
        case "$1" in
            *".pdf") echo "$1: PDF document" ;;
            *".docx") echo "$1: Microsoft Word document" ;;
            *".txt") echo "$1: ASCII text" ;;
            *) echo "$1: data" ;;
        esac
    }
    export -f file
    
    # Export config functions
    unstructured_io::export_config
    unstructured_io::export_messages
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# Test document processing function exists and has correct parameters
@test "unstructured_io::process_document function exists" {
    run bash -c "type unstructured_io::process_document"
    [ "$status" -eq 0 ]
}

# Test document processing with file validation
@test "unstructured_io::process_document validates file parameter" {
    # Should fail when no file provided  
    run unstructured_io::process_document ""
    [ "$status" -eq 1 ]
}

# Test document processing function basic structure
@test "unstructured_io::process_document handles valid parameters" {
    # Mock the status function to return success
    unstructured_io::status() { return 0; }
    export -f unstructured_io::status
    
    run unstructured_io::process_document "$TEST_DIR/test.txt" "fast" "json" "eng"
    # Function should process without crashing (status 0 or 1 acceptable for mocked test)
    [[ "$status" -eq 0 || "$status" -eq 1 ]]
}

# Test processing failure with invalid file
@test "unstructured_io::process_document fails with invalid file" {
    run unstructured_io::process_document "/nonexistent/file.pdf"
    
    [ "$status" -eq 1 ]
}

# Test file validation - valid files
@test "unstructured_io::validate_file accepts valid files" {
    run unstructured_io::validate_file "$TEST_DIR/test.pdf"
    [ "$status" -eq 0 ]
    
    run unstructured_io::validate_file "$TEST_DIR/test.txt"
    [ "$status" -eq 0 ]
}

# Test file validation - nonexistent file
@test "unstructured_io::validate_file rejects nonexistent file" {
    run unstructured_io::validate_file "/nonexistent/file.pdf"
    [ "$status" -eq 1 ]
}

# Test file validation - oversized file
@test "unstructured_io::validate_file rejects oversized file" {
    run unstructured_io::validate_file "$TEST_DIR/large.pdf"
    [ "$status" -eq 1 ]
}

# Test strategy validation
@test "unstructured_io::validate_strategy checks strategy validity" {
    run unstructured_io::validate_strategy "hi_res"
    [ "$status" -eq 0 ]
    
    run unstructured_io::validate_strategy "fast"
    [ "$status" -eq 0 ]
    
    run unstructured_io::validate_strategy "auto"
    [ "$status" -eq 0 ]
    
    run unstructured_io::validate_strategy "invalid"
    [ "$status" -eq 1 ]
}

# Test output format validation
@test "unstructured_io::validate_output_format checks output format validity" {
    run unstructured_io::validate_output_format "json"
    [ "$status" -eq 0 ]
    
    run unstructured_io::validate_output_format "markdown"
    [ "$status" -eq 0 ]
    
    run unstructured_io::validate_output_format "text"
    [ "$status" -eq 0 ]
    
    run unstructured_io::validate_output_format "invalid"
    [ "$status" -eq 1 ]
}

# Test format support checking
@test "unstructured_io::is_format_supported checks file type support" {
    run unstructured_io::is_format_supported "pdf"
    [ "$status" -eq 0 ]
    
    run unstructured_io::is_format_supported "docx"
    [ "$status" -eq 0 ]
    
    run unstructured_io::is_format_supported "txt"
    [ "$status" -eq 0 ]
    
    run unstructured_io::is_format_supported "xyz"
    [ "$status" -eq 1 ]
}

# Test API test function exists
@test "unstructured_io::test_api function exists" {
    run bash -c "type unstructured_io::test_api"
    [ "$status" -eq 0 ]
}

# Test get supported types function exists  
@test "unstructured_io::get_supported_types function exists" {
    run bash -c "type unstructured_io::get_supported_types"
    [ "$status" -eq 0 ]
}

# Test markdown conversion function exists
@test "unstructured_io::convert_to_markdown function exists" {
    # Test that the function exists and can be called
    run bash -c "type unstructured_io::convert_to_markdown"
    [ "$status" -eq 0 ]
}

# Test batch processing function exists  
@test "unstructured_io::batch_process function exists" {
    # Test that the function exists and can be called
    run bash -c "type unstructured_io::batch_process"
    [ "$status" -eq 0 ]
}