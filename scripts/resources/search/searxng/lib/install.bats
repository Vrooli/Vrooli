#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    vrooli_auto_setup
    
    # Load the functions we are testing (required for bats isolation)
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    source "${SCRIPT_DIR}/install.sh"
    
    # Path to the script under test
    SCRIPT_PATH="$BATS_TEST_DIRNAME/install.sh"
    SEARXNG_DIR="$BATS_TEST_DIRNAME/.."
    
    # Source dependencies
    local resources_dir="$SEARXNG_DIR/../.."
    local helpers_dir="$resources_dir/../helpers"
    
    # Source utilities first
    source "$helpers_dir/utils/log.sh"
    source "$helpers_dir/utils/system.sh"
    source "$helpers_dir/utils/ports.sh"
    source "$helpers_dir/utils/flow.sh"
    source "$resources_dir/port-registry.sh"
    source "$resources_dir/common.sh"
    
    # Source config and messages
    source "$SEARXNG_DIR/config/defaults.sh"
    source "$SEARXNG_DIR/config/messages.sh"
    searxng::export_config
    
    # Source dependencies
    source "$SEARXNG_DIR/lib/common.sh"
    source "$SEARXNG_DIR/lib/docker.sh"
    source "$SEARXNG_DIR/lib/config.sh"
    
    # Source the script under test
    source "$SCRIPT_PATH"
    
    # Mock message function
    searxng::message() {
        local type="$1"
        local msg_var="$2"
        echo "$type: ${!msg_var}"
    }
    
    # Mock prerequisite checks
    command() {
        case "$*" in
            "-v docker")
                if [[ "$MOCK_DOCKER_AVAILABLE" == "yes" ]]; then
                    return 0
                else
                    return 1
                fi
                ;;
            "-v curl")
                if [[ "$MOCK_CURL_AVAILABLE" == "yes" ]]; then
                    return 0
                else
                    return 1
                fi
                ;;
            "-v openssl")
                if [[ "$MOCK_OPENSSL_AVAILABLE" == "yes" ]]; then
                    return 0
                else
                    return 1
                fi
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    docker() {
        case "$1" in
            "info")
                if [[ "$MOCK_DOCKER_DAEMON_RUNNING" == "yes" ]]; then
                    return 0
                else
                    return 1
                fi
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    # Mock SearXNG functions
    searxng::is_installed() { [[ "$MOCK_SEARXNG_INSTALLED" == "yes" ]]; }
    searxng::validate_config() { [[ "$MOCK_CONFIG_VALID" == "yes" ]]; }
    searxng::ensure_data_dir() { [[ "$MOCK_DATA_DIR_SUCCESS" == "yes" ]]; }
    searxng::generate_config() { [[ "$MOCK_CONFIG_GENERATE_SUCCESS" == "yes" ]]; }
    searxng::pull_image() { [[ "$MOCK_PULL_SUCCESS" == "yes" ]]; }
    searxng::create_network() { [[ "$MOCK_NETWORK_SUCCESS" == "yes" ]]; }
    searxng::start_container() { [[ "$MOCK_START_SUCCESS" == "yes" ]]; }
    searxng::wait_for_health() { [[ "$MOCK_HEALTH_SUCCESS" == "yes" ]]; }
    searxng::stop_container() { [[ "$MOCK_STOP_SUCCESS" == "yes" ]]; }
    searxng::cleanup() { [[ "$MOCK_CLEANUP_SUCCESS" == "yes" ]]; }
    searxng::compose_up() { [[ "$MOCK_COMPOSE_SUCCESS" == "yes" ]]; }
    searxng::compose_down() { [[ "$MOCK_COMPOSE_SUCCESS" == "yes" ]]; }
    searxng::remove_container() { [[ "$MOCK_REMOVE_SUCCESS" == "yes" ]]; }
    searxng::restart_container() { [[ "$MOCK_RESTART_SUCCESS" == "yes" ]]; }
    searxng::show_info() { echo "Info displayed"; return 0; }
    flow::prompt_yes_no() { [[ "$MOCK_USER_CONFIRMS" == "yes" ]]; }
    
    # Set default successful mocks
    export MOCK_DOCKER_AVAILABLE="yes"
    export MOCK_CURL_AVAILABLE="yes"
    export MOCK_OPENSSL_AVAILABLE="yes"
    export MOCK_DOCKER_DAEMON_RUNNING="yes"
    export MOCK_SEARXNG_INSTALLED="no"
    export MOCK_CONFIG_VALID="yes"
    export MOCK_DATA_DIR_SUCCESS="yes"
    export MOCK_CONFIG_GENERATE_SUCCESS="yes"
    export MOCK_PULL_SUCCESS="yes"
    export MOCK_NETWORK_SUCCESS="yes"
    export MOCK_START_SUCCESS="yes"
    export MOCK_HEALTH_SUCCESS="yes"
    export MOCK_STOP_SUCCESS="yes"
    export MOCK_CLEANUP_SUCCESS="yes"
    export MOCK_COMPOSE_SUCCESS="yes"
    export MOCK_REMOVE_SUCCESS="yes"
    export MOCK_RESTART_SUCCESS="yes"
    export MOCK_USER_CONFIRMS="yes"
    export FORCE="no"
    export YES="no"
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "sourcing install.sh defines required functions" {
    local required_functions=(
        "searxng::check_prerequisites"
        "searxng::install"
        "searxng::uninstall"
        "searxng::upgrade"
        "searxng::reset"
        "searxng::backup"
        "searxng::restore"
        "searxng::migrate_to_compose"
    )
    
    for func in "${required_functions[@]}"; do
        run bash -c "declare -f $func"
        [ "$status" -eq 0 ]
    done
}

