#!/bin/bash

# Visitor Intelligence CLI Installation Script
# Creates symlink in ~/.vrooli/bin/ and ensures it's in PATH

set -e

# Configuration
CLI_NAME="visitor-intelligence"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SCRIPT="$SCRIPT_DIR/$CLI_NAME"
VROOLI_BIN_DIR="$HOME/.vrooli/bin"
SYMLINK_PATH="$VROOLI_BIN_DIR/$CLI_NAME"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Utility functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1" >&2
}

# Check if CLI script exists
if [[ ! -f "$CLI_SCRIPT" ]]; then
    log_error "CLI script not found: $CLI_SCRIPT"
    exit 1
fi

# Check if CLI script is executable
if [[ ! -x "$CLI_SCRIPT" ]]; then
    log_info "Making CLI script executable..."
    chmod +x "$CLI_SCRIPT"
fi

# Create ~/.vrooli/bin directory if it doesn't exist
if [[ ! -d "$VROOLI_BIN_DIR" ]]; then
    log_info "Creating directory: $VROOLI_BIN_DIR"
    mkdir -p "$VROOLI_BIN_DIR"
fi

# Remove existing symlink if it exists
if [[ -L "$SYMLINK_PATH" ]] || [[ -f "$SYMLINK_PATH" ]]; then
    log_info "Removing existing CLI installation..."
    rm -f "$SYMLINK_PATH"
fi

# Create symlink
log_info "Installing CLI: $CLI_NAME"
ln -s "$CLI_SCRIPT" "$SYMLINK_PATH"

# Verify symlink was created
if [[ ! -L "$SYMLINK_PATH" ]]; then
    log_error "Failed to create symlink: $SYMLINK_PATH"
    exit 1
fi

log_success "CLI installed successfully: $SYMLINK_PATH"

# Check if ~/.vrooli/bin is in PATH
if [[ ":$PATH:" != *":$VROOLI_BIN_DIR:"* ]]; then
    log_warn "~/.vrooli/bin is not in your PATH"
    log_info "Add the following line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo ""
    echo "  export PATH=\"\$HOME/.vrooli/bin:\$PATH\""
    echo ""
    log_info "Or run the following command for the current session:"
    echo ""
    echo "  export PATH=\"$VROOLI_BIN_DIR:\$PATH\""
    echo ""
    
    # Try to add to common shell profiles
    for profile in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
        if [[ -f "$profile" ]] && ! grep -q "\.vrooli/bin" "$profile"; then
            if [[ -w "$profile" ]]; then
                log_info "Adding PATH to $profile..."
                echo "" >> "$profile"
                echo "# Added by Visitor Intelligence CLI installer" >> "$profile"
                echo "export PATH=\"\$HOME/.vrooli/bin:\$PATH\"" >> "$profile"
                log_success "PATH updated in $profile"
                break
            fi
        fi
    done
else
    log_success "~/.vrooli/bin is already in your PATH"
fi

# Test CLI installation
log_info "Testing CLI installation..."
if command -v "$CLI_NAME" >/dev/null 2>&1; then
    log_success "CLI is accessible in PATH"
    
    # Test basic functionality
    if "$CLI_NAME" version >/dev/null 2>&1; then
        log_success "CLI is functioning correctly"
    else
        log_warn "CLI installed but may have issues (API might not be running)"
    fi
else
    log_warn "CLI not found in PATH - you may need to restart your shell"
fi

echo ""
log_success "Installation complete!"
echo ""
log_info "Usage examples:"
echo "  $CLI_NAME status                              # Check system status"
echo "  $CLI_NAME track pageview my-scenario          # Track an event"
echo "  $CLI_NAME profile visitor-123                 # Get visitor profile"
echo "  $CLI_NAME analytics my-scenario --timeframe 30d  # Get scenario analytics"
echo "  $CLI_NAME help                                # Show help"
echo ""
log_info "To start using the CLI, either restart your shell or run:"
echo "  export PATH=\"$VROOLI_BIN_DIR:\$PATH\""