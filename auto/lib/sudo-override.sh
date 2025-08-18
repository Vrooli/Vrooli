#!/usr/bin/env bash

# Sudo Override Initialization for Auto/ System
# Handles one-time password entry and persistent sudo access for loop operations

set -euo pipefail

# Source dependencies
AUTO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${AUTO_DIR}/../scripts/lib/utils/log.sh" 2>/dev/null || {
    # Fallback logging if utils not available
    log::info() { echo "[INFO] $*"; }
    log::warn() { echo "[WARN] $*"; }
    log::error() { echo "[ERROR] $*"; }
    log::success() { echo "[SUCCESS] $*"; }
}

# Configuration
SUDO_OVERRIDE_DIR="${AUTO_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)}/data/sudo-override"
SUDO_PASSWORD_FILE="${SUDO_OVERRIDE_DIR}/password"
SUDO_CONFIG_FILE="${SUDO_OVERRIDE_DIR}/config"
SUDO_LOCK_FILE="${SUDO_OVERRIDE_DIR}/lock"
SUDO_PID_FILE="${SUDO_OVERRIDE_DIR}/sudo-daemon.pid"

# Default allowed commands for resource management
DEFAULT_SUDO_COMMANDS="systemctl,service,docker,podman,apt-get,apt,chown,chmod,mkdir,rm,cp,mv,lsof,netstat,ps,kill,pkill,npm,pip,brew,snap,git"

#######################################
# Initialize sudo override system
# Prompts for password once and sets up persistent access
# Returns: 0 on success, 1 on failure
#######################################
sudo_override::init() {
    local allowed_commands="${1:-$DEFAULT_SUDO_COMMANDS}"
    
    log::info "🔧 Initializing sudo override for auto/ system"
    
    # Create sudo override directory
    mkdir -p "$SUDO_OVERRIDE_DIR"
    chmod 700 "$SUDO_OVERRIDE_DIR"
    
    # Check if already initialized
    if sudo_override::is_initialized; then
        log::info "✅ Sudo override already initialized"
        return 0
    fi
    
    # Check if sudo is available
    if ! command -v sudo >/dev/null 2>&1; then
        log::error "❌ Sudo is not available on this system"
        return 1
    fi
    
    # Check if user has sudo privileges
    if sudo -n true 2>/dev/null; then
        log::success "✅ Passwordless sudo access confirmed"
        sudo_override::save_config "$allowed_commands" ""
        return 0
    else
        log::info "🔐 Sudo password required - no passwordless access available"
    fi
    
    # Prompt for sudo password
    log::info "🔐 Sudo password required for auto/ system"
    log::info "This password will be stored securely for the duration of the session"
    
    local password
    local password_confirm
    
    # Read password securely
    read -s -p "Enter sudo password: " password
    echo
    
    # Confirm password
    read -s -p "Confirm sudo password: " password_confirm
    echo
    
    if [[ "$password" != "$password_confirm" ]]; then
        log::error "❌ Passwords do not match"
        return 1
    fi
    
    # Test sudo access with password
    if ! echo "$password" | sudo -S true 2>/dev/null; then
        log::error "❌ Invalid sudo password"
        return 1
    fi
    
    # Save configuration
    if sudo_override::save_config "$allowed_commands" "$password"; then
        log::success "✅ Sudo override initialized successfully"
        log::info "Allowed commands: $allowed_commands"
        return 0
    else
        log::error "❌ Failed to save sudo override configuration"
        return 1
    fi
}

#######################################
# Check if sudo override is initialized
# Returns: 0 if initialized, 1 if not
#######################################
sudo_override::is_initialized() {
    [[ -f "$SUDO_CONFIG_FILE" ]] && [[ -f "$SUDO_PASSWORD_FILE" ]]
}

#######################################
# Save sudo override configuration
# Arguments:
#   $1 - Allowed commands (comma-separated)
#   $2 - Sudo password (empty for passwordless)
# Returns: 0 on success, 1 on failure
#######################################
sudo_override::save_config() {
    local allowed_commands="$1"
    local password="$2"
    
    # Save allowed commands
    echo "$allowed_commands" > "$SUDO_CONFIG_FILE"
    chmod 600 "$SUDO_CONFIG_FILE"
    
    # Save password if provided
    if [[ -n "$password" ]]; then
        echo "$password" > "$SUDO_PASSWORD_FILE"
        chmod 600 "$SUDO_PASSWORD_FILE"
    else
        touch "$SUDO_PASSWORD_FILE"
        chmod 600 "$SUDO_PASSWORD_FILE"
    fi
    
    return 0
}

