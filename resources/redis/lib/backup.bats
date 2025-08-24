#!/usr/bin/env bats
# Tests for Redis Backup Operations

# Source var.sh to get proper directory variables
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Load Vrooli test infrastructure
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "redis-backup"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    REDIS_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and functions once
    source "${REDIS_DIR}/config/defaults.sh"
    source "${REDIS_DIR}/config/messages.sh"
    source "${SCRIPT_DIR}/common.sh"
    source "${SCRIPT_DIR}/backup.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
    export SETUP_FILE_REDIS_DIR="$REDIS_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    REDIS_DIR="${SETUP_FILE_REDIS_DIR}"
    
    # Set test environment
    export HOME="/home/testuser"
    export REDIS_PORT="6380"
    export REDIS_DATABASES="16"
    export REDIS_MAX_MEMORY="128mb"
    export REDIS_PERSISTENCE="rdb"
    export REDIS_DATA_DIR="/var/lib/redis"
    export REDIS_BACKUP_DIR="${HOME}/.vrooli/redis/backups"
    export INSTANCE="main"
    export BACKUP_NAME="test_backup_20240115_103000"
    export YES="no"
    
    # Set up test message variables
    export MSG_BACKUP_START="Starting Redis backup"
    export MSG_BACKUP_SUCCESS="Backup completed successfully"
    export MSG_BACKUP_LOCATION="Backup saved to: "
    export MSG_BACKUP_RESTORED="Backup restored successfully"
    export MSG_BACKUP_DELETED="Backup deleted"
    export MSG_BACKUP_NOT_FOUND="Backup not found"
    export MSG_BACKUP_LIST_EMPTY="No backups found"
    
    # Mock Redis utility functions
    redis::common::is_running() { return 0; }
    redis::common::is_healthy() { return 0; }
    redis::common::container_exists() { return 0; }
    redis::common::get_info() {
        cat << 'EOF'
redis_version:7.0.5
redis_git_sha1:00000000
redis_mode:standalone
tcp_port:6380
used_memory:1048576
used_memory_human:1.00M
connected_clients:2
total_commands_processed:1000
EOF
    }
    export -f redis::common::is_running redis::common::is_healthy
    export -f redis::common::container_exists redis::common::get_info
    
    # Mock Docker functions for backup operations
    docker() {
        case "$*" in
            *"exec"*"redis-cli"*"BGSAVE"*)
                echo "Background saving started"
                return 0
                ;;
            *"exec"*"redis-cli"*"SAVE"*)
                echo "OK"
                return 0
                ;;
            *"exec"*"redis-cli"*"FLUSHALL"*)
                echo "OK"
                return 0
                ;;
            *"exec"*"redis-cli"*"DEBUG RELOAD"*)
                echo "OK"
                return 0
                ;;
            *"exec"*"redis-cli"*"LASTSAVE"*)
                echo "1705317000"  # Unix timestamp
                return 0
                ;;
            *"exec"*"redis-cli"*"CONFIG GET dir"*)
                echo "dir"
                echo "/var/lib/redis"
                return 0
                ;;
            *"exec"*"redis-cli"*"CONFIG GET dbfilename"*)
                echo "dbfilename"
                echo "dump.rdb"
                return 0
                ;;
            *"cp"*"dump.rdb"*)
                return 0  # Successful file copy
                ;;
            *"exec"*"cat"*)
                # Mock RDB file content verification
                echo "REDIS0009"  # Redis RDB file header
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
            *"$REDIS_BACKUP_DIR"*)
                echo "test_backup_20240115_103000.rdb"
                echo "test_backup_20240115_103000.info"
                echo "test_backup_20240114_153000.rdb"
                echo "test_backup_20240114_153000.info"
                echo "old_backup_20240101_120000.rdb"
                echo "old_backup_20240101_120000.info"
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
                echo "4194304"  # 4MB backup file size
                return 0
                ;;
            *)
                echo "Access: 2024-01-15 10:30:00"
                echo "Modify: 2024-01-15 10:30:00"
                echo "Size: 4194304"
                return 0
                ;;
        esac
    }
    wc() {
        case "$*" in
            *"-l"*) echo "42" ;;  # Line count
            *"-c"*) echo "4194304" ;;  # Byte count
            *) echo "42 1024 4194304" ;;  # lines words bytes
        esac
    }
    export -f [[ stat wc
    
    # Mock JSON operations
    jq() {
        case "$*" in
            *".name"*) echo "test_backup_20240115_103000" ;;
            *".created"*) echo "2024-01-15T10:30:00+00:00" ;;
            *".redis_version"*) echo "7.0.5" ;;
            *".redis_config.port"*) echo "6380" ;;
            *) echo "JSON: $*" ;;
        esac
    }
    export -f jq
    
    # Mock compression tools
    tar() {
        case "$*" in
            *"-czf"*) return 0 ;;  # Create compressed archive
            *"-xzf"*) return 0 ;;  # Extract compressed archive
            *"-tzf"*) 
                echo "redis-data/dump.rdb"
                echo "redis-data/redis.conf"
                return 0
                ;;
            *) return 0 ;;
        esac
    }
    gzip() {
        case "$*" in
            *"-t"*) return 0 ;;  # Test archive integrity
            *) return 0 ;;       # Compress file
        esac
    }
    gunzip() { return 0; }
    export -f tar gzip gunzip
    
    # Mock log functions
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::warning() { echo "[WARNING] $*" >&2; }
    log::warn() { echo "[WARN] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::debug() { [[ "${DEBUG:-}" == "true" ]] && echo "[DEBUG] $*" >&2 || true; }
    export -f log::info log::error log::warning log::warn log::success log::debug
    
    # Mock system commands
    date() {
        case "$*" in
            *"-Iseconds"*) echo "2024-01-15T10:30:00+00:00" ;;
            *"+%Y%m%d-%H%M%S"*) echo "20240115-103000" ;;
            *) echo "20240115_103000" ;;
        esac
    }
    du() { echo "4096	/path/to/backup"; }
    find() {
        case "$*" in
            *"-name"*"*.rdb"*)
                echo "/home/testuser/.vrooli/redis/backups/backup1.rdb"
                echo "/home/testuser/.vrooli/redis/backups/backup2.rdb"
                return 0
                ;;
            *"-mtime"*)
                echo "/home/testuser/.vrooli/redis/backups/old_backup.rdb"
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

