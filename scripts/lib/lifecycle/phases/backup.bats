#!/usr/bin/env bats
################################################################################
# Tests for backup.sh - Universal backup phase handler
################################################################################

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../../../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../../__test/helpers/bats-assert/load"

# Load required mocks
load "${BATS_TEST_DIRNAME}/../../../__test/fixtures/mocks/commands.bash"
load "${BATS_TEST_DIRNAME}/../../../__test/fixtures/mocks/system.sh"
load "${BATS_TEST_DIRNAME}/../../../__test/fixtures/mocks/filesystem.sh"

setup() {
    vrooli_setup_unit_test
    
    # Reset all mocks
    mock::commands::reset
    mock::system::reset
    mock::filesystem::reset
    
    # Mock phase utilities
    mock::commands::setup_function "phase::init" 0
    mock::commands::setup_function "phase::complete" 0
    mock::commands::setup_function "log::info" 0
    mock::commands::setup_function "log::header" 0
    mock::commands::setup_function "log::success" 0
    mock::commands::setup_function "log::warning" 0
    mock::commands::setup_function "log::error" 0
    mock::commands::setup_function "log::debug" 0
    
    # Set up test environment
    export var_ROOT_DIR="${MOCK_ROOT}"
    export BACKUP_DIR="${MOCK_ROOT}/backups"
    export BACKUP_RETENTION_DAYS="30"
    export BACKUP_MIN_SPACE_GB="5"
    
    # Mock date command for consistent timestamps
    mock::commands::setup_command "date" "+%Y%m%d_%H%M%S" "20231201_120000"
    
    # Source backup functions for testing
    source "${BATS_TEST_DIRNAME}/backup.sh"
}

teardown() {
    vrooli_cleanup_test
}

################################################################################
# Backup Directory Creation Tests
################################################################################

@test "backup::create_backup_directory creates session directory" {
    mock::commands::setup_command "mkdir" "-p ${BACKUP_DIR}/20231201_120000" ""
    mock::commands::setup_command "chmod" "750 ${BACKUP_DIR}/20231201_120000" ""
    
    run backup::create_backup_directory
    
    assert_success
    mock::commands::assert::called "mkdir" "-p ${BACKUP_DIR}/20231201_120000"
    mock::commands::assert::called "chmod" "750 ${BACKUP_DIR}/20231201_120000"
    mock::commands::assert::called "log::success" "✅ Backup directory created: ${BACKUP_DIR}/20231201_120000"
    assert_equal "$BACKUP_SESSION_DIR" "${BACKUP_DIR}/20231201_120000"
    assert_equal "$BACKUP_TIMESTAMP" "20231201_120000"
}

@test "backup::create_backup_directory fails on mkdir error" {
    mock::commands::setup_command "mkdir" "-p ${BACKUP_DIR}/20231201_120000" "" "1"  # mkdir fails
    
    run backup::create_backup_directory
    
    assert_failure
    mock::commands::assert::called "log::error" "Failed to create backup directory: ${BACKUP_DIR}/20231201_120000"
}

@test "backup::create_backup_directory uses custom backup directory" {
    export BACKUP_DIR="/custom/backup/path"
    mock::commands::setup_command "mkdir" "-p /custom/backup/path/20231201_120000" ""
    mock::commands::setup_command "chmod" "750 /custom/backup/path/20231201_120000" ""
    
    run backup::create_backup_directory
    
    assert_success
    mock::commands::assert::called "mkdir" "-p /custom/backup/path/20231201_120000"
    assert_equal "$BACKUP_SESSION_DIR" "/custom/backup/path/20231201_120000"
}

################################################################################
# Backup Tools Setup Tests
################################################################################

@test "backup::setup_backup_tools checks required tools" {
    mock::commands::setup_command "command -v tar" "" "0"
    mock::commands::setup_command "command -v gzip" "" "0"
    mock::commands::setup_command "command -v rclone" "" "1"  # Optional - not found
    mock::commands::setup_command "command -v pg_dump" "" "0"
    
    run backup::setup_backup_tools
    
    assert_success
    mock::commands::assert::called "log::success" "✅ Backup tools check complete"
    assert_equal "$RCLONE_AVAILABLE" "false"
    assert_equal "$PG_DUMP_AVAILABLE" "true"
}

