#!/usr/bin/env bats
# Unstructured.io Mock Test Suite
#
# Comprehensive tests for the Unstructured.io mock implementation
# Tests all service management commands, document processing, cache management,
# state persistence, and BATS compatibility features

# Source trash module for safe test cleanup
MOCK_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${MOCK_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Test environment setup
setup() {
    # Set up test directory
    export TEST_DIR="$BATS_TEST_TMPDIR/unstructured-io-mock-test"
    mkdir -p "$TEST_DIR"
    cd "$TEST_DIR"
    
    # Set up mock directory
    export MOCK_DIR="${BATS_TEST_DIRNAME}"
    
    # Configure Unstructured.io mock state directory
    export UNSTRUCTURED_IO_MOCK_STATE_DIR="$TEST_DIR/unstructured-io-state"
    mkdir -p "$UNSTRUCTURED_IO_MOCK_STATE_DIR"
    
    # Source the Unstructured.io mock
    source "$MOCK_DIR/unstructured-io.sh"
    
    # Reset Unstructured.io mock to clean state
    mock::unstructured_io::reset
}

teardown() {
    # Clean up test directory
    trash::safe_remove "$TEST_DIR" --test-cleanup
}

# Helper functions for assertions (same as redis.bats for consistency)
assert_success() {
    if [[ "$status" -ne 0 ]]; then
        echo "Expected success but got status $status" >&2
        echo "Output: $output" >&2
        return 1
    fi
}

assert_failure() {
    if [[ "$status" -eq 0 ]]; then
        echo "Expected failure but got success" >&2
        echo "Output: $output" >&2
        return 1
    fi
}

assert_output() {
    local expected="$1"
    if [[ "$1" == "--partial" ]]; then
        expected="$2"
        if [[ ! "$output" =~ "$expected" ]]; then
            echo "Expected output to contain: $expected" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    elif [[ "$1" == "--regexp" ]]; then
        expected="$2"
        if [[ ! "$output" =~ $expected ]]; then
            echo "Expected output to match: $expected" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    else
        if [[ "$output" != "$expected" ]]; then
            echo "Expected: $expected" >&2
            echo "Actual: $output" >&2
            return 1
        fi
    fi
}

refute_output() {
    local pattern="$2"
    if [[ "$1" == "--partial" ]] || [[ "$1" == "--regexp" ]]; then
        if [[ "$output" =~ "$pattern" ]]; then
            echo "Expected output NOT to contain: $pattern" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    fi
}

assert_line() {
    local index expected
    if [[ "$1" == "--index" ]]; then
        index="$2"
        expected="$3"
        local lines=()
        while IFS= read -r line; do
            lines+=("$line")
        done <<< "$output"
        
        if [[ "${lines[$index]}" != "$expected" ]]; then
            echo "Line $index mismatch" >&2
            echo "Expected: $expected" >&2
            echo "Actual: ${lines[$index]}" >&2
            return 1
        fi
    fi
}

# Service management tests
@test "unstructured-io-manage: installation when not installed" {
    run unstructured-io-manage --action install
    assert_success
    assert_output --partial "Installing Unstructured.io service"
    assert_output --partial "Installation completed successfully"
}

@test "unstructured-io-manage: installation when already installed" {
    mock::unstructured_io::set_installed "true"
    
    run unstructured-io-manage --action install
    assert_success
    assert_output --partial "already installed"
}

@test "unstructured-io-manage: uninstallation when installed" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    run unstructured-io-manage --action uninstall
    assert_success
    assert_output --partial "Uninstalling Unstructured.io service"
    assert_output --partial "Uninstallation completed"
}

@test "unstructured-io-manage: uninstallation when not installed" {
    run unstructured-io-manage --action uninstall
    assert_success
    assert_output --partial "not installed"
}

@test "unstructured-io-manage: start when not installed" {
    run unstructured-io-manage --action start
    assert_failure
    assert_output --partial "not installed"
}

@test "unstructured-io-manage: start when installed but not running" {
    mock::unstructured_io::set_installed "true"
    
    run unstructured-io-manage --action start
    assert_success
    assert_output --partial "Starting Unstructured.io service"
    assert_output --partial "started successfully"
}

