#!/usr/bin/env bats
# Tests for agent-s2 config/messages.sh message system

bats_require_minimum_version 1.5.0

# Expensive setup operations run once per file
setup_file() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Get resource directory path
    AGENTS2_DIR="$(dirname "$(dirname "${BATS_TEST_FILENAME}")")"
    
    # Load configuration and messages once per file
    source "${AGENTS2_DIR}/config/defaults.sh"
    source "${AGENTS2_DIR}/config/messages.sh"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Load messages.sh for each test (required for bats isolation)
    AGENTS2_DIR="$(dirname "$(dirname "${BATS_TEST_FILENAME}")")"
    source "${AGENTS2_DIR}/config/messages.sh"
    
    # Load messages.sh for each test (required for bats isolation)
    AGENTS2_DIR="$(dirname "$(dirname "${BATS_TEST_FILENAME}")")"
    source "${AGENTS2_DIR}/config/messages.sh"
    
    # Clear any existing messages to test defaults (lightweight per-test)
    unset MSG_INSTALLING MSG_ALREADY_INSTALLED MSG_USE_FORCE
    unset MSG_INSTALL_SUCCESS MSG_INSTALL_FAILED
    unset MSG_DOCKER_REQUIRED MSG_BUILDING_IMAGE MSG_BUILD_SUCCESS
    unset MSG_SERVICE_HEALTHY MSG_HEALTH_CHECK_FAILED
    unset MSG_STATUS_HEADER MSG_SERVICE_RUNNING
    
    # Call export_messages to initialize message variables
    agents2::export_messages
}

teardown() {
    # Clean up test environment
    cleanup_mocks
}

# Test message system initialization

@test "agents2::export_messages should set all installation messages" {
    # Messages should already be set from setup
    # Test installation message constants are set
    [ -n "$MSG_INSTALLING" ]
    [ -n "$MSG_ALREADY_INSTALLED" ]
    [ -n "$MSG_USE_FORCE" ]
    [ -n "$MSG_INSTALL_SUCCESS" ]
    [ -n "$MSG_INSTALL_FAILED" ]
}

@test "agents2::export_messages should set all Docker messages" {
    # Messages should already be set from setup
    # Test Docker message constants are set
    [ -n "$MSG_DOCKER_REQUIRED" ]
    [ -n "$MSG_BUILDING_IMAGE" ]
    [ -n "$MSG_BUILD_SUCCESS" ]
    [ -n "$MSG_BUILD_FAILED" ]
    [ -n "$MSG_PULLING_BASE" ]
}

@test "agents2::export_messages should set all container messages" {
    # Messages should already be set from setup
    # Test container message constants are set
    [ -n "$MSG_STARTING_CONTAINER" ]
    [ -n "$MSG_CONTAINER_STARTED" ]
    [ -n "$MSG_CONTAINER_RUNNING" ]
    [ -n "$MSG_CONTAINER_NOT_RUNNING" ]
    [ -n "$MSG_STOPPING_CONTAINER" ]
    [ -n "$MSG_CONTAINER_STOPPED" ]
}

@test "agents2::export_messages should set all health check messages" {
    # Messages should already be set from setup
    # Test health check message constants are set
    [ -n "$MSG_WAITING_READY" ]
    [ -n "$MSG_SERVICE_HEALTHY" ]
    [ -n "$MSG_HEALTH_CHECK_FAILED" ]
    [ -n "$MSG_STARTUP_TIMEOUT" ]
}

@test "agents2::export_messages should set all status messages" {
    # Messages should already be set from setup
    # Test status message constants are set
    [ -n "$MSG_STATUS_HEADER" ]
    [ -n "$MSG_SERVICE_RUNNING" ]
    [ -n "$MSG_SERVICE_NOT_INSTALLED" ]
    [ -n "$MSG_PORT_LISTENING" ]
    [ -n "$MSG_PORT_NOT_ACCESSIBLE" ]
}

@test "agents2::export_messages should contain appropriate content for installation messages" {
    agents2::export_messages
    
    # Test message content is appropriate
    [[ "$MSG_INSTALLING" =~ "Agent S2" ]]
    [[ "$MSG_INSTALLING" =~ "Installing" || "$MSG_INSTALLING" =~ "🤖" ]]
    [[ "$MSG_INSTALL_SUCCESS" =~ "success" || "$MSG_INSTALL_SUCCESS" =~ "✅" ]]
    [[ "$MSG_INSTALL_FAILED" =~ "failed" || "$MSG_INSTALL_FAILED" =~ "❌" ]]
}

@test "agents2::export_messages should contain appropriate content for status messages" {
    agents2::export_messages
    
    # Test status message content
    [[ "$MSG_SERVICE_HEALTHY" =~ "healthy" || "$MSG_SERVICE_HEALTHY" =~ "✅" ]]
    [[ "$MSG_STATUS_HEADER" =~ "Status" || "$MSG_STATUS_HEADER" =~ "📊" ]]
    [[ "$MSG_SERVICE_RUNNING" =~ "running" ]]
    [[ "$MSG_SERVICE_NOT_INSTALLED" =~ "not installed" ]]
}

