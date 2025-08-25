#!/usr/bin/env bats
# Tests for minio config/messages.sh message system

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true


# Setup for each test
setup() {
    # Setup mock framework
    export BATS_TEST_TMPDIR="${BATS_TEST_TMPDIR:-$(mktemp -d)}"
    export MOCK_RESPONSES_DIR="$BATS_TEST_TMPDIR/mock_responses"
    mkdir -p "$MOCK_RESPONSES_DIR"
    
    # Load mock framework
    MOCK_DIR="${BATS_TEST_DIRNAME}/../../../tests/bats-fixtures/mocks"
    source "$MOCK_DIR/system_mocks.bash"
    source "$MOCK_DIR/mock_helpers.bash"
    source "$MOCK_DIR/resource_mocks.bash"
    
    # Set test environment
    export HOME="/tmp/test-home"
    export MINIO_CONSOLE_URL="http://localhost:9001"
    mkdir -p "$HOME"
    
    # Clear any existing messages to test initialization
    unset MSG_INSTALL_SUCCESS MSG_START_SUCCESS MSG_STOP_SUCCESS
    unset MSG_RESTART_SUCCESS MSG_UNINSTALL_SUCCESS MSG_BUCKET_CREATED
    unset MSG_CREDENTIALS_GENERATED MSG_BUCKETS_INITIALIZED
    unset MSG_HEALTHY MSG_RUNNING MSG_CONSOLE_AVAILABLE
    unset MSG_CHECKING_DOCKER MSG_CHECKING_PORTS MSG_PULLING_IMAGE
    unset MSG_CREATING_DIRECTORIES MSG_STARTING_CONTAINER
    unset MSG_WAITING_STARTUP MSG_CREATING_BUCKETS MSG_CONFIGURING_POLICIES
    unset MSG_PORT_IN_USE MSG_CONTAINER_EXISTS MSG_LOW_DISK_SPACE
    unset MSG_USING_DEFAULT_CREDS MSG_DOCKER_NOT_FOUND MSG_INSTALL_FAILED
    unset MSG_START_FAILED MSG_HEALTH_CHECK_FAILED MSG_BUCKET_CREATE_FAILED
    unset MSG_INSUFFICIENT_DISK MSG_HELP_ACCESS MSG_HELP_CREDENTIALS
    unset MSG_HELP_PORT_CONFLICT
    
    # Get resource directory path
    MINIO_DIR="$(dirname "${BATS_TEST_DIRNAME}")"
    
    # Load configuration and messages
    source "${MINIO_DIR}/config/defaults.sh"
    source "${MINIO_DIR}/config/messages.sh"
    
    # Initialize message system
    minio::messages::init
}

teardown() {
    # Clean up test environment
    trash::safe_remove "/tmp/test-home" --test-cleanup
    [[ -d "$MOCK_RESPONSES_DIR" ]] && trash::safe_remove "$MOCK_RESPONSES_DIR" --test-cleanup
}

# Test message system initialization

@test "minio::messages::init should set all success messages" {
    # Messages should already be initialized from setup
    [ -n "$MSG_INSTALL_SUCCESS" ]
    [ -n "$MSG_START_SUCCESS" ]
    [ -n "$MSG_STOP_SUCCESS" ]
    [ -n "$MSG_RESTART_SUCCESS" ]
    [ -n "$MSG_UNINSTALL_SUCCESS" ]
    [ -n "$MSG_BUCKET_CREATED" ]
    [ -n "$MSG_CREDENTIALS_GENERATED" ]
    [ -n "$MSG_BUCKETS_INITIALIZED" ]
}

@test "minio::messages::init should set all status messages" {
    # Test status message constants are set
    [ -n "$MSG_HEALTHY" ]
    [ -n "$MSG_RUNNING" ]
    [ -n "$MSG_CONSOLE_AVAILABLE" ]
}

@test "minio::messages::init should set all info messages" {
    # Test info message constants are set
    [ -n "$MSG_CHECKING_DOCKER" ]
    [ -n "$MSG_CHECKING_PORTS" ]
    [ -n "$MSG_PULLING_IMAGE" ]
    [ -n "$MSG_CREATING_DIRECTORIES" ]
    [ -n "$MSG_STARTING_CONTAINER" ]
    [ -n "$MSG_WAITING_STARTUP" ]
    [ -n "$MSG_CREATING_BUCKETS" ]
    [ -n "$MSG_CONFIGURING_POLICIES" ]
}

@test "minio::messages::init should set all warning messages" {
    # Test warning message constants are set
    [ -n "$MSG_PORT_IN_USE" ]
    [ -n "$MSG_CONTAINER_EXISTS" ]
    [ -n "$MSG_LOW_DISK_SPACE" ]
    [ -n "$MSG_USING_DEFAULT_CREDS" ]
}