@test "unstructured-io-manage: start when already running" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    run unstructured-io-manage --action start
    assert_success
    assert_output --partial "already running"
}

@test "unstructured-io-manage: stop when running" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    run unstructured-io-manage --action stop
    assert_success
    assert_output --partial "Stopping Unstructured.io service"
    assert_output --partial "stopped successfully"
}

@test "unstructured-io-manage: stop when not running" {
    mock::unstructured_io::set_installed "true"
    
    run unstructured-io-manage --action stop
    assert_success
    assert_output --partial "not running"
}

@test "unstructured-io-manage: restart service" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    run unstructured-io-manage --action restart
    assert_success
    assert_output --partial "Stopping Unstructured.io service"
    assert_output --partial "Starting Unstructured.io service"
}

@test "unstructured-io-manage: status when not installed" {
    run unstructured-io-manage --action status
    assert_failure
    assert_output --partial "Status: Not installed"
}

@test "unstructured-io-manage: status when installed but not running" {
    mock::unstructured_io::set_installed "true"
    
    run unstructured-io-manage --action status
    assert_failure
    assert_output --partial "Status: Installed"
    assert_output --partial "Service: Stopped"
}

@test "unstructured-io-manage: status when installed and running" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    run unstructured-io-manage --action status
    assert_success
    assert_output --partial "Status: Installed"
    assert_output --partial "Service: Running"
    assert_output --partial "API URL"
    assert_output --partial "Version"
}

# Document processing tests
@test "unstructured-io-manage: process document when service not running" {
    mock::unstructured_io::set_installed "true"
    
    run unstructured-io-manage --action process --file "/path/to/document.pdf"
    assert_failure
    assert_output --partial "service is not running"
}

@test "unstructured-io-manage: process document without file argument" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    run unstructured-io-manage --action process
    assert_failure
    assert_output --partial "No file provided for processing"
}

@test "unstructured-io-manage: process document with default settings" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    run unstructured-io-manage --action process --file "/path/to/document.pdf"
    assert_success
    assert_output --partial "Processing document: document.pdf"
    assert_output --partial "Strategy: auto"
    assert_output --partial "Output format: json"
    assert_output --partial "Processing completed"
}

@test "unstructured-io-manage: process document with custom strategy and output" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    run unstructured-io-manage --action process --file "/path/to/document.pdf" --strategy "hi_res" --output "text" --languages "en,es"
    assert_success
    assert_output --partial "Strategy: hi_res"
    assert_output --partial "Output format: text"
    assert_output --regexp "Document Title"
}

@test "unstructured-io-manage: process document uses cache on second call" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    # First processing
    run unstructured-io-manage --action process --file "/path/to/document.pdf"
    assert_success
    refute_output --partial "Using cached result"
    
    # Second processing should use cache
    run unstructured-io-manage --action process --file "/path/to/document.pdf"
    assert_success
    assert_output --partial "Using cached result"
}

# Table extraction tests
@test "unstructured-io-manage: extract tables from document" {
    run unstructured-io-manage --action extract-tables "/path/to/document.pdf"
    assert_success
    assert_output --partial "Extracting tables from: document.pdf"
    assert_output --regexp '"tables"'
    assert_output --regexp '"table_id"'
    assert_output --regexp '"Header 1"'
}

@test "unstructured-io-manage: extract tables without file argument" {
    run unstructured-io-manage --action extract-tables
    assert_failure
    assert_output --partial "No file provided for table extraction"
}

# Metadata extraction tests
@test "unstructured-io-manage: extract metadata from document" {
    run unstructured-io-manage --action extract-metadata "/path/to/document.pdf"
    assert_success
    assert_output --partial "Extracting metadata from: document.pdf"
    assert_output --regexp '"metadata"'
    assert_output --regexp '"filename"'
    assert_output --regexp '"page_count"'
    assert_output --regexp '"word_count"'
}

@test "unstructured-io-manage: extract metadata without file argument" {
    run unstructured-io-manage --action extract-metadata
    assert_failure
    assert_output --partial "No file provided for metadata extraction"
}

