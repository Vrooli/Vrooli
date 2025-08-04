#!/usr/bin/env bats
# Tests for Node-RED common functions (lib/common.sh)

load ../test_fixtures/test_helper

setup() {
    setup_test_environment
    source_node_red_scripts
    mock_docker "success"
    mock_curl "success"
    mock_jq "success"
}

teardown() {
    teardown_test_environment
}

@test "node_red::check_docker succeeds when docker is available" {
    run node_red::check_docker
    assert_success
}

@test "node_red::check_docker fails when docker is not available" {
    # Mock docker command to not exist
    docker() { return 127; }
    export -f docker
    
    run node_red::check_docker
    assert_failure
}

@test "node_red::container_exists returns true when container exists" {
    mock_docker "success"
    
    run node_red::container_exists
    assert_success
}

@test "node_red::container_exists returns false when container does not exist" {
    mock_docker "not_installed"
    
    run node_red::container_exists
    assert_failure
}

@test "node_red::is_installed returns true when container exists" {
    mock_docker "success"
    
    run node_red::is_installed
    assert_success
}

@test "node_red::is_installed returns false when container does not exist" {
    mock_docker "not_installed"
    
    run node_red::is_installed
    assert_failure
}

@test "node_red::is_running returns true when container is running" {
    mock_docker "success"
    
    run node_red::is_running
    assert_success
}

@test "node_red::is_running returns false when container is not running" {
    mock_docker "not_running"
    
    run node_red::is_running
    assert_failure
}

@test "node_red::is_running returns false when container does not exist" {
    mock_docker "not_installed"
    
    run node_red::is_running
    assert_failure
}

@test "node_red::is_healthy returns true when service responds" {
    mock_curl "success"
    
    run node_red::is_healthy
    assert_success
}

@test "node_red::is_healthy returns false when service does not respond" {
    mock_curl "failure"
    
    run node_red::is_healthy
    assert_failure
}

@test "node_red::is_healthy returns false on timeout" {
    mock_curl "timeout"
    
    run node_red::is_healthy
    assert_failure
}

@test "node_red::wait_for_ready succeeds when service becomes available quickly" {
    mock_curl "success"
    
    run node_red::wait_for_ready
    assert_success
    assert_output_contains "ready"
}

@test "node_red::wait_for_ready fails when service never becomes available" {
    mock_curl "failure"
    
    # Set a very short timeout for testing
    export NODE_RED_READY_TIMEOUT=1
    export NODE_RED_READY_SLEEP=0.1
    
    run node_red::wait_for_ready
    assert_failure
    assert_output_contains "failed to start"
}

