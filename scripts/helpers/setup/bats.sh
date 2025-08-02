#!/usr/bin/env bash
# Posix-compliant script to setup bats and its dependencies
set -euo pipefail

ORIGINAL_DIR=$(pwd)
SETUP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/flow.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/var.sh"

# Ensure var_SCRIPT_TESTS_DIR is defined
if [ -z "${var_SCRIPT_TESTS_DIR:-}" ]; then
    log::error "var_SCRIPT_TESTS_DIR not defined—please check var.sh"
    exit 1
fi

BATS_DEPENDENCIES_DIR="${var_SCRIPT_TESTS_DIR}/helpers"

# Function to determine and export BATS_PREFIX based on environment and sudo mode
bats::determine_prefix() {
    # Default Bats installation prefix unless overridden by environment variable
    local prefix=${BATS_PREFIX:-/usr/local}

    # Check if we need to switch to user-local dir
    if [ "$prefix" = "/usr/local" ]; then
        # Check if we should use local installation
        local use_local=false
        
        if [ "${SUDO_MODE:-error}" = "skip" ]; then
            use_local=true
            log::info "SUDO_MODE=skip: switching to user-local Bats installation"
        elif [ "${SUDO_MODE:-error}" = "error" ]; then
            # In error mode, check if sudo is available without exiting
            # Test for passwordless sudo
            if ! command -v sudo >/dev/null 2>&1 || ! sudo -n true >/dev/null 2>&1; then
                use_local=true
                log::info "Sudo not available: switching to user-local Bats installation"
            fi
        fi
        
        if [ "$use_local" = "true" ]; then
            prefix="$HOME/.local"
            log::info "Will install Bats to $prefix (user directory)"
            # Ensure the local bin directory exists for Bats install and PATH update
            mkdir -p "$prefix/bin" # Bats install.sh might expect the base prefix to exist
        else
            log::info "Using default Bats install prefix: $prefix"
        fi
    elif [ "$prefix" != "/usr/local" ]; then
        log::info "Using user-defined BATS_PREFIX: $prefix"
    fi
    export BATS_PREFIX="$prefix" # Export the final determined value
}

# Function to create directory to store bats core and dependencies
bats::create_dependencies_dir() {
    mkdir -p "$BATS_DEPENDENCIES_DIR"
}

# Function to clone and confirm a bats dependency
bats::install_dependency() {
    local repo_url=$1
    local dir_name=$2
    cd "$BATS_DEPENDENCIES_DIR"
    if [ ! -d "$dir_name" ]; then
        # Handle git clone with TLS compatibility
        # If running under sudo, use the original user's git config
        if [ -n "${SUDO_USER:-}" ] && [ "$SUDO_USER" != "root" ]; then
            # Run git as the original user to inherit their config
            sudo -u "$SUDO_USER" git -c http.sslVersion=tlsv1.2 clone "$repo_url" "$dir_name"
        else
            # Set git config for TLS compatibility to handle GnuTLS handshake issues
            local original_ssl_version
            original_ssl_version=$(git config --global http.sslVersion 2>/dev/null || echo "")
            
            # Temporarily set TLS version
            git config --global http.sslVersion tlsv1.2
            
            # Clone the repository
            git clone "$repo_url" "$dir_name"
            
            # Restore original config or unset if it wasn't set
            if [ -n "$original_ssl_version" ]; then
                git config --global http.sslVersion "$original_ssl_version"
            else
                git config --global --unset http.sslVersion 2>/dev/null || true
            fi
        fi
        
        log::success "$dir_name installed successfully at $(pwd)/$dir_name"
    else
        log::info "$dir_name is already installed"
    fi
    cd "$ORIGINAL_DIR"
}

# Install Bats-core
bats::install_core() {
    bats::determine_prefix # Determine prefix just before installation

    cd "$BATS_DEPENDENCIES_DIR"
    
    # Fix permissions on the dependencies directory if running under sudo
    # This ensures the original user can write to it when cloning
    if [ -n "${SUDO_USER:-}" ] && [ "$SUDO_USER" != "root" ]; then
        log::info "Fixing permissions on $BATS_DEPENDENCIES_DIR for user $SUDO_USER"
        chown -R "$SUDO_USER:$(id -gn "$SUDO_USER")" "$BATS_DEPENDENCIES_DIR"
    fi
    
    if [ ! -d "bats-core" ]; then
        # Handle git clone with TLS compatibility
        # If running under sudo, use the original user's git config
        if [ -n "${SUDO_USER:-}" ] && [ "$SUDO_USER" != "root" ]; then
            # Run git as the original user to inherit their config
            sudo -u "$SUDO_USER" git -c http.sslVersion=tlsv1.2 clone https://github.com/bats-core/bats-core.git
        else
            # Set git config for TLS compatibility to handle GnuTLS handshake issues
            git -c http.sslVersion=tlsv1.2 clone https://github.com/bats-core/bats-core.git
        fi
        cd bats-core
        # Ensure the BATS installation prefix exists (use sudo if required for system paths)
        # Check if we need sudo for system paths
        local need_sudo=false
        if [[ "$BATS_PREFIX" = "/usr/local"* ]]; then
            # For system paths, check if we can use sudo (without exiting on failure)
            if command -v sudo >/dev/null 2>&1 && sudo -n true >/dev/null 2>&1; then
                need_sudo=true
            fi
        fi
        
        # Create directory with or without sudo
        if [ "$need_sudo" = "true" ]; then
            sudo mkdir -p "$BATS_PREFIX"
        else
            mkdir -p "$BATS_PREFIX"
        fi
        
        # Install with or without sudo
        if [ "$need_sudo" = "true" ]; then
            log::info "Installing Bats-core into $BATS_PREFIX (with sudo)"
            sudo ./install.sh "$BATS_PREFIX"
        else
            log::info "Installing Bats-core into $BATS_PREFIX (no sudo)"
            ./install.sh "$BATS_PREFIX"
        fi
        log::success "Bats-core installed successfully into $BATS_PREFIX"
        cd ..
    else
        log::info "Bats-core is already installed"
    fi
    cd "$ORIGINAL_DIR"
}

# Install Bats for testing bash scripts
bats::install() {
    bats::determine_prefix # Determine prefix before potentially updating PATH

    log::header "Installing Bats and dependencies for Bash script testing"

    # Create dependencies directory
    bats::create_dependencies_dir

    # Install Bats-core
    bats::install_core

    # Install other dependencies
    bats::install_dependency "https://github.com/bats-core/bats-support.git" "bats-support"
    bats::install_dependency "https://github.com/jasonkarns/bats-mock.git" "bats-mock"
    bats::install_dependency "https://github.com/bats-core/bats-assert.git" "bats-assert"

    # Ensure Bats bin directory is in PATH without duplicates
    if [[ ":$PATH:" != *":$BATS_PREFIX/bin:"* ]]; then
        export PATH="$BATS_PREFIX/bin:$PATH"
        log::info "Added $BATS_PREFIX/bin to PATH"
    fi

    log::success "Bats and dependencies installed successfully"
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    bats::install "$@"
fi
