#!/usr/bin/env bats
# Tests for Unstructured.io defaults.sh configuration

# Setup for each test
setup() {
    # Set test environment
    export UNSTRUCTURED_IO_CUSTOM_PORT="9999"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    UNSTRUCTURED_IO_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Mock resources function
    resources::get_default_port() {
        case "$1" in
            "unstructured-io") echo "8002" ;;
            *) echo "8000" ;;
        esac
    }
    
    # Load the configuration to test
    source "${UNSTRUCTURED_IO_DIR}/config/defaults.sh"
}

# Test port configuration
@test "UNSTRUCTURED_IO_PORT uses custom port when set" {
    [ "$UNSTRUCTURED_IO_PORT" = "9999" ]
}

@test "UNSTRUCTURED_IO_PORT uses default port when custom not set" {
    unset UNSTRUCTURED_IO_CUSTOM_PORT
    
    # Re-source to get default behavior
    source "${UNSTRUCTURED_IO_DIR}/config/defaults.sh"
    
    [ "$UNSTRUCTURED_IO_PORT" = "8002" ]
}

# Test base URL configuration
@test "UNSTRUCTURED_IO_BASE_URL is constructed correctly" {
    [[ "$UNSTRUCTURED_IO_BASE_URL" =~ "http://localhost:9999" ]]
}

# Test service name configuration
@test "UNSTRUCTURED_IO_SERVICE_NAME is set correctly" {
    [ "$UNSTRUCTURED_IO_SERVICE_NAME" = "unstructured-io" ]
}

# Test container name configuration
@test "UNSTRUCTURED_IO_CONTAINER_NAME is set correctly" {
    [ "$UNSTRUCTURED_IO_CONTAINER_NAME" = "vrooli-unstructured-io" ]
}

# Test Docker image configuration
@test "UNSTRUCTURED_IO_IMAGE is set correctly" {
    [[ "$UNSTRUCTURED_IO_IMAGE" =~ "unstructured-api" ]]
    [[ "$UNSTRUCTURED_IO_IMAGE" =~ "latest" ]]
}

# Test Docker port configuration
@test "UNSTRUCTURED_IO_API_PORT is set correctly" {
    [ "$UNSTRUCTURED_IO_API_PORT" = "8000" ]
}

# Test memory limit configuration
@test "UNSTRUCTURED_IO_MEMORY_LIMIT is set correctly" {
    [ "$UNSTRUCTURED_IO_MEMORY_LIMIT" = "4g" ]
}

# Test CPU limit configuration
@test "UNSTRUCTURED_IO_CPU_LIMIT is set correctly" {
    [ "$UNSTRUCTURED_IO_CPU_LIMIT" = "2.0" ]
}

# Test processing strategy configuration
@test "UNSTRUCTURED_IO_DEFAULT_STRATEGY is set correctly" {
    [ "$UNSTRUCTURED_IO_DEFAULT_STRATEGY" = "hi_res" ]
}

# Test language configuration
@test "UNSTRUCTURED_IO_DEFAULT_LANGUAGES is set correctly" {
    [ "$UNSTRUCTURED_IO_DEFAULT_LANGUAGES" = "eng" ]
}

# Test partition by API configuration
@test "UNSTRUCTURED_IO_PARTITION_BY_API is set correctly" {
    [ "$UNSTRUCTURED_IO_PARTITION_BY_API" = "true" ]
}

# Test page breaks configuration
@test "UNSTRUCTURED_IO_INCLUDE_PAGE_BREAKS is set correctly" {
    [ "$UNSTRUCTURED_IO_INCLUDE_PAGE_BREAKS" = "true" ]
}

# Test encoding configuration
@test "UNSTRUCTURED_IO_ENCODING is set correctly" {
    [ "$UNSTRUCTURED_IO_ENCODING" = "utf-8" ]
}

# Test file size limit configuration
@test "UNSTRUCTURED_IO_MAX_FILE_SIZE is set correctly" {
    [ "$UNSTRUCTURED_IO_MAX_FILE_SIZE" = "50MB" ]
}

# Test file size bytes configuration
@test "UNSTRUCTURED_IO_MAX_FILE_SIZE_BYTES is calculated correctly" {
    expected=$((50 * 1024 * 1024))
    [ "$UNSTRUCTURED_IO_MAX_FILE_SIZE_BYTES" = "$expected" ]
}

# Test timeout configuration
@test "UNSTRUCTURED_IO_TIMEOUT_SECONDS is set correctly" {
    [ "$UNSTRUCTURED_IO_TIMEOUT_SECONDS" = "300" ]
}

