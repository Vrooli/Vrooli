#!/usr/bin/env bats
# Tests for vault config/messages.sh message system

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
    export HOME="/tmp/test-home"
    export VAULT_PORT="8200"
    export VAULT_TOKEN_FILE="${HOME}/.vault/config/root-token"
    export VAULT_UNSEAL_KEYS_FILE="${HOME}/.vault/config/unseal-keys"
    export VAULT_DEV_ROOT_TOKEN_ID="myroot"
    mkdir -p "$HOME"
    
    # Clear any existing messages to test initialization
    unset VAULT_MESSAGES
    
    # Get resource directory path
    VAULT_DIR="$(dirname "${BATS_TEST_DIRNAME}")"
    
    # Mock logging functions
    log::info() { echo "INFO: $*"; }
    log::warn() { echo "WARN: $*"; }
    log::error() { echo "ERROR: $*"; }
    log::success() { echo "SUCCESS: $*"; }
    export -f log::info log::warn log::error log::success
    
    # Load configuration and messages
    source "${VAULT_DIR}/config/defaults.sh"
    source "${VAULT_DIR}/config/messages.sh"
    
    # Initialize message system
    vault::messages::init
}

teardown() {
    # Clean up test environment
    trash::safe_remove "/tmp/test-home" --test-cleanup
    [[ -d "$MOCK_RESPONSES_DIR" ]] && trash::safe_remove "$MOCK_RESPONSES_DIR" --test-cleanup
}

# Test message system initialization

@test "vault::messages::init should initialize VAULT_MESSAGES array" {
    # Messages should already be initialized from setup
    [ -n "${VAULT_MESSAGES[MSG_VAULT_INSTALL_START]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_INSTALL_SUCCESS]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_INSTALL_FAILED]}" ]
}

@test "vault::messages::init should set all installation messages" {
    # Test installation message constants are set
    [ -n "${VAULT_MESSAGES[MSG_VAULT_INSTALL_START]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_INSTALL_SUCCESS]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_INSTALL_FAILED]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_ALREADY_INSTALLED]}" ]
}

@test "vault::messages::init should set all configuration messages" {
    # Test configuration message constants are set
    [ -n "${VAULT_MESSAGES[MSG_VAULT_CONFIG_CREATING]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_CONFIG_SUCCESS]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_CONFIG_FAILED]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_CONFIG_EXISTS]}" ]
}

@test "vault::messages::init should set all startup messages" {
    # Test startup message constants are set
    [ -n "${VAULT_MESSAGES[MSG_VAULT_START_STARTING]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_START_SUCCESS]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_START_FAILED]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_ALREADY_RUNNING]}" ]
}

@test "vault::messages::init should set all initialization messages" {
    # Test initialization message constants are set
    [ -n "${VAULT_MESSAGES[MSG_VAULT_INIT_STARTING]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_INIT_SUCCESS]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_INIT_FAILED]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_ALREADY_INITIALIZED]}" ]
}

@test "vault::messages::init should set all unsealing messages" {
    # Test unsealing message constants are set
    [ -n "${VAULT_MESSAGES[MSG_VAULT_UNSEAL_STARTING]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_UNSEAL_SUCCESS]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_UNSEAL_FAILED]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_ALREADY_UNSEALED]}" ]
}

@test "vault::messages::init should set all secret management messages" {
    # Test secret management message constants are set
    [ -n "${VAULT_MESSAGES[MSG_VAULT_SECRET_PUT_SUCCESS]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_SECRET_PUT_FAILED]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_SECRET_GET_SUCCESS]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_SECRET_GET_FAILED]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_SECRET_DELETE_SUCCESS]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_SECRET_DELETE_FAILED]}" ]
}

@test "vault::messages::init should set all status messages" {
    # Test status message constants are set
    [ -n "${VAULT_MESSAGES[MSG_VAULT_STATUS_HEALTHY]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_STATUS_UNHEALTHY]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_STATUS_STOPPED]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_STATUS_NOT_INSTALLED]}" ]
}