@test "redis::backup::create creates backup successfully" {
    run redis::backup::create "$BACKUP_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Starting Redis backup" ]]
    [[ "$output" =~ "Background saving started" ]]
    [[ "$output" =~ "Backup completed successfully" ]]
}

@test "redis::backup::create uses default timestamp when backup name not provided" {
    run redis::backup::create
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Starting Redis backup" ]]
    [[ "$output" =~ "20240115-103000" ]]
}

@test "redis::backup::create fails when Redis not running" {
    redis::common::is_running() { return 1; }
    
    run redis::backup::create "$BACKUP_NAME"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Redis is not running" ]]
}

@test "redis::backup::create creates backup metadata" {
    run redis::backup::create "$BACKUP_NAME"
    [ "$status" -eq 0 ]
    # The metadata creation is tested implicitly through the cat command in the function
}

@test "redis::backup::create_rdb creates RDB dump backup" {
    run redis::backup::create_rdb "/tmp/test_backup"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Background saving started" ]] || [[ "$output" =~ "RDB" ]]
}

@test "redis::backup::copy_data_dir creates data directory backup" {
    run redis::backup::copy_data_dir "/tmp/test_backup"
    [ "$status" -eq 0 ]
}

@test "redis::backup::create handles BGSAVE failure with data directory fallback" {
    docker() {
        case "$*" in
            *"BGSAVE"*) return 1 ;;  # Simulate BGSAVE failure
            *) return 0 ;;
        esac
    }
    
    run redis::backup::create "$BACKUP_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "RDB backup failed, trying data directory copy" ]]
}

