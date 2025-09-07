#!/bin/bash

# Referral Program Generator CLI Installation Script
# Creates symlink in ~/.vrooli/bin/ for global access

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SCRIPT="$SCRIPT_DIR/referral-program-generator"
CLI_NAME="referral-program-generator"
INSTALL_DIR="$HOME/.vrooli/bin"

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Logging functions
log_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if CLI script exists
if [[ ! -f "$CLI_SCRIPT" ]]; then
    log_error "CLI script not found: $CLI_SCRIPT"
    exit 1
fi

# Make sure CLI script is executable
chmod +x "$CLI_SCRIPT"

# Create install directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Remove existing symlink if it exists
if [[ -L "$INSTALL_DIR/$CLI_NAME" ]]; then
    rm "$INSTALL_DIR/$CLI_NAME"
    log_info "Removed existing symlink"
fi

# Create new symlink
ln -sf "$CLI_SCRIPT" "$INSTALL_DIR/$CLI_NAME"
log_success "Created symlink: $INSTALL_DIR/$CLI_NAME -> $CLI_SCRIPT"

# Check if ~/.vrooli/bin is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    log_info "Adding $INSTALL_DIR to PATH"
    
    # Detect shell and update appropriate rc file
    if [[ -n "${BASH_VERSION:-}" ]]; then
        RC_FILE="$HOME/.bashrc"
    elif [[ -n "${ZSH_VERSION:-}" ]]; then
        RC_FILE="$HOME/.zshrc"
    else
        RC_FILE="$HOME/.profile"
    fi
    
    # Add to PATH if not already present
    if ! grep -q "\.vrooli/bin" "$RC_FILE" 2>/dev/null; then
        echo "" >> "$RC_FILE"
        echo "# Added by Referral Program Generator CLI installer" >> "$RC_FILE"
        echo 'export PATH="$HOME/.vrooli/bin:$PATH"' >> "$RC_FILE"
        log_success "Added ~/.vrooli/bin to PATH in $RC_FILE"
        log_info "Please run 'source $RC_FILE' or restart your terminal"
    else
        log_info "~/.vrooli/bin already in PATH"
    fi
else
    log_success "~/.vrooli/bin already in PATH"
fi

# Test installation
if command -v "$CLI_NAME" >/dev/null 2>&1; then
    log_success "CLI installed successfully!"
    log_info "Test with: $CLI_NAME --help"
else
    log_info "CLI installed, but not immediately available"
    log_info "Run: export PATH=\"\$HOME/.vrooli/bin:\$PATH\""
    log_info "Or restart your terminal"
fi

echo ""
echo "Referral Program Generator CLI v1.0.0 installed"
echo "Usage: $CLI_NAME <command> [options]"
echo "Help:  $CLI_NAME --help"