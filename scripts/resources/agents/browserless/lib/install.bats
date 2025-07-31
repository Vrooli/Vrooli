#!/usr/bin/env bats
# Tests for Browserless install.sh functions

# Expensive setup operations run once per file  
setup_file() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Load dependencies once per file
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    BROWSERLESS_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and messages once
    source "${BROWSERLESS_DIR}/config/defaults.sh"
    source "${BROWSERLESS_DIR}/config/messages.sh"
    browserless::export_config
    browserless::export_messages
    
    # Load install functions once
    source "${SCRIPT_DIR}/install.sh"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    setup_standard_mocks
    
    # Load the functions we're testing (required for bats isolation)
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    source "${SCRIPT_DIR}/install.sh"
    
    # Set test environment (lightweight per-test)
    export BROWSERLESS_CUSTOM_PORT="9999"
    export BROWSERLESS_CONTAINER_NAME="browserless-test"
    export BROWSERLESS_BASE_URL="http://localhost:9999"
    export BROWSERLESS_DATA_DIR="/tmp/browserless-test"
    export BROWSERLESS_MAX_BROWSERS="5"
    export BROWSERLESS_HEADLESS="yes"
    export BROWSERLESS_TIMEOUT="30000"
    export BROWSERLESS_STARTUP_MAX_WAIT="30"
    export BROWSERLESS_INITIALIZATION_WAIT="5"
    export YES="no"
    
    # Mock rollback functions (lightweight)
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
    
    # Mock flow functions
    flow::is_yes() {
        [[ "$1" == "yes" ]]
    }
    
    # Mock common functions - prevent slow operations
    browserless::check_existing_installation() { return 0; }
    browserless::validate_prerequisites() { return 0; }
    browserless::wait_for_ready() { return 0; }
    browserless::wait_for_healthy() { return 0; }
    browserless::backup_data() { echo "BACKUP: $1"; }
    
    # Mock docker functions - prevent real Docker calls and timeouts
    browserless::create_network() { echo "CREATE_NETWORK"; }
    browserless::docker_run() { return 0; }
    browserless::docker_remove() { echo "DOCKER_REMOVE: $1"; }
    
    # Export all mock functions
    export -f resources::add_rollback_action resources::start_rollback_context
    export -f resources::handle_error resources::update_config resources::remove_config
    export -f flow::is_yes browserless::check_existing_installation
    export -f browserless::validate_prerequisites browserless::wait_for_ready
    export -f browserless::wait_for_healthy browserless::backup_data
    export -f browserless::create_network browserless::docker_run browserless::docker_remove
}

# Test directory creation success
@test "browserless::create_directories succeeds when mkdir works" {
    # Mock successful mkdir
    mkdir() {
        if [[ "$1" == "-p" && "$2" == "$BROWSERLESS_DATA_DIR" ]]; then
            return 0
        fi
        return 1
    }
    
    run browserless::create_directories
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Creating Browserless data directory"* ]]
    [[ "$output" == *"Browserless directories created"* ]]
    [[ "$output" == *"ROLLBACK: Remove Browserless data directory"* ]]
}

# Test directory creation failure
@test "browserless::create_directories fails when mkdir fails" {
    # Mock failed mkdir
    mkdir() {
        return 1
    }
    
    run browserless::create_directories
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Failed to create Browserless directories"* ]]
}

# Test configuration update success
@test "browserless::update_config creates proper JSON configuration" {
    run browserless::update_config
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"UPDATE_CONFIG: agents browserless $BROWSERLESS_BASE_URL"* ]]
}

# Test configuration update failure
@test "browserless::update_config handles update failure gracefully" {
    # Mock failed config update
    resources::update_config() {
        return 1
    }
    
    run browserless::update_config
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Failed to update Vrooli configuration"* ]]
}

# Test installation success display
@test "browserless::show_installation_success displays all required information" {
    run browserless::show_installation_success
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Browserless is running and healthy on port $BROWSERLESS_PORT"* ]]
    [[ "$output" == *"Browserless Access Information"* ]]
    [[ "$output" == *"URL: $BROWSERLESS_BASE_URL"* ]]
    [[ "$output" == *"Status Check: $BROWSERLESS_BASE_URL/pressure"* ]]
    [[ "$output" == *"Max Browsers: $BROWSERLESS_MAX_BROWSERS"* ]]
    [[ "$output" == *"Headless Mode: $BROWSERLESS_HEADLESS"* ]]
    [[ "$output" == *"Next Steps"* ]]
    [[ "$output" == *"Test with: $0 --action usage"* ]]
}

