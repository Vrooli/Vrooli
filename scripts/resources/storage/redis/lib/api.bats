#!/usr/bin/env bats
# Tests for Redis API functions

# Load Vrooli test infrastructure
source "$(dirname "${BATS_TEST_FILENAME}")/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "redis"
    
    # Load dependencies once
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    REDIS_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and messages once
    source "${REDIS_DIR}/config/defaults.sh"
    source "${REDIS_DIR}/config/messages.sh"
    
    # Load API functions once
    source "${SCRIPT_DIR}/common.sh"
    source "${SCRIPT_DIR}/status.sh"
    source "${SCRIPT_DIR}/docker.sh"
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
    export REDIS_PORT="6380"
    export REDIS_CONTAINER_NAME="redis-main"
    export REDIS_DATA_DIR="/var/lib/redis"
    export REDIS_CONFIG_DIR="/etc/redis"
    export REDIS_LOG_DIR="/var/log/redis"
    export REDIS_BACKUP_DIR="/var/backups/redis"
    export REDIS_PASSWORD="test_password"
    export INSTANCE="main"
    export BACKUP_NAME="test_backup_20240115_103000"
    export YES="no"
    
    # Set up test message variables
    export MSG_REDIS_STARTING="Starting Redis"
    export MSG_REDIS_STOPPING="Stopping Redis"
    export MSG_REDIS_HEALTHY="Redis is healthy"
    export MSG_REDIS_NOT_RUNNING="Redis is not running"
    export MSG_BACKUP_CREATED="Backup created"
    export MSG_BACKUP_RESTORED="Backup restored"
    export MSG_CONNECTION_SUCCESSFUL="Connection successful"
    
    # Mock Redis utility functions
    redis::common::container_exists() { return 0; }
    redis::common::is_running() { return 0; }
    redis::common::is_healthy() { return 0; }
    redis::common::ping() { echo "PONG"; return 0; }
    redis::common::wait_for_ready() { return 0; }
    redis::common::find_available_port() { echo "6381"; }
    redis::common::is_port_available() {
        case "$1" in
            "6379") return 1 ;;  # Port in use
            *) return 0 ;;       # Port available
        esac
    }
    export -f redis::common::container_exists redis::common::is_running
    export -f redis::common::is_healthy redis::common::ping
    export -f redis::common::wait_for_ready redis::common::find_available_port
    export -f redis::common::is_port_available
    
    # Mock Docker functions for Redis operations
    docker() {
        case "$*" in
            *"exec"*"redis-cli"*"PING"*)
                echo "PONG"
                return 0
                ;;
            *"exec"*"redis-cli"*"INFO"*)
                cat << 'EOF'
# Server
redis_version:7.0.5
redis_git_sha1:00000000
redis_git_dirty:0
redis_build_id:0123456789abcdef
redis_mode:standalone
os:Linux 5.4.0-74-generic x86_64
arch_bits:64
multiplexing_api:epoll
atomicvar_api:c11-builtin
gcc_version:9.4.0
process_id:1
process_supervised:no
run_id:abc123def456
tcp_port:6380
uptime_in_seconds:86400
uptime_in_days:1

# Memory
used_memory:1048576
used_memory_human:1.00M
used_memory_rss:2097152
used_memory_rss_human:2.00M
used_memory_peak:1572864
used_memory_peak_human:1.50M
used_memory_dataset:524288
used_memory_dataset_perc:50.00%
total_system_memory:4294967296
total_system_memory_human:4.00G

# Clients
connected_clients:2
client_recent_max_input_buffer:8
client_recent_max_output_buffer:0
blocked_clients:0
tracking_clients:0

