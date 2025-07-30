#!/usr/bin/env bats
# Tests for RESOURCE_NAME config/defaults.sh configuration management
#
# Template Usage:
# 1. Copy this file to RESOURCE_NAME/config/defaults.bats
# 2. Replace RESOURCE_NAME with your resource name (e.g., "ollama", "n8n")
# 3. Replace CONFIG_VARIABLES with your actual configuration variables
# 4. Implement resource-specific configuration tests
# 5. Remove this header comment block

bats_require_minimum_version 1.5.0

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    setup_standard_mocks
    
    # Set test environment
    export RESOURCE_NAME_PORT="8080"  # Replace with actual default port
    export RESOURCE_NAME_CONTAINER_NAME="RESOURCE_NAME-test"
    export RESOURCE_NAME_BASE_URL="http://localhost:8080"
    
    # Clear any existing configuration to test defaults
    unset RESOURCE_NAME_CUSTOM_CONFIG
    
    # Get resource directory path
    RESOURCE_NAME_DIR="$(dirname "$(dirname "${BATS_TEST_FILENAME}")")"
    
    # Load configuration
    source "${RESOURCE_NAME_DIR}/config/defaults.sh"
}

teardown() {
    # Clean up environment
    cleanup_mocks
}

# Test configuration loading and initialization

@test "RESOURCE_NAME::export_config should set all required variables" {
    run RESOURCE_NAME::export_config
    [ "$status" -eq 0 ]
    
    # Test that essential configuration variables are set
    # Replace these with your actual configuration variables:
    [ -n "$RESOURCE_NAME_PORT" ]
    [ -n "$RESOURCE_NAME_CONTAINER_NAME" ]
    [ -n "$RESOURCE_NAME_BASE_URL" ]
    [ -n "$RESOURCE_NAME_IMAGE" ]
    [ -n "$RESOURCE_NAME_VERSION" ]
}

@test "RESOURCE_NAME::export_config should use defaults when no custom config" {
    # Clear environment
    unset RESOURCE_NAME_PORT
    
    run RESOURCE_NAME::export_config
    [ "$status" -eq 0 ]
    
    # Should use default port (replace 8080 with your actual default)
    [ "$RESOURCE_NAME_PORT" = "8080" ]
}

@test "RESOURCE_NAME::export_config should respect environment overrides" {
    # Set custom values
    export RESOURCE_NAME_PORT="9090"
    export RESOURCE_NAME_CONTAINER_NAME="custom-name"
    
    run RESOURCE_NAME::export_config
    [ "$status" -eq 0 ]
    
    # Should respect overrides
    [ "$RESOURCE_NAME_PORT" = "9090" ]
    [ "$RESOURCE_NAME_CONTAINER_NAME" = "custom-name" ]
}

# Test configuration validation functions

@test "RESOURCE_NAME::validate_config should accept valid configuration" {
    RESOURCE_NAME::export_config
    
    run RESOURCE_NAME::validate_config
    [ "$status" -eq 0 ]
}

@test "RESOURCE_NAME::validate_config should reject invalid port" {
    export RESOURCE_NAME_PORT="invalid"
    
    run RESOURCE_NAME::validate_config
    [ "$status" -ne 0 ]
    [[ "$output" =~ "port" || "$output" =~ "invalid" ]]
}

@test "RESOURCE_NAME::validate_config should reject port out of range" {
    export RESOURCE_NAME_PORT="70000"
    
    run RESOURCE_NAME::validate_config
    [ "$status" -ne 0 ]
    [[ "$output" =~ "port" || "$output" =~ "range" ]]
}

@test "RESOURCE_NAME::validate_config should validate container name format" {
    export RESOURCE_NAME_CONTAINER_NAME="Invalid Name With Spaces"
    
    run RESOURCE_NAME::validate_config
    [ "$status" -ne 0 ]
    [[ "$output" =~ "container" || "$output" =~ "name" ]]
}

# Test configuration display functions

@test "RESOURCE_NAME::show_config should display all configuration" {
    RESOURCE_NAME::export_config
    
    run RESOURCE_NAME::show_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "RESOURCE_NAME_PORT" ]]
    [[ "$output" =~ "RESOURCE_NAME_CONTAINER_NAME" ]]
    [[ "$output" =~ "RESOURCE_NAME_BASE_URL" ]]
}

@test "RESOURCE_NAME::show_config should mask sensitive values" {
    export RESOURCE_NAME_API_KEY="secret123"
    RESOURCE_NAME::export_config
    
    run RESOURCE_NAME::show_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "***" || "$output" =~ "hidden" ]]
    [[ ! "$output" =~ "secret123" ]]
}