#######################################
# Load sudo override configuration
# Exports environment variables for Claude Code
# Returns: 0 on success, 1 on failure
#######################################
sudo_override::load_config() {
    if ! sudo_override::is_initialized; then
        log::error "❌ Sudo override not initialized"
        log::info "Run: sudo_override::init to initialize"
        return 1
    fi
    
    # Load allowed commands
    if [[ -f "$SUDO_CONFIG_FILE" ]]; then
        export SUDO_COMMANDS=$(cat "$SUDO_CONFIG_FILE")
    else
        export SUDO_COMMANDS="$DEFAULT_SUDO_COMMANDS"
    fi
    
    # Load password if available
    if [[ -f "$SUDO_PASSWORD_FILE" ]]; then
        local password
        password=$(cat "$SUDO_PASSWORD_FILE")
        if [[ -n "$password" ]]; then
            export SUDO_PASSWORD="$password"
        fi
    fi
    
    # Enable sudo override
    export SUDO_OVERRIDE="yes"
    
    log::info "✅ Sudo override configuration loaded"
    return 0
}

#######################################
# Test sudo override functionality
# Returns: 0 on success, 1 on failure
#######################################
sudo_override::test() {
    log::info "🧪 Testing sudo override functionality"
    
    if ! sudo_override::load_config; then
        return 1
    fi
    
    # Test basic sudo access
    if sudo -n echo "Sudo test" 2>/dev/null; then
        log::success "✅ Passwordless sudo access confirmed"
        return 0
    else
        # Test with password if available
        local password="${SUDO_PASSWORD:-}"
        if [[ -n "$password" ]]; then
            if echo "$password" | sudo -S echo "Sudo test with password" 2>/dev/null; then
                log::success "✅ Sudo access with password confirmed"
                return 0
            else
                log::error "❌ Sudo access with password failed - password may be incorrect"
                return 1
            fi
        else
            log::error "❌ No sudo password available for testing"
            log::info "💡 Run 'sudo-init' to set up sudo override with password"
            return 1
        fi
    fi
}

#######################################
# Clean up sudo override configuration
# Removes stored password and configuration
# Returns: 0 on success, 1 on failure
#######################################
sudo_override::cleanup() {
    log::info "🧹 Cleaning up sudo override configuration"
    
    # Remove password file
    if [[ -f "$SUDO_PASSWORD_FILE" ]]; then
        rm -f "$SUDO_PASSWORD_FILE"
    fi
    
    # Remove config file
    if [[ -f "$SUDO_CONFIG_FILE" ]]; then
        rm -f "$SUDO_CONFIG_FILE"
    fi
    
    # Remove lock file
    if [[ -f "$SUDO_LOCK_FILE" ]]; then
        rm -f "$SUDO_LOCK_FILE"
    fi
    
    # Remove PID file
    if [[ -f "$SUDO_PID_FILE" ]]; then
        rm -f "$SUDO_PID_FILE"
    fi
    
    # Clear environment variables
    unset SUDO_OVERRIDE
    unset SUDO_COMMANDS
    unset SUDO_PASSWORD
    
    log::success "✅ Sudo override configuration cleaned up"
    return 0
}

#######################################
# Get sudo override status
# Returns: 0 if active, 1 if not
#######################################
sudo_override::status() {
    if sudo_override::is_initialized; then
        log::info "✅ Sudo override is initialized"
        if [[ -f "$SUDO_CONFIG_FILE" ]]; then
            local commands
            commands=$(cat "$SUDO_CONFIG_FILE")
            log::info "Allowed commands: $commands"
        fi
        return 0
    else
        log::info "❌ Sudo override is not initialized"
        return 1
    fi
}

#######################################
# Main function for command-line usage
#######################################
sudo_override::main() {
    local action="${1:-help}"
    
    case "$action" in
        "init")
            local commands="${2:-$DEFAULT_SUDO_COMMANDS}"
            sudo_override::init "$commands"
            ;;
        "test")
            sudo_override::test
            ;;
        "status")
            sudo_override::status
            ;;
        "cleanup")
            sudo_override::cleanup
            ;;
        "help"|"--help"|"-h")
            cat << EOF
Sudo Override Management for Auto/ System

Usage: sudo_override::main <action> [options]

Actions:
  init [commands]    Initialize sudo override with allowed commands
  test               Test sudo override functionality
  status             Show sudo override status
  cleanup            Remove sudo override configuration
  help               Show this help message

Examples:
  sudo_override::main init "systemctl,docker,apt-get"
  sudo_override::main test
  sudo_override::main status
  sudo_override::main cleanup

Default allowed commands: $DEFAULT_SUDO_COMMANDS
EOF
            ;;
        *)
            log::error "Unknown action: $action"
            sudo_override::main help
            return 1
            ;;
    esac
}

# Export functions for use in other scripts
export -f sudo_override::init
export -f sudo_override::is_initialized
export -f sudo_override::save_config
export -f sudo_override::load_config
export -f sudo_override::test
export -f sudo_override::cleanup
export -f sudo_override::status
export -f sudo_override::main

# If this script is run directly, execute main function
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    sudo_override::main "$@"
fi 