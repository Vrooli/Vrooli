#!/bin/bash

# Scenario Dependency Analyzer CLI Installation Script
# Installs the CLI tool globally for system-wide access

set -euo pipefail

CLI_NAME="scenario-dependency-analyzer"
INSTALL_DIR="$HOME/.vrooli/bin"
CLI_SOURCE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$CLI_NAME"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# Check if source CLI exists
if [[ ! -f "$CLI_SOURCE" ]]; then
    log_error "CLI source file not found: $CLI_SOURCE"
    exit 1
fi

# Create install directory if it doesn't exist
if [[ ! -d "$INSTALL_DIR" ]]; then
    log_info "Creating install directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
fi

# Copy CLI to install directory
log_info "Installing $CLI_NAME to $INSTALL_DIR"
cp "$CLI_SOURCE" "$INSTALL_DIR/$CLI_NAME"
chmod +x "$INSTALL_DIR/$CLI_NAME"

# Check if install directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    log_warning "$INSTALL_DIR is not in PATH"
    log_info "Adding $INSTALL_DIR to PATH in shell configuration"
    
    # Determine shell configuration file
    if [[ -n "${ZSH_VERSION:-}" ]]; then
        SHELL_CONFIG="$HOME/.zshrc"
    elif [[ -n "${BASH_VERSION:-}" ]]; then
        if [[ -f "$HOME/.bashrc" ]]; then
            SHELL_CONFIG="$HOME/.bashrc"
        else
            SHELL_CONFIG="$HOME/.bash_profile"
        fi
    else
        SHELL_CONFIG="$HOME/.profile"
    fi
    
    # Add PATH export to shell config if not already present
    if [[ -f "$SHELL_CONFIG" ]] && ! grep -q "$INSTALL_DIR" "$SHELL_CONFIG"; then
        echo "" >> "$SHELL_CONFIG"
        echo "# Added by Vrooli scenario-dependency-analyzer installer" >> "$SHELL_CONFIG"
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_CONFIG"
        log_info "Added PATH export to $SHELL_CONFIG"
        log_warning "Please restart your shell or run: source $SHELL_CONFIG"
    elif [[ ! -f "$SHELL_CONFIG" ]]; then
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" > "$SHELL_CONFIG"
        log_info "Created $SHELL_CONFIG with PATH export"
        log_warning "Please restart your shell or run: source $SHELL_CONFIG"
    else
        log_info "PATH already configured in $SHELL_CONFIG"
    fi
else
    log_success "$INSTALL_DIR is already in PATH"
fi

# Test installation
if command -v "$CLI_NAME" >/dev/null 2>&1; then
    log_success "$CLI_NAME installed successfully!"
    
    # Show version
    "$CLI_NAME" version
    
    echo
    log_info "Quick start commands:"
    echo "  $CLI_NAME status              # Check system status"
    echo "  $CLI_NAME analyze all         # Analyze all scenarios"
    echo "  $CLI_NAME graph combined      # Generate dependency graph"
    echo "  $CLI_NAME help                # Show full help"
    
elif [[ -x "$INSTALL_DIR/$CLI_NAME" ]]; then
    log_success "$CLI_NAME installed to $INSTALL_DIR"
    log_warning "CLI may not be immediately available in current shell"
    log_info "Try running: $INSTALL_DIR/$CLI_NAME version"
    log_info "Or restart your shell to update PATH"
else
    log_error "Installation failed - CLI not executable"
    exit 1
fi

echo
log_info "For detailed usage information, run: $CLI_NAME help"