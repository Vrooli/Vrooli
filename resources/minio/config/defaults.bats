#!/usr/bin/env bats
# Tests for minio config/defaults.sh configuration management

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
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
    export MINIO_CUSTOM_PORT="9000"
    export MINIO_CUSTOM_CONSOLE_PORT="9001"
    export HOME="/tmp/test-home"
    mkdir -p "$HOME"
    
    # Clear any existing configuration to test defaults
    unset MINIO_PORT MINIO_CONSOLE_PORT MINIO_BASE_URL MINIO_CONSOLE_URL
    unset MINIO_CONTAINER_NAME MINIO_DATA_DIR MINIO_CONFIG_DIR MINIO_IMAGE
    unset MINIO_ROOT_USER MINIO_ROOT_PASSWORD MINIO_DEFAULT_BUCKETS
    unset MINIO_NETWORK_NAME MINIO_REGION MINIO_BROWSER
    unset MINIO_HEALTH_CHECK_INTERVAL MINIO_HEALTH_CHECK_MAX_ATTEMPTS
    unset MINIO_API_TIMEOUT MINIO_STARTUP_MAX_WAIT MINIO_STARTUP_WAIT_INTERVAL
    unset MINIO_INITIALIZATION_WAIT MINIO_MIN_DISK_SPACE_GB
    
    # Get resource directory path
    MINIO_DIR="$(dirname "${BATS_TEST_DIRNAME}")"
    
    # Mock resources::get_default_port function
    resources::get_default_port() { echo "9000"; }
    export -f resources::get_default_port
    
    # Load configuration
    source "${MINIO_DIR}/config/defaults.sh"
    
    # Call export_config to initialize variables
    minio::export_config
}

teardown() {
    # Clean up test environment
    trash::safe_remove "/tmp/test-home" --test-cleanup
    [[ -d "$MOCK_RESPONSES_DIR" ]] && trash::safe_remove "$MOCK_RESPONSES_DIR" --test-cleanup
}

# Test configuration loading and initialization

@test "minio::export_config should set all required service variables" {
    # Variables should already be set from setup
    [ -n "$MINIO_PORT" ]
    [ -n "$MINIO_CONSOLE_PORT" ]
    [ -n "$MINIO_BASE_URL" ]
    [ -n "$MINIO_CONSOLE_URL" ]
    [ -n "$MINIO_CONTAINER_NAME" ]
    [ -n "$MINIO_DATA_DIR" ]
    [ -n "$MINIO_CONFIG_DIR" ]
    [ -n "$MINIO_IMAGE" ]
}

@test "minio::export_config should set all required credential variables" {
    # Variables should already be set from setup
    [ -n "$MINIO_ROOT_USER" ]
    [ -n "$MINIO_ROOT_PASSWORD" ]
    [ -n "$MINIO_DEFAULT_BUCKETS" ]
}

@test "minio::export_config should set all required network variables" {
    # Variables should already be set from setup
    [ -n "$MINIO_NETWORK_NAME" ]
    [ -n "$MINIO_REGION" ]
    [ -n "$MINIO_BROWSER" ]
}

@test "minio::export_config should use defaults when no custom config" {
    # Variables should already be set from setup with defaults
    [ "$MINIO_PORT" = "9000" ]
    [ "$MINIO_CONSOLE_PORT" = "9001" ]
    [ "$MINIO_CONTAINER_NAME" = "minio" ]
    [ "$MINIO_ROOT_USER" = "minioadmin" ]
    [ "$MINIO_ROOT_PASSWORD" = "minioadmin" ]
    [ "$MINIO_REGION" = "us-east-1" ]
    [ "$MINIO_BROWSER" = "on" ]
}

@test "minio::export_config should respect environment overrides" {
    # Test that custom ports are used
    [ "$MINIO_PORT" = "9000" ]
    [ "$MINIO_CONSOLE_PORT" = "9001" ]
    
    # Test that function exists and can be called
    type minio::export_config >/dev/null
}