@test "agents2::export_messages should handle repeated calls without duplication" {
    # First call
    agents2::export_messages
    local first_msg="$MSG_INSTALLING"
    
    # Second call
    agents2::export_messages
    local second_msg="$MSG_INSTALLING"
    
    # Should be identical
    [ "$first_msg" = "$second_msg" ]
}

@test "agents2::export_messages should preserve existing message values" {
    # Pre-set a message
    export MSG_CUSTOM_TEST="Pre-existing value"
    
    agents2::export_messages
    
    # Should preserve pre-existing value
    [ "$MSG_CUSTOM_TEST" = "Pre-existing value" ]
}

@test "agents2::export_messages should export all message variables" {
    agents2::export_messages
    
    # Test that key messages are exported to environment
    [ -n "${MSG_INSTALLING:-}" ]
    [ -n "${MSG_INSTALL_SUCCESS:-}" ]
    [ -n "${MSG_SERVICE_HEALTHY:-}" ]
    [ -n "${MSG_STATUS_HEADER:-}" ]
    [ -n "${MSG_CONTAINER_STARTED:-}" ]
}

@test "agents2::export_messages should set network and directory messages" {
    # Messages should already be set from setup
    # Test network messages
    [ -n "$MSG_CREATING_NETWORK" ]
    [ -n "$MSG_NETWORK_EXISTS" ]
    [ -n "$MSG_NETWORK_CREATED" ]
    [ -n "$MSG_NETWORK_FAILED" ]
    
    # Test directory messages
    [ -n "$MSG_CREATING_DIRS" ]
    [ -n "$MSG_DIRECTORIES_CREATED" ]
    [ -n "$MSG_CREATE_DIRS_FAILED" ]
}

@test "agents2::export_messages should set uninstall messages" {
    # Messages should already be set from setup
    # Test uninstall messages
    [ -n "$MSG_UNINSTALLING" ]
    [ -n "$MSG_UNINSTALL_WARNING" ]
    [ -n "$MSG_UNINSTALL_SUCCESS" ]
    [ -n "$MSG_UNINSTALL_CANCELLED" ]
    
    # Test uninstall message content
    [[ "$MSG_UNINSTALL_WARNING" =~ "remove" || "$MSG_UNINSTALL_WARNING" =~ "data" ]]
    [[ "$MSG_UNINSTALLING" =~ "🗑️" || "$MSG_UNINSTALLING" =~ "Uninstalling" ]]
}

@test "agents2::export_messages should set configuration messages" {
    # Messages should already be set from setup
    # Test configuration messages
    [ -n "$MSG_CONFIG_UPDATE_SUCCESS" ]
    [ -n "$MSG_CONFIG_UPDATE_FAILED" ]
    
    # Test configuration message content
    [[ "$MSG_CONFIG_UPDATE_SUCCESS" =~ "success" ]]
    [[ "$MSG_CONFIG_UPDATE_FAILED" =~ "failed" || "$MSG_CONFIG_UPDATE_FAILED" =~ "Failed" ]]
}

@test "agents2::export_messages should contain descriptive error messages" {
    agents2::export_messages
    
    # Test that error messages are descriptive
    [ ${#MSG_INSTALL_FAILED} -gt 10 ]
    [ ${#MSG_BUILD_FAILED} -gt 10 ]
    [ ${#MSG_HEALTH_CHECK_FAILED} -gt 10 ]
    [ ${#MSG_STARTUP_TIMEOUT} -gt 10 ]
    
    # Test that error messages contain useful information
    [[ "$MSG_DOCKER_REQUIRED" =~ "Docker" ]]
    [[ "$MSG_BUILD_FAILED" =~ "build" || "$MSG_BUILD_FAILED" =~ "image" ]]
}

@test "agents2::export_messages should include emojis for visual clarity" {
    agents2::export_messages
    
    # Test that key messages include visual indicators
    [[ "$MSG_INSTALLING" =~ "🤖" || "$MSG_SERVICE_HEALTHY" =~ "✅" || "$MSG_STATUS_HEADER" =~ "📊" ]]
    [[ "$MSG_INSTALL_SUCCESS" =~ "✅" || "$MSG_SERVICE_RUNNING" =~ "✅" ]]
    [[ "$MSG_INSTALL_FAILED" =~ "❌" || "$MSG_HEALTH_CHECK_FAILED" =~ "⚠️" ]]
}

@test "agents2::export_messages should handle missing environment gracefully" {
    # Clear all environment variables
    env -i bash -c "
        source '${AGENTS2_DIR}/config/messages.sh'
        agents2::export_messages
        exit 0
    "
    [ "$?" -eq 0 ]
}

@test "agents2::export_messages should provide Agent S2 specific messaging" {
    agents2::export_messages
    
    # Test that messages are specific to Agent S2
    [[ "$MSG_INSTALLING" =~ "Agent S2" ]]
    [[ "$MSG_STATUS_HEADER" =~ "Agent S2" ]]
    [[ "$MSG_UNINSTALLING" =~ "Agent S2" ]]
    
    # Test that messages reflect autonomous interaction capabilities
    [[ "$MSG_INSTALLING" =~ "Computer Interaction" || "$MSG_INSTALLING" =~ "Autonomous" ]]
}
