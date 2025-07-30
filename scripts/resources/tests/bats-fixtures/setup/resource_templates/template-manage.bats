#!/usr/bin/env bats
# Tests for RESOURCE_NAME manage.sh entry point
# 
# Template Usage:
# 1. Copy this file to RESOURCE_NAME/manage.bats
# 2. Replace RESOURCE_NAME with your resource name (e.g., "ollama", "n8n")
# 3. Replace MODULE_PATH with the path to your manage.sh file
# 4. Implement resource-specific test cases
# 5. Remove this header comment block

bats_require_minimum_version 1.5.0

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    setup_standard_mocks
    
    # Set test environment variables
    export RESOURCE_NAME_PORT="8080"  # Replace with actual port
    export RESOURCE_NAME_CONTAINER_NAME="RESOURCE_NAME-test"
    export RESOURCE_NAME_BASE_URL="http://localhost:8080"
    
    # Create test directories if needed
    RESOURCE_NAME_DIR="$(dirname "$SCRIPT_DIR")"
    mkdir -p "/tmp/RESOURCE_NAME-test"
    
    # Configure resource-specific mocks
    mock::docker::set_container_state "RESOURCE_NAME-test" "running"
    mock::http::set_endpoint_state "http://localhost:8080" "healthy"
    
    # Load configuration and the script to test
    source "${RESOURCE_NAME_DIR}/config/defaults.sh"
    source "${RESOURCE_NAME_DIR}/config/messages.sh"
    RESOURCE_NAME::export_config
    RESOURCE_NAME::messages::init
    
    # Load the manage script
    MODULE_PATH="${RESOURCE_NAME_DIR}/manage.sh"
}

teardown() {
    # Clean up test environment
    cleanup_mocks
    rm -rf "/tmp/RESOURCE_NAME-test"
}

# Test script argument parsing and validation

@test "manage.sh should display help when called with --help" {
    run bash "$MODULE_PATH" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "--action" ]]
}

@test "manage.sh should display help when called with -h" {
    run bash "$MODULE_PATH" -h
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "manage.sh should display version when called with --version" {
    run bash "$MODULE_PATH" --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "RESOURCE_NAME" ]]
}

@test "manage.sh should exit with error when no arguments provided" {
    run bash "$MODULE_PATH"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Usage:" || "$output" =~ "Required:" ]]
}

@test "manage.sh should exit with error for invalid action" {
    run bash "$MODULE_PATH" --action invalid
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Invalid action" || "$output" =~ "Unknown action" ]]
}

# Test valid actions (customize based on your resource)

@test "manage.sh should accept install action" {
    run bash "$MODULE_PATH" --action install --yes yes
    # Customize assertion based on expected behavior
    [ "$status" -eq 0 ]
}

@test "manage.sh should accept status action" {
    run bash "$MODULE_PATH" --action status
    [ "$status" -eq 0 ]
}

@test "manage.sh should accept start action" {
    run bash "$MODULE_PATH" --action start
    # Customize assertion based on expected behavior
    [ "$status" -eq 0 ]
}

@test "manage.sh should accept stop action" {
    run bash "$MODULE_PATH" --action stop
    # Customize assertion based on expected behavior
    [ "$status" -eq 0 ]
}

@test "manage.sh should accept logs action" {
    run bash "$MODULE_PATH" --action logs
    [ "$status" -eq 0 ]
}

# Test argument combinations

@test "manage.sh should handle force flag" {
    run bash "$MODULE_PATH" --action install --force yes --yes yes
    [ "$status" -eq 0 ]
}

@test "manage.sh should handle dry-run mode" {
    run bash "$MODULE_PATH" --action install --dry-run yes
    [ "$status" -eq 0 ]
    [[ "$output" =~ "DRY RUN" || "$output" =~ "Would" ]]
}

# Test error handling

@test "manage.sh should handle missing dependencies gracefully" {
    # Mock missing dependency
    docker() { return 127; }
    export -f docker
    
    run bash "$MODULE_PATH" --action status
    [ "$status" -ne 0 ]
    [[ "$output" =~ "dependency" || "$output" =~ "Docker" || "$output" =~ "required" ]]
}

@test "manage.sh should validate required parameters" {
    # Test case where required parameter is missing (customize as needed)
    run bash "$MODULE_PATH" --action install
    # Some resources might require additional parameters
    # Customize this test based on your resource requirements
    [ "$status" -eq 0 ] || [[ "$output" =~ "required" ]]
}

# Test configuration loading

@test "manage.sh should load configuration correctly" {
    # Test that configuration is properly loaded
    run bash "$MODULE_PATH" --action status
    [ "$status" -eq 0 ]
    # Add assertions to verify configuration was loaded
}

@test "manage.sh should handle configuration errors" {
    # Mock configuration file error
    RESOURCE_NAME_CONFIG_FILE="/nonexistent/config"
    export RESOURCE_NAME_CONFIG_FILE
    
    run bash "$MODULE_PATH" --action status
    # Should either gracefully fallback to defaults or show clear error
    [[ "$status" -eq 0 || "$output" =~ "configuration" ]]
}

# Add resource-specific tests below:

# Example resource-specific tests (customize for your resource):

# @test "manage.sh should validate RESOURCE_NAME-specific parameters" {
#     run bash "$MODULE_PATH" --action install --resource-param invalid
#     [ "$status" -ne 0 ]
#     [[ "$output" =~ "Invalid parameter" ]]
# }

# @test "manage.sh should handle RESOURCE_NAME service interactions" {
#     run bash "$MODULE_PATH" --action health-check
#     [ "$status" -eq 0 ]
#     [[ "$output" =~ "healthy" || "$output" =~ "running" ]]
# }