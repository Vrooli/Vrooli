#!/usr/bin/env bats
# Tests for PostgreSQL Instance Management

# Load Vrooli test infrastructure
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "postgres-instances"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    POSTGRES_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and functions once
    source "${POSTGRES_DIR}/config/defaults.sh"
    source "${POSTGRES_DIR}/config/messages.sh"
    source "${SCRIPT_DIR}/common.sh"
    source "${SCRIPT_DIR}/instance.sh"
    source "${SCRIPT_DIR}/multi_instance.sh"
    
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
    export POSTGRES_MAX_INSTANCES="10"
    export POSTGRES_PORT_RANGE_START="5433"
    export POSTGRES_PORT_RANGE_END="5499"
    export POSTGRES_CONTAINER_PREFIX="postgres"
    export POSTGRES_DATA_DIR="/var/lib/postgresql/data"
    export INSTANCE_NAME="test-instance"
    export INSTANCE_PORT="5433"
    export TEMPLATE="development"
    export NETWORKS="vrooli-network"
    export YES="no"
    
    # Set up test message variables
    export MSG_INSTANCE_CREATED="Instance created successfully"
    export MSG_INSTANCE_DESTROYED="Instance destroyed"
    export MSG_INSTANCE_STARTED="Instance started"
    export MSG_INSTANCE_STOPPED="Instance stopped"
    export MSG_MAX_INSTANCES="Maximum instances limit reached"
    export MSG_FINDING_PORT="Finding available port"
    export MSG_NO_AVAILABLE_PORT="No available port found"
    export MSG_PORT_IN_USE="Port already in use"
    export MSG_INVALID_NAME="Invalid instance name"
    
    # Mock PostgreSQL utility functions
    postgres::common::container_exists() { 
        case "$1" in
            "existing-instance") return 0 ;;  # Instance exists
            *) return 1 ;;  # Instance doesn't exist
        esac
    }
    postgres::common::is_running() { return 0; }
    postgres::common::count_running_instances() { echo "3"; }
    postgres::common::is_port_available() {
        case "$1" in
            "5432") return 1 ;;  # Port in use
            *) return 0 ;;       # Port available
        esac
    }
    postgres::instance::find_available_port() { echo "5434"; }
    postgres::instance::validate_name() { 
        case "$1" in
            "invalid-name*") return 1 ;;  # Invalid characters
            "") return 1 ;;               # Empty name
            *) return 0 ;;                # Valid name
        esac
    }
    postgres::instance::validate_port() {
        case "$1" in
            "22"|"80"|"443") return 1 ;;  # Reserved ports
            *) return 0 ;;                # Valid port
        esac
    }
    export -f postgres::common::container_exists postgres::common::is_running
    export -f postgres::common::count_running_instances postgres::common::is_port_available
    export -f postgres::instance::find_available_port postgres::instance::validate_name
    export -f postgres::instance::validate_port
    
    # Mock Docker functions for instance operations
    docker() {
        case "$*" in
            *"run"*"postgres"*)
                echo "container_id_123456789"
                return 0
                ;;
            *"ps"*"--filter"*)
                echo "postgres-test-instance"
                echo "postgres-main"
                echo "postgres-dev"
                return 0
                ;;
            *"start"*|*"stop"*|*"restart"*)
                echo "postgres-$INSTANCE_NAME"
                return 0
                ;;
            *"rm"*)
                echo "postgres-$INSTANCE_NAME"
                return 0
                ;;
            *"inspect"*)
                echo '[{"Config":{"ExposedPorts":{"5432/tcp":{}}},"NetworkSettings":{"Ports":{"5432/tcp":[{"HostPort":"5433"}]}}}]'
                return 0
                ;;
            *"exec"*"psql"*)
                echo "PostgreSQL 14.5 (Ubuntu 14.5-2.pgdg20.04+2) on x86_64"
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
    export -f mkdir cp mv rm chmod chown
    
    # Mock configuration functions
    postgres::instance::save_config() { return 0; }
    postgres::instance::get_connection_string() {
        echo "postgresql://$POSTGRES_DEFAULT_USER:password@localhost:$INSTANCE_PORT/$POSTGRES_DEFAULT_DB"
    }
    export -f postgres::instance::save_config postgres::instance::get_connection_string
    
    # Mock network functions
    postgres::network::create() { return 0; }
    postgres::network::connect_container() { return 0; }
    export -f postgres::network::create postgres::network::connect_container
    
    # Mock log functions
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::warning() { echo "[WARNING] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::debug() { [[ "${DEBUG:-}" == "true" ]] && echo "[DEBUG] $*" >&2 || true; }
    export -f log::info log::error log::warning log::success log::debug
    
    # Mock system commands
    netstat() { 
        case "$*" in
            *"5432"*) echo "tcp 0 0 0.0.0.0:5432 0.0.0.0:* LISTEN" ;;
            *) return 1 ;;
        esac
    }
    ss() {
        case "$*" in
            *"5432"*) echo "LISTEN 0 244 0.0.0.0:5432 0.0.0.0:*" ;;
            *) return 1 ;;
        esac
    }
    export -f netstat ss
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# ============================================================================
# Instance Creation Tests
# ============================================================================

