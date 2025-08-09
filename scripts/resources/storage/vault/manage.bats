#!/usr/bin/env bats
# Tests for Vault manage.sh script

# Load Vrooli test infrastructure
# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../../../../__test/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Load var.sh to get directory variables
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/../../../../lib/utils/var.sh"
    
    # Use Vrooli service test setup
    vrooli_setup_service_test "vault"
    
    # Load Vault specific configuration once per file
    # shellcheck disable=SC1091
    source "$var_SCRIPTS_RESOURCES_DIR/storage/vault/config/defaults.sh"
    # shellcheck disable=SC1091
    source "$var_SCRIPTS_RESOURCES_DIR/storage/vault/config/messages.sh"
    # shellcheck disable=SC1091
    source "$var_SCRIPTS_RESOURCES_DIR/storage/vault/manage.sh"
}

# Lightweight per-test setup
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Set test environment variables (lightweight per-test)
    export VAULT_PORT="8200"
    export VAULT_MODE="dev"
    export VAULT_DATA_DIR="/tmp/vault-test"
    export VAULT_CONFIG_DIR="/tmp/vault-test"
    export VAULT_CONTAINER_NAME="vault-test"
    export ACTION="status"
    export SECRET_PATH=""
    export SECRET_VALUE=""
    export ENV_FILE=""
    export VAULT_PREFIX=""
    export FOLLOW_LOGS="no"
    export MONITOR_INTERVAL="30"
    
    # Export config functions
    vault::export_config
    vault::export_messages
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "manage.sh loads without errors" {
    # Should load successfully in setup_file
    [ "$?" -eq 0 ]
}

@test "manage.sh defines required functions" {
    # Functions should be available from setup_file
    declare -f vault::parse_arguments >/dev/null
    declare -f vault::main >/dev/null
    declare -f vault::install >/dev/null
    declare -f vault::show_status >/dev/null
}

# ============================================================================
# Argument Parsing Tests
# ============================================================================

@test "vault::parse_arguments sets default action to status" {
    vault::parse_arguments
    [ "$ACTION" = "status" ]
}

@test "vault::parse_arguments accepts install action" {
    vault::parse_arguments --action install
    [ "$ACTION" = "install" ]
}

@test "vault::parse_arguments accepts status action" {
    vault::parse_arguments --action status
    [ "$ACTION" = "status" ]
}

@test "vault::parse_arguments accepts init-dev action" {
    vault::parse_arguments --action init-dev
    [ "$ACTION" = "init-dev" ]
}

@test "vault::parse_arguments accepts init-prod action" {
    vault::parse_arguments --action init-prod
    [ "$ACTION" = "init-prod" ]
}

@test "vault::parse_arguments accepts put-secret action" {
    vault::parse_arguments --action put-secret
    [ "$ACTION" = "put-secret" ]
}

@test "vault::parse_arguments accepts get-secret action" {
    vault::parse_arguments --action get-secret
    [ "$ACTION" = "get-secret" ]
}

@test "vault::parse_arguments accepts secret management arguments" {
    vault::parse_arguments --action put-secret --path test/key --value secret123
    [ "$SECRET_PATH" = "test/key" ]
    [ "$SECRET_VALUE" = "secret123" ]
}

@test "vault::parse_arguments accepts mode argument" {
    vault::parse_arguments --mode prod
    [ "$VAULT_MODE" = "prod" ]
}

@test "vault::parse_arguments accepts migration arguments" {
    vault::parse_arguments --action migrate-env --env-file .env --vault-prefix dev
    [ "$ENV_FILE" = ".env" ]
    [ "$VAULT_PREFIX" = "dev" ]
}

@test "vault::parse_arguments sets follow logs flag" {
    vault::parse_arguments --follow yes
    [ "$FOLLOW_LOGS" = "yes" ]
}

@test "vault::parse_arguments accepts monitor interval" {
    vault::parse_arguments --interval 60
    [ "$MONITOR_INTERVAL" = "60" ]
}

# ============================================================================
# Help and Usage Tests
# ============================================================================

@test "vault::usage displays help text" {
    run vault::usage
    [ "$status" -eq 0 ]
    [[ "$output" == *"HashiCorp Vault Secret Management"* ]]
    [[ "$output" == *"--action"* ]]
    [[ "$output" == *"USAGE:"* ]]
}

# ============================================================================
# Main Function Tests
# ============================================================================

@test "vault::main calls correct function for install action" {
    # Mock vault::install function
    vault::install() { echo 'vault::install called'; return 0; }
    export -f vault::install
    
    export ACTION='install'
    run vault::main
    [ "$status" -eq 0 ]
    [[ "$output" =~ "vault::install called" ]]
}

@test "vault::main calls correct function for status action" {
    # Mock vault::show_status function
    vault::show_status() { echo 'vault::show_status called'; return 0; }
    export -f vault::show_status
    
    export ACTION='status'
    run vault::main
    [ "$status" -eq 0 ]
    [[ "$output" =~ "vault::show_status called" ]]
}

@test "vault::main validates put-secret arguments" {
    export ACTION='put-secret'
    export SECRET_PATH=''
    export SECRET_VALUE='test'
    
    run vault::main
    [ "$status" -eq 1 ]
    [[ "$output" =~ "--path and --value are required" ]]
}