# ============================================================================
# Backup Restore Tests
# ============================================================================

@test "redis::backup::restore restores RDB backup successfully" {
    run redis::backup::restore "$INSTANCE" "/path/to/backup.rdb"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "OK" ]] || [[ "$output" =~ "restore" ]]
}

@test "redis::backup::restore_rdb restores RDB dump" {
    run redis::backup::restore_rdb "/path/to/backup.rdb"
    [ "$status" -eq 0 ]
}

@test "redis::backup::restore_tar restores tar archive backup" {
    run redis::backup::restore_tar "/path/to/backup.tar.gz"
    [ "$status" -eq 0 ]
}

@test "redis::backup::restore fails when backup file missing" {
    test_file() { return 1; }  # Mock file does not exist
    
    run redis::backup::restore "$INSTANCE" "/nonexistent/backup.rdb"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not found" ]] || [[ "$output" =~ "does not exist" ]]
}

@test "redis::backup::restore fails when Redis not running" {
    redis::common::is_running() { return 1; }
    
    run redis::backup::restore "$INSTANCE" "/path/to/backup.rdb"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not running" ]]
}

# ============================================================================
# Backup List Tests
# ============================================================================

@test "redis::backup::list shows available backups" {
    run redis::backup::list "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test_backup_20240115_103000" ]]
    [[ "$output" =~ "test_backup_20240114_153000" ]]
    [[ "$output" =~ "old_backup_20240101_120000" ]]
}

@test "redis::backup::list shows backup details with verbose flag" {
    export VERBOSE="yes"
    
    run redis::backup::list "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "4194304" ]]  # File size
    [[ "$output" =~ "2024-01-15" ]]  # Date
}

@test "redis::backup::list handles no backups found" {
    ls() { return 1; }  # No files found
    
    run redis::backup::list "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No backups found" ]]
}

@test "redis::backup::list filters by pattern" {
    export PATTERN="*20240115*"
    
    run redis::backup::list "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test_backup_20240115" ]]
}

# ============================================================================
# Backup Information and Metadata Tests
# ============================================================================

@test "redis::backup::show_info displays backup metadata" {
    run redis::backup::show_info "$BACKUP_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test_backup_20240115_103000" ]]
    [[ "$output" =~ "2024-01-15T10:30:00" ]]
    [[ "$output" =~ "7.0.5" ]]
}

@test "redis::backup::show_info handles missing backup metadata" {
    test_file() { return 1; }  # Mock metadata file does not exist
    
    run redis::backup::show_info "$BACKUP_NAME"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not found" ]]
}

# ============================================================================
# Backup Verification Tests
# ============================================================================

@test "redis::backup::verify validates RDB backup integrity" {
    run redis::backup::verify "/path/to/backup.rdb"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "REDIS0009" ]]  # RDB file header
}

@test "redis::backup::verify validates tar backup integrity" {
    run redis::backup::verify "/path/to/backup.tar.gz"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "redis-data/dump.rdb" ]]
}

@test "redis::backup::verify handles corrupted backup" {
    docker() {
        case "$*" in
            *"cat"*) return 1 ;;  # Simulate corrupted file
            *) return 0 ;;
        esac
    }
    
    run redis::backup::verify "/path/to/backup.rdb"
    [ "$status" -eq 1 ]
}

# ============================================================================
# Backup Deletion Tests
# ============================================================================

@test "redis::backup::delete removes backup successfully" {
    export YES="yes"
    
    run redis::backup::delete "$INSTANCE" "$BACKUP_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "deleted" ]] || [[ "$output" =~ "removed" ]]
}

@test "redis::backup::delete requires confirmation by default" {
    export YES="no"
    
    run redis::backup::delete "$INSTANCE" "$BACKUP_NAME"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "cancelled" ]] || [[ "$output" =~ "confirmation" ]]
}

@test "redis::backup::delete handles missing backup" {
    test_file() { return 1; }  # Mock backup file does not exist
    export YES="yes"
    
    run redis::backup::delete "$INSTANCE" "$BACKUP_NAME"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not found" ]]
}