# Directory processing tests
@test "unstructured-io-manage: process directory with default settings" {
    run unstructured-io-manage --action process-directory --directory "/path/to/documents"
    assert_success
    assert_output --partial "Processing documents in directory: /path/to/documents"
    assert_output --partial "Strategy: auto"
    assert_output --partial "Output format: json"
    assert_output --partial "Recursive: false"
    assert_output --partial "Processed 4 documents"
}

@test "unstructured-io-manage: process directory with custom settings" {
    run unstructured-io-manage --action process-directory --directory "/path/to/documents" --strategy "hi_res" --output "text" --recursive "true"
    assert_success
    assert_output --partial "Strategy: hi_res"
    assert_output --partial "Output format: text"
    assert_output --partial "Recursive: true"
}

@test "unstructured-io-manage: process directory without directory argument" {
    run unstructured-io-manage --action process-directory
    assert_failure
    assert_output --partial "No directory provided"
}

# Report generation tests
@test "unstructured-io-manage: create report for document" {
    run unstructured-io-manage --action create-report --file "/path/to/document.pdf"
    assert_success
    assert_output --partial "Creating processing report for: document.pdf"
    assert_output --partial "Report file: processing_report.html"
    assert_output --partial "Report created"
}

@test "unstructured-io-manage: create report with custom filename" {
    run unstructured-io-manage --action create-report --file "/path/to/document.pdf" --report-file "custom_report.html"
    assert_success
    assert_output --partial "Report file: custom_report.html"
}

@test "unstructured-io-manage: create report without file argument" {
    run unstructured-io-manage --action create-report
    assert_failure
    assert_output --partial "No file provided for report generation"
}

# Cache management tests
@test "unstructured-io-manage: show empty cache stats" {
    run unstructured-io-manage --action cache-stats
    assert_success
    assert_output --partial "Cache Statistics"
    assert_output --partial "Cached items: 0"
    assert_output --partial "Processed documents: 0"
}

@test "unstructured-io-manage: show cache stats after processing" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    # Process a document to populate cache
    unstructured-io-manage --action process --file "/path/to/document.pdf"
    
    run unstructured-io-manage --action cache-stats
    assert_success
    assert_output --partial "Cached items: 1"
    assert_output --partial "Processed documents: 1"
    assert_output --partial "Cache contents:"
    assert_output --partial "document.pdf"
}

@test "unstructured-io-manage: clear all cache" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    # Process a document to populate cache
    unstructured-io-manage --action process --file "/path/to/document.pdf"
    
    run unstructured-io-manage --action clear-cache
    assert_success
    assert_output --partial "Cleared all cache"
    
    # Verify cache is empty
    run unstructured-io-manage --action cache-stats
    assert_success
    assert_output --partial "Cached items: 0"
}

@test "unstructured-io-manage: clear cache for specific file" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    # Process multiple documents
    unstructured-io-manage --action process --file "/path/to/document1.pdf"
    unstructured-io-manage --action process --file "/path/to/document2.pdf"
    
    run unstructured-io-manage --action clear-cache --file "/path/to/document1.pdf"
    assert_success
    assert_output --partial "Cleared cache for: document1.pdf"
    
    # Verify only one document remains cached
    run unstructured-io-manage --action cache-stats
    assert_success
    assert_output --partial "Cached items: 1"
    assert_output --partial "document2.pdf"
    refute_output --partial "document1.pdf"
}

# Service information and testing
@test "unstructured-io-manage: show service information" {
    run unstructured-io-manage --action info
    assert_success
    assert_output --partial "Unstructured.io Service Information"
    assert_output --partial "Version: 0.10.30"
    assert_output --partial "API URL: http://localhost:8000"
    assert_output --partial "Container: unstructured-io-mock-container"
    assert_output --partial "Health: healthy"
    assert_output --partial "Cache: true"
    assert_output --partial "Supported formats:"
    assert_output --partial "Default strategy: auto"
}

@test "unstructured-io-manage: test API when service not running" {
    mock::unstructured_io::set_installed "true"
    
    run unstructured-io-manage --action test
    assert_failure
    assert_output --partial "Service is not running"
}

