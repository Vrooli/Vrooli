#!/usr/bin/env bats
# Tests for PostgreSQL Backup Operations

# Load Vrooli test infrastructure
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "postgres-backup"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    POSTGRES_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and functions once
    source "${POSTGRES_DIR}/config/defaults.sh"
    source "${POSTGRES_DIR}/config/messages.sh"
    source "${SCRIPT_DIR}/common.sh"
    source "${SCRIPT_DIR}/backup.sh"
    
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
    export POSTGRES_BACKUP_DIR="/var/backups/postgres"
    export POSTGRES_CONTAINER_PREFIX="postgres"
    export POSTGRES_DEFAULT_USER="postgres"
    export POSTGRES_DEFAULT_DB="vrooli"
    export INSTANCE="test-instance"
    export DATABASE_NAME="test_db"
    export BACKUP_NAME="test_backup_20240115_103000"
    export BACKUP_TYPE="full"
    export YES="no"
    
    # Set up test message variables
    export MSG_INSTANCE_NOT_FOUND="Instance not found"
    export MSG_BACKUP_CREATED="Backup created successfully"
    export MSG_BACKUP_RESTORED="Backup restored successfully"
    export MSG_BACKUP_DELETED="Backup deleted"
    export MSG_BACKUP_NOT_FOUND="Backup not found"
    export MSG_BACKUP_VERIFIED="Backup verified"
    
    # Mock PostgreSQL utility functions
    postgres::common::container_exists() { return 0; }
    postgres::common::is_running() { return 0; }
    postgres::common::health_check() { return 0; }
    postgres::common::wait_for_ready() { return 0; }
    export -f postgres::common::container_exists postgres::common::is_running
    export -f postgres::common::health_check postgres::common::wait_for_ready
    
    # Mock Docker functions for backup operations
    docker() {
        case "$*" in
            *"exec"*"pg_dump"*"--schema-only"*)
                # Mock schema-only backup
                echo "-- PostgreSQL database dump (schema only)"
                echo "-- Dumped by pg_dump version 14.5"
                echo "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT);"
                return 0
                ;;
            *"exec"*"pg_dump"*"--data-only"*)
                # Mock data-only backup
                echo "-- PostgreSQL database dump (data only)"
                echo "COPY users (id, name) FROM stdin;"
                echo "1	Alice"
                echo "2	Bob"
                echo "\\."
                return 0
                ;;
            *"exec"*"pg_dump"*)
                # Mock full backup
                echo "-- PostgreSQL database dump"
                echo "-- Dumped by pg_dump version 14.5"
                echo "-- Database: $DATABASE_NAME"
                echo "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT);"
                echo "COPY users (id, name) FROM stdin;"
                echo "1	Alice"
                echo "\\."
                return 0
                ;;
            *"exec"*"psql"*"-f"*)
                # Mock restore operation
                echo "CREATE TABLE"
                echo "COPY 1"
                return 0
                ;;
            *"exec"*"pg_isready"*)
                # Mock health check
                echo "postgres-test-instance:5432 - accepting connections"
                return 0
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f docker
    
    # Mock file system operations
    mkdir() { return 0; }
    cp() { return 0; }
    mv() { return 0; }
    rm() { return 0; }
    chmod() { return 0; }
    chown() { return 0; }
    ls() {
        case "$*" in
            *"$POSTGRES_BACKUP_DIR"*)
                echo "test_backup_20240115_103000"
                echo "test_backup_20240114_153000"
                echo "old_backup_20240101_120000"
                return 0
                ;;
            *) return 0 ;;
        esac
    }
    export -f mkdir cp mv rm chmod chown ls
    
    # Mock file operations
    test_file() { return 0; }  # Mock file existence check
    export -f test_file
    stat() {
        case "$*" in
            *"--format=%s"*)
                echo "1048576"  # 1MB backup file size
                return 0
                ;;
            *)
                echo "Access: 2024-01-15 10:30:00"
                echo "Modify: 2024-01-15 10:30:00"
                return 0
                ;;
        esac
    }
    export -f [[ stat
    
    # Mock compression tools
    gzip() {
        case "$*" in
            *"-t"*) return 0 ;;  # Test archive integrity
            *) return 0 ;;       # Compress file
        esac
    }
    gunzip() { return 0; }
    export -f gzip gunzip
    
    # Mock log functions
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::warning() { echo "[WARNING] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::debug() { [[ "${DEBUG:-}" == "true" ]] && echo "[DEBUG] $*" >&2 || true; }
    export -f log::info log::error log::warning log::success log::debug
    
    # Mock system commands
    date() { echo "20240115_103000"; }
    du() { echo "1024	/path/to/backup"; }
    find() {
        case "$*" in
            *"-name"*"*.sql"*)
                echo "/var/backups/postgres/test-instance/backup1.sql"
                echo "/var/backups/postgres/test-instance/backup2.sql"
                return 0
                ;;
            *"-mtime"*)
                echo "/var/backups/postgres/test-instance/old_backup.sql"
                return 0
                ;;
            *) return 0 ;;
        esac
    }
    export -f date du find
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# ============================================================================
# Backup Creation Tests
# ============================================================================