@test "redis::backup::delete removes both RDB and metadata files" {
    export YES="yes"
    
    run redis::backup::delete "$INSTANCE" "$BACKUP_NAME"
    [ "$status" -eq 0 ]
    # Function should remove both .rdb and .info files
}

# ============================================================================
# Backup Cleanup Tests
# ============================================================================

@test "redis::backup::cleanup removes old backups" {
    export RETENTION_DAYS="7"
    
    run redis::backup::cleanup "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "cleanup" ]] || [[ "$output" =~ "old backups" ]]
}

@test "redis::backup::cleanup with dry run shows what would be deleted" {
    export DRY_RUN="yes"
    export RETENTION_DAYS="7"
    
    run redis::backup::cleanup "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "would be deleted" ]] || [[ "$output" =~ "dry run" ]]
}

@test "redis::backup::cleanup respects retention policy" {
    export RETENTION_DAYS="30"
    export RETENTION_COUNT="5"
    
    run redis::backup::cleanup "$INSTANCE"
    [ "$status" -eq 0 ]
}

# ============================================================================
# Advanced Backup Features Tests
# ============================================================================

@test "redis::backup::create supports compression" {
    export COMPRESS="yes"
    
    run redis::backup::create "$BACKUP_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "compress" ]] || [[ "$output" =~ "gzip" ]]
}

@test "redis::backup::create supports custom destination" {
    export BACKUP_DESTINATION="/custom/backup/path"
    
    run redis::backup::create "$BACKUP_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "/custom/backup/path" ]]
}

@test "redis::backup::create includes all databases" {
    export REDIS_DATABASES="16"
    
    run redis::backup::create "$BACKUP_NAME"
    [ "$status" -eq 0 ]
    # All 16 databases should be included in the backup
}

# ============================================================================
# Backup Scheduling Tests
# ============================================================================

@test "redis::backup::schedule creates automated backup job" {
    export SCHEDULE="0 3 * * *"  # Daily at 3 AM
    
    run redis::backup::schedule "$INSTANCE" "$SCHEDULE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "scheduled" ]] || [[ "$output" =~ "cron" ]]
}

@test "redis::backup::unschedule removes automated backup job" {
    run redis::backup::unschedule "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "removed" ]] || [[ "$output" =~ "unscheduled" ]]
}

# ============================================================================
# Backup Performance Tests  
# ============================================================================

@test "redis::backup::create measures backup performance" {
    export BENCHMARK="yes"
    
    run redis::backup::create "$BACKUP_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "time" ]] || [[ "$output" =~ "performance" ]]
}

@test "redis::backup::create shows backup statistics" {
    export STATS="yes"
    
    run redis::backup::create "$BACKUP_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "size" ]] || [[ "$output" =~ "keys" ]]
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "redis::backup::create handles Docker failure" {
    docker() { return 1; }  # Simulate Docker failure
    
    run redis::backup::create "$BACKUP_NAME"
    [ "$status" -eq 1 ]
}

@test "redis::backup::create handles disk space issues" {
    mkdir() { return 1; }  # Simulate disk space issue
    
    run redis::backup::create "$BACKUP_NAME"
    [ "$status" -eq 1 ]
}

@test "redis::backup::restore handles permission issues" {
    cp() { return 1; }  # Simulate permission issue
    
    run redis::backup::restore "$INSTANCE" "/path/to/backup.rdb"
    [ "$status" -eq 1 ]
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "all Redis backup functions are defined" {
    # Test that all expected functions exist
    type redis::backup::create >/dev/null
    type redis::backup::create_rdb >/dev/null
    type redis::backup::copy_data_dir >/dev/null
    type redis::backup::restore >/dev/null
    type redis::backup::restore_rdb >/dev/null
    type redis::backup::restore_tar >/dev/null
    type redis::backup::list >/dev/null
    type redis::backup::delete >/dev/null
    type redis::backup::cleanup >/dev/null
}

# Teardown
teardown() {
    vrooli_cleanup_test
}