@test "unstructured-io-manage: test API when service running" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    run unstructured-io-manage --action test
    assert_success
    assert_output --partial "Testing Unstructured.io API"
    assert_output --partial "Health check: OK"
    assert_output --partial "API endpoint: Responding"
    assert_output --partial "Document processing: Available"
    assert_output --partial "All tests passed"
}

@test "unstructured-io-manage: view service logs" {
    run unstructured-io-manage --action logs
    assert_success
    assert_output --partial "Unstructured.io Service Logs"
    assert_output --partial "Starting unstructured server on port 8000"
    assert_output --partial "Health check endpoint available"
    assert_output --partial "Document processing endpoint available"
    assert_output --partial "Service ready to accept requests"
}

# Installation validation tests
@test "unstructured-io-manage: validate installation when not installed" {
    run unstructured-io-manage --action validate-installation
    assert_failure
    assert_output --partial "Validating Unstructured.io installation"
    assert_output --partial "Unstructured.io is not installed"
    assert_output --partial "Service is not running"
    assert_output --partial "Found 2 issue(s)"
}

@test "unstructured-io-manage: validate installation when fully working" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    run unstructured-io-manage --action validate-installation
    assert_success
    assert_output --partial "Installation: OK"
    assert_output --partial "Service status: OK"
    assert_output --partial "API endpoint: Available"
    assert_output --partial "Docker connectivity: OK"
    assert_output --partial "Port availability: OK"
    assert_output --partial "All validation checks passed!"
}

# Error handling tests
@test "unstructured-io-manage: unknown action" {
    run unstructured-io-manage --action unknown-action
    assert_failure
    assert_output --partial "Unknown action: unknown-action"
    assert_output --partial "Usage: unstructured-io-manage --action"
}

@test "unstructured-io-manage: error injection - service unavailable" {
    mock::unstructured_io::set_error "service_unavailable"
    
    run unstructured-io-manage --action status
    assert_failure
    assert_output --partial "service is unavailable"
}

@test "unstructured-io-manage: error injection - installation failed" {
    mock::unstructured_io::set_error "installation_failed"
    
    run unstructured-io-manage --action install
    assert_failure
    assert_output --partial "Installation failed"
}

@test "unstructured-io-manage: error injection - processing error" {
    mock::unstructured_io::set_error "processing_error"
    
    run unstructured-io-manage --action process --file "/path/to/document.pdf"
    assert_failure
    assert_output --partial "Document processing failed"
}

# State persistence tests
@test "unstructured-io state persistence across subshells" {
    # Set state in parent shell
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    mock::unstructured_io::save_state
    
    # Verify in subshell
    output=$(
        source "$MOCK_DIR/unstructured-io.sh"
        unstructured-io-manage --action status
    )
    [[ "$output" =~ "Service: Running" ]]
}

@test "unstructured-io state file creation and loading" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    mock::unstructured_io::save_state
    
    # Check state file exists
    [[ -f "$UNSTRUCTURED_IO_MOCK_STATE_DIR/unstructured-io-state.sh" ]]
    
    # Reset without saving state, then reload
    mock::unstructured_io::reset false
    mock::unstructured_io::load_state
    
    # Verify state was restored
    run mock::unstructured_io::assert_installed
    assert_success
    run mock::unstructured_io::assert_running
    assert_success
}

# Test helper function tests
@test "mock::unstructured_io::reset clears all data" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    mock::unstructured_io::set_error "service_unavailable"
    
    # Populate some data
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    unstructured-io-manage --action process --file "/path/to/document.pdf" || true
    
    mock::unstructured_io::reset
    
    # Verify everything is reset
    run mock::unstructured_io::assert_installed
    assert_failure
    
    run mock::unstructured_io::assert_running
    assert_failure
    
    run unstructured-io-manage --action status
    assert_failure  # Should return 1 when not installed, but without error mode message
    refute_output --partial "service is unavailable"
}

