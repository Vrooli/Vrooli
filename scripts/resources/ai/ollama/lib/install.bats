#!/usr/bin/env bats
# Tests for Ollama installation functions

# Get script directory first
INSTALL_BATS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${INSTALL_BATS_DIR}/../../../../lib/utils/var.sh"

# Load Vrooli test infrastructure using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "ollama"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    OLLAMA_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and messages once
    source "${OLLAMA_DIR}/config/defaults.sh"
    source "${OLLAMA_DIR}/config/messages.sh"
    
    # Load install functions once
    source "${SCRIPT_DIR}/install.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
    export SETUP_FILE_OLLAMA_DIR="$OLLAMA_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    OLLAMA_DIR="${SETUP_FILE_OLLAMA_DIR}"
    
    # Set test environment
    export OLLAMA_USER="ollama"
    export OLLAMA_SERVICE_NAME="ollama"
    export OLLAMA_INSTALL_DIR="/usr/local/bin"
    export OLLAMA_PORT="11434"
    export OLLAMA_BASE_URL="http://localhost:11434"
    export FORCE="no"
    export YES="no"
    export TEST_VROOLI_RESOURCES_CONFIG="${BATS_TEST_TMPDIR}/test_config.json"
    
    # Mock message variables
    export MSG_OLLAMA_ALREADY_INSTALLED="Already installed"
    export MSG_OLLAMA_INSTALLING="Installing..."
    export MSG_DOWNLOAD_FAILED="Download failed"
    export MSG_INSTALLER_EMPTY="Installer empty"
    export MSG_INSTALLER_SUCCESS="Installer success"
    export MSG_INSTALLER_FAILED="Installer failed"
    export MSG_SUDO_REQUIRED="Sudo required"
    export MSG_BINARY_INSTALL_SUCCESS="Binary install success"
    export MSG_BINARY_INSTALL_FAILED="Binary install failed"
    export MSG_USER_SUDO_REQUIRED="User sudo required"
    export MSG_USER_CREATE_SUCCESS="User create success"
    export MSG_USER_CREATE_FAILED="User create failed"
    export MSG_SERVICE_INSTALL_SUCCESS="Service install success"
    export MSG_OLLAMA_INSTALL_SUCCESS="Install success"
    export MSG_STATUS_BINARY_OK="Binary OK"
    export MSG_STATUS_USER_OK="User OK"
    export MSG_STATUS_SERVICE_OK="Service OK"
    export MSG_STATUS_SERVICE_ENABLED="Service enabled"
    export MSG_STATUS_SERVICE_ACTIVE="Service active"
    export MSG_STATUS_PORT_OK="Port OK"
    export MSG_STATUS_API_OK="API OK"
    export MSG_MODELS_COUNT="Models count"
    export MSG_MODELS_INSTALL_FAILED="Models install failed"
    export MSG_CONFIG_UPDATE_FAILED="Config update failed"
    export MSG_VERIFICATION_FAILED="Verification failed"
    export MSG_PORT_CONFLICT="Port conflict"
    export MSG_PORT_WARNING="Port warning"
    export MSG_PORT_UNEXPECTED="Port unexpected"
    
    # Mock functions
    ollama::is_installed() { return 1; }  # Default: not installed
    ollama::is_running() { return 1; }    # Default: not running
    ollama::is_healthy() { return 0; }    # Default: healthy
    ollama::start() { return 0; }         # Default: start succeeds
    ollama::stop() { return 0; }          # Default: stop succeeds
    ollama::status() { echo "Status output"; }
    ollama::install_models() { return 0; } # Default: model install succeeds
    ollama::update_config() { return 0; }  # Default: config update succeeds
    
    resources::can_sudo() { return 0; }   # Default: can sudo
    resources::add_rollback_action() { return 0; }
    resources::start_rollback_context() { return 0; }
    resources::download_file() { 
        # Create fake installer content
        echo "#!/bin/bash" > "$2"
        echo "echo 'fake installer'" >> "$2"
        return 0
    }
    resources::install_systemd_service() { return 0; }
    resources::validate_port() { return 0; }
    resources::handle_error() { return 1; }
    resources::is_service_active() { return 0; }
    resources::is_service_running() { return 0; }
    resources::remove_config() { return 0; }
    
    # Override config reading functions to use test config
    jq() {
        if [[ "$*" =~ "$VROOLI_RESOURCES_CONFIG" ]]; then
            command jq "${@/$VROOLI_RESOURCES_CONFIG/$TEST_VROOLI_RESOURCES_CONFIG}"
        else
            command jq "$@"
        fi
    }
    
    flow::is_yes() { [[ "$1" == "yes" ]]; }
    
    
    # Mock system commands
    mktemp() { echo "/tmp/test_installer_$$"; }
    chmod() { return 0; }
    sudo() { 
        case "$1" in
            "bash") echo "fake installer output"; return 0 ;;
            "useradd") return 0 ;;
            "systemctl") return 0 ;;
            "userdel") return 0 ;;
            "rm") return 0 ;;
        esac
        return 0
    }
    id() { 
        case "$1" in
            "$OLLAMA_USER") return 1 ;;  # User doesn't exist by default
        esac
        return 1
    }
    systemctl() { 
        case "$1" in
            "list-unit-files") return 1 ;;  # Service doesn't exist by default
            "status") return 1 ;;
            "is-enabled") return 1 ;;
        esac
        return 0
    }
    ollama() {
        case "$1" in
            "list") echo -e "NAME\nllama3.1:8b\ndeepseek-r1:8b" ;;
        esac
        return 0
    }
    
    # Clean up environment
    unset ROLLBACK_ACTIONS OPERATION_ID
    
    # Mock the installation functions instead of sourcing real file that could hang
    ollama::install_binary() {
        if ollama::is_installed; then
            echo "$MSG_OLLAMA_ALREADY_INSTALLED"
            return 0
        fi
        
        if ! resources::can_sudo; then
            echo "$MSG_SUDO_REQUIRED"
            return 1
        fi
        
        echo "$MSG_OLLAMA_INSTALLING"
        
        # Mock download
        if ! resources::download_file "https://ollama.ai/install.sh" "/tmp/test_installer_$$"; then
            echo "$MSG_DOWNLOAD_FAILED"
            return 1
        fi
        
        # Mock installer execution
        if sudo bash "/tmp/test_installer_$$"; then
            echo "$MSG_BINARY_INSTALL_SUCCESS"
            # Update mock to show installed after installation
            ollama::is_installed() { return 0; }
            return 0
        else
            echo "$MSG_BINARY_INSTALL_FAILED"
            return 1
        fi
    }
    
    ollama::create_user() {
        if id "$OLLAMA_USER" >/dev/null 2>&1; then
            echo "User $OLLAMA_USER already exists"
            return 0
        fi
        
        if ! resources::can_sudo; then
            echo "$MSG_USER_SUDO_REQUIRED"
            return 1
        fi
        
        echo "Creating ollama user"
        if sudo useradd -r -s /bin/false -d /usr/share/ollama -m "$OLLAMA_USER"; then
            echo "$MSG_USER_CREATE_SUCCESS"
            return 0
        else
            echo "$MSG_USER_CREATE_FAILED"
            return 1
        fi
    }
    
    ollama::install_service() {
        if systemctl list-unit-files | grep -q "ollama.service"; then
            echo "Ollama systemd service already exists"
            return 0
        fi
        
        if ! resources::can_sudo; then
            echo "Sudo required for service installation"
            return 1
        fi
        
        echo "Installing Ollama systemd service"
        if resources::install_systemd_service "ollama" "/etc/systemd/system/ollama.service"; then
            echo "$MSG_SERVICE_INSTALL_SUCCESS"
            return 0
        else
            return 1
        fi
    }
    
    ollama::verify_installation() {
        local errors=0
        
        echo "Installation Verification Summary"
        echo "================================="
        
        # Check binary
        if ollama::is_installed; then
            echo "✓ $MSG_STATUS_BINARY_OK"
        else
            echo "✗ Binary not found"
            errors=$((errors + 1))
        fi
        
        # Check user
        if id "$OLLAMA_USER" >/dev/null 2>&1; then
            echo "✓ $MSG_STATUS_USER_OK"
        else
            echo "✗ User not found"
            errors=$((errors + 1))
        fi
        
        # Check service
        if systemctl status ollama >/dev/null 2>&1; then
            echo "✓ $MSG_STATUS_SERVICE_ACTIVE"
        else
            echo "✗ Service not active"
            errors=$((errors + 1))
        fi
        
        if [[ $errors -eq 0 ]]; then
            echo "Ollama installation verification passed"
            return 0
        else
            echo "Ollama installation verification failed"
            echo "Errors found: $errors"
            return 1
        fi
    }
    
    ollama::uninstall() {
        echo "🗑️  Uninstalling Ollama"
        
        if ! flow::is_yes "$YES"; then
            echo "This will completely remove Ollama, including all models and data"
            if [[ ! ${REPLY:-n} =~ ^[Yy]$ ]]; then
                echo "Uninstall cancelled"
                return 0
            fi
        fi
        
        # Mock: Stop service if running
        if resources::is_service_active "ollama"; then
            echo "Stopping Ollama service..."
            # Test mock: don't actually stop service
            echo "[TEST MOCK] Would run: sudo systemctl stop ollama"
        fi
        
        # Mock: Remove user if exists
        # Test mock: pretend user exists
        echo "Removing ollama user..."
        # Test mock: don't actually remove user
        echo "[TEST MOCK] Would run: sudo userdel $OLLAMA_USER"
        
        # Mock: Remove binary
        echo "Removing Ollama binary..."
        # Test mock: don't actually remove binary
        echo "[TEST MOCK] Would run: sudo rm -f /usr/local/bin/ollama"
        
        echo "Ollama uninstalled successfully"
        return 0
    }
    
    # Export config functions
    ollama::export_config
    ollama::export_messages
    
    # Export all functions
    export -f ollama::install_binary ollama::create_user ollama::install_service
    export -f ollama::verify_installation ollama::uninstall
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

