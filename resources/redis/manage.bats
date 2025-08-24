#!/usr/bin/env bats
# Tests for Redis manage.sh script

# Source var.sh to get proper directory variables
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/.." && builtin pwd)}"
_REDIS_BATS_DIR="$APP_ROOT/resources/redis"
# shellcheck disable=SC1091
source "${APP_ROOT}/lib/utils/var.sh"

# Load Vrooli test infrastructure
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

# Setup for each test
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set Redis-specific test environment
    export REDIS_CUSTOM_PORT="9999"
    export REDIS_CONTAINER_NAME="redis-test"
    export REDIS_PASSWORD="test-password"
    export DATABASE="0"
    export REMOVE_DATA="no"
    export LOG_LINES="50"
    export MONITOR_INTERVAL="5"
    export BACKUP_NAME=""
    export CLIENT_ID=""
    export FORCE="no"
    export YES="no"
    
    # Load the script without executing main
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    source "${SCRIPT_DIR}/manage.sh" || true
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "redis script loads without errors" {
    # Script loading happens in setup_file, this verifies it worked
    declare -f redis::parse_arguments > /dev/null
    [ "$?" -eq 0 ]
}

@test "redis defines all required functions" {
    declare -f redis::parse_arguments > /dev/null
    [ "$?" -eq 0 ]
    declare -f redis::main > /dev/null
    [ "$?" -eq 0 ]
}

# ============================================================================
# Argument Parsing Tests
# ============================================================================

@test "redis::parse_arguments sets default action to status" {
    redis::parse_arguments
    [ "$ACTION" = "status" ]
}

@test "redis::parse_arguments accepts install action" {
    redis::parse_arguments --action install
    [ "$ACTION" = "install" ]
}

@test "redis::parse_arguments accepts uninstall action" {
    redis::parse_arguments --action uninstall
    [ "$ACTION" = "uninstall" ]
}

@test "redis::parse_arguments accepts start action" {
    redis::parse_arguments --action start
    [ "$ACTION" = "start" ]
}

@test "redis::parse_arguments accepts stop action" {
    redis::parse_arguments --action stop
    [ "$ACTION" = "stop" ]
}

@test "redis::parse_arguments accepts restart action" {
    redis::parse_arguments --action restart
    [ "$ACTION" = "restart" ]
}

@test "redis::parse_arguments accepts backup action" {
    redis::parse_arguments --action backup
    [ "$ACTION" = "backup" ]
}

@test "redis::parse_arguments accepts restore action" {
    redis::parse_arguments --action restore
    [ "$ACTION" = "restore" ]
}

@test "redis::parse_arguments accepts flush action" {
    redis::parse_arguments --action flush
    [ "$ACTION" = "flush" ]
}

@test "redis::parse_arguments accepts cli action" {
    redis::parse_arguments --action cli
    [ "$ACTION" = "cli" ]
}

@test "redis::parse_arguments accepts benchmark action" {
    redis::parse_arguments --action benchmark
    [ "$ACTION" = "benchmark" ]
}

@test "redis::parse_arguments accepts monitor action" {
    redis::parse_arguments --action monitor
    [ "$ACTION" = "monitor" ]
}

@test "redis::parse_arguments accepts diagnose action" {
    redis::parse_arguments --action diagnose
    [ "$ACTION" = "diagnose" ]
}

# ============================================================================
# Parameter Handling Tests
# ============================================================================

@test "redis::parse_arguments handles backup-name parameter" {
    redis::parse_arguments --action backup --backup-name test-backup
    [ "$ACTION" = "backup" ]
    [ "$BACKUP_NAME" = "test-backup" ]
}

@test "redis::parse_arguments handles client-id parameter" {
    redis::parse_arguments --action create-client --client-id test-client
    [ "$ACTION" = "create-client" ]
    [ "$CLIENT_ID" = "test-client" ]
}

@test "redis::parse_arguments handles database parameter" {
    redis::parse_arguments --database 5
    [ "$DATABASE" = "5" ]
}

@test "redis::parse_arguments handles remove-data parameter" {
    redis::parse_arguments --action uninstall --remove-data yes
    [ "$ACTION" = "uninstall" ]
    [ "$REMOVE_DATA" = "yes" ]
}

@test "redis::parse_arguments handles lines parameter" {
    redis::parse_arguments --action logs --lines 100
    [ "$ACTION" = "logs" ]
    [ "$LOG_LINES" = "100" ]
}

@test "redis::parse_arguments handles interval parameter" {
    redis::parse_arguments --action monitor --interval 10
    [ "$ACTION" = "monitor" ]
    [ "$MONITOR_INTERVAL" = "10" ]
}

# ============================================================================
# Default Values Tests
# ============================================================================

@test "redis::parse_arguments sets default database to 0" {
    redis::parse_arguments
    [ "$DATABASE" = "0" ]
}

@test "redis::parse_arguments sets default remove-data to no" {
    redis::parse_arguments
    [ "$REMOVE_DATA" = "no" ]
}

@test "redis::parse_arguments sets default lines to 50" {
    redis::parse_arguments
    [ "$LOG_LINES" = "50" ]
}

@test "redis::parse_arguments sets default interval to 5" {
    redis::parse_arguments
    [ "$MONITOR_INTERVAL" = "5" ]
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "redis::parse_arguments function is defined" {
    declare -f redis::parse_arguments > /dev/null
    [ "$?" -eq 0 ]
}

@test "redis::main function is defined" {
    declare -f redis::main > /dev/null
    [ "$?" -eq 0 ]
}

# ============================================================================
# Configuration Tests
# ============================================================================

@test "redis config directory exists" {
    [ -d "${BATS_TEST_DIRNAME}/config" ]
}

@test "redis lib directory exists" {
    [ -d "${BATS_TEST_DIRNAME}/lib" ]
}

@test "redis defaults.sh config exists" {
    [ -f "${BATS_TEST_DIRNAME}/config/defaults.sh" ]
}

@test "redis messages.sh config exists" {
    [ -f "${BATS_TEST_DIRNAME}/config/messages.sh" ]
}

# ============================================================================
# Multi-Client Operation Tests
# ============================================================================

@test "redis::parse_arguments accepts create-client action" {
    redis::parse_arguments --action create-client
    [ "$ACTION" = "create-client" ]
}

@test "redis::parse_arguments accepts destroy-client action" {
    redis::parse_arguments --action destroy-client
    [ "$ACTION" = "destroy-client" ]
}

@test "redis::parse_arguments accepts list-clients action" {
    redis::parse_arguments --action list-clients
    [ "$ACTION" = "list-clients" ]
}

# ============================================================================
# Backup Management Tests
# ============================================================================

@test "redis::parse_arguments accepts list-backups action" {
    redis::parse_arguments --action list-backups
    [ "$ACTION" = "list-backups" ]
}

@test "redis::parse_arguments accepts delete-backup action" {
    redis::parse_arguments --action delete-backup
    [ "$ACTION" = "delete-backup" ]
}

@test "redis::parse_arguments accepts cleanup-backups action" {
    redis::parse_arguments --action cleanup-backups
    [ "$ACTION" = "cleanup-backups" ]
}

# ============================================================================
# Configuration Management Tests
# ============================================================================

@test "redis::parse_arguments accepts config action" {
    redis::parse_arguments --action config
    [ "$ACTION" = "config" ]
}

@test "redis::parse_arguments accepts upgrade action" {
    redis::parse_arguments --action upgrade
    [ "$ACTION" = "upgrade" ]
}