# ============================================================================
# Prerequisites Check Tests
# ============================================================================

@test "searxng::check_prerequisites passes when all dependencies available" {
    run searxng::check_prerequisites
    [ "$status" -eq 0 ]
}

@test "searxng::check_prerequisites fails when docker missing" {
    export MOCK_DOCKER_AVAILABLE="no"
    
    run searxng::check_prerequisites
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Missing required dependencies" ]]
    [[ "$output" =~ "docker" ]]
}

@test "searxng::check_prerequisites fails when docker daemon not running" {
    export MOCK_DOCKER_DAEMON_RUNNING="no"
    
    run searxng::check_prerequisites
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Docker daemon is not running" ]]
}

@test "searxng::check_prerequisites fails when curl missing" {
    export MOCK_CURL_AVAILABLE="no"
    
    run searxng::check_prerequisites
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Missing required dependencies" ]]
    [[ "$output" =~ "curl" ]]
}

@test "searxng::check_prerequisites warns when openssl missing" {
    export MOCK_OPENSSL_AVAILABLE="no"
    
    run searxng::check_prerequisites
    [ "$status" -eq 0 ]
    [[ "$output" =~ "WARNING:" ]]
    [[ "$output" =~ "OpenSSL not found" ]]
}

# ============================================================================
# Installation Tests
# ============================================================================

@test "searxng::install completes full installation successfully" {
    run searxng::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "info: MSG_SEARXNG_SETUP_START" ]]
    [[ "$output" =~ "success: MSG_SEARXNG_INSTALL_SUCCESS" ]]
}

@test "searxng::install skips when already installed without force" {
    export MOCK_SEARXNG_INSTALLED="yes"
    export FORCE="no"
    
    run searxng::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "WARNING:" ]]
    [[ "$output" =~ "already installed" ]]
}

@test "searxng::install proceeds when already installed with force" {
    export MOCK_SEARXNG_INSTALLED="yes"
    export FORCE="yes"
    
    run searxng::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "success: MSG_SEARXNG_INSTALL_SUCCESS" ]]
}

@test "searxng::install fails when prerequisites not met" {
    export MOCK_DOCKER_AVAILABLE="no"
    
    run searxng::install
    [ "$status" -eq 1 ]
}

@test "searxng::install fails when config validation fails" {
    export MOCK_CONFIG_VALID="no"
    
    run searxng::install
    [ "$status" -eq 1 ]
    [[ "$output" =~ "error: MSG_SEARXNG_CONFIG_FAILED" ]]
}

@test "searxng::install fails when data directory creation fails" {
    export MOCK_DATA_DIR_SUCCESS="no"
    
    run searxng::install
    [ "$status" -eq 1 ]
}

@test "searxng::install fails when config generation fails" {
    export MOCK_CONFIG_GENERATE_SUCCESS="no"
    
    run searxng::install
    [ "$status" -eq 1 ]
    [[ "$output" =~ "error: MSG_SEARXNG_CONFIG_FAILED" ]]
}

@test "searxng::install fails when image pull fails" {
    export MOCK_PULL_SUCCESS="no"
    
    run searxng::install
    [ "$status" -eq 1 ]
}

