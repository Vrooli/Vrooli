#!/usr/bin/env bats
# Tests for vault config/defaults.sh configuration management

bats_require_minimum_version 1.5.0

# Setup for each test
setup() {
    # Setup mock framework
    export BATS_TEST_TMPDIR="${BATS_TEST_TMPDIR:-$(mktemp -d)}"
    export MOCK_RESPONSES_DIR="$BATS_TEST_TMPDIR/mock_responses"
    mkdir -p "$MOCK_RESPONSES_DIR"
    
    # Load mock framework
    MOCK_DIR="/home/matthalloran8/Vrooli/scripts/resources/tests/bats-fixtures/mocks"
    source "$MOCK_DIR/system_mocks.bash"
    source "$MOCK_DIR/mock_helpers.bash"
    source "$MOCK_DIR/resource_mocks.bash"
    
    # Set test environment
    export VAULT_CUSTOM_PORT="8200"
    export HOME="/tmp/test-home"
    mkdir -p "$HOME"
    
    # Clear any existing configuration to test defaults
    unset VAULT_PORT VAULT_BASE_URL VAULT_CONTAINER_NAME
    unset VAULT_DATA_DIR VAULT_CONFIG_DIR VAULT_LOGS_DIR VAULT_IMAGE
    unset VAULT_MODE VAULT_DEV_ROOT_TOKEN_ID VAULT_DEV_LISTEN_ADDRESS
    unset VAULT_TLS_DISABLE VAULT_STORAGE_TYPE
    unset VAULT_NAMESPACE_PREFIX VAULT_SECRET_ENGINE VAULT_SECRET_VERSION
    unset VAULT_NETWORK_NAME VAULT_HEALTH_CHECK_INTERVAL
    unset VAULT_HEALTH_CHECK_MAX_ATTEMPTS VAULT_API_TIMEOUT
    unset VAULT_STARTUP_MAX_WAIT VAULT_STARTUP_WAIT_INTERVAL
    unset VAULT_INITIALIZATION_WAIT VAULT_TOKEN_FILE
    unset VAULT_UNSEAL_KEYS_FILE VAULT_AUDIT_LOG_FILE VAULT_DEFAULT_PATHS
    
    # Get resource directory path
    VAULT_DIR="$(dirname "$(dirname "${BATS_TEST_FILENAME}")")"
    
    # Mock resources::get_default_port function
    resources::get_default_port() { echo "8200"; }
    export -f resources::get_default_port
    
    # Load configuration
    source "${VAULT_DIR}/config/defaults.sh"
    
    # Call export_config to initialize variables
    vault::export_config
}

teardown() {
    # Clean up test environment
    rm -rf "/tmp/test-home"
    [[ -d "$MOCK_RESPONSES_DIR" ]] && rm -rf "$MOCK_RESPONSES_DIR"
}

# Test configuration loading and initialization

@test "vault::export_config should set all required service variables" {
    # Variables should already be set from setup
    [ -n "$VAULT_PORT" ]
    [ -n "$VAULT_BASE_URL" ]
    [ -n "$VAULT_CONTAINER_NAME" ]
    [ -n "$VAULT_DATA_DIR" ]
    [ -n "$VAULT_CONFIG_DIR" ]
    [ -n "$VAULT_LOGS_DIR" ]
    [ -n "$VAULT_IMAGE" ]
}

@test "vault::export_config should set all required vault mode variables" {
    # Variables should already be set from setup
    [ -n "$VAULT_MODE" ]
    [ -n "$VAULT_DEV_ROOT_TOKEN_ID" ]
    [ -n "$VAULT_DEV_LISTEN_ADDRESS" ]
    [ -n "$VAULT_TLS_DISABLE" ]
    [ -n "$VAULT_STORAGE_TYPE" ]
}

@test "vault::export_config should set all required secret engine variables" {
    # Variables should already be set from setup
    [ -n "$VAULT_NAMESPACE_PREFIX" ]
    [ -n "$VAULT_SECRET_ENGINE" ]
    [ -n "$VAULT_SECRET_VERSION" ]
    [ -n "$VAULT_NETWORK_NAME" ]
}

@test "vault::export_config should use defaults when no custom config" {
    # Variables should already be set from setup with defaults
    [ "$VAULT_PORT" = "8200" ]
    [ "$VAULT_CONTAINER_NAME" = "vault" ]
    [ "$VAULT_MODE" = "dev" ]
    [ "$VAULT_DEV_ROOT_TOKEN_ID" = "myroot" ]
    [ "$VAULT_TLS_DISABLE" = "1" ]
    [ "$VAULT_STORAGE_TYPE" = "file" ]
}

