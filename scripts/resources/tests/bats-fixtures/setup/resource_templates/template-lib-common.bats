#!/usr/bin/env bats
# Tests for RESOURCE_NAME lib/common.sh utility functions
#
# Template Usage:
# 1. Copy this file to RESOURCE_NAME/lib/common.bats
# 2. Replace RESOURCE_NAME with your resource name (e.g., "ollama", "n8n")
# 3. Replace FUNCTION_NAMES with your actual function names
# 4. Implement resource-specific utility function tests
# 5. Remove this header comment block

bats_require_minimum_version 1.5.0

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    setup_standard_mocks
    
    # Set test environment
    export RESOURCE_NAME_PORT="8080"  # Replace with actual port
    export RESOURCE_NAME_CONTAINER_NAME="RESOURCE_NAME-test"
    export RESOURCE_NAME_BASE_URL="http://localhost:8080"
    export RESOURCE_NAME_DATA_DIR="/tmp/RESOURCE_NAME-test/data"
    export RESOURCE_NAME_CONFIG_DIR="/tmp/RESOURCE_NAME-test/config"
    
    # Get resource directory path
    RESOURCE_NAME_DIR="$(dirname "$(dirname "${BATS_TEST_FILENAME}")")"
    
    # Create test directories
    mkdir -p "$RESOURCE_NAME_DATA_DIR"
    mkdir -p "$RESOURCE_NAME_CONFIG_DIR"
    
    # Configure resource-specific mocks
    mock::docker::set_container_state "RESOURCE_NAME-test" "running"
    mock::http::set_endpoint_state "http://localhost:8080" "healthy"
    
    # Load configuration and the functions to test
    source "${RESOURCE_NAME_DIR}/config/defaults.sh"
    source "${RESOURCE_NAME_DIR}/config/messages.sh"
    RESOURCE_NAME::export_config
    RESOURCE_NAME::messages::init
    
    # Load the common utilities
    source "${RESOURCE_NAME_DIR}/lib/common.sh"
}

teardown() {
    # Clean up test environment
    cleanup_mocks
    rm -rf "/tmp/RESOURCE_NAME-test"
}

# Test utility functions (customize based on your actual functions)

@test "RESOURCE_NAME::is_running should detect running container" {
    # Mock running container
    mock::docker::set_container_state "RESOURCE_NAME-test" "running"
    
    run RESOURCE_NAME::is_running
    [ "$status" -eq 0 ]
}

@test "RESOURCE_NAME::is_running should detect stopped container" {
    # Mock stopped container
    mock::docker::set_container_state "RESOURCE_NAME-test" "stopped"
    
    run RESOURCE_NAME::is_running
    [ "$status" -ne 0 ]
}

@test "RESOURCE_NAME::is_running should handle missing container" {
    # Mock missing container
    mock::docker::set_container_state "RESOURCE_NAME-test" "missing"
    
    run RESOURCE_NAME::is_running
    [ "$status" -ne 0 ]
}

@test "RESOURCE_NAME::get_container_id should return valid container ID" {
    # Mock container with ID
    mock::docker::set_container_id "RESOURCE_NAME-test" "abc123def456"
    
    run RESOURCE_NAME::get_container_id
    [ "$status" -eq 0 ]
    [ "$output" = "abc123def456" ]
}

@test "RESOURCE_NAME::get_container_id should handle missing container" {
    # Mock missing container
    mock::docker::set_container_state "RESOURCE_NAME-test" "missing"
    
    run RESOURCE_NAME::get_container_id
    [ "$status" -ne 0 ]
    [ -z "$output" ]
}

@test "RESOURCE_NAME::health_check should return healthy status" {
    # Mock healthy service
    mock::http::set_endpoint_response "http://localhost:8080/health" "200" '{"status":"healthy"}'
    
    run RESOURCE_NAME::health_check
    [ "$status" -eq 0 ]
    [[ "$output" =~ "healthy" ]]
}

@test "RESOURCE_NAME::health_check should handle unhealthy status" {
    # Mock unhealthy service
    mock::http::set_endpoint_response "http://localhost:8080/health" "503" '{"status":"unhealthy"}'
    
    run RESOURCE_NAME::health_check
    [ "$status" -ne 0 ]
    [[ "$output" =~ "unhealthy" || "$output" =~ "error" ]]
}