@test "minio::messages::init should set all error messages" {
    # Test error message constants are set
    [ -n "$MSG_DOCKER_NOT_FOUND" ]
    [ -n "$MSG_INSTALL_FAILED" ]
    [ -n "$MSG_START_FAILED" ]
    [ -n "$MSG_HEALTH_CHECK_FAILED" ]
    [ -n "$MSG_BUCKET_CREATE_FAILED" ]
    [ -n "$MSG_INSUFFICIENT_DISK" ]
}

@test "minio::messages::init should set all help messages" {
    # Test help message constants are set
    [ -n "$MSG_HELP_ACCESS" ]
    [ -n "$MSG_HELP_CREDENTIALS" ]
    [ -n "$MSG_HELP_PORT_CONFLICT" ]
}

@test "minio::messages::init should contain appropriate content for success messages" {
    # Test message content is appropriate
    [[ "$MSG_INSTALL_SUCCESS" =~ "MinIO" && "$MSG_INSTALL_SUCCESS" =~ "success" ]]
    [[ "$MSG_START_SUCCESS" =~ "MinIO" && "$MSG_START_SUCCESS" =~ "success" ]]
    [[ "$MSG_INSTALL_SUCCESS" =~ "✅" ]]
    [[ "$MSG_START_SUCCESS" =~ "✅" ]]
    [[ "$MSG_BUCKET_CREATED" =~ "Bucket" && "$MSG_BUCKET_CREATED" =~ "success" ]]
}

@test "minio::messages::init should contain appropriate content for status messages" {
    # Test status message content
    [[ "$MSG_HEALTHY" =~ "healthy" || "$MSG_HEALTHY" =~ "✅" ]]
    [[ "$MSG_RUNNING" =~ "running" || "$MSG_RUNNING" =~ "✅" ]]
    [[ "$MSG_CONSOLE_AVAILABLE" =~ "Console" ]]
}

@test "minio::messages::init should contain appropriate content for error messages" {
    # Test error message content
    [[ "$MSG_INSTALL_FAILED" =~ "failed" ]]
    [[ "$MSG_START_FAILED" =~ "Failed" ]]
    [[ "$MSG_DOCKER_NOT_FOUND" =~ "Docker" ]]
    [[ "$MSG_HEALTH_CHECK_FAILED" =~ "health check" && "$MSG_HEALTH_CHECK_FAILED" =~ "failed" ]]
}

@test "minio::messages::init should handle repeated calls without duplication" {
    # First call
    minio::messages::init
    local first_msg="$MSG_INSTALL_SUCCESS"
    
    # Second call
    minio::messages::init
    local second_msg="$MSG_INSTALL_SUCCESS"
    
    # Should be identical
    [ "$first_msg" = "$second_msg" ]
}

@test "minio::messages::init should preserve existing message values" {
    # Test that readonly variables are preserved
    local original_msg="$MSG_INSTALL_SUCCESS"
    
    # Try to reinitialize
    minio::messages::init
    
    # Should preserve original value (readonly protection)
    [ "$MSG_INSTALL_SUCCESS" = "$original_msg" ]
}

@test "minio::messages::init should contain MinIO-specific terminology" {
    # Test that messages are specific to MinIO
    [[ "$MSG_INSTALL_SUCCESS" =~ "MinIO" ]]
    [[ "$MSG_HEALTHY" =~ "MinIO" ]]
    [[ "$MSG_RUNNING" =~ "MinIO" ]]
    [[ "$MSG_CONSOLE_AVAILABLE" =~ "MinIO Console" ]]
}

@test "minio::messages::init should include bucket-related messages" {
    # Test bucket-specific messages
    [[ "$MSG_BUCKET_CREATED" =~ "Bucket" ]]
    [[ "$MSG_CREATING_BUCKETS" =~ "bucket" ]]
    [[ "$MSG_BUCKET_CREATE_FAILED" =~ "bucket" && "$MSG_BUCKET_CREATE_FAILED" =~ "Failed" ]]
    [[ "$MSG_BUCKETS_INITIALIZED" =~ "bucket" ]]
}

@test "minio::messages::init should include security warnings" {
    # Test security-related messages
    [[ "$MSG_USING_DEFAULT_CREDS" =~ "default credentials" ]]
    [[ "$MSG_USING_DEFAULT_CREDS" =~ "production" ]]
    [[ "$MSG_USING_DEFAULT_CREDS" =~ "⚠️" ]]
    [[ "$MSG_CREDENTIALS_GENERATED" =~ "credentials" ]]
}