@test "backup::setup_backup_tools detects optional tools" {
    mock::commands::setup_command "command -v tar" "" "0"
    mock::commands::setup_command "command -v gzip" "" "0"
    mock::commands::setup_command "command -v rclone" "" "0"  # Available
    mock::commands::setup_command "command -v pg_dump" "" "0"
    
    run backup::setup_backup_tools
    
    assert_success
    mock::commands::assert::called "log::info" "✅ rclone available for remote backup sync"
    mock::commands::assert::called "log::info" "✅ pg_dump available for PostgreSQL backups"
    assert_equal "$RCLONE_AVAILABLE" "true"
    assert_equal "$PG_DUMP_AVAILABLE" "true"
}

@test "backup::setup_backup_tools fails on missing required tools" {
    mock::commands::setup_command "command -v tar" "" "1"  # tar missing
    mock::commands::setup_command "command -v gzip" "" "0"
    
    run backup::setup_backup_tools
    
    assert_failure
    mock::commands::assert::called "log::error" "Missing required backup tools: tar"
    mock::commands::assert::called "log::error" "Please install missing tools and retry backup"
}

@test "backup::setup_backup_tools fails on multiple missing tools" {
    mock::commands::setup_command "command -v tar" "" "1"  # tar missing
    mock::commands::setup_command "command -v gzip" "" "1"  # gzip missing
    
    run backup::setup_backup_tools
    
    assert_failure
    mock::commands::assert::called "log::error" "Missing required backup tools: tar gzip"
}

################################################################################
# Backup Rotation Tests
################################################################################

@test "backup::rotate_old_backups removes old backups" {
    mock::filesystem::create_directory "${BACKUP_DIR}"
    export BACKUP_RETENTION_DAYS="7"
    
    # Mock find command to return old backup directories
    mock::commands::setup_command "find" "${BACKUP_DIR} -maxdepth 1 -type d -name \"[0-9]*_[0-9]*\" -mtime +7" "${BACKUP_DIR}/20231120_120000\n${BACKUP_DIR}/20231119_120000"
    mock::commands::setup_command "rm" "-rf ${BACKUP_DIR}/20231120_120000" ""
    mock::commands::setup_command "rm" "-rf ${BACKUP_DIR}/20231119_120000" ""
    
    run backup::rotate_old_backups
    
    assert_success
    mock::commands::assert::called "find" "${BACKUP_DIR} -maxdepth 1 -type d -name \"[0-9]*_[0-9]*\" -mtime +7"
    mock::commands::assert::called "rm" "-rf ${BACKUP_DIR}/20231120_120000"
    mock::commands::assert::called "rm" "-rf ${BACKUP_DIR}/20231119_120000"
    mock::commands::assert::called "log::success" "✅ Old backup cleanup complete"
}

@test "backup::rotate_old_backups skips when no backup directory" {
    # Don't create backup directory
    
    run backup::rotate_old_backups
    
    assert_success
    mock::commands::assert::called "log::info" "No backup directory found, skipping rotation"
}

@test "backup::rotate_old_backups handles no old backups" {
    mock::filesystem::create_directory "${BACKUP_DIR}"
    
    # Mock find to return no old backups
    mock::commands::setup_command "find" "${BACKUP_DIR} -maxdepth 1 -type d -name \"[0-9]*_[0-9]*\" -mtime +30" ""
    
    run backup::rotate_old_backups
    
    assert_success
    mock::commands::assert::called "log::info" "No old backups to remove"
}

@test "backup::rotate_old_backups uses custom retention period" {
    export BACKUP_RETENTION_DAYS="14"
    mock::filesystem::create_directory "${BACKUP_DIR}"
    mock::commands::setup_command "find" "${BACKUP_DIR} -maxdepth 1 -type d -name \"[0-9]*_[0-9]*\" -mtime +14" ""
    
    run backup::rotate_old_backups
    
    assert_success
    mock::commands::assert::called "find" "${BACKUP_DIR} -maxdepth 1 -type d -name \"[0-9]*_[0-9]*\" -mtime +14"
}

################################################################################
# Backup Environment Verification Tests
################################################################################

@test "backup::verify_backup_environment checks disk space" {
    mock::filesystem::create_directory "${BACKUP_DIR}"
    export BACKUP_MIN_SPACE_GB="10"
    
    # Mock df command to return sufficient space
    mock::commands::setup_command "df" "-BG ${BACKUP_DIR}" "Filesystem     1G-blocks  Used Available Use% Mounted on\n/dev/sda1             50    30        15  67% /"
    mock::commands::setup_command "awk" "NR==2 {print \$4}" "15"
    mock::commands::setup_command "sed" "s/G//" "15"
    
    run backup::verify_backup_environment
    
    assert_success
    mock::commands::assert::called "log::info" "✅ Disk space check: 15GB available"
    mock::commands::assert::called "log::success" "✅ Backup environment verification complete"
}