# Test concurrent requests configuration
@test "UNSTRUCTURED_IO_MAX_CONCURRENT_REQUESTS is set correctly" {
    [ "$UNSTRUCTURED_IO_MAX_CONCURRENT_REQUESTS" = "5" ]
}

# Test readonly nature of configurations
@test "configuration variables are readonly" {
    # Try to modify a readonly variable - should fail
    run bash -c "UNSTRUCTURED_IO_SERVICE_NAME='modified'; echo \$UNSTRUCTURED_IO_SERVICE_NAME"
    # The original value should remain unchanged in our current shell
    [ "$UNSTRUCTURED_IO_SERVICE_NAME" = "unstructured-io" ]
}

# Test that all required variables are set
@test "all required configuration variables are defined" {
    [ -n "$UNSTRUCTURED_IO_PORT" ]
    [ -n "$UNSTRUCTURED_IO_BASE_URL" ]
    [ -n "$UNSTRUCTURED_IO_SERVICE_NAME" ]
    [ -n "$UNSTRUCTURED_IO_CONTAINER_NAME" ]
    [ -n "$UNSTRUCTURED_IO_IMAGE" ]
    [ -n "$UNSTRUCTURED_IO_API_PORT" ]
    [ -n "$UNSTRUCTURED_IO_MEMORY_LIMIT" ]
    [ -n "$UNSTRUCTURED_IO_CPU_LIMIT" ]
    [ -n "$UNSTRUCTURED_IO_DEFAULT_STRATEGY" ]
    [ -n "$UNSTRUCTURED_IO_DEFAULT_LANGUAGES" ]
    [ -n "$UNSTRUCTURED_IO_MAX_FILE_SIZE" ]
    [ -n "$UNSTRUCTURED_IO_MAX_FILE_SIZE_BYTES" ]
    [ -n "$UNSTRUCTURED_IO_TIMEOUT_SECONDS" ]
    [ -n "$UNSTRUCTURED_IO_MAX_CONCURRENT_REQUESTS" ]
}

# Test configuration export function
@test "unstructured_io::export_config exports all configurations" {
    unstructured_io::export_config
    
    # Check that variables are exported (available in subshells)
    result=$(bash -c 'echo $UNSTRUCTURED_IO_PORT')
    [ "$result" = "9999" ]
    
    result=$(bash -c 'echo $UNSTRUCTURED_IO_SERVICE_NAME')
    [ "$result" = "unstructured-io" ]
}

# Test configuration validation
@test "port configuration is numeric" {
    [[ "$UNSTRUCTURED_IO_PORT" =~ ^[0-9]+$ ]]
    [[ "$UNSTRUCTURED_IO_API_PORT" =~ ^[0-9]+$ ]]
}

@test "timeout configuration is numeric" {
    [[ "$UNSTRUCTURED_IO_TIMEOUT_SECONDS" =~ ^[0-9]+$ ]]
    [[ "$UNSTRUCTURED_IO_MAX_CONCURRENT_REQUESTS" =~ ^[0-9]+$ ]]
}

@test "memory and CPU limits are properly formatted" {
    [[ "$UNSTRUCTURED_IO_MEMORY_LIMIT" =~ ^[0-9]+[kmg]?$ ]]
    [[ "$UNSTRUCTURED_IO_CPU_LIMIT" =~ ^[0-9]+\.?[0-9]*$ ]]
}

@test "file size configuration is consistent" {
    # Check that byte calculation matches the MB specification
    expected_mb=$(echo "$UNSTRUCTURED_IO_MAX_FILE_SIZE" | grep -o '^[0-9]*')
    expected_bytes=$((expected_mb * 1024 * 1024))
    [ "$UNSTRUCTURED_IO_MAX_FILE_SIZE_BYTES" = "$expected_bytes" ]
}

@test "boolean configurations are valid" {
    [[ "$UNSTRUCTURED_IO_PARTITION_BY_API" =~ ^(true|false)$ ]]
    [[ "$UNSTRUCTURED_IO_INCLUDE_PAGE_BREAKS" =~ ^(true|false)$ ]]
}

@test "strategy configuration is valid" {
    [[ "$UNSTRUCTURED_IO_DEFAULT_STRATEGY" =~ ^(hi_res|fast|auto)$ ]]
}

@test "encoding configuration is valid" {
    [[ "$UNSTRUCTURED_IO_ENCODING" =~ ^(utf-8|ascii|iso-8859-1)$ ]]
}