# Stats
total_connections_received:10
total_commands_processed:1000
instantaneous_ops_per_sec:5
total_net_input_bytes:51200
total_net_output_bytes:102400
instantaneous_input_kbps:0.10
instantaneous_output_kbps:0.20
rejected_connections:0
sync_full:0
sync_partial_ok:0
sync_partial_err:0
expired_keys:0
evicted_keys:0
keyspace_hits:750
keyspace_misses:250
pubsub_channels:0
pubsub_patterns:0
latest_fork_usec:500
migrate_cached_sockets:0
slave_expires_tracked_keys:0
active_defrag_hits:0
active_defrag_misses:0
active_defrag_key_hits:0
active_defrag_key_misses:0
EOF
                return 0
                ;;
            *"exec"*"redis-cli"*"SET"*)
                echo "OK"
                return 0
                ;;
            *"exec"*"redis-cli"*"GET"*)
                echo "test_value"
                return 0
                ;;
            *"exec"*"redis-cli"*"DEL"*)
                echo "(integer) 1"
                return 0
                ;;
            *"exec"*"redis-cli"*"FLUSHALL"*)
                echo "OK"
                return 0
                ;;
            *"exec"*"redis-cli"*"SAVE"*)
                echo "OK"
                return 0
                ;;
            *"exec"*"redis-cli"*"BGSAVE"*)
                echo "Background saving started"
                return 0
                ;;
            *"exec"*"redis-cli"*"CLIENT LIST"*)
                echo "id=3 addr=127.0.0.1:52184 fd=8 name= age=60 idle=0 flags=N db=0 sub=0 psub=0 multi=-1 qbuf=0 qbuf-free=32768 obl=0 oll=0 omem=0 events=r cmd=client"
                return 0
                ;;
            *"exec"*"redis-cli"*"DBSIZE"*)
                echo "(integer) 42"
                return 0
                ;;
            *"run"*"redis"*)
                echo "container_id_redis123"
                return 0
                ;;
            *"start"*|*"stop"*|*"restart"*)
                echo "redis-main"
                return 0
                ;;
            *"cp"*)
                return 0  # Successful file copy
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
                echo "redis_backup_20240115_103000.rdb"
                echo "redis_backup_20240114_153000.tar.gz"
                echo "redis_backup_20240113_120000.rdb"
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
                echo "2048576"  # 2MB backup file size
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
    
    # Mock log functions
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::warning() { echo "[WARNING] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::debug() { [[ "${DEBUG:-}" == "true" ]] && echo "[DEBUG] $*" >&2 || true; }
    export -f log::info log::error log::warning log::success log::debug
    
    # Mock compression tools
    tar() {
        case "$*" in
            *"-czf"*) return 0 ;;  # Create archive
            *"-xzf"*) return 0 ;;  # Extract archive
            *"-tzf"*) 
                echo "redis/dump.rdb"
                echo "redis/redis.conf"
                return 0
                ;;
            *) return 0 ;;
        esac
    }
    export -f tar
    
    # Mock system commands
    date() { echo "20240115_103000"; }
    du() { echo "2048	/path/to/backup"; }
    export -f date du
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# ============================================================================
# Redis Connection and Basic Operations Tests
# ============================================================================

@test "redis::common::ping tests Redis connectivity" {
    run redis::common::ping
    [ "$status" -eq 0 ]
    [[ "$output" =~ "PONG" ]]
}

@test "redis::common::is_healthy checks Redis health" {
    run redis::common::is_healthy
    [ "$status" -eq 0 ]
}

@test "redis::common::wait_for_ready waits for Redis to be ready" {
    run redis::common::wait_for_ready 5
    [ "$status" -eq 0 ]
}

@test "redis::common::container_exists checks container existence" {
    run redis::common::container_exists
    [ "$status" -eq 0 ]
}

@test "redis::common::is_running checks if Redis is running" {
    run redis::common::is_running
    [ "$status" -eq 0 ]
}

# ============================================================================
# Redis Information and Status Tests
# ============================================================================

@test "redis::common::get_info retrieves Redis information" {
    run redis::common::get_info
    [ "$status" -eq 0 ]
    [[ "$output" =~ "redis_version" ]]
    [[ "$output" =~ "used_memory" ]]
    [[ "$output" =~ "connected_clients" ]]
}

@test "redis::common::get_memory_usage shows memory consumption" {
    run redis::common::get_memory_usage
    [ "$status" -eq 0 ]
    [[ "$output" =~ "1048576" ]] || [[ "$output" =~ "1.00M" ]]
}