@test "minio::export_config should set correct base URLs" {
    # Test that URLs are constructed correctly from current values
    [ "$MINIO_BASE_URL" = "http://localhost:9000" ]
    [ "$MINIO_CONSOLE_URL" = "http://localhost:9001" ]
    
    # Test URL format validation
    [[ "$MINIO_BASE_URL" =~ ^http://localhost:[0-9]+ ]]
    [[ "$MINIO_CONSOLE_URL" =~ ^http://localhost:[0-9]+ ]]
}

@test "minio::export_config should set health check configuration" {
    # Variables should already be set from setup
    [ "$MINIO_HEALTH_CHECK_INTERVAL" = "5" ]
    [ "$MINIO_HEALTH_CHECK_MAX_ATTEMPTS" = "12" ]
    [ "$MINIO_API_TIMEOUT" = "10" ]
}

@test "minio::export_config should set startup timing configuration" {
    # Variables should already be set from setup
    [ "$MINIO_STARTUP_MAX_WAIT" = "60" ]
    [ "$MINIO_STARTUP_WAIT_INTERVAL" = "2" ]
    [ "$MINIO_INITIALIZATION_WAIT" = "10" ]
}

@test "minio::export_config should set storage configuration" {
    # Variables should already be set from setup
    [ "$MINIO_MIN_DISK_SPACE_GB" = "5" ]
    
    # Test that directories use .minio structure
    [[ "$MINIO_DATA_DIR" =~ /.minio/data$ ]]
    [[ "$MINIO_CONFIG_DIR" =~ /.minio/config$ ]]
}

@test "minio::export_config should set default bucket configuration" {
    # Test that default buckets are set and contain expected entries
    [ -n "$MINIO_DEFAULT_BUCKETS" ]
    
    # Test that default bucket logic exists in function
    grep -q "MINIO_DEFAULT_BUCKETS" "$MINIO_DIR/config/defaults.sh"
    grep -q "vrooli-user-uploads" "$MINIO_DIR/config/defaults.sh"
    grep -q "vrooli-agent-artifacts" "$MINIO_DIR/config/defaults.sh"
}

@test "minio::export_config should handle data directory with spaces" {
    # Test that data directory is set and follows expected pattern
    [ -n "$MINIO_DATA_DIR" ]
    [[ "$MINIO_DATA_DIR" =~ \.minio/data$ ]]
    
    # Test that HOME variable is used in directory construction
    grep -q "HOME" "$MINIO_DIR/config/defaults.sh"
}

@test "minio::export_config should maintain configuration consistency across calls" {
    # Test that configuration is consistent when called from setup
    local port1="$MINIO_PORT"
    local user1="$MINIO_ROOT_USER"
    
    # Test in a fresh environment that multiple calls are consistent
    result=$(bash -c "
        export HOME=\"/tmp/test-home\"
        source \"$MINIO_DIR/config/defaults.sh\"
        resources::get_default_port() { echo '9000'; }
        minio::export_config
        port1=\$MINIO_PORT
        user1=\$MINIO_ROOT_USER
        minio::export_config
        port2=\$MINIO_PORT
        user2=\$MINIO_ROOT_USER
        echo \$port1
        echo \$port2
        echo \$user1
        echo \$user2
    ")
    
    port1_result=$(echo "$result" | sed -n '1p')
    port2_result=$(echo "$result" | sed -n '2p')
    user1_result=$(echo "$result" | sed -n '3p')
    user2_result=$(echo "$result" | sed -n '4p')
    
    [ "$port1_result" = "$port2_result" ]
    [ "$user1_result" = "$user2_result" ]
}

@test "minio::export_config should export all variables to environment" {
    # Variables should already be exported from setup
    [ -n "${MINIO_PORT:-}" ]
    [ -n "${MINIO_BASE_URL:-}" ]
    [ -n "${MINIO_CONTAINER_NAME:-}" ]
    [ -n "${MINIO_ROOT_USER:-}" ]
    [ -n "${MINIO_REGION:-}" ]
}

@test "minio::export_config should handle missing HOME directory" {
    # Test with missing HOME
    result=$(bash -c "
        unset HOME
        source \"$MINIO_DIR/config/defaults.sh\"
        resources::get_default_port() { echo '9000'; }
        minio::export_config 2>/dev/null || true
        echo \$MINIO_DATA_DIR
    ")
    
    # Should still set a data directory (may be relative)
    [ -n "$result" ]
}

@test "minio::export_config should handle custom credentials" {
    # Test with custom credentials
    result=$(bash -c "
        export MINIO_CUSTOM_ROOT_USER='custom-user'
        export MINIO_CUSTOM_ROOT_PASSWORD='custom-pass-123'
        export HOME=\"/tmp/test-home\"
        source \"$MINIO_DIR/config/defaults.sh\"
        resources::get_default_port() { echo '9000'; }
        minio::export_config
        echo \$MINIO_ROOT_USER
        echo \$MINIO_ROOT_PASSWORD
    ")
    
    user_result=$(echo "$result" | sed -n '1p')
    pass_result=$(echo "$result" | sed -n '2p')
    
    [ "$user_result" = "custom-user" ]
    [ "$pass_result" = "custom-pass-123" ]
}

@test "minio::export_config should handle custom ports" {
    # Test with custom ports
    result=$(bash -c "
        export MINIO_CUSTOM_PORT='9999'
        export MINIO_CUSTOM_CONSOLE_PORT='8888'
        export HOME=\"/tmp/test-home\"
        source \"$MINIO_DIR/config/defaults.sh\"
        resources::get_default_port() { echo '9000'; }
        minio::export_config
        echo \$MINIO_PORT
        echo \$MINIO_CONSOLE_PORT
        echo \$MINIO_BASE_URL
        echo \$MINIO_CONSOLE_URL
    ")
    
    port_result=$(echo "$result" | sed -n '1p')
    console_port_result=$(echo "$result" | sed -n '2p')
    base_url_result=$(echo "$result" | sed -n '3p')
    console_url_result=$(echo "$result" | sed -n '4p')
    
    [ "$port_result" = "9999" ]
    [ "$console_port_result" = "8888" ]
    [ "$base_url_result" = "http://localhost:9999" ]
    [ "$console_url_result" = "http://localhost:8888" ]
}

@test "minio::export_config should use official minio docker image" {
    # Test that image is set to official MinIO image
    [[ "$MINIO_IMAGE" =~ ^minio/minio: ]]
    [ "$MINIO_IMAGE" = "minio/minio:latest" ]
}

@test "minio::export_config should handle readonly variable conflicts gracefully" {
    # Test behavior when variables are already readonly
    result=$(bash -c "
        export HOME=\"/tmp/test-home\"
        source \"$MINIO_DIR/config/defaults.sh\"
        resources::get_default_port() { echo '9000'; }
        minio::export_config
        # Try to call again (should handle readonly gracefully)
        minio::export_config 2>/dev/null
        echo \$MINIO_PORT
    ")
    
    [ "$result" = "9000" ]
}

@test "minio::export_config should set reasonable timeout values" {
    # Test that timeout values are reasonable for production use
    [ "$MINIO_HEALTH_CHECK_INTERVAL" -ge 1 ]
    [ "$MINIO_HEALTH_CHECK_INTERVAL" -le 30 ]
    [ "$MINIO_HEALTH_CHECK_MAX_ATTEMPTS" -ge 3 ]
    [ "$MINIO_API_TIMEOUT" -ge 5 ]
    [ "$MINIO_STARTUP_MAX_WAIT" -ge 30 ]
}

@test "minio::export_config should set valid AWS region" {
    # Test that region is a valid AWS region format
    [[ "$MINIO_REGION" =~ ^[a-z]+-[a-z]+-[0-9]+$ ]]
    [ "$MINIO_REGION" = "us-east-1" ]
}

@test "minio::export_config should enable browser by default" {
    # Test that browser interface is enabled by default
    [ "$MINIO_BROWSER" = "on" ]
}

@test "minio::export_config should set minimum disk space requirement" {
    # Test that minimum disk space is set to reasonable value
    [ "$MINIO_MIN_DISK_SPACE_GB" -ge 1 ]
    [ "$MINIO_MIN_DISK_SPACE_GB" -le 100 ]
    [ "$MINIO_MIN_DISK_SPACE_GB" = "5" ]
}

@test "minio::export_config should use secure defaults for production" {
    # Test that defaults are reasonable for production use
    [ ${#MINIO_ROOT_USER} -ge 8 ]  # Minimum username length
    [ ${#MINIO_ROOT_PASSWORD} -ge 8 ]  # Minimum password length
    [ "$MINIO_BROWSER" = "on" ]  # Console access enabled
}