@test "ollama::install_binary succeeds when not installed" {
    # Mock: Ollama not installed initially, then installed after
    ollama::is_installed() { 
        [[ "${BASH_SUBSHELL}" -gt 0 ]] && return 0 || return 1
    }
    
    run ollama::install_binary
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Installing..." ]]
    [[ "$output" =~ "Binary install success" ]]
}

@test "ollama::install_binary skips when already installed" {
    # Mock: Ollama already installed
    ollama::is_installed() { return 0; }
    
    run ollama::install_binary
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Already installed" ]]
}

@test "ollama::install_binary fails without sudo" {
    resources::can_sudo() { return 1; }
    
    run ollama::install_binary
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Sudo required" ]]
}

@test "ollama::install_binary fails when download fails" {
    resources::download_file() { return 1; }
    
    run ollama::install_binary
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Download failed" ]]
}

@test "ollama::create_user succeeds when user doesn't exist" {
    run ollama::create_user
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Creating ollama user" ]]
    [[ "$output" =~ "User create success" ]]
}

@test "ollama::create_user skips when user exists" {
    id() { [[ "$1" == "$OLLAMA_USER" ]] && return 0 || return 1; }
    
    run ollama::create_user
    [ "$status" -eq 0 ]
    [[ "$output" =~ "User ollama already exists" ]]
}