@test "postgres::backup::create creates full backup successfully" {
    run postgres::backup::create "$INSTANCE" "$BACKUP_NAME" "full"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Creating full backup" ]]
    [[ "$output" =~ "PostgreSQL database dump" ]]
}

@test "postgres::backup::create creates schema-only backup" {
    run postgres::backup::create "$INSTANCE" "$BACKUP_NAME" "schema"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Creating schema backup" ]]
    [[ "$output" =~ "schema only" ]]
}

@test "postgres::backup::create creates data-only backup" {
    run postgres::backup::create "$INSTANCE" "$BACKUP_NAME" "data"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Creating data backup" ]]
    [[ "$output" =~ "data only" ]]
}

@test "postgres::backup::create uses default timestamp when backup name not provided" {
    run postgres::backup::create "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Creating full backup" ]]
    [[ "$output" =~ "20240115_103000" ]]
}

@test "postgres::backup::create fails when instance name missing" {
    run postgres::backup::create ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Instance name is required" ]]
}

@test "postgres::backup::create fails when instance not found" {
    postgres::common::container_exists() { return 1; }
    
    run postgres::backup::create "$INSTANCE" "$BACKUP_NAME"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Instance not found" ]]
}

@test "postgres::backup::create fails when instance not running" {
    postgres::common::is_running() { return 1; }
    
    run postgres::backup::create "$INSTANCE" "$BACKUP_NAME"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "is not running" ]]
}

@test "postgres::backup::create fails with invalid backup type" {
    run postgres::backup::create "$INSTANCE" "$BACKUP_NAME" "invalid"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Invalid backup type" ]]
}

@test "postgres::backup::create handles pg_dump failure" {
    docker() {
        case "$*" in
            *"pg_dump"*) return 1 ;;  # Simulate pg_dump failure
            *) return 0 ;;
        esac
    }
    
    run postgres::backup::create "$INSTANCE" "$BACKUP_NAME"
    [ "$status" -eq 1 ]
}

# ============================================================================
# Backup Restore Tests
# ============================================================================

@test "postgres::backup::restore restores backup successfully" {
    run postgres::backup::restore "$INSTANCE" "$DATABASE_NAME" "/path/to/backup.sql"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "CREATE TABLE" ]]
    [[ "$output" =~ "COPY" ]]
}

@test "postgres::backup::restore fails when instance name missing" {
    run postgres::backup::restore "" "$DATABASE_NAME" "/path/to/backup.sql"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Instance name is required" ]]
}

@test "postgres::backup::restore fails when database name missing" {
    run postgres::backup::restore "$INSTANCE" "" "/path/to/backup.sql"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Database name is required" ]]
}

@test "postgres::backup::restore fails when backup file missing" {
    run postgres::backup::restore "$INSTANCE" "$DATABASE_NAME" ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Backup file path is required" ]]
}

@test "postgres::backup::restore fails when backup file does not exist" {
    test_file() { return 1; }  # Mock file does not exist
    
    run postgres::backup::restore "$INSTANCE" "$DATABASE_NAME" "/nonexistent/backup.sql"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Backup file not found" ]]
}

@test "postgres::backup::restore handles psql failure" {
    docker() {
        case "$*" in
            *"psql"*"-f"*) return 1 ;;  # Simulate psql failure
            *) return 0 ;;
        esac
    }
    
    run postgres::backup::restore "$INSTANCE" "$DATABASE_NAME" "/path/to/backup.sql"
    [ "$status" -eq 1 ]
}

# ============================================================================
# Backup List Tests
# ============================================================================

@test "postgres::backup::list shows available backups" {
    run postgres::backup::list "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test_backup_20240115_103000" ]]
    [[ "$output" =~ "test_backup_20240114_153000" ]]
    [[ "$output" =~ "old_backup_20240101_120000" ]]
}

@test "postgres::backup::list shows backup details with verbose flag" {
    export VERBOSE="yes"
    
    run postgres::backup::list "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "1024" ]]  # File size
    [[ "$output" =~ "2024-01-15" ]]  # Date
}

@test "postgres::backup::list handles no backups found" {
    ls() { return 1; }  # No files found
    
    run postgres::backup::list "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No backups found" ]]
}

# ============================================================================
# Backup Information Tests
# ============================================================================

@test "postgres::backup::show_backup_info displays backup metadata" {
    run postgres::backup::show_backup_info "$INSTANCE" "$BACKUP_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Backup Name:" ]]
    [[ "$output" =~ "Size:" ]]
    [[ "$output" =~ "Created:" ]]
}

