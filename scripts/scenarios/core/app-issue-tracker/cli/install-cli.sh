#!/usr/bin/env bash
# Install App Issue Tracker CLI globally

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SCRIPT="$SCRIPT_DIR/app-issue-tracker.sh"
INSTALL_DIR="${CLI_INSTALL_DIR:-/usr/local/bin}"
CLI_NAME="app-issue-tracker"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

# Check if running as root for system-wide installation
if [ "$INSTALL_DIR" = "/usr/local/bin" ] && [ "$EUID" -ne 0 ]; then
    log_info "Installing to $INSTALL_DIR requires sudo privileges"
    
    # Try to install with sudo
    if command -v sudo &> /dev/null; then
        log_info "Using sudo to install CLI system-wide..."
        sudo cp "$CLI_SCRIPT" "$INSTALL_DIR/$CLI_NAME"
        sudo chmod +x "$INSTALL_DIR/$CLI_NAME"
        log_success "App Issue Tracker CLI installed to $INSTALL_DIR/$CLI_NAME"
    else
        log_error "sudo not available. Please run as root or set CLI_INSTALL_DIR to a writable directory"
        exit 1
    fi
else
    # Create install directory if it doesn't exist
    mkdir -p "$INSTALL_DIR"
    
    # Copy and make executable
    cp "$CLI_SCRIPT" "$INSTALL_DIR/$CLI_NAME"
    chmod +x "$INSTALL_DIR/$CLI_NAME"
    
    log_success "App Issue Tracker CLI installed to $INSTALL_DIR/$CLI_NAME"
fi

# Check if install directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    log_info "Note: $INSTALL_DIR is not in your PATH"
    log_info "Add this line to your ~/.bashrc or ~/.zshrc:"
    echo "export PATH=\"$INSTALL_DIR:\$PATH\""
fi

log_success "Installation complete!"
log_info "Usage: $CLI_NAME --help"