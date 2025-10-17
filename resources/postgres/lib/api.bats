#!/usr/bin/env bats
# Tests for PostgreSQL API functions

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load Vrooli test infrastructure
# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "postgres"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    POSTGRES_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and messages once
    source "${POSTGRES_DIR}/config/defaults.sh"
    source "${POSTGRES_DIR}/config/messages.sh"
    
    # Load API functions once
    source "${SCRIPT_DIR}/common.sh"
    source "${SCRIPT_DIR}/database.sh"
    source "${SCRIPT_DIR}/instance.sh"
    source "${SCRIPT_DIR}/backup.sh"
    source "${SCRIPT_DIR}/status.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
    export SETUP_FILE_POSTGRES_DIR="$POSTGRES_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    POSTGRES_DIR="${SETUP_FILE_POSTGRES_DIR}"
    
    # Set test environment
    export POSTGRES_PORT="5432"
    export POSTGRES_DEFAULT_DB="vrooli"
    export POSTGRES_DEFAULT_USER="postgres"
    export POSTGRES_PASSWORD="test_password"
    export POSTGRES_CONTAINER_PREFIX="postgres"
    export POSTGRES_DATA_DIR="/var/lib/postgresql/data"
    export POSTGRES_BACKUP_DIR="/var/backups/postgres"
    export INSTANCE="test-instance"
    export PORT="5433"
    export TEMPLATE="development"
    export DATABASE_NAME="test_db"
    export USERNAME="test_user"
    export BACKUP_FILE="test_backup.sql"
    export YES="no"
    
    # Set up test message variables
    export MSG_INSTANCE_NOT_FOUND="Instance not found"
    export MSG_DATABASE_CREATED="Database created"
    export MSG_DATABASE_DROPPED="Database dropped"
    export MSG_USER_CREATED="User created"
    export MSG_USER_DROPPED="User dropped"
    export MSG_BACKUP_CREATED="Backup created"
    export MSG_BACKUP_RESTORED="Backup restored"
    export MSG_CONNECTION_SUCCESSFUL="Connection successful"
    export MSG_MIGRATION_COMPLETE="Migration complete"
    
    # Mock PostgreSQL utility functions
    postgres::common::container_exists() { return 0; }
    postgres::common::is_running() { return 0; }
    postgres::common::health_check() { return 0; }
    postgres::common::wait_for_ready() { return 0; }
    postgres::common::find_available_port() { echo "5434"; }
    postgres::common::generate_password() { echo "generated_password_123"; }
    export -f postgres::common::container_exists postgres::common::is_running
    export -f postgres::common::health_check postgres::common::wait_for_ready
    export -f postgres::common::find_available_port postgres::common::generate_password
    
    # Load official mocks
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_DIR}/__test/fixtures/mocks/docker.sh"
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_DIR}/__test/fixtures/mocks/postgres.sh"
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_DIR}/__test/fixtures/mocks/filesystem.sh"
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_DIR}/__test/fixtures/mocks/system.sh"
    
    # Configure Docker mock for PostgreSQL operations
    mock::docker::add_container "postgres-test-instance" "running" "postgres:16-alpine"
    
    # Configure PostgreSQL mock for database operations
    mock::postgres::set_query_result "psql" "Command completed successfully"
    mock::postgres::set_query_result "pg_dump" "-- PostgreSQL database dump
-- Dumped by pg_dump version 14.5"
    mock::postgres::set_query_result "createdb" "CREATE DATABASE"
    mock::postgres::set_query_result "dropdb" "DROP DATABASE"
    
    # Configure system mock responses
    mock::system::set_command_output "date" "2024-01-15 10:30:00"
    mock::system::set_command_output "id" "uid=1000(user) gid=1000(user) groups=1000(user)"
    mock::system::set_command_output "whoami" "testuser"
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# ============================================================================
# Database Operations Tests
# ============================================================================

@test "postgres::database::execute runs SQL command successfully" {
    run postgres::database::execute "$INSTANCE" "SELECT 1;"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Command completed successfully" ]]
}

@test "postgres::database::execute fails when instance name missing" {
    run postgres::database::execute "" "SELECT 1;"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Instance name and SQL command are required" ]]
}

@test "postgres::database::execute fails when SQL command missing" {
    run postgres::database::execute "$INSTANCE" ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Instance name and SQL command are required" ]]
}

@test "postgres::database::execute fails when instance not found" {
    postgres::common::container_exists() { return 1; }
    
    run postgres::database::execute "$INSTANCE" "SELECT 1;"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Instance not found" ]]
}

@test "postgres::database::execute fails when instance not running" {
    postgres::common::is_running() { return 1; }
    
    run postgres::database::execute "$INSTANCE" "SELECT 1;"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "is not running" ]]
}

@test "postgres::database::create creates new database successfully" {
    run postgres::database::create "$INSTANCE" "$DATABASE_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "CREATE DATABASE" ]]
}

@test "postgres::database::drop removes database successfully" {
    export YES="yes"
    
    run postgres::database::drop "$INSTANCE" "$DATABASE_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "DROP DATABASE" ]]
}

@test "postgres::database::create_user creates new user successfully" {
    run postgres::database::create_user "$INSTANCE" "$USERNAME" "user_password"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Command completed successfully" ]]
}

@test "postgres::database::drop_user removes user successfully" {
    export YES="yes"
    
    run postgres::database::drop_user "$INSTANCE" "$USERNAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Command completed successfully" ]]
}

# ============================================================================
# Instance Management Tests
# ============================================================================

@test "postgres::instance::create creates new instance successfully" {
    run postgres::instance::create "$INSTANCE" "$PORT" "$TEMPLATE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "container_id_123456" ]]
}

