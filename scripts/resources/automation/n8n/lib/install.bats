#!/usr/bin/env bats
# Tests for n8n install.sh functions

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "n8n"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    export SETUP_FILE_N8N_DIR="$(dirname "${BATS_TEST_DIRNAME}")"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    N8N_DIR="${SETUP_FILE_N8N_DIR}"
    
    # Set test environment BEFORE sourcing config files to avoid readonly conflicts
    export N8N_CUSTOM_PORT="5678"
    export N8N_DATA_DIR="/tmp/n8n-test"
    export DATABASE_TYPE="postgres"
    export N8N_ENCRYPTION_KEY="test-encryption-key"
    export YES="no"
    
    # Create test directory
    mkdir -p "$N8N_DATA_DIR"
    
    # Mock resources functions that are called during config loading
    resources::get_default_port() {
        case "$1" in
            "n8n") echo "5678" ;;
            *) echo "8080" ;;
        esac
    }
    
    # Now source the config and library files
    source "${N8N_DIR}/config/defaults.sh"
    source "${N8N_DIR}/config/messages.sh"
    source "${N8N_DIR}/lib/install.sh"
    
    # Export config and messages
    n8n::export_config
    n8n::export_messages
    
    # Mock rollback functions
    resources::add_rollback_action() {
        echo "ROLLBACK: $1"
        return 0
    }
    
    resources::start_rollback_context() {
        echo "START_ROLLBACK: $1"
        return 0
    }
    
    resources::handle_error() {
        echo "HANDLE_ERROR: $1 - $2 - $3"
        return 1
    }
    
    resources::update_config() {
        echo "UPDATE_CONFIG: $1 $2 $3"
        return 0
    }
    
    resources::remove_config() {
        echo "REMOVE_CONFIG: $1 $2"
        return 0
    }
    
    # Mock system functions
    system::is_port_in_use() {
        return 1  # Port available
    }
    
    # Mock openssl command
    openssl() {
        case "$1" in
            "rand")
                echo "generated-random-key-12345"
                ;;
            *) return 0 ;;
        esac
    }
    
    # Mock n8n functions
    docker::check_daemon() { return 0; }
    n8n::container_exists() { return 1; }  # No existing container
    n8n::is_running() { return 0; }
    n8n::check_port_available() { return 0; }
    n8n::start_postgres() { return 0; }
    n8n::init_database() { return 0; }
    n8n::status() { return 0; }
}

# Cleanup after each test
teardown() {
    trash::safe_remove "$N8N_DATA_DIR" --test-cleanup
    vrooli_cleanup_test
}

# Test successful installation
@test "n8n::install installs successfully with clean environment" {
    result=$(n8n::install "no")
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Installing" ]]
    [[ "$result" =~ "DOCKER_PULL:" ]]
    [[ "$result" =~ "DOCKER_RUN:" ]]
    [[ "$result" =~ "UPDATE_CONFIG:" ]]
    [[ "$result" =~ "SUCCESS:" ]]
}

# Test installation with Docker unavailable
@test "n8n::install fails with Docker unavailable" {
    # Override Docker check to fail
    docker::check_daemon() {
        log::error "Docker not available"
        return 1
    }
    
    run n8n::install "no"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Docker not available" ]]
}

# Test installation with existing container (no force)
@test "n8n::install skips when container exists without force" {
    # Override container check to return existing container
    n8n::container_exists() { return 0; }
    
    result=$(n8n::install "no")
    
    [[ "$result" =~ "already installed" ]]
    [[ ! "$result" =~ "DOCKER_RUN:" ]]
}

# Test installation with existing container (force)
@test "n8n::install reinstalls when container exists with force" {
    # Override container check to return existing container
    n8n::container_exists() { return 0; }
    
    # Mock uninstall function
    n8n::uninstall() {
        echo "UNINSTALL_CALLED"
        return 0
    }
    
    result=$(n8n::install "yes")
    
    [[ "$result" =~ "UNINSTALL_CALLED" ]]
    [[ "$result" =~ "DOCKER_PULL:" ]]
    [[ "$result" =~ "DOCKER_RUN:" ]]
}

# Test installation with port conflict
@test "n8n::install fails with port conflict" {
    # Override port check to return conflict
    n8n::check_port_available() {
        log::error "Port $N8N_PORT is in use"
        return 1
    }
    
    run n8n::install "no"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Port $N8N_PORT is in use" ]]
}

# Test installation with database setup
@test "n8n::install sets up database when using PostgreSQL" {
    export DATABASE_TYPE="postgres"
    
    result=$(n8n::install "no")
    
    [[ "$result" =~ "Setting up database" ]]
    [[ "$result" =~ "postgres" ]]
}

# Test installation with SQLite
@test "n8n::install configures SQLite correctly" {
    export DATABASE_TYPE="sqlite"
    
    result=$(n8n::install "no")
    
    [[ "$result" =~ "SQLite" ]] || [[ "$result" =~ "sqlite" ]]
}

# Test uninstallation
@test "n8n::uninstall removes containers and data successfully" {
    # Override container check to return existing container
    n8n::container_exists() { return 0; }
    
    result=$(n8n::uninstall)
    
    [[ "$result" =~ "INFO: Removing" ]]
    [[ "$result" =~ "DOCKER_RM:" ]]
    [[ "$result" =~ "REMOVE_CONFIG:" ]]
}

