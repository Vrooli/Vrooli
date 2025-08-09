#!/usr/bin/env bats
# Tests for PostgreSQL manage.sh script

# Load Vrooli test infrastructure
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

# Setup for each test
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set PostgreSQL-specific test environment
    export POSTGRES_CUSTOM_PORT="9999"
    export POSTGRES_CONTAINER_NAME="postgres-test"
    export INSTANCE="main"
    export TEMPLATE="development"
    export PORT="5432"
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

@test "postgres script loads without errors" {
    # Script loading happens in setup, this verifies it worked
    declare -f postgres::parse_arguments > /dev/null
    [ "$?" -eq 0 ]
}

@test "postgres defines all required functions" {
    declare -f postgres::parse_arguments > /dev/null
    [ "$?" -eq 0 ]
    declare -f postgres::main > /dev/null
    [ "$?" -eq 0 ]
}

# ============================================================================
# Argument Parsing Tests
# ============================================================================

@test "postgres::parse_arguments sets default action to status" {
    postgres::parse_arguments
    [ "$ACTION" = "status" ]
}

@test "postgres::parse_arguments accepts install action" {
    postgres::parse_arguments --action install
    [ "$ACTION" = "install" ]
}

@test "postgres::parse_arguments accepts uninstall action" {
    postgres::parse_arguments --action uninstall
    [ "$ACTION" = "uninstall" ]
}

@test "postgres::parse_arguments accepts create action" {
    postgres::parse_arguments --action create
    [ "$ACTION" = "create" ]
}

@test "postgres::parse_arguments accepts destroy action" {
    postgres::parse_arguments --action destroy
    [ "$ACTION" = "destroy" ]
}

@test "postgres::parse_arguments accepts backup action" {
    postgres::parse_arguments --action backup
    [ "$ACTION" = "backup" ]
}

@test "postgres::parse_arguments accepts restore action" {
    postgres::parse_arguments --action restore
    [ "$ACTION" = "restore" ]
}

@test "postgres::parse_arguments accepts migrate action" {
    postgres::parse_arguments --action migrate
    [ "$ACTION" = "migrate" ]
}

@test "postgres::parse_arguments handles instance parameter" {
    postgres::parse_arguments --action create --instance test-db
    [ "$ACTION" = "create" ]
    [ "$INSTANCE" = "test-db" ]
}

@test "postgres::parse_arguments handles port parameter" {
    postgres::parse_arguments --action create --port 5433
    [ "$ACTION" = "create" ]
    [ "$PORT" = "5433" ]
}

@test "postgres::parse_arguments handles template parameter" {
    postgres::parse_arguments --action create --template production
    [ "$ACTION" = "create" ]
    [ "$TEMPLATE" = "production" ]
}

@test "postgres::parse_arguments sets default instance to main" {
    postgres::parse_arguments
    [ "$INSTANCE" = "main" ]
}

@test "postgres::parse_arguments sets default template to development" {
    postgres::parse_arguments
    [ "$TEMPLATE" = "development" ]
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "postgres::parse_arguments function is defined" {
    declare -f postgres::parse_arguments > /dev/null
    [ "$?" -eq 0 ]
}

@test "postgres::main function is defined" {
    declare -f postgres::main > /dev/null
    [ "$?" -eq 0 ]
}

@test "postgres required functions are defined" {
    # Check for main management functions
    declare -f postgres::parse_arguments > /dev/null
    [ "$?" -eq 0 ]
    declare -f postgres::main > /dev/null  
    [ "$?" -eq 0 ]
}

# ============================================================================
# Configuration Tests
# ============================================================================

@test "postgres config directory exists" {
    [ -d "${BATS_TEST_DIRNAME}/config" ]
}

@test "postgres lib directory exists" {
    [ -d "${BATS_TEST_DIRNAME}/lib" ]
}

@test "postgres defaults.sh config exists" {
    [ -f "${BATS_TEST_DIRNAME}/config/defaults.sh" ]
}

@test "postgres messages.sh config exists" {
    [ -f "${BATS_TEST_DIRNAME}/config/messages.sh" ]
}

# ============================================================================
# Multi-Instance Operation Tests
# ============================================================================

@test "postgres::parse_arguments accepts multi-start action" {
    postgres::parse_arguments --action multi-start
    [ "$ACTION" = "multi-start" ]
}

@test "postgres::parse_arguments accepts multi-stop action" {
    postgres::parse_arguments --action multi-stop
    [ "$ACTION" = "multi-stop" ]
}

@test "postgres::parse_arguments accepts multi-status action" {
    postgres::parse_arguments --action multi-status
    [ "$ACTION" = "multi-status" ]
}

# ============================================================================
# Database Management Tests
# ============================================================================

@test "postgres::parse_arguments accepts create-db action" {
    postgres::parse_arguments --action create-db
    [ "$ACTION" = "create-db" ]
}

@test "postgres::parse_arguments accepts drop-db action" {
    postgres::parse_arguments --action drop-db
    [ "$ACTION" = "drop-db" ]
}

@test "postgres::parse_arguments accepts create-user action" {
    postgres::parse_arguments --action create-user
    [ "$ACTION" = "create-user" ]
}

@test "postgres::parse_arguments accepts drop-user action" {
    postgres::parse_arguments --action drop-user
    [ "$ACTION" = "drop-user" ]
}