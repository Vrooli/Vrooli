#!/usr/bin/env bats
# Tests for config/defaults.sh

load ../test_fixtures/test_helper

setup() {
    setup_test_environment
    source_node_red_scripts
}

teardown() {
    teardown_test_environment
}

# Tests for config/defaults.sh
@test "defaults.sh exports all required constants" {
    # Source just the defaults to test
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    
    # Check that all required constants are defined
    [[ -n "$NODE_RED_PORT" ]]
    [[ -n "$NODE_RED_BASE_URL" ]]
    [[ -n "$CONTAINER_NAME" ]]
    [[ -n "$IMAGE_NAME" ]]
    [[ -n "$VOLUME_NAME" ]]
    [[ -n "$NETWORK_NAME" ]]
    [[ -n "$OFFICIAL_IMAGE" ]]
}

@test "defaults.sh uses correct default port" {
    unset NODE_RED_CUSTOM_PORT
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    
    [[ "$NODE_RED_PORT" == "1880" ]]
}

@test "defaults.sh respects custom port environment variable" {
    export NODE_RED_CUSTOM_PORT="8080"
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    
    [[ "$NODE_RED_PORT" == "8080" ]]
}

@test "defaults.sh creates correct base URL" {
    export NODE_RED_CUSTOM_PORT="9999"
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    
    [[ "$NODE_RED_BASE_URL" == "http://localhost:9999" ]]
}

@test "defaults.sh defines proper resource names" {
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    
    [[ "$CONTAINER_NAME" == "node-red" ]]
    [[ "$IMAGE_NAME" == "node-red-vrooli:latest" ]]
    [[ "$VOLUME_NAME" == "node-red-data" ]]
    [[ "$NETWORK_NAME" == "vrooli-network" ]]
}

@test "defaults.sh sets appropriate timeout values" {
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    
    [[ -n "$NODE_RED_HEALTH_CHECK_TIMEOUT" ]]
    [[ -n "$NODE_RED_API_TIMEOUT" ]]
    [[ -n "$HTTP_REQUEST_TIMEOUT" ]]
    
    # Timeouts should be reasonable numbers
    [[ "$NODE_RED_HEALTH_CHECK_TIMEOUT" -gt 0 ]]
    [[ "$NODE_RED_API_TIMEOUT" -gt 0 ]]
    [[ "$HTTP_REQUEST_TIMEOUT" -gt 0 ]]
}

@test "defaults.sh creates proper file paths" {
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    
    [[ -n "$DEFAULT_FLOW_FILE" ]]
    
    # File names should be valid
    [[ "$DEFAULT_FLOW_FILE" =~ \.json$ ]]
}

@test "defaults.sh generates secure secret" {
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    
    [[ -n "$DEFAULT_SECRET" ]]
    # Secret should be reasonably long
    [[ ${#DEFAULT_SECRET} -ge 16 ]]
}

@test "node_red::export_config exports all configuration" {
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    
    node_red::export_config
    
    # Check that key variables are available after export
    [[ -n "$NODE_RED_PORT" ]]
    [[ -n "$NODE_RED_BASE_URL" ]]
    [[ -n "$CONTAINER_NAME" ]]
}

@test "node_red::export_config makes variables available to subprocesses" {
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    node_red::export_config
    
    # Test that exported variables are available in subshells
    local port_in_subshell=$(bash -c 'echo $NODE_RED_PORT')
    [[ "$port_in_subshell" == "$NODE_RED_PORT" ]]
}

@test "config handles missing environment variables gracefully" {
    # Clear all Node-RED related environment variables
    unset NODE_RED_PORT NODE_RED_CUSTOM_PORT RESOURCE_PORT
    
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    
    # Should still set default values
    [[ -n "$NODE_RED_PORT" ]]
    [[ "$NODE_RED_PORT" == "1880" ]]
}

@test "config respects system environment overrides" {
    # Set system-wide overrides
    export NODE_RED_CUSTOM_PORT="5000"
    export TZ="America/New_York"
    
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    
    [[ "$NODE_RED_PORT" == "5000" ]]
    [[ "$NODE_RED_BASE_URL" == "http://localhost:5000" ]]
}

@test "config provides comprehensive Docker settings" {
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    
    # Check Docker-related configuration
    [[ -n "$CONTAINER_NAME" ]]
    [[ -n "$IMAGE_NAME" ]]
    [[ -n "$VOLUME_NAME" ]]
    [[ -n "$NETWORK_NAME" ]]
    [[ -n "$OFFICIAL_IMAGE" ]]
    
    # Names should follow Docker conventions
    [[ ! "$CONTAINER_NAME" =~ [^a-zA-Z0-9_.-] ]]
    [[ ! "$VOLUME_NAME" =~ [^a-zA-Z0-9_.-] ]]
    [[ ! "$NETWORK_NAME" =~ [^a-zA-Z0-9_.-] ]]
}

@test "config provides resource metadata" {
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    
    [[ -n "$RESOURCE_NAME" ]]
    [[ -n "$RESOURCE_CATEGORY" ]]
    [[ -n "$RESOURCE_DESC" ]]
    
    [[ "$RESOURCE_NAME" == "node-red" ]]
    [[ "$RESOURCE_CATEGORY" == "automation" ]]
}

@test "config includes comprehensive timeout settings" {
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    
    # All timeouts should be positive integers
    [[ "$NODE_RED_HEALTH_CHECK_TIMEOUT" =~ ^[0-9]+$ ]]
    [[ "$NODE_RED_API_TIMEOUT" =~ ^[0-9]+$ ]]
    [[ "$HTTP_REQUEST_TIMEOUT" =~ ^[0-9]+$ ]]
    
    # Timeouts should be reasonable
    [[ "$NODE_RED_HEALTH_CHECK_TIMEOUT" -le 300 ]]  # Max 5 minutes
    [[ "$NODE_RED_API_TIMEOUT" -le 300 ]]     # Max 5 minutes
    [[ "$HTTP_REQUEST_TIMEOUT" -le 300000 ]]  # Max 5 minutes in ms
}

@test "config handles invalid port numbers gracefully" {
    export NODE_RED_CUSTOM_PORT="invalid"
    
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    
    # Configuration uses the value as-is (validation happens elsewhere)
    [[ "$NODE_RED_PORT" == "invalid" ]]
}

@test "config handles extreme port numbers" {
    export NODE_RED_CUSTOM_PORT="99999"
    
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    
    [[ "$NODE_RED_PORT" == "99999" ]]
    [[ "$NODE_RED_BASE_URL" == "http://localhost:99999" ]]
}

@test "config secret generation is secure" {
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    
    # Check the default secret
    [[ -n "$DEFAULT_SECRET" ]]
    
    # Should be long enough
    [[ ${#DEFAULT_SECRET} -ge 32 ]]
    
    # Should contain hex characters (or be the fallback)
    [[ "$DEFAULT_SECRET" =~ ^[a-f0-9]+$ || "$DEFAULT_SECRET" == "default-insecure-secret" ]]
}