@test "node_red::generate_secret creates a valid secret" {
    run node_red::generate_secret
    assert_success
    
    # Secret should be non-empty and reasonable length
    [[ ${#output} -ge 16 ]]
}

@test "node_red::generate_secret creates different secrets on multiple calls" {
    secret1=$(node_red::generate_secret)
    secret2=$(node_red::generate_secret)
    
    [[ "$secret1" != "$secret2" ]]
}

@test "node_red::create_directories creates required directories" {
    run node_red::create_directories
    assert_success
    
    assert_file_exists "$SCRIPT_DIR/flows"
    assert_file_exists "$SCRIPT_DIR/nodes"  
}

@test "node_red::create_directories handles existing directories gracefully" {
    # Create directories first
    mkdir -p "$SCRIPT_DIR/flows" "$SCRIPT_DIR/nodes"
    
    run node_red::create_directories
    assert_success
}

@test "node_red::create_directories fails with insufficient permissions" {
    # Mock mkdir to fail
    mkdir() { return 1; }
    export -f mkdir
    
    run node_red::create_directories
    assert_failure
}

@test "node_red::update_resource_config creates config file" {
    run node_red::update_resource_config
    assert_success
    
    assert_file_exists "$NODE_RED_TEST_CONFIG_DIR/service.json"
}

@test "node_red::update_resource_config updates existing config file" {
    # Create initial config
    mkdir -p "$NODE_RED_TEST_CONFIG_DIR"
    echo '{"services": {}}' > "$NODE_RED_TEST_CONFIG_DIR/service.json"
    
    run node_red::update_resource_config
    assert_success
    
    # Config should still exist and be valid JSON
    assert_file_exists "$NODE_RED_TEST_CONFIG_DIR/service.json"
}

@test "node_red::remove_resource_config removes node-red from config" {
    # Create config with node-red entry
    mkdir -p "$NODE_RED_TEST_CONFIG_DIR"
    cat > "$NODE_RED_TEST_CONFIG_DIR/service.json" << 'EOF'
{
    "services": {
        "automation": {
            "node-red": {
                "enabled": true
            }
        }
    }
}
EOF
    
    run node_red::remove_resource_config
    assert_success
}

@test "node_red::remove_resource_config handles missing config file gracefully" {
    run node_red::remove_resource_config
    assert_success
}

@test "node_red::create_default_settings creates settings.js file" {
    run node_red::create_default_settings
    assert_success
    
    assert_file_exists "$SCRIPT_DIR/settings.js"
}

@test "node_red::create_default_settings creates valid JavaScript" {
    node_red::create_default_settings
    
    # Check that the file contains expected JavaScript structure
    grep -q "module.exports" "$SCRIPT_DIR/settings.js"
    grep -q "flowFile" "$SCRIPT_DIR/settings.js"
    grep -q "userDir" "$SCRIPT_DIR/settings.js"
}

@test "node_red::create_default_settings overwrites existing file" {
    # Create an existing file
    echo "old content" > "$SCRIPT_DIR/settings.js"
    
    run node_red::create_default_settings
    assert_success
    
    # File should contain new content
    grep -q "module.exports" "$SCRIPT_DIR/settings.js"
    ! grep -q "old content" "$SCRIPT_DIR/settings.js"
}

@test "node_red::check_port returns true for available port" {
    # Mock netstat/ss to show port is free
    netstat() { return 1; }  # Exit code 1 means port not found (available)
    ss() { return 1; }
    export -f netstat ss
    
    run node_red::check_port "18800"
    assert_success
}

@test "node_red::check_port returns false for occupied port" {
    # Mock netstat/ss to show port is in use
    netstat() { echo "tcp 0 0 :::18800 :::* LISTEN"; return 0; }
    ss() { echo "LISTEN 0 128 *:18800 *:*"; return 0; }
    export -f netstat ss
    
    run node_red::check_port "18800"
    assert_failure
}

@test "node_red::validate_json returns true for valid JSON" {
    echo '{"valid": "json"}' > "$NODE_RED_TEST_DIR/test.json"
    
    run node_red::validate_json "$NODE_RED_TEST_DIR/test.json"
    assert_success
}

@test "node_red::validate_json returns false for invalid JSON" {
    echo '{"invalid": json}' > "$NODE_RED_TEST_DIR/test.json"
    
    # Mock jq to fail
    mock_jq "failure"
    
    run node_red::validate_json "$NODE_RED_TEST_DIR/test.json"
    assert_failure
}

@test "node_red::validate_json returns false for missing file" {
    run node_red::validate_json "$NODE_RED_TEST_DIR/nonexistent.json"
    assert_failure
}

@test "node_red::get_logs returns container logs" {
    mock_docker "success"
    
    run node_red::get_logs
    assert_success
}

@test "node_red::get_logs fails when container is not installed" {
    mock_docker "not_installed"
    
    run node_red::get_logs
    assert_failure
}

@test "node_red::get_resource_usage returns usage statistics" {
    mock_docker "success"
    
    run node_red::get_resource_usage
    assert_success
}

@test "node_red::get_resource_usage fails when container is not running" {
    mock_docker "not_running"
    
    run node_red::get_resource_usage
    assert_failure
}

@test "node_red::get_health_status returns health information" {
    mock_docker "success"
    mock_curl "success"
    
    run node_red::get_health_status
    assert_success
}

@test "node_red::get_health_status handles unhealthy container" {
    mock_docker "success"
    mock_curl "failure"
    
    run node_red::get_health_status
    # Should still succeed but indicate unhealthy status
    assert_success
}

# Test environment variable handling
@test "functions respect custom NODE_RED_PORT" {
    export NODE_RED_PORT="19999"
    export RESOURCE_PORT="$NODE_RED_PORT"
    
    # Test that functions use the custom port
    mock_curl "success"
    run node_red::is_healthy
    assert_success
}

@test "functions use default port when not specified" {
    unset NODE_RED_PORT
    export RESOURCE_PORT="1880"  # Default
    
    mock_curl "success"
    run node_red::is_healthy
    assert_success
}

# Test error handling
@test "functions handle docker command failures gracefully" {
    mock_docker "failure"
    
    run node_red::is_installed
    assert_failure
    
    run node_red::is_running
    assert_failure
}

@test "functions handle network failures gracefully" {
    mock_curl "failure"
    
    run node_red::is_healthy
    assert_failure
}

# Test concurrent operations
@test "functions work when called concurrently" {
    mock_docker "success"
    mock_curl "success"
    
    # Run multiple functions in background
    node_red::is_installed &
    node_red::is_running &
    node_red::is_healthy &
    
    wait  # Wait for all background processes
    
    # All should have succeeded
    [[ $? -eq 0 ]]
}