@test "RESOURCE_NAME::health_check should handle connection failure" {
    # Mock connection failure
    mock::http::set_endpoint_response "http://localhost:8080/health" "connection_refused" ""
    
    run RESOURCE_NAME::health_check
    [ "$status" -ne 0 ]
    [[ "$output" =~ "connection" || "$output" =~ "failed" ]]
}

@test "RESOURCE_NAME::wait_for_ready should wait for service to be ready" {
    # Mock service becoming ready
    mock::http::set_endpoint_sequence "http://localhost:8080/health" "503,503,200" "unhealthy,unhealthy,healthy"
    
    run timeout 10 RESOURCE_NAME::wait_for_ready
    [ "$status" -eq 0 ]
}

@test "RESOURCE_NAME::wait_for_ready should timeout when service never ready" {
    # Mock service never becoming ready
    mock::http::set_endpoint_response "http://localhost:8080/health" "503" "unhealthy"
    
    run timeout 2 RESOURCE_NAME::wait_for_ready
    [ "$status" -ne 0 ]
}

# Test configuration helpers

@test "RESOURCE_NAME::get_config_value should return configuration values" {
    export RESOURCE_NAME_TEST_VALUE="test123"
    
    run RESOURCE_NAME::get_config_value "TEST_VALUE"
    [ "$status" -eq 0 ]
    [ "$output" = "test123" ]
}

@test "RESOURCE_NAME::get_config_value should handle missing values" {
    run RESOURCE_NAME::get_config_value "NONEXISTENT_VALUE"
    [ "$status" -ne 0 ]
    [ -z "$output" ]
}

@test "RESOURCE_NAME::set_config_value should update configuration" {
    run RESOURCE_NAME::set_config_value "TEST_VALUE" "new_value"
    [ "$status" -eq 0 ]
    [ "$RESOURCE_NAME_TEST_VALUE" = "new_value" ]
}

# Test directory management

@test "RESOURCE_NAME::ensure_directories should create required directories" {
    # Remove test directories
    rm -rf "/tmp/RESOURCE_NAME-test"
    
    run RESOURCE_NAME::ensure_directories
    [ "$status" -eq 0 ]
    [ -d "$RESOURCE_NAME_DATA_DIR" ]
    [ -d "$RESOURCE_NAME_CONFIG_DIR" ]
}

@test "RESOURCE_NAME::ensure_directories should handle existing directories" {
    # Directories already exist from setup
    run RESOURCE_NAME::ensure_directories
    [ "$status" -eq 0 ]
}

@test "RESOURCE_NAME::clean_directories should remove temporary files" {
    # Create test files
    touch "$RESOURCE_NAME_DATA_DIR/test.tmp"
    touch "$RESOURCE_NAME_DATA_DIR/important.data"
    
    run RESOURCE_NAME::clean_directories
    [ "$status" -eq 0 ]
    [ ! -f "$RESOURCE_NAME_DATA_DIR/test.tmp" ]
    [ -f "$RESOURCE_NAME_DATA_DIR/important.data" ]  # Should preserve non-temp files
}

# Test validation helpers

@test "RESOURCE_NAME::validate_port should accept valid ports" {
    run RESOURCE_NAME::validate_port "8080"
    [ "$status" -eq 0 ]
    
    run RESOURCE_NAME::validate_port "80"
    [ "$status" -eq 0 ]
    
    run RESOURCE_NAME::validate_port "65535"
    [ "$status" -eq 0 ]
}

@test "RESOURCE_NAME::validate_port should reject invalid ports" {
    run RESOURCE_NAME::validate_port "0"
    [ "$status" -ne 0 ]
    
    run RESOURCE_NAME::validate_port "70000"
    [ "$status" -ne 0 ]
    
    run RESOURCE_NAME::validate_port "abc"
    [ "$status" -ne 0 ]
    
    run RESOURCE_NAME::validate_port ""
    [ "$status" -ne 0 ]
}

@test "RESOURCE_NAME::validate_url should accept valid URLs" {
    run RESOURCE_NAME::validate_url "http://localhost:8080"
    [ "$status" -eq 0 ]
    
    run RESOURCE_NAME::validate_url "https://example.com"
    [ "$status" -eq 0 ]
}

