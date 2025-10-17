#!/bin/bash

# Data Backup Manager CLI Installation Script
# Installs the CLI to ~/.vrooli/bin/ and adds to PATH if needed

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SOURCE="${SCRIPT_DIR}/data-backup-manager"

# Installation directories
VROOLI_BIN_DIR="${HOME}/.vrooli/bin"
LOCAL_BIN_DIR="${HOME}/.local/bin"

log_info "Installing Data Backup Manager CLI..."

# Check if source CLI exists
if [[ ! -f "$CLI_SOURCE" ]]; then
    log_error "CLI source file not found: $CLI_SOURCE"
    exit 1
fi

# Check if CLI is executable
if [[ ! -x "$CLI_SOURCE" ]]; then
    log_info "Making CLI executable..."
    chmod +x "$CLI_SOURCE"
fi

# Create ~/.vrooli/bin directory if it doesn't exist
if [[ ! -d "$VROOLI_BIN_DIR" ]]; then
    log_info "Creating directory: $VROOLI_BIN_DIR"
    mkdir -p "$VROOLI_BIN_DIR"
fi

# Create ~/.local/bin directory if it doesn't exist (fallback)
if [[ ! -d "$LOCAL_BIN_DIR" ]]; then
    log_info "Creating directory: $LOCAL_BIN_DIR"
    mkdir -p "$LOCAL_BIN_DIR"
fi

# Install to ~/.vrooli/bin (primary location)
VROOLI_TARGET="${VROOLI_BIN_DIR}/data-backup-manager"
log_info "Installing CLI to: $VROOLI_TARGET"

if [[ -L "$VROOLI_TARGET" ]]; then
    log_info "Removing existing symlink: $VROOLI_TARGET"
    rm "$VROOLI_TARGET"
elif [[ -f "$VROOLI_TARGET" ]]; then
    log_info "Backing up existing file: $VROOLI_TARGET"
    mv "$VROOLI_TARGET" "${VROOLI_TARGET}.backup.$(date +%s)"
fi

ln -sf "$CLI_SOURCE" "$VROOLI_TARGET"
log_success "CLI installed to $VROOLI_TARGET"

# Install to ~/.local/bin (fallback location)
LOCAL_TARGET="${LOCAL_BIN_DIR}/data-backup-manager"
log_info "Installing CLI to: $LOCAL_TARGET (fallback)"

if [[ -L "$LOCAL_TARGET" ]]; then
    log_info "Removing existing symlink: $LOCAL_TARGET"
    rm "$LOCAL_TARGET"
elif [[ -f "$LOCAL_TARGET" ]]; then
    log_info "Backing up existing file: $LOCAL_TARGET"
    mv "$LOCAL_TARGET" "${LOCAL_TARGET}.backup.$(date +%s)"
fi

ln -sf "$CLI_SOURCE" "$LOCAL_TARGET"
log_success "CLI installed to $LOCAL_TARGET"

# Check if ~/.vrooli/bin is in PATH
if [[ ":$PATH:" != *":$VROOLI_BIN_DIR:"* ]]; then
    log_warning "~/.vrooli/bin is not in your PATH"
    
    # Determine which shell config file to update
    SHELL_CONFIG=""
    if [[ -n "$ZSH_VERSION" ]]; then
        SHELL_CONFIG="$HOME/.zshrc"
    elif [[ -n "$BASH_VERSION" ]]; then
        if [[ -f "$HOME/.bashrc" ]]; then
            SHELL_CONFIG="$HOME/.bashrc"
        elif [[ -f "$HOME/.bash_profile" ]]; then
            SHELL_CONFIG="$HOME/.bash_profile"
        fi
    fi

    if [[ -n "$SHELL_CONFIG" ]]; then
        log_info "Adding ~/.vrooli/bin to PATH in $SHELL_CONFIG"
        
        # Add PATH export if not already present
        if ! grep -q "export PATH.*\.vrooli/bin" "$SHELL_CONFIG" 2>/dev/null; then
            echo "" >> "$SHELL_CONFIG"
            echo "# Added by Vrooli Data Backup Manager installation" >> "$SHELL_CONFIG"
            echo 'export PATH="$HOME/.vrooli/bin:$PATH"' >> "$SHELL_CONFIG"
            log_success "Added ~/.vrooli/bin to PATH in $SHELL_CONFIG"
            log_info "Please run 'source $SHELL_CONFIG' or restart your terminal"
        else
            log_info "PATH already configured in $SHELL_CONFIG"
        fi
    else
        log_warning "Could not determine shell config file"
        log_info "Please manually add ~/.vrooli/bin to your PATH:"
        log_info "  export PATH=\"\$HOME/.vrooli/bin:\$PATH\""
    fi
fi

# Check if ~/.local/bin is in PATH (fallback)
if [[ ":$PATH:" != *":$LOCAL_BIN_DIR:"* ]]; then
    log_warning "~/.local/bin is not in your PATH"
    log_info "Many systems include ~/.local/bin in PATH by default"
    log_info "If the CLI is not found, add ~/.local/bin to your PATH:"
    log_info "  export PATH=\"\$HOME/.local/bin:\$PATH\""
fi

# Verify installation
log_info "Verifying installation..."

if command -v data-backup-manager &> /dev/null; then
    log_success "CLI installation verified - command found in PATH"
    
    # Test CLI functionality
    if data-backup-manager version &> /dev/null; then
        log_success "CLI functionality verified"
        log_info "Installation complete!"
        echo ""
        log_info "Usage: data-backup-manager --help"
        log_info "Quick start: data-backup-manager status"
    else
        log_warning "CLI installed but may have functionality issues"
        log_info "Try: data-backup-manager --help"
    fi
else
    log_warning "CLI not found in PATH"
    log_info "You can run it directly from:"
    log_info "  $VROOLI_TARGET"
    log_info "  $LOCAL_TARGET"
    log_info ""
    log_info "Or add one of these directories to your PATH:"
    log_info "  ~/.vrooli/bin"
    log_info "  ~/.local/bin"
fi

echo ""
log_info "Data Backup Manager CLI installation completed"
log_info "Configuration directory: ~/.vrooli/data-backup-manager/"
log_info "For help: data-backup-manager help"