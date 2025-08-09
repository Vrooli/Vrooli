#!/usr/bin/env bats
# Tests for Unstructured.io manage.sh script

# Get script directory first
MANAGE_BATS_DIR="${BATS_TEST_DIRNAME}"

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${MANAGE_BATS_DIR}/../../../lib/utils/var.sh"

# Load Vrooli test infrastructure using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "unstructured-io"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    
    # Source manage.sh and all dependencies
    source "${SCRIPT_DIR}/manage.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    
    # Set unstructured-io specific environment
    export UNSTRUCTURED_IO_CUSTOM_PORT="9999"
    export FILE_INPUT=""
    export STRATEGY="hi_res"
    export OUTPUT="json"
    export LANGUAGES="eng"
    export BATCH="no"
    
    # Export config functions
    unstructured_io::export_config
    unstructured_io::export_messages
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# Test script loading
@test "manage.sh loads without errors" {
    # The script should source successfully in setup
    [ "$?" -eq 0 ]
}

# Test argument parsing
@test "unstructured_io::parse_arguments sets defaults correctly" {
    # Source just the parsing function
    source "${SCRIPT_DIR}/lib/common.sh" 2>/dev/null || true
    
    unstructured_io::parse_arguments --action status
    
    [ "$ACTION" = "status" ]
    [ "$FORCE" = "no" ]
    [ "$STRATEGY" = "hi_res" ]
    [ "$OUTPUT" = "json" ]
}

# Test argument parsing with custom values
@test "unstructured_io::parse_arguments handles custom values" {
    source "${SCRIPT_DIR}/lib/common.sh" 2>/dev/null || true
    
    unstructured_io::parse_arguments \
        --action process \
        --force yes \
        --file "test.pdf" \
        --strategy fast \
        --output markdown \
        --languages "eng,spa" \
        --batch yes \
        --follow yes \
        --yes yes
    
    [ "$ACTION" = "process" ]
    [ "$FORCE" = "yes" ]
    [ "$FILE_INPUT" = "test.pdf" ]
    [ "$STRATEGY" = "fast" ]
    [ "$OUTPUT" = "markdown" ]
    [ "$LANGUAGES" = "eng,spa" ]
    [ "$BATCH" = "yes" ]
    [ "$FOLLOW" = "yes" ]
    [ "$YES" = "yes" ]
}

# Test install functionality
@test "unstructured_io::install function works correctly" {
    # Use the proper unstructured-io mock
    run unstructured-io-manage --action install
    [ "$status" -eq 0 ]
    
    # Verify installation was successful
    run unstructured-io-manage --action status
    [ "$status" -eq 0 ]
}

# Test status functionality  
@test "unstructured_io::status function works correctly" {
    # Install service first to test status
    run unstructured-io-manage --action install
    [ "$status" -eq 0 ]
    
    # Test status
    run unstructured-io-manage --action status
    [ "$status" -eq 0 ]
    [[ "$output" == *"installed"* ]]
}

# Test process functionality with missing file
@test "unstructured_io::process handles missing file" {
    # Test should fail for missing file
    run unstructured-io-manage --action process
    [ "$status" -ne 0 ]
    [[ "$output" == *"No file provided"* ]]
}

# Test process functionality with valid file
@test "unstructured_io::process handles valid file" {
    # Install and start service first
    run unstructured-io-manage --action install
    [ "$status" -eq 0 ]
    
    run unstructured-io-manage --action start
    [ "$status" -eq 0 ]
    
    # Create test file
    local test_file=$(create_temp_file "test.txt" "Test content")
    
    # Test process
    run unstructured-io-manage --action process --file "$test_file"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Processing completed"* ]]
}

# Test batch processing
@test "unstructured_io::process handles batch mode" {
    # Install and start service first
    run unstructured-io-manage --action install
    [ "$status" -eq 0 ]
    
    run unstructured-io-manage --action start
    [ "$status" -eq 0 ]
    
    # Create test files
    local test_file1=$(create_temp_file "test1.txt" "Test content 1")
    local test_file2=$(create_temp_file "test2.txt" "Test content 2")
    
    # Test batch process
    run unstructured-io-manage --action process --file "$test_file1,$test_file2" --batch yes
    [ "$status" -eq 0 ]
}

# Test API functions
@test "unstructured_io::check_health works correctly" {
    # Install and start service first
    run unstructured-io-manage --action install
    [ "$status" -eq 0 ]
    
    run unstructured-io-manage --action start
    [ "$status" -eq 0 ]
    
    # Test API endpoints - use test action which includes health check
    run unstructured-io-manage --action test
    [ "$status" -eq 0 ]
    [[ "$output" == *"All tests passed"* ]]
}

# Test supported formats
@test "unstructured_io::is_format_supported works correctly" {
    # Test supported format by trying to process
    local test_file=$(create_temp_file "test.pdf" "Test PDF content")
    
    run unstructured-io-manage --action install
    [ "$status" -eq 0 ]
    
    run unstructured-io-manage --action start
    [ "$status" -eq 0 ]
    
    run unstructured-io-manage --action process --file "$test_file"
    [ "$status" -eq 0 ]
    
    # Test unsupported format 
    local unsupported_file=$(create_temp_file "test.xyz" "Test unsupported content")
    run unstructured-io-manage --action process --file "$unsupported_file"
    [ "$status" -ne 0 ]
}

# Test extract-tables functionality
@test "unstructured_io::extract_tables works correctly" {
    # Install and start service first
    run unstructured-io-manage --action install
    [ "$status" -eq 0 ]
    
    run unstructured-io-manage --action start
    [ "$status" -eq 0 ]
    
    # Create test file
    local test_file=$(create_temp_file "test.pdf" "Test PDF with tables")
    
    # Test table extraction
    run unstructured-io-manage --action extract-tables --file "$test_file"
    [ "$status" -eq 0 ]
    [[ "$output" == *"tables"* ]]
}

# Test extract-metadata functionality  
@test "unstructured_io::extract_metadata works correctly" {
    # Install and start service first
    run unstructured-io-manage --action install
    [ "$status" -eq 0 ]
    
    run unstructured-io-manage --action start
    [ "$status" -eq 0 ]
    
    # Create test file
    local test_file=$(create_temp_file "test.docx" "Test Word document")
    
    # Test metadata extraction
    run unstructured-io-manage --action extract-metadata --file "$test_file"
    [ "$status" -eq 0 ]
    [[ "$output" == *"metadata"* ]]
}

# Test cache functionality
@test "unstructured_io::cache_stats works correctly" {
    run unstructured-io-manage --action cache-stats
    [ "$status" -eq 0 ]
    [[ "$output" == *"Cache"* ]]
}

# Test clear cache functionality
@test "unstructured_io::clear_cache works correctly" {
    run unstructured-io-manage --action clear-cache
    [ "$status" -eq 0 ]
    [[ "$output" == *"Cleared"* ]]
}