@test "mock::unstructured_io::assert_installed" {
    # Test assertion failure
    run mock::unstructured_io::assert_installed
    assert_failure
    assert_output --partial "Unstructured.io is not installed"
    
    # Test assertion success
    mock::unstructured_io::set_installed "true"
    run mock::unstructured_io::assert_installed
    assert_success
}

@test "mock::unstructured_io::assert_running" {
    mock::unstructured_io::set_installed "true"
    
    # Test assertion failure
    run mock::unstructured_io::assert_running
    assert_failure
    assert_output --partial "Unstructured.io is not running"
    
    # Test assertion success
    mock::unstructured_io::set_running "true"
    run mock::unstructured_io::assert_running
    assert_success
}

@test "mock::unstructured_io::assert_processed" {
    # Test assertion failure
    run mock::unstructured_io::assert_processed "/path/to/document.pdf"
    assert_failure
    assert_output --partial "File '/path/to/document.pdf' has not been processed"
    
    # Process a file then test assertion success
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    unstructured-io-manage --action process --file "/path/to/document.pdf"
    
    run mock::unstructured_io::assert_processed "/path/to/document.pdf"
    assert_success
}

@test "mock::unstructured_io::assert_cached" {
    # Test assertion failure
    run mock::unstructured_io::assert_cached "/path/to/document.pdf"
    assert_failure
    assert_output --partial "File '/path/to/document.pdf' is not cached"
    
    # Process a file then test assertion success
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    unstructured-io-manage --action process --file "/path/to/document.pdf"
    
    run mock::unstructured_io::assert_cached "/path/to/document.pdf"
    assert_success
}

@test "mock::unstructured_io::dump_state shows current state" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    mock::unstructured_io::set_config "port" "9000"
    
    run mock::unstructured_io::dump_state
    assert_success
    assert_output --partial "Unstructured.io Mock State"
    assert_output --partial "Configuration:"
    assert_output --partial "installed: true"
    assert_output --partial "running: true"
    assert_output --partial "port: 9000"
}

# Configuration management tests
@test "mock::unstructured_io::set_config updates configuration" {
    mock::unstructured_io::set_config "version" "0.11.0"
    
    run unstructured-io-manage --action info
    assert_success
    assert_output --partial "Version: 0.11.0"
}

@test "mock::unstructured_io::set_installed updates installation state" {
    mock::unstructured_io::set_installed "true"
    
    run mock::unstructured_io::assert_installed
    assert_success
    
    mock::unstructured_io::set_installed "false"
    
    run mock::unstructured_io::assert_installed
    assert_failure
}

@test "mock::unstructured_io::set_running updates running state" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    run mock::unstructured_io::assert_running
    assert_success
    
    mock::unstructured_io::set_running "false"
    
    run mock::unstructured_io::assert_running
    assert_failure
}

# Complex scenario tests
@test "unstructured-io: complete workflow from installation to processing" {
    # Install service
    run unstructured-io-manage --action install
    assert_success
    
    # Start service
    run unstructured-io-manage --action start
    assert_success
    
    # Verify status
    run unstructured-io-manage --action status
    assert_success
    assert_output --partial "Service: Running"
    
    # Test API
    run unstructured-io-manage --action test
    assert_success
    
    # Process a document
    run unstructured-io-manage --action process --file "/path/to/document.pdf" --strategy "hi_res"
    assert_success
    assert_output --partial "Processing completed"
    
    # Verify caching on second call
    run unstructured-io-manage --action process --file "/path/to/document.pdf" --strategy "hi_res"
    assert_success
    assert_output --partial "Using cached result"
    
    # Check cache stats
    run unstructured-io-manage --action cache-stats
    assert_success
    assert_output --partial "Cached items: 1"
    
    # Extract tables
    run unstructured-io-manage --action extract-tables "/path/to/document.pdf"
    assert_success
    assert_output --regexp '"tables"'
    
    # Extract metadata
    run unstructured-io-manage --action extract-metadata "/path/to/document.pdf"
    assert_success
    assert_output --regexp '"metadata"'
    
    # Create report
    run unstructured-io-manage --action create-report --file "/path/to/document.pdf"
    assert_success
    
    # Validate installation
    run unstructured-io-manage --action validate-installation
    assert_success
    
    # Stop service
    run unstructured-io-manage --action stop
    assert_success
    
    # Uninstall
    run unstructured-io-manage --action uninstall
    assert_success
}