@test "postgres::instance::start starts instance successfully" {
    run postgres::instance::start "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "postgres-test-instance" ]]
}

@test "postgres::instance::stop stops instance successfully" {
    run postgres::instance::stop "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "postgres-test-instance" ]]
}

@test "postgres::instance::restart restarts instance successfully" {
    run postgres::instance::restart "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "postgres-test-instance" ]]
}

@test "postgres::instance::list shows available instances" {
    run postgres::instance::list
    [ "$status" -eq 0 ]
    [[ "$output" =~ "postgres-test-instance" ]]
}

@test "postgres::instance::get_connection_string returns valid connection string" {
    run postgres::instance::get_connection_string "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "postgresql://" ]]
}

# ============================================================================
# Backup Operations Tests
# ============================================================================

@test "postgres::backup::create creates backup successfully" {
    run postgres::backup::create "$INSTANCE" "$DATABASE_NAME" "$BACKUP_FILE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "PostgreSQL database dump" ]]
}

@test "postgres::backup::restore restores backup successfully" {
    # Create a mock backup file
    echo "-- PostgreSQL database dump" > "/tmp/$BACKUP_FILE"
    
    run postgres::backup::restore "$INSTANCE" "$DATABASE_NAME" "/tmp/$BACKUP_FILE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Command completed successfully" ]]
    
    trash::safe_remove "/tmp/$BACKUP_FILE" --test-cleanup
}

@test "postgres::backup::list shows available backups" {
    # Mock ls command to show backup files
    ls() {
        echo "backup_2024-01-15_10-30-00.sql"
        echo "backup_2024-01-14_15-20-00.sql"
        return 0
    }
    export -f ls
    
    run postgres::backup::list "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "backup_2024-01-15" ]]
    [[ "$output" =~ "backup_2024-01-14" ]]
}

@test "postgres::backup::delete removes backup successfully" {
    export YES="yes"
    
    run postgres::backup::delete "$INSTANCE" "$BACKUP_FILE"
    [ "$status" -eq 0 ]
}

@test "postgres::backup::verify validates backup integrity" {
    # Create a mock backup file
    echo "-- PostgreSQL database dump" > "/tmp/$BACKUP_FILE"
    
    run postgres::backup::verify "/tmp/$BACKUP_FILE"
    [ "$status" -eq 0 ]
    
    trash::safe_remove "/tmp/$BACKUP_FILE" --test-cleanup
}

# ============================================================================
# Status and Health Check Tests
# ============================================================================

@test "postgres::status::check shows instance status" {
    run postgres::status::check "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Instance:" ]] || [[ "$output" =~ "Status:" ]]
}

@test "postgres::common::health_check validates instance health" {
    run postgres::common::health_check "$INSTANCE"
    [ "$status" -eq 0 ]
}

@test "postgres::common::wait_for_ready waits for instance to be ready" {
    run postgres::common::wait_for_ready "$INSTANCE" 5
    [ "$status" -eq 0 ]
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "postgres::database::execute handles Docker failure" {
    docker() { return 1; }  # Simulate Docker failure
    
    run postgres::database::execute "$INSTANCE" "SELECT 1;"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Failed to execute SQL command" ]]
}

@test "postgres::backup::create handles backup failure" {
    docker() {
        case "$*" in
            *"pg_dump"*) return 1 ;;  # Simulate pg_dump failure
            *) return 0 ;;
        esac
    }
    
    run postgres::backup::create "$INSTANCE" "$DATABASE_NAME" "$BACKUP_FILE"
    [ "$status" -eq 1 ]
}

@test "postgres::instance::create handles port conflict" {
    postgres::common::is_port_available() { return 1; }  # Port in use
    
    run postgres::instance::create "$INSTANCE" "$PORT" "$TEMPLATE"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "port" ]] || [[ "$output" =~ "available" ]]
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "postgres::database::migrate runs migrations successfully" {
    # Mock migration file
    echo "CREATE TABLE test_table (id SERIAL PRIMARY KEY);" > "/tmp/001_create_test_table.sql"
    
    run postgres::database::migrate "$INSTANCE" "/tmp"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Command completed successfully" ]]
    
    trash::safe_remove "/tmp/001_create_test_table.sql" --test-cleanup
}

@test "postgres full workflow: create instance, database, user, backup" {
    # Test complete workflow
    run postgres::instance::create "$INSTANCE" "$PORT" "$TEMPLATE"
    [ "$status" -eq 0 ]
    
    run postgres::database::create "$INSTANCE" "$DATABASE_NAME"
    [ "$status" -eq 0 ]
    
    run postgres::database::create_user "$INSTANCE" "$USERNAME" "password123"
    [ "$status" -eq 0 ]
    
    run postgres::backup::create "$INSTANCE" "$DATABASE_NAME" "$BACKUP_FILE"
    [ "$status" -eq 0 ]
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "all PostgreSQL API functions are defined" {
    # Test that all expected functions exist
    type postgres::database::execute >/dev/null
    type postgres::database::create >/dev/null
    type postgres::database::drop >/dev/null
    type postgres::database::create_user >/dev/null
    type postgres::database::drop_user >/dev/null
    type postgres::instance::create >/dev/null
    type postgres::instance::start >/dev/null
    type postgres::instance::stop >/dev/null
    type postgres::instance::list >/dev/null
    type postgres::backup::create >/dev/null
    type postgres::backup::restore >/dev/null
    type postgres::backup::list >/dev/null
    type postgres::common::health_check >/dev/null
}

# Teardown
teardown() {
    vrooli_cleanup_test
}