#!/bin/bash
# Standard Mock Framework for BATS Tests
# This file provides consistent mocking patterns across all test files

#######################################
# Setup standard mock functions that are commonly needed
# Globals: None
# Arguments: None
# Returns: 0 on success
#######################################
setup_standard_mock_framework() {
    # Prevent infinite loops in mocks and duplicate setups
    if [[ "${MOCK_FRAMEWORK_LOADED:-}" == "true" ]]; then
        return 0
    fi
    export MOCK_FRAMEWORK_LOADED="true"
    
    declare -g MOCK_CALL_COUNT=0
    
    # Mock system commands
    docker() {
        MOCK_CALL_COUNT=$((MOCK_CALL_COUNT + 1))
        if [[ $MOCK_CALL_COUNT -gt 50 ]]; then
            echo "ERROR: Too many docker calls, preventing infinite loop" >&2
            return 1
        fi
        
        case "$1" in
            "ps"|"images"|"network"|"volume") echo "mock_docker_output" ;;
            "run"|"build"|"pull"|"start"|"stop"|"rm") return 0 ;;
            "exec") echo "mock_exec_output" ;;
            "inspect") echo '{"State": {"Status": "running", "Health": {"Status": "healthy"}}}' ;;
            *) return 0 ;;
        esac
    }
    
    curl() {
        MOCK_CALL_COUNT=$((MOCK_CALL_COUNT + 1))
        if [[ $MOCK_CALL_COUNT -gt 20 ]]; then
            echo "ERROR: Too many curl calls, preventing infinite loop" >&2
            return 1
        fi
        
        # Handle all common curl patterns
        case "$*" in
            *"/health"*) echo '{"status": "healthy"}' ;;
            *"/api/"*) echo '{"data": []}' ;;
            *"/status"*) echo '{"status": "ok", "version": "1.0.0"}' ;;
            *"-s"*|*"--silent"*) echo '{"success": true}' ;;
            *"-f"*|*"--fail"*) return 0 ;;
            *"localhost"*|*"127.0.0.1"*) echo '{"status": "running"}' ;;
            *) echo '{"success": true}' ;;
        esac
        return 0
    }
    
    jq() {
        case "$*" in
            *".status"*) echo "healthy" ;;
            *".enabled"*) echo "true" ;;
            *) echo "mock_value" ;;
        esac
    }
    
    # Mock system checks
    command() {
        case "$2" in
            "docker"|"curl"|"jq"|"openssl"|"sudo") return 0 ;;
            *) return 1 ;;
        esac
    }
    
    which() { 
        case "$1" in
            "docker"|"curl"|"jq"|"openssl"|"sudo") echo "/usr/bin/$1" ;;
            *) return 1 ;;
        esac
    }
    
    # Mock read with protection against infinite loops
    declare -g READ_CALL_COUNT=0
    read() {
        READ_CALL_COUNT=$((READ_CALL_COUNT + 1))
        if [[ $READ_CALL_COUNT -gt 5 ]]; then
            echo "ERROR: Too many read calls, preventing infinite loop" >&2
            REPLY="n"  # Set REPLY to prevent hangs
            return 1
        fi
        
        # Set REPLY variable properly for read -r usage
        case "$*" in
            *"-p"*"y/N"*|*"confirm"*) REPLY="y" ;;
            *"-p"*|*"prompt"*) REPLY="mock_input" ;;
            *"-s"*) REPLY="mock_password" ;;
            *) REPLY="mock_input" ;;
        esac
        return 0
    }
    
    # Mock logging functions
    log::info() { echo "INFO: $*"; }
    log::error() { echo "ERROR: $*"; }
    log::success() { echo "SUCCESS: $*"; }
    log::warn() { echo "WARN: $*"; }
    log::warning() { echo "WARNING: $*"; }
    log::header() { echo "HEADER: $*"; }
    log::debug() { echo "DEBUG: $*"; }
    
    # Mock flow functions
    flow::is_yes() { [[ "$1" == "yes" ]]; }
    
    # Mock system functions
    system::is_command() {
        case "$1" in
            "docker"|"curl"|"jq"|"openssl"|"sudo"|"systemctl"|"useradd"|"userdel") return 0 ;;
            *) return 1 ;;
        esac
    }
    
    # Mock resource functions
    resources::can_sudo() { return 0; }
    resources::ensure_docker() { return 0; }
    resources::validate_port() { return 0; }
    resources::update_config() { echo "Config updated"; return 0; }
    resources::remove_config() { return 0; }
    resources::start_rollback_context() { return 0; }
    resources::add_rollback_action() { return 0; }
    resources::execute_rollback() { return 0; }
    resources::get_default_port() {
        case "$1" in
            "agent-s2") echo "4113" ;;
            "ollama") echo "11434" ;;
            "unstructured-io") echo "8000" ;;
            "whisper") echo "9000" ;;
            "comfyui") echo "8188" ;;
            "n8n") echo "5678" ;;
            *) echo "8080" ;;
        esac
    }
    resources::download_file() {
        # Create fake file
        echo "#!/bin/bash" > "$2"
        echo "echo 'mock installer'" >> "$2"
        return 0
    }
    resources::install_systemd_service() { return 0; }
    resources::is_service_active() { return 0; }
    resources::is_service_running() { return 0; }
    
    # Mock common export functions that resources expect
    unstructured_io::export_messages() { return 0; }
    unstructured_io::export_config() { return 0; }
    agents2::export_messages() { return 0; }
    agents2::export_config() { return 0; }
    ollama::export_messages() { return 0; }
    ollama::export_config() { return 0; }
    whisper::export_messages() { return 0; }
    whisper::export_config() { return 0; }
    comfyui::export_messages() { return 0; }
    comfyui::export_config() { return 0; }
    n8n::export_messages() { return 0; }
    n8n::export_config() { return 0; }
    
    # Mock common system utilities
    sudo() {
        case "$1" in
            "bash"|"systemctl"|"useradd"|"userdel"|"mkdir"|"rm"|"chown"|"chmod") return 0 ;;
            *) return 0 ;;
        esac
    }
    
    # Mock git for bats tests
    git() {
        case "$1" in
            "clone") echo "Cloning..." ;;
            "pull") echo "Already up to date." ;;
            *) return 0 ;;
        esac
    }
    
    systemctl() {
        case "$1" in
            "status"|"start"|"stop"|"enable"|"disable"|"is-active"|"is-enabled") return 0 ;;
            "list-unit-files") echo "mock.service" ;;
            *) return 0 ;;
        esac
    }
    
    id() {
        case "$1" in
            "root") return 0 ;;
            *) return 1 ;;  # User doesn't exist by default
        esac
    }
    
    # Mock sleep to prevent timeouts in wait loops
    sleep() {
        # Don't actually sleep in tests
        return 0
    }
    
    # Export all functions
    export -f docker curl jq command which read
    export -f log::info log::error log::success log::warn log::warning log::header log::debug
    export -f flow::is_yes system::is_command
    export -f resources::can_sudo resources::ensure_docker resources::validate_port
    export -f resources::update_config resources::remove_config resources::start_rollback_context
    export -f resources::add_rollback_action resources::execute_rollback resources::download_file
    export -f resources::install_systemd_service resources::is_service_active resources::is_service_running
    export -f resources::get_default_port
    export -f sudo systemctl id sleep git
    export -f unstructured_io::export_messages unstructured_io::export_config
    export -f agents2::export_messages agents2::export_config
    export -f ollama::export_messages ollama::export_config
    export -f whisper::export_messages whisper::export_config
    export -f comfyui::export_messages comfyui::export_config
    export -f n8n::export_messages n8n::export_config
    
    return 0
}

#######################################
# Clean up mock framework
# Globals: MOCK_CALL_COUNT, READ_CALL_COUNT
# Arguments: None
# Returns: 0 on success
#######################################
cleanup_standard_mock_framework() {
    # Reset counters
    MOCK_CALL_COUNT=0
    READ_CALL_COUNT=0
    
    # Clear framework loaded flag to allow re-setup in next test
    unset MOCK_FRAMEWORK_LOADED
    
    # Kill any background processes
    jobs -p | xargs -r kill 2>/dev/null || true
    
    # Clean up any temporary files
    rm -rf "${MOCK_RESPONSES_DIR:-}"/* 2>/dev/null || true
    
    return 0
}

# Source additional mock helpers
MOCK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "${MOCK_DIR}/file_operation_mocks.bash" ]]; then
    source "${MOCK_DIR}/file_operation_mocks.bash"
fi
if [[ -f "${MOCK_DIR}/http_mock_helpers.bash" ]]; then
    source "${MOCK_DIR}/http_mock_helpers.bash"
fi
if [[ -f "${MOCK_DIR}/docker_mock_helpers.bash" ]]; then
    source "${MOCK_DIR}/docker_mock_helpers.bash"
fi

# Auto-setup if sourced
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    setup_standard_mock_framework
fi