# Test configuration file handling

@test "RESOURCE_NAME::load_config_file should load existing config file" {
    # Create temporary config file
    local config_file="/tmp/RESOURCE_NAME-test-config"
    cat > "$config_file" << EOF
RESOURCE_NAME_PORT=9999
RESOURCE_NAME_CONTAINER_NAME=test-container
EOF
    
    run RESOURCE_NAME::load_config_file "$config_file"
    [ "$status" -eq 0 ]
    [ "$RESOURCE_NAME_PORT" = "9999" ]
    [ "$RESOURCE_NAME_CONTAINER_NAME" = "test-container" ]
    
    # Cleanup
    rm -f "$config_file"
}

@test "RESOURCE_NAME::load_config_file should handle missing config file gracefully" {
    run RESOURCE_NAME::load_config_file "/nonexistent/config"
    # Should either succeed with defaults or fail gracefully
    [[ "$status" -eq 0 || "$output" =~ "not found" ]]
}

@test "RESOURCE_NAME::save_config should create valid config file" {
    RESOURCE_NAME::export_config
    local config_file="/tmp/RESOURCE_NAME-test-save"
    
    run RESOURCE_NAME::save_config "$config_file"
    [ "$status" -eq 0 ]
    [ -f "$config_file" ]
    
    # Verify config file content
    grep -q "RESOURCE_NAME_PORT=" "$config_file"
    grep -q "RESOURCE_NAME_CONTAINER_NAME=" "$config_file"
    
    # Cleanup
    rm -f "$config_file"
}

# Test configuration dependency checks

@test "RESOURCE_NAME::check_dependencies should verify required tools" {
    run RESOURCE_NAME::check_dependencies
    [ "$status" -eq 0 ]
}

@test "RESOURCE_NAME::check_dependencies should fail when Docker missing" {
    # Mock missing Docker
    docker() { return 127; }
    export -f docker
    
    run RESOURCE_NAME::check_dependencies
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Docker" || "$output" =~ "dependency" ]]
}

# Test resource-specific configuration (customize based on your resource)

@test "RESOURCE_NAME specific configuration validation" {
    # Example for AI resources:
    # export RESOURCE_NAME_MODEL="invalid-model"
    # run RESOURCE_NAME::validate_model
    # [ "$status" -ne 0 ]
    
    # Example for automation resources:
    # export RESOURCE_NAME_WORKFLOW_DIR="/nonexistent"
    # run RESOURCE_NAME::validate_workflow_dir
    # [ "$status" -ne 0 ]
    
    # Example for storage resources:
    # export RESOURCE_NAME_DATA_DIR="/readonly"
    # run RESOURCE_NAME::validate_data_dir
    # [ "$status" -ne 0 ]
    
    # Replace this with actual resource-specific tests
    skip "Implement resource-specific configuration tests"
}

# Test configuration edge cases

@test "RESOURCE_NAME::export_config should handle special characters in values" {
    export RESOURCE_NAME_CUSTOM_VALUE="value with spaces and symbols!@#"
    
    run RESOURCE_NAME::export_config
    [ "$status" -eq 0 ]
    [ "$RESOURCE_NAME_CUSTOM_VALUE" = "value with spaces and symbols!@#" ]
}

@test "RESOURCE_NAME::validate_config should handle empty values appropriately" {
    export RESOURCE_NAME_OPTIONAL_VALUE=""
    
    run RESOURCE_NAME::validate_config
    # Should either accept empty optional values or provide clear error
    [[ "$status" -eq 0 || "$output" =~ "required" ]]
}

@test "RESOURCE_NAME::export_config should maintain configuration consistency" {
    RESOURCE_NAME::export_config
    local port1="$RESOURCE_NAME_PORT"
    
    RESOURCE_NAME::export_config
    local port2="$RESOURCE_NAME_PORT"
    
    [ "$port1" = "$port2" ]
}

# Add more resource-specific configuration tests here:

# Example templates for different resource types:

# For AI resources:
# @test "RESOURCE_NAME should validate model configuration" { ... }
# @test "RESOURCE_NAME should handle GPU/CPU mode selection" { ... }

# For automation resources:
# @test "RESOURCE_NAME should validate workflow directory" { ... }
# @test "RESOURCE_NAME should handle authentication configuration" { ... }

# For storage resources:
# @test "RESOURCE_NAME should validate storage paths" { ... }
# @test "RESOURCE_NAME should handle backup configuration" { ... }