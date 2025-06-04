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
    # Condition: Default prefix is used AND (sudo mode is skip OR sudo cannot be run)
    if [ "$prefix" = "/usr/local" ] && { [ "${SUDO_MODE:-error}" = "skip" ] || ! flow::can_run_sudo; }; then
        prefix="$HOME/.local"
        log::info "Sudo unavailable/skipped: switching Bats install prefix to $prefix"
        # Ensure the local bin directory exists for Bats install and PATH update
        mkdir -p "$prefix/bin" # Bats install.sh might expect the base prefix to exist
    elif [ "$prefix" != "/usr/local" ]; then
        log::info "Using user-defined BATS_PREFIX: $prefix"
    else
        log::info "Using default Bats install prefix: $prefix"
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
        git clone "$repo_url" "$dir_name"
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
    if [ ! -d "bats-core" ]; then
        git clone https://github.com/bats-core/bats-core.git
        cd bats-core
        # Ensure the BATS installation prefix exists (use sudo if required for system paths)
        if flow::can_run_sudo && [[ "$BATS_PREFIX" = "/usr/local"* ]]; then
            sudo mkdir -p "$BATS_PREFIX"
        else
            mkdir -p "$BATS_PREFIX"
        fi
        if flow::can_run_sudo; then
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