@test "searxng::install fails when network creation fails" {
    export MOCK_NETWORK_SUCCESS="no"
    
    run searxng::install
    [ "$status" -eq 1 ]
    [[ "$output" =~ "error: MSG_SEARXNG_NETWORK_FAILED" ]]
}

@test "searxng::install fails when container start fails" {
    export MOCK_START_SUCCESS="no"
    
    run searxng::install
    [ "$status" -eq 1 ]
}

@test "searxng::install warns when health check fails but continues" {
    export MOCK_HEALTH_SUCCESS="no"
    
    run searxng::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "WARNING:" ]]
    [[ "$output" =~ "may not be fully ready yet" ]]
}

# ============================================================================
# Uninstallation Tests
# ============================================================================

@test "searxng::uninstall completes when SearXNG installed" {
    export MOCK_SEARXNG_INSTALLED="yes"
    export YES="yes"  # Auto-confirm
    
    run searxng::uninstall
    [ "$status" -eq 0 ]
    [[ "$output" =~ "success: MSG_SEARXNG_UNINSTALL_SUCCESS" ]]
}

@test "searxng::uninstall succeeds when not installed" {
    export MOCK_SEARXNG_INSTALLED="no"
    
    run searxng::uninstall
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SearXNG is not installed" ]]
}

@test "searxng::uninstall prompts for confirmation when not forced" {
    export MOCK_SEARXNG_INSTALLED="yes"
    export FORCE="no"
    export YES="no"
    export MOCK_USER_CONFIRMS="no"
    
    run searxng::uninstall
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Uninstallation cancelled" ]]
}

@test "searxng::uninstall removes data in force mode" {
    export MOCK_SEARXNG_INSTALLED="yes"
    export FORCE="yes"
    
    run searxng::uninstall
    [ "$status" -eq 0 ]
    [[ "$output" =~ "success: MSG_SEARXNG_UNINSTALL_SUCCESS" ]]
}

# ============================================================================
# Upgrade Tests
# ============================================================================

@test "searxng::upgrade works when SearXNG installed" {
    export MOCK_SEARXNG_INSTALLED="yes"
    export MOCK_RESTART_SUCCESS="yes"
    
    run searxng::upgrade
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Upgrading SearXNG" ]]
    [[ "$output" =~ "SUCCESS:" ]]
    [[ "$output" =~ "upgraded successfully" ]]
}

@test "searxng::upgrade fails when not installed" {
    export MOCK_SEARXNG_INSTALLED="no"
    
    run searxng::upgrade
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "not installed" ]]
}

@test "searxng::upgrade fails when image pull fails" {
    export MOCK_SEARXNG_INSTALLED="yes"
    export MOCK_PULL_SUCCESS="no"
    
    run searxng::upgrade
    [ "$status" -eq 1 ]
}

@test "searxng::upgrade fails when restart fails" {
    export MOCK_SEARXNG_INSTALLED="yes"
    export MOCK_RESTART_SUCCESS="no"
    
    run searxng::upgrade
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "Failed to upgrade" ]]
}

# ============================================================================
# Reset Tests
# ============================================================================

@test "searxng::reset regenerates config and restarts" {
    # Mock is_running to return true initially
    searxng::is_running() { return 0; }
    
    run searxng::reset
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Resetting SearXNG installation" ]]
    [[ "$output" =~ "SUCCESS:" ]]
    [[ "$output" =~ "reset successfully" ]]
}

@test "searxng::reset fails when config generation fails" {
    export MOCK_CONFIG_GENERATE_SUCCESS="no"
    
    run searxng::reset
    [ "$status" -eq 1 ]
    [[ "$output" =~ "error: MSG_SEARXNG_CONFIG_FAILED" ]]
}

@test "searxng::reset fails when container start fails" {
    export MOCK_START_SUCCESS="no"
    
    run searxng::reset
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "Failed to reset" ]]
}

# ============================================================================
# Backup and Restore Tests
# ============================================================================

@test "searxng::backup creates backup successfully" {
    # Create mock data directory
    export SEARXNG_DATA_DIR="/tmp/searxng-test-data"
    mkdir -p "$SEARXNG_DATA_DIR"
    echo "test config" > "$SEARXNG_DATA_DIR/settings.yml"
    
    run searxng::backup
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Creating SearXNG backup" ]]
    [[ "$output" =~ "SUCCESS:" ]]
    [[ "$output" =~ "backup created successfully" ]]
    
    # Cleanup
    rm -rf "$SEARXNG_DATA_DIR"
    rm -rf ~/.searxng-backup-*
}