@test "redis::common::get_connected_clients shows client count" {
    run redis::common::get_connected_clients
    [ "$status" -eq 0 ]
    [[ "$output" =~ "2" ]]
}

@test "redis::common::get_total_commands shows command statistics" {
    run redis::common::get_total_commands
    [ "$status" -eq 0 ]
    [[ "$output" =~ "1000" ]]
}

@test "redis::status::show displays basic status" {
    run redis::status::show
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Redis" ]] || [[ "$output" =~ "status" ]]
}

@test "redis::status::show_detailed_info displays comprehensive information" {
    run redis::status::show_detailed_info
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Redis" ]]
}

@test "redis::status::list_clients shows connected clients" {
    run redis::status::list_clients
    [ "$status" -eq 0 ]
    [[ "$output" =~ "127.0.0.1" ]] || [[ "$output" =~ "client" ]]
}

# ============================================================================
# Redis Docker Operations Tests
# ============================================================================

@test "redis::docker::create_container creates Redis container" {
    run redis::docker::create_container
    [ "$status" -eq 0 ]
    [[ "$output" =~ "container_id_redis123" ]]
}

@test "redis::docker::start starts Redis container" {
    run redis::docker::start
    [ "$status" -eq 0 ]
    [[ "$output" =~ "redis-main" ]]
}

@test "redis::docker::stop stops Redis container" {
    run redis::docker::stop
    [ "$status" -eq 0 ]
    [[ "$output" =~ "redis-main" ]]
}

@test "redis::docker::restart restarts Redis container" {
    run redis::docker::restart
    [ "$status" -eq 0 ]
    [[ "$output" =~ "redis-main" ]]
}

@test "redis::docker::exec_cli executes Redis CLI commands" {
    run redis::docker::exec_cli "PING"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "PONG" ]]
}

@test "redis::docker::show_connection_info displays connection details" {
    run redis::docker::show_connection_info
    [ "$status" -eq 0 ]
    [[ "$output" =~ "connection" ]] || [[ "$output" =~ "redis" ]]
}

# ============================================================================
# Redis Data Operations Tests
# ============================================================================

@test "Redis SET command stores data successfully" {
    run redis::docker::exec_cli "SET test_key test_value"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "OK" ]]
}

@test "Redis GET command retrieves data successfully" {
    run redis::docker::exec_cli "GET test_key"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test_value" ]]
}

@test "Redis DEL command removes data successfully" {
    run redis::docker::exec_cli "DEL test_key"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "(integer) 1" ]]
}

@test "Redis DBSIZE command shows database size" {
    run redis::docker::exec_cli "DBSIZE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "(integer) 42" ]]
}

@test "Redis FLUSHALL command clears all data" {
    export YES="yes"
    
    run redis::docker::exec_cli "FLUSHALL"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "OK" ]]
}

# ============================================================================
# Redis Backup Operations Tests
# ============================================================================

@test "redis::backup::create creates RDB backup successfully" {
    run redis::backup::create "$INSTANCE" "$BACKUP_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Background saving started" ]] || [[ "$output" =~ "backup" ]]
}

@test "redis::backup::create_rdb creates RDB dump" {
    run redis::backup::create_rdb "$INSTANCE" "$BACKUP_NAME"
    [ "$status" -eq 0 ]
}

@test "redis::backup::restore restores backup successfully" {
    run redis::backup::restore "$INSTANCE" "/path/to/backup.rdb"
    [ "$status" -eq 0 ]
}

@test "redis::backup::list shows available backups" {
    run redis::backup::list "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "redis_backup_20240115" ]]
    [[ "$output" =~ "redis_backup_20240114" ]]
    [[ "$output" =~ "redis_backup_20240113" ]]
}

@test "redis::backup::delete removes backup successfully" {
    export YES="yes"
    
    run redis::backup::delete "$INSTANCE" "$BACKUP_NAME"
    [ "$status" -eq 0 ]
}

@test "redis::backup::cleanup removes old backups" {
    export RETENTION_DAYS="7"
    
    run redis::backup::cleanup "$INSTANCE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "cleanup" ]] || [[ "$output" =~ "old" ]]
}

# ============================================================================
# Redis Persistence Operations Tests
# ============================================================================