# Test uninstallation with no existing containers
@test "n8n::uninstall handles missing containers gracefully" {
    # Override container check to return no container
    n8n::container_exists() { return 1; }
    
    result=$(n8n::uninstall)
    
    [[ "$result" =~ "not found" ]] || [[ "$result" =~ "SUCCESS:" ]]
}

# Test service start
@test "n8n::start starts services successfully" {
    # Override container check to return existing but stopped container
    n8n::container_exists() { return 0; }
    n8n::is_running() { return 1; }  # Not running
    
    result=$(n8n::start)
    
    [[ "$result" =~ "INFO: Starting" ]]
    [[ "$result" =~ "DOCKER_START:" ]]
}

# Test service start with missing container
@test "n8n::start fails with missing container" {
    # Override container check to return no container
    n8n::container_exists() { return 1; }
    
    run n8n::start
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not found" ]]
}

# Test service start with already running container
@test "n8n::start handles already running container" {
    # Override checks to return running container
    n8n::container_exists() { return 0; }
    n8n::is_running() { return 0; }  # Already running
    
    result=$(n8n::start)
    
    [[ "$result" =~ "already running" ]]
}

# Test service stop
@test "n8n::stop stops services successfully" {
    # Override checks to return running container
    n8n::container_exists() { return 0; }
    n8n::is_running() { return 0; }  # Running
    
    result=$(n8n::stop)
    
    [[ "$result" =~ "INFO: Stopping" ]]
    [[ "$result" =~ "DOCKER_STOP:" ]]
}

# Test service stop with stopped container
@test "n8n::stop handles already stopped container" {
    # Override checks to return stopped container
    n8n::container_exists() { return 0; }
    n8n::is_running() { return 1; }  # Not running
    
    result=$(n8n::stop)
    
    [[ "$result" =~ "not running" ]] || [[ "$result" =~ "already stopped" ]]
}

# Test service restart
@test "n8n::restart performs stop and start sequence" {
    # Mock stop and start functions
    n8n::stop() {
        echo "STOP_CALLED"
        return 0
    }
    
    n8n::start() {
        echo "START_CALLED"
        return 0
    }
    
    result=$(n8n::restart)
    
    [[ "$result" =~ "STOP_CALLED" ]]
    [[ "$result" =~ "START_CALLED" ]]
}

# Test encryption key generation
@test "n8n::generate_encryption_key generates secure key" {
    result=$(n8n::generate_encryption_key)
    
    [[ "$result" =~ "generated-random-key" ]]
    [ ${#result} -gt 10 ]  # Should be reasonably long
}

# Test data directory setup
@test "n8n::setup_data_directory creates necessary directories" {
    result=$(n8n::setup_data_directory)
    
    [[ "$result" =~ "Setting up data directory" ]]
    [ -d "$N8N_DATA_DIR" ]
}

# Test environment validation
@test "n8n::validate_environment checks system requirements" {
    result=$(n8n::validate_environment)
    
    [[ "$result" =~ "Validating environment" ]]
    [[ "$result" =~ "Docker" ]]
}

# Test environment validation with missing requirements
@test "n8n::validate_environment detects missing requirements" {
    # Override system check to fail
    system::is_command() {
        case "$1" in
            "docker") return 1 ;;  # Docker not available
            *) return 0 ;;
        esac
    }
    
    run n8n::validate_environment
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
}

# Test container startup verification
@test "n8n::wait_for_startup verifies container startup" {
    # Mock status check
    n8n::status() {
        echo "SERVICE_READY"
        return 0
    }
    
    result=$(n8n::wait_for_startup)
    
    [[ "$result" =~ "Waiting for startup" ]]
    [[ "$result" =~ "SERVICE_READY" ]]
}

# Test container startup verification timeout
@test "n8n::wait_for_startup handles startup timeout" {
    # Mock status check to always fail
    n8n::status() {
        return 1
    }
    
    # Mock sleep to avoid actual waiting
    sleep() { return 0; }
    
    # Set short timeout for testing
    export N8N_STARTUP_MAX_WAIT="1"
    
    run n8n::wait_for_startup
    [ "$status" -eq 1 ]
    [[ "$output" =~ "timeout" ]]
}

# Test configuration backup
@test "n8n::backup_configuration creates configuration backup" {
    echo "test config" > "$N8N_DATA_DIR/config.json"
    
    result=$(n8n::backup_configuration)
    
    [[ "$result" =~ "Backing up configuration" ]]
    [ -f "$N8N_DATA_DIR/config.json.bak" ] || [[ "$result" =~ "backup" ]]
}

# Test configuration restore
@test "n8n::restore_configuration restores configuration from backup" {
    echo "backup config" > "$N8N_DATA_DIR/config.json.bak"
    
    result=$(n8n::restore_configuration)
    
    [[ "$result" =~ "Restoring configuration" ]]
}

# Test container health verification
@test "n8n::verify_installation verifies successful installation" {
    result=$(n8n::verify_installation)
    
    [[ "$result" =~ "Verifying installation" ]]
    [[ "$result" =~ "SUCCESS:" ]] || [[ "$result" =~ "verified" ]]
}

# Test installation cleanup on failure
@test "n8n::cleanup_failed_installation cleans up after failure" {
    result=$(n8n::cleanup_failed_installation)
    
    [[ "$result" =~ "Cleaning up failed installation" ]]
    [[ "$result" =~ "DOCKER_RM:" ]] || [[ "$result" =~ "cleanup" ]]
}