@test "searxng::backup fails when data directory missing" {
    export SEARXNG_DATA_DIR="/nonexistent/directory"
    
    run searxng::backup
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "data directory not found" ]]
}

@test "searxng::restore restores from backup successfully" {
    # Create mock backup directory
    local backup_dir="/tmp/searxng-backup-test"
    mkdir -p "$backup_dir"
    echo "backup config" > "$backup_dir/settings.yml"
    
    # Create mock data directory
    export SEARXNG_DATA_DIR="/tmp/searxng-test-data"
    mkdir -p "$SEARXNG_DATA_DIR"
    
    # Mock is_running to return true initially
    searxng::is_running() { return 0; }
    
    run searxng::restore "$backup_dir"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Restoring SearXNG from backup" ]]
    [[ "$output" =~ "SUCCESS:" ]]
    [[ "$output" =~ "restore completed successfully" ]]
    
    # Cleanup
    rm -rf "$backup_dir"
    rm -rf "$SEARXNG_DATA_DIR"
}

@test "searxng::restore fails when backup directory missing" {
    run searxng::restore "/nonexistent/backup"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "Backup directory not found" ]]
}

@test "searxng::restore fails when no backup directory provided" {
    run searxng::restore ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "Backup directory path is required" ]]
}

# ============================================================================
# Migration Tests
# ============================================================================

@test "searxng::migrate_to_compose migrates successfully" {
    # Mock is_running to return true initially
    searxng::is_running() { return 0; }
    
    run searxng::migrate_to_compose
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Migrating SearXNG to Docker Compose" ]]
    [[ "$output" =~ "SUCCESS:" ]]
    [[ "$output" =~ "migrated to Docker Compose successfully" ]]
}

@test "searxng::migrate_to_compose fails when compose up fails" {
    export MOCK_COMPOSE_SUCCESS="no"
    
    run searxng::migrate_to_compose
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "Failed to migrate" ]]
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "installation flow handles all error scenarios gracefully" {
    # Test each failure point
    local failure_scenarios=(
        "MOCK_DOCKER_AVAILABLE=no"
        "MOCK_CONFIG_VALID=no"
        "MOCK_DATA_DIR_SUCCESS=no"
        "MOCK_CONFIG_GENERATE_SUCCESS=no"
        "MOCK_PULL_SUCCESS=no"
        "MOCK_NETWORK_SUCCESS=no"
        "MOCK_START_SUCCESS=no"
    )
    
    for scenario in "${failure_scenarios[@]}"; do
        # Reset to defaults for each test
        export MOCK_DOCKER_AVAILABLE="yes"
        export MOCK_CURL_AVAILABLE="yes"
        export MOCK_OPENSSL_AVAILABLE="yes"
        export MOCK_DOCKER_DAEMON_RUNNING="yes"
        export MOCK_SEARXNG_INSTALLED="no"
        export MOCK_CONFIG_VALID="yes"
        export MOCK_DATA_DIR_SUCCESS="yes"
        export MOCK_CONFIG_GENERATE_SUCCESS="yes"
        export MOCK_PULL_SUCCESS="yes"
        export MOCK_NETWORK_SUCCESS="yes"
        export MOCK_START_SUCCESS="yes"
        export MOCK_HEALTH_SUCCESS="yes"
        export MOCK_STOP_SUCCESS="yes"
        export MOCK_CLEANUP_SUCCESS="yes"
        export MOCK_COMPOSE_SUCCESS="yes"
        export MOCK_REMOVE_SUCCESS="yes"
        export MOCK_RESTART_SUCCESS="yes"
        export MOCK_USER_CONFIRMS="yes"
        export FORCE="no"
        export YES="no"
        
        # Apply the specific failure scenario
        eval "export $scenario"
        
        run searxng::install
        [ "$status" -eq 1 ]
    done
}

@test "functions properly handle force and yes parameters" {
    # Test force installation
    export MOCK_SEARXNG_INSTALLED="yes"
    export FORCE="yes"
    
    run searxng::install
    [ "$status" -eq 0 ]
    
    # Test auto-confirmation
    export FORCE="no"
    export YES="yes"
    
    run searxng::uninstall
    [ "$status" -eq 0 ]
}