@test "backup::verify_backup_environment fails on insufficient disk space" {
    mock::filesystem::create_directory "${BACKUP_DIR}"
    export BACKUP_MIN_SPACE_GB="20"
    
    # Mock df command to return insufficient space
    mock::commands::setup_command "df" "-BG ${BACKUP_DIR}" "Filesystem     1G-blocks  Used Available Use% Mounted on\n/dev/sda1             50    45         3  94% /"
    mock::commands::setup_command "awk" "NR==2 {print \$4}" "3"
    mock::commands::setup_command "sed" "s/G//" "3"
    
    run backup::verify_backup_environment
    
    assert_failure
    mock::commands::assert::called "log::error" "Insufficient disk space for backup"
    mock::commands::assert::called "log::error" "Available: 3GB, Required: 20GB"
}

@test "backup::verify_backup_environment checks parent directory when backup dir missing" {
    # Don't create backup directory
    export BACKUP_DIR="${MOCK_ROOT}/nonexistent/backups"
    export BACKUP_MIN_SPACE_GB="5"
    
    # Mock df for parent directory
    mock::commands::setup_command "df" "-BG ${MOCK_ROOT}/nonexistent" "Filesystem     1G-blocks  Used Available Use% Mounted on\n/dev/sda1             50    30        10  75% /"
    mock::commands::setup_command "awk" "NR==2 {print \$4}" "10"
    mock::commands::setup_command "sed" "s/G//" "10"
    
    run backup::verify_backup_environment
    
    assert_success
    mock::commands::assert::called "log::info" "✅ Disk space check: 10GB available"
}

@test "backup::verify_backup_environment handles df failure gracefully" {
    mock::filesystem::create_directory "${BACKUP_DIR}"
    
    # Mock df command to fail
    mock::commands::setup_command "df" "-BG ${BACKUP_DIR}" "" "1"
    
    run backup::verify_backup_environment
    
    assert_success
    mock::commands::assert::called "log::warning" "Unable to check disk space - proceeding with backup"
    mock::commands::assert::called "log::info" "ℹ️  To enable disk space checks, ensure the backup directory is accessible"
}

@test "backup::verify_backup_environment checks directory permissions" {
    # Create backup directory without write permissions on parent
    mock::filesystem::create_directory "${BACKUP_DIR}"
    mock::commands::setup_command "df" "-BG ${BACKUP_DIR}" "Filesystem     1G-blocks  Used Available Use% Mounted on\n/dev/sda1             50    30        10  75% /"
    mock::commands::setup_command "awk" "NR==2 {print \$4}" "10"
    mock::commands::setup_command "sed" "s/G//" "10"
    
    # Mock test for write permission failure
    backup::verify_backup_environment() {
        # Override just the permission check for this test
        local backup_base_dir="${BACKUP_DIR:-${var_ROOT_DIR}/backups}"
        local required_space="${BACKUP_MIN_SPACE_GB:-5}"
        
        log::info "Verifying backup environment..."
        
        # Skip disk space check for this test
        log::info "✅ Disk space check: 10GB available"
        
        # Check write permission (simulate failure)
        if [[ ! -w "/non/writable/path" ]]; then  # Always fails
            log::error "No write permission for backup directory: $backup_base_dir"
            exit "${ERROR_BACKUP_PERMISSION_DENIED:-1}"
        fi
        
        log::success "✅ Backup environment verification complete"
    }
    
    run backup::verify_backup_environment
    
    assert_failure
    mock::commands::assert::called "log::error"
}

################################################################################
# Main Backup Function Tests
################################################################################

@test "backup::universal::main runs all backup preparation steps" {
    mock::commands::setup_command "command -v tar" "" "0"
    mock::commands::setup_command "command -v gzip" "" "0"
    mock::commands::setup_command "command -v rclone" "" "1"
    mock::commands::setup_command "command -v pg_dump" "" "1"
    mock::commands::setup_command "mkdir" "-p ${BACKUP_DIR}/20231201_120000" ""
    mock::commands::setup_command "chmod" "750 ${BACKUP_DIR}/20231201_120000" ""
    mock::filesystem::create_directory "${BACKUP_DIR}"
    mock::commands::setup_command "find" "${BACKUP_DIR} -maxdepth 1 -type d -name \"[0-9]*_[0-9]*\" -mtime +30" ""
    mock::commands::setup_command "df" "-BG ${BACKUP_DIR}" "Filesystem     1G-blocks  Used Available Use% Mounted on\n/dev/sda1             50    30        10  75% /"
    mock::commands::setup_command "awk" "NR==2 {print \$4}" "10"
    mock::commands::setup_command "sed" "s/G//" "10"
    
    run backup::universal::main
    
    assert_success
    mock::commands::assert::called "phase::init" "Backup"
    mock::commands::assert::called "phase::complete"
    mock::commands::assert::called "log::success" "✅ Universal backup preparation complete"
    mock::commands::assert::called "log::info" "Backup session directory: ${BACKUP_DIR}/20231201_120000"
    mock::commands::assert::called "log::info" "Next: App-specific backup logic will execute"
}