@test "vault::main validates get-secret arguments" {
    export ACTION='get-secret'
    export SECRET_PATH=''
    
    run vault::main
    [ "$status" -eq 1 ]
    [[ "$output" =~ "--path is required" ]]
}

@test "vault::main validates migrate-env arguments" {
    export ACTION='migrate-env'
    export ENV_FILE=''
    export VAULT_PREFIX='dev'
    
    run vault::main
    [ "$status" -eq 1 ]
    [[ "$output" =~ "--env-file and --vault-prefix are required" ]]
}

@test "vault::main handles unknown action" {
    export ACTION='unknown-action'
    
    run vault::main
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown action: unknown-action" ]]
}

# ============================================================================
# Additional Action Tests
# ============================================================================

@test "vault::parse_arguments accepts uninstall action" {
    vault::parse_arguments --action uninstall
    [ "$ACTION" = "uninstall" ]
}

@test "vault::parse_arguments accepts start action" {
    vault::parse_arguments --action start
    [ "$ACTION" = "start" ]
}

@test "vault::parse_arguments accepts stop action" {
    vault::parse_arguments --action stop
    [ "$ACTION" = "stop" ]
}

@test "vault::parse_arguments accepts restart action" {
    vault::parse_arguments --action restart
    [ "$ACTION" = "restart" ]
}

@test "vault::parse_arguments accepts logs action" {
    vault::parse_arguments --action logs
    [ "$ACTION" = "logs" ]
}

@test "vault::parse_arguments accepts unseal action" {
    vault::parse_arguments --action unseal
    [ "$ACTION" = "unseal" ]
}

@test "vault::parse_arguments accepts list-secrets action" {
    vault::parse_arguments --action list-secrets
    [ "$ACTION" = "list-secrets" ]
}

@test "vault::parse_arguments accepts delete-secret action" {
    vault::parse_arguments --action delete-secret
    [ "$ACTION" = "delete-secret" ]
}

@test "vault::parse_arguments accepts backup action" {
    vault::parse_arguments --action backup
    [ "$ACTION" = "backup" ]
}

@test "vault::parse_arguments accepts restore action" {
    vault::parse_arguments --action restore --backup-file /tmp/backup.tar
    [ "$ACTION" = "restore" ]
    [ "$BACKUP_FILE" = "/tmp/backup.tar" ]
}

@test "vault::parse_arguments accepts diagnose action" {
    vault::parse_arguments --action diagnose
    [ "$ACTION" = "diagnose" ]
}

@test "vault::parse_arguments accepts monitor action" {
    vault::parse_arguments --action monitor
    [ "$ACTION" = "monitor" ]
}

@test "vault::parse_arguments accepts remove-data flag" {
    vault::parse_arguments --remove-data yes
    [ "$VAULT_REMOVE_DATA" = "yes" ]
}

@test "vault::parse_arguments accepts storage-strategy argument" {
    vault::parse_arguments --storage-strategy bind
    [ "$VAULT_STORAGE_STRATEGY" = "bind" ]
}

@test "vault::main validates list-secrets arguments" {
    export ACTION='list-secrets'
    export SECRET_PATH=''
    
    run vault::main
    [ "$status" -eq 1 ]
    [[ "$output" =~ "--path is required" ]]
}

@test "vault::main validates delete-secret arguments" {
    export ACTION='delete-secret'
    export SECRET_PATH=''
    
    run vault::main
    [ "$status" -eq 1 ]
    [[ "$output" =~ "--path is required" ]]
}

@test "vault::main validates restore arguments" {
    export ACTION='restore'
    export BACKUP_FILE=''
    
    run vault::main
    [ "$status" -eq 1 ]
    [[ "$output" =~ "--backup-file is required" ]]
}

@test "vault::main calls correct function for uninstall action" {
    # Mock vault::uninstall function
    vault::uninstall() { echo 'vault::uninstall called'; return 0; }
    export -f vault::uninstall
    
    export ACTION='uninstall'
    run vault::main
    [ "$status" -eq 0 ]
    [[ "$output" =~ "vault::uninstall called" ]]
}

@test "vault::main calls correct function for logs action with follow" {
    # Mock vault::docker::show_logs function
    vault::docker::show_logs() { echo "vault::docker::show_logs called with $*"; return 0; }
    export -f vault::docker::show_logs
    
    export ACTION='logs'
    export FOLLOW_LOGS='yes'
    export LOG_LINES='100'
    run vault::main
    [ "$status" -eq 0 ]
    [[ "$output" =~ "vault::docker::show_logs called with 100 follow" ]]
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "vault::main executes status action without arguments" {
    # Mock vault functions to avoid actual execution
    vault::show_status() { echo 'Status: not_installed'; return 0; }
    vault::cleanup() { : ; }
    export -f vault::show_status vault::cleanup
    
    export ACTION=status
    run vault::main
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Status:" ]]
}

@test "configuration is exported correctly" {
    vault::export_config
    
    [ -n "$VAULT_PORT" ]
    [ -n "$VAULT_CONTAINER_NAME" ]
    [ -n "$VAULT_IMAGE" ]
}

@test "messages are initialized correctly" {
    vault::export_messages
    
    [ -n "$MSG_INSTALL_SUCCESS" ]
    [ -n "$MSG_ALREADY_INSTALLED" ]
    [ -n "$MSG_HEALTHY" ]
}