@test "minio::messages::init should include storage-related messages" {
    # Test storage-specific messages
    [[ "$MSG_LOW_DISK_SPACE" =~ "disk space" ]]
    [[ "$MSG_INSUFFICIENT_DISK" =~ "disk space" ]]
    [[ "$MSG_CREATING_DIRECTORIES" =~ "director" ]]
}

@test "minio::messages::init should include Docker-related messages" {
    # Test Docker-specific messages
    [[ "$MSG_CHECKING_DOCKER" =~ "Docker" ]]
    [[ "$MSG_DOCKER_NOT_FOUND" =~ "Docker" ]]
    [[ "$MSG_PULLING_IMAGE" =~ "image" ]]
    [[ "$MSG_STARTING_CONTAINER" =~ "container" ]]
    [[ "$MSG_CONTAINER_EXISTS" =~ "container" ]]
}

@test "minio::messages::init should include emojis for visual clarity" {
    # Test that key messages include visual indicators
    [[ "$MSG_INSTALL_SUCCESS" =~ "✅" ]]
    [[ "$MSG_START_SUCCESS" =~ "✅" ]]
    [[ "$MSG_HEALTHY" =~ "✅" ]]
    [[ "$MSG_USING_DEFAULT_CREDS" =~ "⚠️" ]]
}

@test "minio::messages::init should handle missing environment gracefully" {
    # Test initialization without environment variables
    result=$(env -i bash -c "
        source '${MINIO_DIR}/config/messages.sh' 2>/dev/null
        minio::messages::init 2>/dev/null
        echo \$MSG_INSTALL_SUCCESS
    ")
    
    # Should still initialize messages
    [[ "$result" =~ "MinIO" && "$result" =~ "success" ]]
}

@test "minio::messages::init should include help content" {
    # Test help message content
    [[ "$MSG_HELP_ACCESS" =~ "Console" ]]
    [[ "$MSG_HELP_CREDENTIALS" =~ "credentials" ]]
    [[ "$MSG_HELP_PORT_CONFLICT" =~ "port" ]]
    [[ "$MSG_HELP_PORT_CONFLICT" =~ "MINIO_CUSTOM_PORT" ]]
}

@test "minio::messages::init should handle port configuration in help" {
    # Test that help messages use configuration
    [[ "$MSG_HELP_ACCESS" =~ "localhost" ]]
    # Should contain port reference (either default or variable)
    [[ "$MSG_HELP_ACCESS" =~ "900" || "$MSG_HELP_ACCESS" =~ "MINIO_CONSOLE_URL" ]]
}

@test "minio::messages::init should provide actionable error messages" {
    # Test that error messages are actionable
    [ ${#MSG_INSTALL_FAILED} -gt 10 ]
    [ ${#MSG_START_FAILED} -gt 10 ]
    [ ${#MSG_DOCKER_NOT_FOUND} -gt 20 ]
    
    # Test that messages contain useful information
    [[ "$MSG_DOCKER_NOT_FOUND" =~ "Docker" ]]
    [[ "$MSG_PORT_IN_USE" =~ "Port" ]]
}

@test "minio::messages::init should provide comprehensive bucket messages" {
    # Test bucket operation messages
    [[ "$MSG_CREATING_BUCKETS" =~ "Creating" && "$MSG_CREATING_BUCKETS" =~ "bucket" ]]
    [[ "$MSG_CONFIGURING_POLICIES" =~ "Configuring" && "$MSG_CONFIGURING_POLICIES" =~ "policies" ]]
    [[ "$MSG_BUCKETS_INITIALIZED" =~ "initialized" ]]
}

@test "minio::messages::init should provide startup sequence messages" {
    # Test startup sequence messages in logical order
    [ -n "$MSG_CHECKING_DOCKER" ]
    [ -n "$MSG_CHECKING_PORTS" ]
    [ -n "$MSG_PULLING_IMAGE" ]
    [ -n "$MSG_CREATING_DIRECTORIES" ]
    [ -n "$MSG_STARTING_CONTAINER" ]
    [ -n "$MSG_WAITING_STARTUP" ]
    [ -n "$MSG_CREATING_BUCKETS" ]
    
    # Test that messages reflect startup sequence
    [[ "$MSG_CHECKING_DOCKER" =~ "Checking" ]]
    [[ "$MSG_WAITING_STARTUP" =~ "Waiting" ]]
}

@test "minio::messages::init should handle credential generation messages" {
    # Test credential-related messages
    [[ "$MSG_CREDENTIALS_GENERATED" =~ "credentials" && "$MSG_CREDENTIALS_GENERATED" =~ "generated" ]]
    [[ "$MSG_HELP_CREDENTIALS" =~ "show-credentials" ]]
    [[ "$MSG_USING_DEFAULT_CREDS" =~ "not secure" ]]
}