@test "postgres::backup::show_backup_info handles missing backup" {
    [[ () { return 1; }  # Mock backup file does not exist
    
    run postgres::backup::show_backup_info "$INSTANCE" "$BACKUP_NAME"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Backup not found" ]]
}

# ============================================================================
# Backup Deletion Tests
# ============================================================================

@test "postgres::backup::delete removes backup successfully" {
    export YES="yes"
    
    run postgres::backup::delete "$INSTANCE" "$BACKUP_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "deleted" ]] || [[ "$output" =~ "removed" ]]
}

@test "postgres::backup::delete requires confirmation by default" {
    export YES="no"
    
    run postgres::backup::delete "$INSTANCE" "$BACKUP_NAME"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "cancelled" ]] || [[ "$output" =~ "confirmation" ]]
}

@test "postgres::backup::delete handles missing backup" {
    [[ () { return 1; }  # Mock backup file does not exist
    export YES="yes"
    
    run postgres::backup::delete "$INSTANCE" "$BACKUP_NAME"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Backup not found" ]]
}

# ============================================================================
# Backup Cleanup Tests
# ============================================================================

@test "postgres::backup::cleanup removes old backups" {
    export RETENTION_DAYS="7"
    
    run postgres::backup::cleanup "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "cleanup" ]] || [[ "$output" =~ "old backups" ]]
}

@test "postgres::backup::cleanup with dry run shows what would be deleted" {
    export DRY_RUN="yes"
    export RETENTION_DAYS="7"
    
    run postgres::backup::cleanup "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "would be deleted" ]] || [[ "$output" =~ "dry run" ]]
}

# ============================================================================
# Backup Verification Tests
# ============================================================================

@test "postgres::backup::verify validates backup integrity" {
    run postgres::backup::verify "/path/to/backup.sql"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "valid" ]] || [[ "$output" =~ "verified" ]]
}

@test "postgres::backup::verify handles corrupted backup" {
    gzip() {
        case "$*" in
            *"-t"*) return 1 ;;  # Simulate corruption
            *) return 0 ;;
        esac
    }
    
    run postgres::backup::verify "/path/to/backup.sql.gz"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "corrupt" ]] || [[ "$output" =~ "invalid" ]]
}

@test "postgres::backup::verify handles missing backup file" {
    test_file() { return 1; }  # Mock file does not exist
    
    run postgres::backup::verify "/nonexistent/backup.sql"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not found" ]]
}

# ============================================================================
# Backup Metadata Tests
# ============================================================================

@test "postgres::backup::create_metadata generates backup metadata" {
    run postgres::backup::create_metadata "$INSTANCE" "$BACKUP_NAME" "full"
    [ "$status" -eq 0 ]
}

@test "postgres::backup::finalize_metadata updates backup completion status" {
    run postgres::backup::finalize_metadata "$INSTANCE" "$BACKUP_NAME" "success"
    [ "$status" -eq 0 ]
}

# ============================================================================
# Advanced Backup Features Tests
# ============================================================================

@test "postgres::backup::create supports compression" {
    export COMPRESS="yes"
    
    run postgres::backup::create "$INSTANCE" "$BACKUP_NAME" "full"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "compress" ]] || [[ "$output" =~ "gzip" ]]
}

@test "postgres::backup::create supports custom destination" {
    export BACKUP_DESTINATION="/custom/backup/path"
    
    run postgres::backup::create "$INSTANCE" "$BACKUP_NAME" "full"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "/custom/backup/path" ]]
}

@test "postgres::backup::create includes multiple databases" {
    export DATABASES="db1,db2,db3"
    
    run postgres::backup::create "$INSTANCE" "$BACKUP_NAME" "full"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "multiple databases" ]] || [[ "$output" =~ "db1" ]]
}

# ============================================================================
# Backup Scheduling Tests
# ============================================================================

@test "postgres::backup::schedule creates automated backup job" {
    export SCHEDULE="0 2 * * *"  # Daily at 2 AM
    
    run postgres::backup::schedule "$INSTANCE" "$SCHEDULE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "scheduled" ]] || [[ "$output" =~ "cron" ]]
}

@test "postgres::backup::unschedule removes automated backup job" {
    run postgres::backup::unschedule "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "removed" ]] || [[ "$output" =~ "unscheduled" ]]
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "all PostgreSQL backup functions are defined" {
    # Test that all expected functions exist
    type postgres::backup::create >/dev/null
    type postgres::backup::restore >/dev/null
    type postgres::backup::list >/dev/null
    type postgres::backup::show_backup_info >/dev/null
    type postgres::backup::delete >/dev/null
    type postgres::backup::cleanup >/dev/null
    type postgres::backup::verify >/dev/null
    type postgres::backup::create_metadata >/dev/null
    type postgres::backup::finalize_metadata >/dev/null
}

# Teardown
teardown() {
    vrooli_cleanup_test
}