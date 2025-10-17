#!/bin/bash

# Data Structurer CLI Installation Script
# Installs the data-structurer command globally

set -euo pipefail

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
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Configuration
VROOLI_BIN_DIR="$HOME/.vrooli/bin"
CLI_SOURCE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/data-structurer"
CLI_TARGET="$VROOLI_BIN_DIR/data-structurer"

# Create Vrooli bin directory if it doesn't exist
mkdir -p "$VROOLI_BIN_DIR"

# Check if source CLI exists
if [[ ! -f "$CLI_SOURCE" ]]; then
    log_error "CLI source file not found: $CLI_SOURCE"
    exit 1
fi

# Check if CLI is executable
if [[ ! -x "$CLI_SOURCE" ]]; then
    log_warning "Making CLI executable"
    chmod +x "$CLI_SOURCE"
fi

# Copy CLI to target location
log_info "Installing data-structurer CLI..."
cp "$CLI_SOURCE" "$CLI_TARGET"
chmod +x "$CLI_TARGET"

# Check if ~/.vrooli/bin is in PATH
if [[ ":$PATH:" != *":$VROOLI_BIN_DIR:"* ]]; then
    log_warning "~/.vrooli/bin is not in PATH"
    
    # Determine shell configuration file
    SHELL_CONFIG=""
    if [[ -n "${ZSH_VERSION:-}" ]]; then
        SHELL_CONFIG="$HOME/.zshrc"
    elif [[ -n "${BASH_VERSION:-}" ]]; then
        if [[ -f "$HOME/.bash_profile" ]]; then
            SHELL_CONFIG="$HOME/.bash_profile"
        else
            SHELL_CONFIG="$HOME/.bashrc"
        fi
    fi
    
    if [[ -n "$SHELL_CONFIG" ]]; then
        log_info "Adding ~/.vrooli/bin to PATH in $SHELL_CONFIG"
        
        # Add PATH export to shell config if not already present
        if ! grep -q "/.vrooli/bin" "$SHELL_CONFIG" 2>/dev/null; then
            echo "" >> "$SHELL_CONFIG"
            echo "# Add Vrooli CLI tools to PATH" >> "$SHELL_CONFIG"
            echo 'export PATH="$HOME/.vrooli/bin:$PATH"' >> "$SHELL_CONFIG"
            
            log_success "Added ~/.vrooli/bin to PATH in $SHELL_CONFIG"
            log_info "Please run: source $SHELL_CONFIG"
            log_info "Or start a new terminal session to use the data-structurer command"
        else
            log_info "~/.vrooli/bin already configured in $SHELL_CONFIG"
        fi
    else
        log_warning "Could not determine shell configuration file"
        log_info "Please manually add this line to your shell configuration:"
        log_info 'export PATH="$HOME/.vrooli/bin:$PATH"'
    fi
else
    log_success "~/.vrooli/bin is already in PATH"
fi

# Test CLI installation
log_info "Testing CLI installation..."
if "$CLI_TARGET" version --json > /dev/null 2>&1; then
    log_success "CLI installed successfully!"
else
    log_warning "CLI installed but test failed - check API connectivity"
fi

# Show installation summary
echo
log_success "Data Structurer CLI Installation Complete"
echo
echo "Command: data-structurer"
echo "Location: $CLI_TARGET"
echo
echo "Usage examples:"
echo "  data-structurer status"
echo "  data-structurer help"
echo "  data-structurer list-schemas"
echo "  data-structurer create-schema 'my-schema' ./schema.json"
echo
if [[ ":$PATH:" != *":$VROOLI_BIN_DIR:"* ]]; then
    echo "Note: You may need to restart your terminal or run 'source ~/.bashrc' to use the command."
fi