# Test complete installation workflow success
@test "browserless::install_service completes full installation successfully" {
    # Mock all dependencies to succeed
    browserless::create_directories() { 
        echo "Directories created"
        return 0 
    }
    browserless::show_installation_success() {
        echo "Installation success displayed"
    }
    browserless::update_config() {
        echo "Config updated"
        return 0
    }
    
    # Mock sleep
    sleep() { return 0; }
    
    # Mock rollback clearing
    ROLLBACK_ACTIONS=()
    OPERATION_ID=""
    
    run browserless::install_service
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Installing Browserless Browser Automation"* ]]
    [[ "$output" == *"START_ROLLBACK: install_browserless_docker"* ]]
    [[ "$output" == *"Directories created"* ]]
    [[ "$output" == *"CREATE_NETWORK"* ]]
    [[ "$output" == *"Installation success displayed"* ]]
}

# Test installation with existing installation check failure
@test "browserless::install_service skips when already installed" {
    # Mock existing installation check to fail (meaning already installed)
    browserless::check_existing_installation() {
        return 1
    }
    
    run browserless::install_service
    
    [ "$status" -eq 0 ]
    # Should exit early without doing installation steps
    [[ ! "$output" == *"CREATE_NETWORK"* ]]
}

# Test installation with prerequisite validation failure
@test "browserless::install_service fails when prerequisites not met" {
    # Mock prerequisite validation to fail
    browserless::validate_prerequisites() {
        return 1
    }
    
    run browserless::install_service
    
    [ "$status" -eq 1 ]
}

# Test installation with directory creation failure
@test "browserless::install_service handles directory creation failure" {
    # Mock directory creation to fail
    browserless::create_directories() {
        return 1
    }
    
    run browserless::install_service
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"HANDLE_ERROR:"* ]]
}

# Test installation with container start failure
@test "browserless::install_service handles container start failure" {
    # Mock directory creation to succeed but docker run to fail
    browserless::create_directories() { return 0; }
    browserless::docker_run() { return 1; }
    
    run browserless::install_service
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"HANDLE_ERROR:"* ]]
}

# Test installation with service readiness timeout
@test "browserless::install_service handles service startup timeout" {
    # Mock directory creation and docker run to succeed but wait_for_ready to fail
    browserless::create_directories() { return 0; }
    browserless::docker_run() { return 0; }
    browserless::wait_for_ready() { return 1; }
    
    run browserless::install_service
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"HANDLE_ERROR:"* ]]
}

# Test installation with health check failure
@test "browserless::install_service handles health check failure gracefully" {
    # Mock everything to succeed except health check
    browserless::create_directories() { return 0; }
    browserless::docker_run() { return 0; }
    browserless::wait_for_ready() { return 0; }
    browserless::wait_for_healthy() { return 1; }
    sleep() { return 0; }
    
    run browserless::install_service
    
    [ "$status" -eq 0 ]  # Should still return success but with warning
    [[ "$output" == *"Browserless started but health check failed"* ]]
}

# Test uninstallation with user confirmation
@test "browserless::uninstall_service completes uninstallation with confirmation" {
    export YES="yes"  # Auto-confirm
    
    # Mock backup and docker removal
    browserless::backup_data() {
        echo "BACKUP: $1"
    }
    
    # Mock read command for data directory removal
    read() {
        if [[ "$1" == "-p" ]]; then
            REPLY="y"
        fi
    }
    
    run browserless::uninstall_service
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Uninstalling Browserless"* ]]
    [[ "$output" == *"BACKUP: uninstall"* ]]
    [[ "$output" == *"DOCKER_REMOVE:"* ]]
    [[ "$output" == *"REMOVE_CONFIG: agents browserless"* ]]
    [[ "$output" == *"Browserless uninstalled successfully"* ]]
}

# Test uninstallation cancellation
@test "browserless::uninstall_service handles user cancellation" {
    export YES="no"  # Don't auto-confirm
    
    # Mock read command to simulate user saying no
    read() {
        if [[ "$1" == "-p" ]]; then
            REPLY="n"
        fi
    }
    
    run browserless::uninstall_service
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Uninstall cancelled"* ]]
    [[ ! "$output" == *"DOCKER_REMOVE:"* ]]  # Should not proceed with removal
}

# Test uninstallation with data directory preservation
@test "browserless::uninstall_service preserves data when user chooses to" {
    export YES="yes"  # Auto-confirm main uninstall
    
    # Mock data directory existence
    test() {
        if [[ "$1" == "-d" && "$2" == "$BROWSERLESS_DATA_DIR" ]]; then
            return 0  # Directory exists
        fi
        return 1
    }
    
    # Mock read command for data directory - user says no to removal
    local read_count=0
    read() {
        if [[ "$1" == "-p" ]]; then
            read_count=$((read_count + 1))
            if [[ $read_count -eq 1 ]]; then
                REPLY="n"  # Don't remove data directory
            fi
        fi
    }
    
    run browserless::uninstall_service
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"DOCKER_REMOVE: no"* ]]  # Should pass 'no' for data removal
}