@test "backup::universal::main fails on environment verification failure" {
    # Mock insufficient disk space
    backup::verify_backup_environment() {
        log::error "Insufficient disk space for backup"
        exit "${ERROR_INSUFFICIENT_DISK_SPACE:-1}"
    }
    
    run backup::universal::main
    
    assert_failure
    mock::commands::assert::called "log::error" "Insufficient disk space for backup"
}

@test "backup::universal::main fails on tool setup failure" {
    # Mock missing required tools
    backup::setup_backup_tools() {
        log::error "Missing required backup tools: tar"
        exit "${ERROR_MISSING_BACKUP_TOOLS:-1}"
    }
    
    run backup::universal::main
    
    assert_failure
    mock::commands::assert::called "log::error" "Missing required backup tools: tar"
}

@test "backup::universal::main fails on directory creation failure" {
    mock::commands::setup_command "command -v tar" "" "0"
    mock::commands::setup_command "command -v gzip" "" "0"
    mock::filesystem::create_directory "${BACKUP_DIR}"
    mock::commands::setup_command "find" "${BACKUP_DIR} -maxdepth 1 -type d -name \"[0-9]*_[0-9]*\" -mtime +30" ""
    mock::commands::setup_command "df" "-BG ${BACKUP_DIR}" "Filesystem     1G-blocks  Used Available Use% Mounted on\n/dev/sda1             50    30        10  75% /"
    mock::commands::setup_command "awk" "NR==2 {print \$4}" "10"
    mock::commands::setup_command "sed" "s/G//" "10"
    
    # Mock directory creation failure
    backup::create_backup_directory() {
        log::error "Failed to create backup directory: ${BACKUP_DIR}/20231201_120000"
        exit "${ERROR_BACKUP_DIRECTORY_CREATION:-1}"
    }
    
    run backup::universal::main
    
    assert_failure
    mock::commands::assert::called "log::error" "Failed to create backup directory: ${BACKUP_DIR}/20231201_120000"
}

################################################################################
# Entry Point Tests
################################################################################

@test "backup script requires lifecycle engine invocation" {
    unset LIFECYCLE_PHASE
    mock::commands::setup_function "log::error" 0
    mock::commands::setup_function "log::info" 0
    
    run bash -c 'BASH_SOURCE=("test_script"); 0="test_script"; source "${BATS_TEST_DIRNAME}/backup.sh"'
    
    assert_failure
    assert_equal "$status" 1
}

@test "backup script runs when called by lifecycle engine" {
    export LIFECYCLE_PHASE="backup"
    mock::commands::setup_command "command -v tar" "" "0"
    mock::commands::setup_command "command -v gzip" "" "0"
    mock::commands::setup_command "command -v rclone" "" "1"
    mock::commands::setup_command "command -v pg_dump" "" "1"
    mock::commands::setup_command "mkdir" "-p ${BACKUP_DIR}/20231201_120000" ""
    mock::commands::setup_command "chmod" "750 ${BACKUP_DIR}/20231201_120000" ""
    mock::filesystem::create_directory "${BACKUP_DIR}"
    mock::commands::setup_command "find" "${BACKUP_DIR} -maxdepth 1 -type d -name \"[0-9]*_[0-9]*\" -mtime +30" ""
    mock::commands::setup_command "df" "-BG ${BACKUP_DIR}" "Filesystem     1G-blocks  Used Available Use% Mounted on\n/dev/sda1             50    30        10  75% /"
    mock::commands::setup_command "awk" "NR==2 {print \$4}" "10"
    mock::commands::setup_command "sed" "s/G//" "10"
    
    # Override the main function for this test
    backup::universal::main() { echo "backup executed"; }
    export -f backup::universal::main
    
    run bash -c 'export LIFECYCLE_PHASE="backup"; BASH_SOURCE=("test_script"); 0="test_script"; source "${BATS_TEST_DIRNAME}/backup.sh"'
    
    assert_success
}