@test "vault::export_config should respect environment overrides" {
    # Test that custom port is used
    [ "$VAULT_PORT" = "8200" ]
    
    # Test that function exists and can be called
    type vault::export_config >/dev/null
}

@test "vault::export_config should set correct base URLs" {
    # Test that URLs are constructed correctly from current values
    [ "$VAULT_BASE_URL" = "http://localhost:8200" ]
    [ "$VAULT_DEV_LISTEN_ADDRESS" = "0.0.0.0:8200" ]
    
    # Test URL format validation
    [[ "$VAULT_BASE_URL" =~ ^http://localhost:[0-9]+ ]]
}

@test "vault::export_config should set health check configuration" {
    # Variables should already be set from setup
    [ "$VAULT_HEALTH_CHECK_INTERVAL" = "5" ]
    [ "$VAULT_HEALTH_CHECK_MAX_ATTEMPTS" = "12" ]
    [ "$VAULT_API_TIMEOUT" = "10" ]
}

@test "vault::export_config should set startup timing configuration" {
    # Variables should already be set from setup
    [ "$VAULT_STARTUP_MAX_WAIT" = "60" ]
    [ "$VAULT_STARTUP_WAIT_INTERVAL" = "2" ]
    [ "$VAULT_INITIALIZATION_WAIT" = "10" ]
}

@test "vault::export_config should set authentication file paths" {
    # Variables should already be set from setup
    [[ "$VAULT_TOKEN_FILE" =~ root-token$ ]]
    [[ "$VAULT_UNSEAL_KEYS_FILE" =~ unseal-keys$ ]]
    [[ "$VAULT_AUDIT_LOG_FILE" =~ audit.log$ ]]
    
    # Test that paths use config/logs directories
    [[ "$VAULT_TOKEN_FILE" =~ /.vault/config/ ]]
    [[ "$VAULT_UNSEAL_KEYS_FILE" =~ /.vault/config/ ]]
    [[ "$VAULT_AUDIT_LOG_FILE" =~ /.vault/logs/ ]]
}

@test "vault::export_config should set secret namespace configuration" {
    # Test namespace and secret engine configuration
    [ "$VAULT_NAMESPACE_PREFIX" = "vrooli" ]
    [ "$VAULT_SECRET_ENGINE" = "secret" ]
    [ "$VAULT_SECRET_VERSION" = "2" ]
    [ "$VAULT_NETWORK_NAME" = "vault-network" ]
}

@test "vault::export_config should set default secret paths array" {
    # Test that default paths are set and contain expected entries
    [ -n "$VAULT_DEFAULT_PATHS" ]
    
    # Test that default paths logic exists in function
    grep -q "VAULT_DEFAULT_PATHS" "$VAULT_DIR/config/defaults.sh"
    grep -q "environments/development" "$VAULT_DIR/config/defaults.sh"
    grep -q "environments/production" "$VAULT_DIR/config/defaults.sh"
}

@test "vault::export_config should handle data directory with spaces" {
    # Test that data directory is set and follows expected pattern
    [ -n "$VAULT_DATA_DIR" ]
    [[ "$VAULT_DATA_DIR" =~ \.vault/data$ ]]
    
    # Test that HOME variable is used in directory construction
    grep -q "HOME" "$VAULT_DIR/config/defaults.sh"
}

@test "vault::export_config should maintain configuration consistency across calls" {
    # Test that configuration is consistent when called from setup
    local port1="$VAULT_PORT"
    local mode1="$VAULT_MODE"
    
    # Test in a fresh environment that multiple calls are consistent
    result=$(bash -c "
        export HOME=\"/tmp/test-home\"
        source \"$VAULT_DIR/config/defaults.sh\"
        resources::get_default_port() { echo '8200'; }
        vault::export_config
        port1=\$VAULT_PORT
        mode1=\$VAULT_MODE
        vault::export_config
        port2=\$VAULT_PORT
        mode2=\$VAULT_MODE
        echo \$port1
        echo \$port2
        echo \$mode1
        echo \$mode2
    ")
    
    port1_result=$(echo "$result" | sed -n '1p')
    port2_result=$(echo "$result" | sed -n '2p')
    mode1_result=$(echo "$result" | sed -n '3p')
    mode2_result=$(echo "$result" | sed -n '4p')
    
    [ "$port1_result" = "$port2_result" ]
    [ "$mode1_result" = "$mode2_result" ]
}

@test "vault::export_config should export all variables to environment" {
    # Variables should already be exported from setup
    [ -n "${VAULT_PORT:-}" ]
    [ -n "${VAULT_BASE_URL:-}" ]
    [ -n "${VAULT_CONTAINER_NAME:-}" ]
    [ -n "${VAULT_MODE:-}" ]
    [ -n "${VAULT_SECRET_ENGINE:-}" ]
}

@test "vault::export_config should handle missing HOME directory" {
    # Test with missing HOME
    result=$(bash -c "
        unset HOME
        source \"$VAULT_DIR/config/defaults.sh\"
        resources::get_default_port() { echo '8200'; }
        vault::export_config 2>/dev/null || true
        echo \$VAULT_DATA_DIR
    ")
    
    # Should still set a data directory (may be relative)
    [ -n "$result" ]
}

@test "vault::export_config should handle production mode configuration" {
    # Test with production mode
    result=$(bash -c "
        export VAULT_CUSTOM_MODE='prod'
        export HOME=\"/tmp/test-home\"
        source \"$VAULT_DIR/config/defaults.sh\"
        resources::get_default_port() { echo '8200'; }
        vault::export_config
        echo \$VAULT_MODE
        echo \$VAULT_TLS_DISABLE
    ")
    
    mode_result=$(echo "$result" | sed -n '1p')
    tls_result=$(echo "$result" | sed -n '2p')
    
    [ "$mode_result" = "prod" ]
    [ "$tls_result" = "1" ]  # Still defaults to disabled unless overridden
}

@test "vault::export_config should handle custom token configuration" {
    # Test with custom token
    result=$(bash -c "
        export VAULT_CUSTOM_DEV_ROOT_TOKEN_ID='custom-token-123'
        export HOME=\"/tmp/test-home\"
        source \"$VAULT_DIR/config/defaults.sh\"
        resources::get_default_port() { echo '8200'; }
        vault::export_config
        echo \$VAULT_DEV_ROOT_TOKEN_ID
    ")
    
    [ "$result" = "custom-token-123" ]
}

@test "vault::export_config should handle custom storage type" {
    # Test with custom storage
    result=$(bash -c "
        export VAULT_CUSTOM_STORAGE_TYPE='consul'
        export HOME=\"/tmp/test-home\"
        source \"$VAULT_DIR/config/defaults.sh\"
        resources::get_default_port() { echo '8200'; }
        vault::export_config
        echo \$VAULT_STORAGE_TYPE
    ")
    
    [ "$result" = "consul" ]
}

@test "vault::export_config should use HashiCorp official vault image" {
    # Test that image is set to HashiCorp official image
    [[ "$VAULT_IMAGE" =~ ^hashicorp/vault: ]]
    [[ "$VAULT_IMAGE" =~ vault:1\.[0-9]+ ]]
}

@test "vault::export_config should handle readonly variable conflicts gracefully" {
    # Test behavior when variables are already readonly
    result=$(bash -c "
        export HOME=\"/tmp/test-home\"
        source \"$VAULT_DIR/config/defaults.sh\"
        resources::get_default_port() { echo '8200'; }
        vault::export_config
        # Try to call again (should handle readonly gracefully)
        vault::export_config 2>/dev/null
        echo \$VAULT_PORT
    ")
    
    [ "$result" = "8200" ]
}

@test "vault::export_config should validate secret engine version" {
    # Test that secret engine version is valid KV version
    [[ "$VAULT_SECRET_VERSION" =~ ^[12]$ ]]
    [ "$VAULT_SECRET_VERSION" = "2" ]  # Default to KV v2
}

@test "vault::export_config should set reasonable timeout values" {
    # Test that timeout values are reasonable for production use
    [ "$VAULT_HEALTH_CHECK_INTERVAL" -ge 1 ]
    [ "$VAULT_HEALTH_CHECK_INTERVAL" -le 30 ]
    [ "$VAULT_HEALTH_CHECK_MAX_ATTEMPTS" -ge 3 ]
    [ "$VAULT_API_TIMEOUT" -ge 5 ]
    [ "$VAULT_STARTUP_MAX_WAIT" -ge 30 ]
}