@test "vault::messages::init should contain appropriate content for installation messages" {
    # Test message content is appropriate
    [[ "${VAULT_MESSAGES[MSG_VAULT_INSTALL_START]}" =~ "Installing" ]]
    [[ "${VAULT_MESSAGES[MSG_VAULT_INSTALL_START]}" =~ "Vault" ]]
    [[ "${VAULT_MESSAGES[MSG_VAULT_INSTALL_SUCCESS]}" =~ "success" || "${VAULT_MESSAGES[MSG_VAULT_INSTALL_SUCCESS]}" =~ "✅" ]]
    [[ "${VAULT_MESSAGES[MSG_VAULT_INSTALL_FAILED]}" =~ "failed" || "${VAULT_MESSAGES[MSG_VAULT_INSTALL_FAILED]}" =~ "❌" ]]
}

@test "vault::messages::init should contain appropriate content for status messages" {
    # Test status message content
    [[ "${VAULT_MESSAGES[MSG_VAULT_STATUS_HEALTHY]}" =~ "healthy" || "${VAULT_MESSAGES[MSG_VAULT_STATUS_HEALTHY]}" =~ "✅" ]]
    [[ "${VAULT_MESSAGES[MSG_VAULT_STATUS_UNHEALTHY]}" =~ "unhealthy" || "${VAULT_MESSAGES[MSG_VAULT_STATUS_UNHEALTHY]}" =~ "⚠️" ]]
    [[ "${VAULT_MESSAGES[MSG_VAULT_STATUS_STOPPED]}" =~ "not running" || "${VAULT_MESSAGES[MSG_VAULT_STATUS_STOPPED]}" =~ "🛑" ]]
}

@test "vault::message function should display messages correctly" {
    # Test the vault::message function
    run vault::message "info" "MSG_VAULT_INSTALL_START"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "INFO:" ]]
    [[ "$output" =~ "Installing" ]]
    [[ "$output" =~ "Vault" ]]
}

@test "vault::message function should handle different message types" {
    # Test different message types
    run vault::message "success" "MSG_VAULT_INSTALL_SUCCESS"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SUCCESS:" ]]
    
    run vault::message "error" "MSG_VAULT_INSTALL_FAILED"
    [ "$status" -eq 1 ]  # log::error should return 1
    [[ "$output" =~ "ERROR:" ]]
    
    run vault::message "warn" "MSG_VAULT_DEV_MODE_WARNING"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "WARN:" ]]
}

@test "vault::messages::init should set security warning messages" {
    # Test security message constants are set
    [ -n "${VAULT_MESSAGES[MSG_VAULT_SECURITY_WARNING]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_TOKEN_LOCATION]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_UNSEAL_KEYS_LOCATION]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_DEV_MODE_WARNING]}" ]
    
    # Test that security messages contain appropriate warnings
    [[ "${VAULT_MESSAGES[MSG_VAULT_SECURITY_WARNING]}" =~ "SECURITY" ]]
    [[ "${VAULT_MESSAGES[MSG_VAULT_DEV_MODE_WARNING]}" =~ "production" ]]
}

@test "vault::messages::init should set troubleshooting messages" {
    # Test troubleshooting message constants are set
    [ -n "${VAULT_MESSAGES[MSG_VAULT_TROUBLESHOOT_LOGS]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_TROUBLESHOOT_CONFIG]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_TROUBLESHOOT_PORT]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_TROUBLESHOOT_RESTART]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_TROUBLESHOOT_REINIT]}" ]
    
    # Test that troubleshooting messages contain helpful commands
    [[ "${VAULT_MESSAGES[MSG_VAULT_TROUBLESHOOT_LOGS]}" =~ "logs" ]]
    [[ "${VAULT_MESSAGES[MSG_VAULT_TROUBLESHOOT_PORT]}" =~ "netstat" ]]
}

@test "vault::messages::init should set integration messages" {
    # Test integration message constants are set
    [ -n "${VAULT_MESSAGES[MSG_VAULT_INTEGRATION_N8N]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_INTEGRATION_NODERED]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_INTEGRATION_AGENT_S2]}" ]
    
    # Test that integration messages reference the services
    [[ "${VAULT_MESSAGES[MSG_VAULT_INTEGRATION_N8N]}" =~ "n8n" ]]
    [[ "${VAULT_MESSAGES[MSG_VAULT_INTEGRATION_NODERED]}" =~ "Node-RED" ]]
    [[ "${VAULT_MESSAGES[MSG_VAULT_INTEGRATION_AGENT_S2]}" =~ "Agent-S2" ]]
}

