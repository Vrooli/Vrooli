#!/bin/bash

# Install script for personal-digital-twin CLI

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="personal-digital-twin"
INSTALL_DIR="/usr/local/bin"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

main() {
    log_info "Installing Personal Digital Twin CLI..."
    
    # Check if running as root or with sudo
    if [[ $EUID -eq 0 ]]; then
        SUDO_CMD=""
    else
        SUDO_CMD="sudo"
        log_info "Requesting sudo privileges for installation..."
    fi
    
    # Check if CLI script exists
    if [[ ! -f "$SCRIPT_DIR/$CLI_NAME" ]]; then
        log_error "CLI script not found at $SCRIPT_DIR/$CLI_NAME"
        exit 1
    fi
    
    # Make sure script is executable
    chmod +x "$SCRIPT_DIR/$CLI_NAME"
    
    # Copy to install directory
    if $SUDO_CMD cp "$SCRIPT_DIR/$CLI_NAME" "$INSTALL_DIR/$CLI_NAME"; then
        log_success "CLI installed to $INSTALL_DIR/$CLI_NAME"
    else
        log_error "Failed to install CLI to $INSTALL_DIR"
        log_info "You may need to run this script with sudo"
        exit 1
    fi
    
    # Verify installation
    if command -v "$CLI_NAME" &> /dev/null; then
        log_success "Installation verified: $CLI_NAME is now available globally"
        log_info "Usage: $CLI_NAME --help"
    else
        log_error "Installation verification failed"
        log_info "Make sure $INSTALL_DIR is in your PATH"
        exit 1
    fi
}

main "$@"