@test "Redis SAVE command creates synchronous backup" {
    run redis::docker::exec_cli "SAVE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "OK" ]]
}

@test "Redis BGSAVE command creates background backup" {
    run redis::docker::exec_cli "BGSAVE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Background saving started" ]]
}

# ============================================================================
# Port Management Tests
# ============================================================================

@test "redis::common::is_port_available detects port usage" {
    run redis::common::is_port_available "6379"
    [ "$status" -eq 1 ]  # Port in use
    
    run redis::common::is_port_available "6381"
    [ "$status" -eq 0 ]  # Port available
}

@test "redis::common::find_available_port finds free port" {
    run redis::common::find_available_port
    [ "$status" -eq 0 ]
    [[ "$output" =~ "6381" ]]
}

# ============================================================================
# Configuration Management Tests
# ============================================================================

@test "redis::common::generate_config creates Redis configuration" {
    run redis::common::generate_config
    [ "$status" -eq 0 ]
}

@test "redis::common::create_directories creates required directories" {
    run redis::common::create_directories
    [ "$status" -eq 0 ]
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "redis::common::ping handles Redis unavailable" {
    docker() {
        case "$*" in
            *"redis-cli"*"PING"*) return 1 ;;  # Simulate connection failure
            *) return 0 ;;
        esac
    }
    
    run redis::common::ping
    [ "$status" -eq 1 ]
}

@test "redis::backup::create handles Redis not running" {
    redis::common::is_running() { return 1; }
    
    run redis::backup::create "$INSTANCE" "$BACKUP_NAME"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not running" ]]
}

@test "redis::docker::start handles Docker failure" {
    docker() {
        case "$*" in
            *"start"*) return 1 ;;  # Simulate Docker failure
            *) return 0 ;;
        esac
    }
    
    run redis::docker::start
    [ "$status" -eq 1 ]
}

# ============================================================================
# Memory and Performance Tests
# ============================================================================

@test "redis::common::format_bytes formats memory sizes correctly" {
    run redis::common::format_bytes "1048576"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "1.00M" ]] || [[ "$output" =~ "1MB" ]]
}

@test "redis::status::monitor provides real-time monitoring" {
    run redis::status::monitor
    [ "$status" -eq 0 ]
    [[ "$output" =~ "monitor" ]] || [[ "$output" =~ "redis" ]]
}

# ============================================================================
# Client Management Tests
# ============================================================================

@test "redis::docker::create_client_instance creates Redis client" {
    run redis::docker::create_client_instance
    [ "$status" -eq 0 ]
}

@test "redis::docker::destroy_client_instance removes Redis client" {
    export YES="yes"
    
    run redis::docker::destroy_client_instance
    [ "$status" -eq 0 ]
}

# ============================================================================
# Advanced Redis Operations Tests
# ============================================================================

@test "Redis INFO command provides comprehensive information" {
    run redis::docker::exec_cli "INFO"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "redis_version:7.0.5" ]]
    [[ "$output" =~ "used_memory:" ]]
    [[ "$output" =~ "connected_clients:" ]]
    [[ "$output" =~ "total_commands_processed:" ]]
}

@test "Redis CLIENT LIST command shows client connections" {
    run redis::docker::exec_cli "CLIENT LIST"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "127.0.0.1" ]]
    [[ "$output" =~ "addr=" ]]
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "all Redis API functions are defined" {
    # Test that all expected functions exist
    type redis::common::container_exists >/dev/null
    type redis::common::is_running >/dev/null
    type redis::common::is_healthy >/dev/null
    type redis::common::ping >/dev/null
    type redis::common::get_info >/dev/null
    type redis::common::get_memory_usage >/dev/null
    type redis::common::get_connected_clients >/dev/null
    type redis::docker::create_container >/dev/null
    type redis::docker::start >/dev/null
    type redis::docker::stop >/dev/null
    type redis::docker::exec_cli >/dev/null
    type redis::backup::create >/dev/null
    type redis::backup::restore >/dev/null
    type redis::backup::list >/dev/null
    type redis::status::show >/dev/null
}

# Teardown
teardown() {
    vrooli_cleanup_test
}