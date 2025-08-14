#!/usr/bin/env bash
# Install Document Manager CLI globally

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SCRIPT="$SCRIPT_DIR/document-manager"
INSTALL_DIR="/usr/local/bin"
CLI_NAME="document-manager"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

main() {
    log_info "Installing Document Manager CLI..."
    
    # Check if script exists
    if [[ ! -f "$CLI_SCRIPT" ]]; then
        log_error "CLI script not found at $CLI_SCRIPT"
        return 1
    fi
    
    # Check if install directory exists
    if [[ ! -d "$INSTALL_DIR" ]]; then
        log_error "Install directory $INSTALL_DIR does not exist"
        return 1
    fi
    
    # Check write permissions
    if [[ ! -w "$INSTALL_DIR" ]]; then
        log_error "No write permission to $INSTALL_DIR. Try running with sudo:"
        echo "sudo $0"
        return 1
    fi
    
    # Copy CLI script
    cp "$CLI_SCRIPT" "$INSTALL_DIR/$CLI_NAME"
    chmod +x "$INSTALL_DIR/$CLI_NAME"
    
    log_success "Document Manager CLI installed successfully"
    log_info "You can now use: $CLI_NAME --help"
    
    # Test installation
    if command -v "$CLI_NAME" >/dev/null 2>&1; then
        log_success "CLI is available in PATH"
        "$CLI_NAME" version
    else
        log_error "CLI not found in PATH. Check your PATH configuration."
        return 1
    fi
}

main "$@"