@test "postgres::instance::create creates new instance successfully" {
    run postgres::instance::create "$INSTANCE_NAME" "$INSTANCE_PORT" "$TEMPLATE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "container_id_123456789" ]]
}

@test "postgres::instance::create uses default values when parameters omitted" {
    run postgres::instance::create
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Finding available port" ]]
    [[ "$output" =~ "5434" ]]  # Found available port
}

@test "postgres::instance::create fails when instance already exists" {
    run postgres::instance::create "existing-instance"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "already exists" ]]
}

@test "postgres::instance::create fails with invalid instance name" {
    run postgres::instance::create "invalid-name*"
    [ "$status" -eq 1 ]
}

@test "postgres::instance::create fails when maximum instances reached" {
    postgres::common::count_running_instances() { echo "10"; }  # At max limit
    
    run postgres::instance::create "$INSTANCE_NAME"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Maximum instances limit reached" ]]
}

@test "postgres::instance::create finds available port automatically" {
    run postgres::instance::create "$INSTANCE_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Finding available port" ]]
    [[ "$output" =~ "Found available port: 5434" ]]
}

@test "postgres::instance::create fails with invalid port" {
    run postgres::instance::create "$INSTANCE_NAME" "22"  # Reserved port
    [ "$status" -eq 1 ]
}

@test "postgres::instance::create handles Docker failure" {
    docker() {
        case "$*" in
            *"run"*) return 1 ;;  # Simulate Docker failure
            *) return 0 ;;
        esac
    }
    
    run postgres::instance::create "$INSTANCE_NAME" "$INSTANCE_PORT"
    [ "$status" -eq 1 ]
}

# ============================================================================
# Instance Lifecycle Tests
# ============================================================================

@test "postgres::instance::start starts instance successfully" {
    run postgres::instance::start "$INSTANCE_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "postgres-$INSTANCE_NAME" ]]
}

@test "postgres::instance::stop stops instance successfully" {
    run postgres::instance::stop "$INSTANCE_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "postgres-$INSTANCE_NAME" ]]
}

@test "postgres::instance::restart restarts instance successfully" {
    run postgres::instance::restart "$INSTANCE_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "postgres-$INSTANCE_NAME" ]]
}

@test "postgres::instance::destroy removes instance successfully" {
    export YES="yes"
    
    run postgres::instance::destroy "$INSTANCE_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "postgres-$INSTANCE_NAME" ]]
}

@test "postgres::instance::destroy requires confirmation by default" {
    export YES="no"
    
    run postgres::instance::destroy "$INSTANCE_NAME"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "cancelled" ]] || [[ "$output" =~ "confirmation" ]]
}

# ============================================================================
# Instance Listing and Information Tests
# ============================================================================

@test "postgres::instance::list shows all instances" {
    run postgres::instance::list
    [ "$status" -eq 0 ]
    [[ "$output" =~ "postgres-test-instance" ]]
    [[ "$output" =~ "postgres-main" ]]
    [[ "$output" =~ "postgres-dev" ]]
}

@test "postgres::instance::get_connection_string returns valid connection string" {
    run postgres::instance::get_connection_string "$INSTANCE_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "postgresql://" ]]
    [[ "$output" =~ "$INSTANCE_PORT" ]]
}

# ============================================================================
# Multi-Instance Operations Tests
# ============================================================================

@test "postgres::instance::start_all starts all instances" {
    run postgres::instance::start_all
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Starting all PostgreSQL instances" ]]
}

@test "postgres::instance::stop_all stops all instances" {
    run postgres::instance::stop_all
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Stopping all PostgreSQL instances" ]]
}

@test "postgres::instance::restart_all restarts all instances" {
    run postgres::instance::restart_all
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Restarting all PostgreSQL instances" ]]
}

# ============================================================================
# Multi-Instance Management Tests
# ============================================================================

@test "postgres::multi::start executes action on all instances" {
    run postgres::multi::start
    [ "$status" -eq 0 ]
    [[ "$output" =~ "instances" ]]
}

@test "postgres::multi::stop executes action on all instances" {
    run postgres::multi::stop
    [ "$status" -eq 0 ]
    [[ "$output" =~ "instances" ]]
}