@test "unstructured-io: batch directory processing workflow" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    # Process directory with different settings
    run unstructured-io-manage --action process-directory --directory "/path/to/documents" --strategy "fast" --output "text" --recursive "true"
    assert_success
    assert_output --partial "Processing documents in directory"
    assert_output --partial "Strategy: fast"
    assert_output --partial "Recursive: true"
    assert_output --partial "Processed 4 documents"
    assert_output --partial "Results saved to"
}

# Edge cases and error conditions
@test "unstructured-io: handling of missing required arguments" {
    # Process without file
    run unstructured-io-manage --action process
    assert_failure
    
    # Extract tables without file
    run unstructured-io-manage --action extract-tables
    assert_failure
    
    # Extract metadata without file
    run unstructured-io-manage --action extract-metadata
    assert_failure
    
    # Process directory without directory
    run unstructured-io-manage --action process-directory
    assert_failure
    
    # Create report without file
    run unstructured-io-manage --action create-report
    assert_failure
}

@test "unstructured-io: various file extensions and processing strategies" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    # Test different file types
    local files=(
        "/path/to/document.pdf"
        "/path/to/document.docx"
        "/path/to/document.txt"
        "/path/to/document.html"
    )
    
    local strategies=("auto" "hi_res" "fast")
    local formats=("json" "text")
    
    for file in "${files[@]}"; do
        for strategy in "${strategies[@]}"; do
            for format in "${formats[@]}"; do
                run unstructured-io-manage --action process --file "$file" --strategy "$strategy" --output "$format"
                assert_success
                assert_output --partial "$(basename "$file")"
                assert_output --partial "Strategy: $strategy"
                assert_output --partial "Output format: $format"
            done
        done
    done
}

@test "unstructured-io: state persistence with multiple operations" {
    # Perform multiple operations that modify state
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    mock::unstructured_io::set_config "port" "9000"
    
    # Process multiple documents
    unstructured-io-manage --action process --file "/doc1.pdf"
    unstructured-io-manage --action process --file "/doc2.docx" --strategy "hi_res"
    unstructured-io-manage --action process --file "/doc3.txt" --output "text"
    
    # Save state
    mock::unstructured_io::save_state
    
    # Reset and reload
    mock::unstructured_io::reset false
    mock::unstructured_io::load_state
    
    # Verify all state was preserved
    run mock::unstructured_io::assert_installed
    assert_success
    
    run mock::unstructured_io::assert_running
    assert_success
    
    run unstructured-io-manage --action info
    assert_success
    assert_output --partial "9000"  # Custom port
    
    # Verify documents are still cached
    run mock::unstructured_io::assert_cached "/doc1.pdf"
    assert_success
    
    run mock::unstructured_io::assert_cached "/doc2.docx"
    assert_success
    
    run mock::unstructured_io::assert_cached "/doc3.txt"
    assert_success
}

# Performance and stress testing
@test "unstructured-io: cache behavior with many documents" {
    mock::unstructured_io::set_installed "true"
    mock::unstructured_io::set_running "true"
    
    # Process many documents
    for i in {1..10}; do
        unstructured-io-manage --action process --file "/path/to/document${i}.pdf" --strategy "auto"
    done
    
    # Check cache stats
    run unstructured-io-manage --action cache-stats
    assert_success
    assert_output --partial "Cached items: 10"
    assert_output --partial "Processed documents: 10"
    
    # Clear cache for odd-numbered documents
    for i in {1..10..2}; do
        unstructured-io-manage --action clear-cache --file "/path/to/document${i}.pdf"
    done
    
    # Verify cache now has 5 items
    run unstructured-io-manage --action cache-stats
    assert_success
    assert_output --partial "Cached items: 5"
    
    # Clear all cache
    run unstructured-io-manage --action clear-cache
    assert_success
    
    # Verify cache is empty
    run unstructured-io-manage --action cache-stats
    assert_success
    assert_output --partial "Cached items: 0"
}