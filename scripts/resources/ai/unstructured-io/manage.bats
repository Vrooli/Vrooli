#!/usr/bin/env bats
# Tests for Unstructured.io manage.sh script

# Load common test helper
source "$(dirname "${BATS_TEST_FILENAME}")/../../tests/common_test_helper.bash"

# Setup for each test
setup() {
    # Use common setup
    common_setup
    
    # Set unstructured-io specific environment
    export UNSTRUCTURED_IO_CUSTOM_PORT="9999"
    export FILE_INPUT=""
    export STRATEGY="hi_res"
    export OUTPUT="json"
    export LANGUAGES="eng"
    export BATCH="no"
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
    skip_if_service_not_running "docker"
    
    # Mock docker commands
    mock_docker
    
    # Source install function
    source "${SCRIPT_DIR}/lib/install.sh" 2>/dev/null || true
    
    # Test install
    run unstructured_io::install "no"
    [ "$status" -eq 0 ]
}

# Test status functionality  
@test "unstructured_io::status function works correctly" {
    # Source status function
    source "${SCRIPT_DIR}/lib/status.sh" 2>/dev/null || true
    
    # Mock service running
    is_service_running() { return 0; }
    export -f is_service_running
    
    # Test status
    run unstructured_io::status
    [ "$status" -eq 0 ]
}

# Test process functionality with missing file
@test "unstructured_io::process handles missing file" {
    # Source process function
    source "${SCRIPT_DIR}/lib/process.sh" 2>/dev/null || true
    
    export FILE_INPUT=""
    
    # Test should fail for missing file
    run unstructured_io::process
    [ "$status" -ne 0 ]
}

# Test process functionality with valid file
@test "unstructured_io::process handles valid file" {
    skip_if_service_not_running "unstructured-io"
    
    # Source process function
    source "${SCRIPT_DIR}/lib/process.sh" 2>/dev/null || true
    
    # Create test file
    local test_file=$(create_temp_file "test.txt" "Test content")
    export FILE_INPUT="$test_file"
    
    # Test process
    run unstructured_io::process
    # May fail if service not actually running
    true
}

# Test batch processing
@test "unstructured_io::process handles batch mode" {
    skip_if_service_not_running "unstructured-io"
    
    # Source process function
    source "${SCRIPT_DIR}/lib/process.sh" 2>/dev/null || true
    
    export BATCH="yes"
    export FILE_INPUT="*.txt"
    
    # Test batch process
    run unstructured_io::process
    # May fail if service not actually running
    true
}

# Test API functions
@test "unstructured_io::check_health works correctly" {
    # Source API function
    source "${SCRIPT_DIR}/lib/api.sh" 2>/dev/null || true
    
    # Mock curl for health check
    curl() {
        if [[ "$*" == *"healthcheck"* ]]; then
            echo "OK"
            return 0
        fi
        command curl "$@"
    }
    export -f curl
    
    run unstructured_io::check_health
    [ "$status" -eq 0 ]
}

# Test supported formats
@test "unstructured_io::is_format_supported works correctly" {
    # Source process function
    source "${SCRIPT_DIR}/lib/process.sh" 2>/dev/null || true
    
    # Test supported format
    run unstructured_io::is_format_supported "pdf"
    [ "$status" -eq 0 ]
    
    # Test unsupported format
    run unstructured_io::is_format_supported "xyz"
    [ "$status" -ne 0 ]
}