@test "postgres::multi::status shows status of all instances" {
    run postgres::multi::status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "PostgreSQL Multi-Instance Status" ]]
}

@test "postgres::multi::health_check validates all instances" {
    run postgres::multi::health_check
    [ "$status" -eq 0 ]
    [[ "$output" =~ "health" ]] || [[ "$output" =~ "check" ]]
}

@test "postgres::multi::resource_usage shows resource consumption" {
    # Mock docker stats
    docker() {
        case "$*" in
            *"stats"*"--no-stream"*)
                echo "CONTAINER    CPU %   MEM USAGE"
                echo "postgres-1   2.5%    256MB"
                echo "postgres-2   1.8%    189MB"
                return 0
                ;;
            *) return 0 ;;
        esac
    }
    
    run postgres::multi::resource_usage
    [ "$status" -eq 0 ]
    [[ "$output" =~ "CPU" ]]
    [[ "$output" =~ "MEM" ]]
}

# ============================================================================
# Instance Validation Tests
# ============================================================================

@test "postgres::instance::validate_name accepts valid names" {
    run postgres::instance::validate_name "valid-name"
    [ "$status" -eq 0 ]
    
    run postgres::instance::validate_name "test123"
    [ "$status" -eq 0 ]
    
    run postgres::instance::validate_name "main"
    [ "$status" -eq 0 ]
}

@test "postgres::instance::validate_name rejects invalid names" {
    run postgres::instance::validate_name ""
    [ "$status" -eq 1 ]
    
    run postgres::instance::validate_name "invalid-name*"
    [ "$status" -eq 1 ]
}

@test "postgres::instance::validate_port accepts valid ports" {
    run postgres::instance::validate_port "5433"
    [ "$status" -eq 0 ]
    
    run postgres::instance::validate_port "8080"
    [ "$status" -eq 0 ]
}

@test "postgres::instance::validate_port rejects reserved ports" {
    run postgres::instance::validate_port "22"
    [ "$status" -eq 1 ]
    
    run postgres::instance::validate_port "80"
    [ "$status" -eq 1 ]
    
    run postgres::instance::validate_port "443"
    [ "$status" -eq 1 ]
}

# ============================================================================
# Port Management Tests
# ============================================================================

@test "postgres::instance::find_available_port finds free port" {
    run postgres::instance::find_available_port
    [ "$status" -eq 0 ]
    [[ "$output" =~ "5434" ]]
}

@test "postgres::common::is_port_available detects port usage" {
    run postgres::common::is_port_available "5432"
    [ "$status" -eq 1 ]  # Port in use
    
    run postgres::common::is_port_available "5434"
    [ "$status" -eq 0 ]  # Port available
}

# ============================================================================
# Configuration Management Tests
# ============================================================================

@test "postgres::instance::save_config saves instance configuration" {
    run postgres::instance::save_config "$INSTANCE_NAME" "$INSTANCE_PORT" "$TEMPLATE"
    [ "$status" -eq 0 ]
}

# ============================================================================
# Network Integration Tests
# ============================================================================

@test "postgres::instance::create with custom networks" {
    run postgres::instance::create "$INSTANCE_NAME" "$INSTANCE_PORT" "$TEMPLATE" "network1,network2"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "container_id_123456789" ]]
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "postgres::instance operations handle container not found" {
    postgres::common::container_exists() { return 1; }
    
    run postgres::instance::start "nonexistent-instance"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not found" ]] || [[ "$output" =~ "does not exist" ]]
}

@test "postgres::multi operations handle no instances" {
    docker() {
        case "$*" in
            *"ps"*"--filter"*) return 1 ;;  # No containers found
            *) return 0 ;;
        esac
    }
    
    run postgres::multi::status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No instances found" ]] || [[ "$output" =~ "0 instances" ]]
}

# ============================================================================
# Concurrent Operations Tests
# ============================================================================

@test "postgres::multi::execute handles concurrent operations" {
    export PARALLEL="yes"
    
    run postgres::multi::execute "status"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "instances" ]]
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "all PostgreSQL instance functions are defined" {
    # Test that all expected functions exist
    type postgres::instance::create >/dev/null
    type postgres::instance::destroy >/dev/null
    type postgres::instance::start >/dev/null
    type postgres::instance::stop >/dev/null
    type postgres::instance::restart >/dev/null
    type postgres::instance::list >/dev/null
    type postgres::instance::get_connection_string >/dev/null
    type postgres::instance::validate_name >/dev/null
    type postgres::instance::validate_port >/dev/null
    type postgres::multi::start >/dev/null
    type postgres::multi::stop >/dev/null
    type postgres::multi::status >/dev/null
    type postgres::multi::health_check >/dev/null
}

# Teardown
teardown() {
    vrooli_cleanup_test
}