@test "RESOURCE_NAME::validate_url should reject invalid URLs" {
    run RESOURCE_NAME::validate_url "not-a-url"
    [ "$status" -ne 0 ]
    
    run RESOURCE_NAME::validate_url ""
    [ "$status" -ne 0 ]
    
    run RESOURCE_NAME::validate_url "ftp://example.com"  # If only HTTP(S) allowed
    [ "$status" -ne 0 ]
}

# Test logging helpers

@test "RESOURCE_NAME::log_info should output info messages" {
    run RESOURCE_NAME::log_info "Test message"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Test message" ]]
    [[ "$output" =~ "INFO" || ! "$output" =~ "ERROR" ]]
}

@test "RESOURCE_NAME::log_error should output error messages" {
    run RESOURCE_NAME::log_error "Error message"
    [ "$status" -eq 0 ]  # Logging should succeed even for errors
    [[ "$output" =~ "Error message" ]]
    [[ "$output" =~ "ERROR" ]]
}

@test "RESOURCE_NAME::log_debug should respect debug mode" {
    export RESOURCE_NAME_DEBUG="true"
    run RESOURCE_NAME::log_debug "Debug message"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Debug message" ]]
    
    export RESOURCE_NAME_DEBUG="false"
    run RESOURCE_NAME::log_debug "Debug message"
    [ "$status" -eq 0 ]
    [[ ! "$output" =~ "Debug message" ]]
}

# Test resource-specific utility functions

@test "RESOURCE_NAME specific utility functions" {
    # Example for AI resources:
    # run RESOURCE_NAME::list_models
    # [ "$status" -eq 0 ]
    # [[ "$output" =~ "model" ]]
    
    # Example for automation resources:
    # run RESOURCE_NAME::validate_workflow "test_workflow"
    # [ "$status" -eq 0 ]
    
    # Example for storage resources:
    # run RESOURCE_NAME::check_disk_space
    # [ "$status" -eq 0 ]
    # [[ "$output" =~ "MB" || "$output" =~ "GB" ]]
    
    # Replace this with actual resource-specific tests
    skip "Implement resource-specific utility function tests"
}

# Test error handling

@test "RESOURCE_NAME::handle_error should process errors correctly" {
    run RESOURCE_NAME::handle_error "Test error" 1
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Test error" ]]
}

@test "RESOURCE_NAME functions should handle missing dependencies" {
    # Mock missing curl
    curl() { return 127; }
    export -f curl
    
    run RESOURCE_NAME::health_check
    [ "$status" -ne 0 ]
    [[ "$output" =~ "curl" || "$output" =~ "dependency" ]]
}

@test "RESOURCE_NAME functions should handle permission errors" {
    # Mock permission denied
    chmod 000 "$RESOURCE_NAME_DATA_DIR" 2>/dev/null || skip "Cannot test permission errors"
    
    run RESOURCE_NAME::ensure_directories
    [ "$status" -ne 0 ]
    [[ "$output" =~ "permission" || "$output" =~ "denied" ]]
    
    # Restore permissions
    chmod 755 "$RESOURCE_NAME_DATA_DIR"
}

# Test performance (basic)

@test "RESOURCE_NAME::health_check should complete within reasonable time" {
    # Mock fast response
    mock::http::set_endpoint_response "http://localhost:8080/health" "200" "healthy"
    
    start_time=$(date +%s)
    run RESOURCE_NAME::health_check
    end_time=$(date +%s)
    
    [ "$status" -eq 0 ]
    [ $((end_time - start_time)) -lt 5 ]  # Should complete within 5 seconds
}

# Add more resource-specific common utility tests here:

# Example templates for different resource types:

# For AI resources:
# @test "RESOURCE_NAME::download_model should handle model downloads" { ... }
# @test "RESOURCE_NAME::check_gpu should detect GPU availability" { ... }

# For automation resources:
# @test "RESOURCE_NAME::backup_workflows should create backups" { ... }
# @test "RESOURCE_NAME::restore_workflows should restore from backup" { ... }

# For storage resources:
# @test "RESOURCE_NAME::check_storage_health should verify storage" { ... }
# @test "RESOURCE_NAME::cleanup_old_data should remove old data" { ... }