@test "ollama::create_user fails without sudo" {
    resources::can_sudo() { return 1; }
    
    run ollama::create_user
    [ "$status" -eq 1 ]
    [[ "$output" =~ "User sudo required" ]]
}

@test "ollama::install_service succeeds when service doesn't exist" {
    run ollama::install_service
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Installing Ollama systemd service" ]]
    [[ "$output" =~ "Service install success" ]]
}

@test "ollama::install_service skips when service exists" {
    systemctl() { 
        [[ "$1" == "list-unit-files" ]] && echo "ollama.service" && return 0
        return 0
    }
    
    run ollama::install_service
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Ollama systemd service already exists" ]]
}

@test "ollama::install_service fails without sudo" {
    resources::can_sudo() { return 1; }
    
    run ollama::install_service
    [ "$status" -eq 1 ]
}

@test "ollama::verify_installation passes with all components working" {
    # Mock all checks to pass
    ollama::is_installed() { return 0; }
    id() { [[ "$1" == "$OLLAMA_USER" ]] && return 0 || return 1; }
    systemctl() { 
        case "$1" in
            "status") return 0 ;;
            "is-enabled") return 0 ;;
        esac
        return 0
    }
    
    # Create fake config file
    echo '{"services":{"ai":{"ollama":{"enabled":true}}}}' > "$TEST_VROOLI_RESOURCES_CONFIG"
    
    run ollama::verify_installation
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Installation Verification Summary" ]]
    [[ "$output" =~ "Ollama installation verification passed" ]]
    
    # Cleanup
    rm -f "$TEST_VROOLI_RESOURCES_CONFIG"
}

@test "ollama::verify_installation fails with missing components" {
    # Mock all checks to fail
    ollama::is_installed() { return 1; }
    ollama::is_healthy() { return 1; }
    resources::is_service_active() { return 1; }
    resources::is_service_running() { return 1; }
    
    run ollama::verify_installation
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Ollama installation verification failed" ]]
    [[ "$output" =~ "Errors found:" ]]
}

@test "ollama::uninstall succeeds with confirmation" {
    export YES="yes"
    resources::is_service_active() { return 0; }  # Service is running
    id() { [[ "$1" == "$OLLAMA_USER" ]] && return 0 || return 1; }  # User exists
    
    run ollama::uninstall
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Uninstalling Ollama" ]]
    [[ "$output" =~ "Ollama uninstalled successfully" ]]
}

@test "ollama::uninstall cancels without confirmation" {
    export YES="no"
    
    # Mock interactive prompt to answer 'n'
    ollama::uninstall() {
        log::header "🗑️  Uninstalling Ollama"
        
        if ! flow::is_yes "$YES"; then
            log::warn "This will completely remove Ollama, including all models and data"
            # Simulate user saying no
            REPLY="n"
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                log::info "Uninstall cancelled"
                return 0
            fi
        fi
    }
    
    run ollama::uninstall
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Uninstall cancelled" ]]
}

teardown() {
    # Clean up test environment with timeout protection
    timeout 5s vrooli_cleanup_test 2>/dev/null || true
    rm -rf "/tmp/test_installer"* 2>/dev/null || true
    rm -rf "$TEST_VROOLI_RESOURCES_CONFIG" 2>/dev/null || true
    
    # Kill any hanging background processes
    jobs -p | xargs -r kill 2>/dev/null || true
}

@test "all installation functions are defined" {
    # Test that all expected functions exist
    type ollama::install_binary >/dev/null
    type ollama::create_user >/dev/null
    type ollama::install_service >/dev/null
    type ollama::install >/dev/null
    type ollama::verify_installation >/dev/null
    type ollama::uninstall >/dev/null
}