@test "vault::messages::init should handle repeated calls without duplication" {
    # First call
    vault::messages::init
    local first_msg="${VAULT_MESSAGES[MSG_VAULT_INSTALL_START]}"
    
    # Second call
    vault::messages::init
    local second_msg="${VAULT_MESSAGES[MSG_VAULT_INSTALL_START]}"
    
    # Should be identical
    [ "$first_msg" = "$second_msg" ]
}

@test "vault::messages::init should set backup and restore messages" {
    # Test backup/restore message constants are set
    [ -n "${VAULT_MESSAGES[MSG_VAULT_BACKUP_START]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_BACKUP_SUCCESS]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_BACKUP_FAILED]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_RESTORE_START]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_RESTORE_SUCCESS]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_RESTORE_FAILED]}" ]
    
    # Test that messages contain appropriate action words
    [[ "${VAULT_MESSAGES[MSG_VAULT_BACKUP_START]}" =~ "backup" ]]
    [[ "${VAULT_MESSAGES[MSG_VAULT_RESTORE_START]}" =~ "restore" ]]
}

@test "vault::messages::init should include emojis for visual clarity" {
    # Test that key messages include visual indicators
    [[ "${VAULT_MESSAGES[MSG_VAULT_INSTALL_START]}" =~ "🔧" ]]
    [[ "${VAULT_MESSAGES[MSG_VAULT_INSTALL_SUCCESS]}" =~ "✅" ]]
    [[ "${VAULT_MESSAGES[MSG_VAULT_INSTALL_FAILED]}" =~ "❌" ]]
    [[ "${VAULT_MESSAGES[MSG_VAULT_SECURITY_WARNING]}" =~ "🔐" ]]
    [[ "${VAULT_MESSAGES[MSG_VAULT_DEV_MODE_WARNING]}" =~ "⚠️" ]]
}

@test "vault::message function should handle missing message keys gracefully" {
    # Test with non-existent message key
    run vault::message "info" "NONEXISTENT_MESSAGE_KEY"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "NONEXISTENT_MESSAGE_KEY" ]]
}

@test "vault::show_help function should display comprehensive help" {
    # Test that help function exists and shows content
    run vault::show_help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "HashiCorp Vault" ]]
    [[ "$output" =~ "ACTIONS:" ]]
    [[ "$output" =~ "install" ]]
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "EXAMPLES:" ]]
    [[ "$output" =~ "SECURITY NOTES:" ]]
}

@test "vault::messages::init should set migration messages" {
    # Test migration message constants are set
    [ -n "${VAULT_MESSAGES[MSG_VAULT_MIGRATION_START]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_MIGRATION_SUCCESS]}" ]
    [ -n "${VAULT_MESSAGES[MSG_VAULT_MIGRATION_FAILED]}" ]
    
    # Test that migration messages reference migration
    [[ "${VAULT_MESSAGES[MSG_VAULT_MIGRATION_START]}" =~ "migration" ]]
    [[ "${VAULT_MESSAGES[MSG_VAULT_MIGRATION_SUCCESS]}" =~ "success" ]]
}

@test "vault::messages::init should handle missing environment gracefully" {
    # Test initialization without environment variables
    result=$(env -i bash -c "
        source '${VAULT_DIR}/config/messages.sh' 2>/dev/null
        vault::messages::init 2>/dev/null
        echo \${VAULT_MESSAGES[MSG_VAULT_INSTALL_START]}
    ")
    
    # Should still initialize messages
    [[ "$result" =~ "Installing" && "$result" =~ "Vault" ]]
}

@test "vault::messages::init should provide vault-specific messaging" {
    # Test that messages are specific to Vault
    [[ "${VAULT_MESSAGES[MSG_VAULT_INSTALL_START]}" =~ "Vault" ]]
    [[ "${VAULT_MESSAGES[MSG_VAULT_STATUS_HEALTHY]}" =~ "Vault" ]]
    
    # Test that messages reflect secret management capabilities
    [[ "${VAULT_MESSAGES[MSG_VAULT_INSTALL_START]}" =~ "secret" ]]
    [[ "${VAULT_MESSAGES[MSG_VAULT_SECRET_PUT_SUCCESS]}" =~ "Secret" ]]
}
