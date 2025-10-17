#!/usr/bin/env bash

# Comment System CLI Installation Script
# Installs the comment-system command globally

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SCRIPT="$SCRIPT_DIR/comment-system"
VROOLI_BIN_DIR="$HOME/.vrooli/bin"
TARGET_PATH="$VROOLI_BIN_DIR/comment-system"

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

# Check if CLI script exists
if [[ ! -f "$CLI_SCRIPT" ]]; then
    log_error "CLI script not found at: $CLI_SCRIPT"
    exit 1
fi

# Create Vrooli bin directory if it doesn't exist
if [[ ! -d "$VROOLI_BIN_DIR" ]]; then
    log_info "Creating Vrooli bin directory: $VROOLI_BIN_DIR"
    mkdir -p "$VROOLI_BIN_DIR"
fi

# Remove existing symlink/file if it exists
if [[ -L "$TARGET_PATH" ]] || [[ -f "$TARGET_PATH" ]]; then
    log_info "Removing existing comment-system command"
    rm -f "$TARGET_PATH"
fi

# Create symlink
log_info "Creating symlink: $TARGET_PATH -> $CLI_SCRIPT"
ln -s "$CLI_SCRIPT" "$TARGET_PATH"

# Make CLI script executable
chmod +x "$CLI_SCRIPT"

# Check if Vrooli bin directory is in PATH
if [[ ":$PATH:" != *":$VROOLI_BIN_DIR:"* ]]; then
    log_info "Adding $VROOLI_BIN_DIR to PATH"
    
    # Determine shell configuration file
    SHELL_CONFIG=""
    if [[ -n "$BASH_VERSION" ]]; then
        if [[ -f "$HOME/.bashrc" ]]; then
            SHELL_CONFIG="$HOME/.bashrc"
        elif [[ -f "$HOME/.bash_profile" ]]; then
            SHELL_CONFIG="$HOME/.bash_profile"
        fi
    elif [[ -n "$ZSH_VERSION" ]]; then
        SHELL_CONFIG="$HOME/.zshrc"
    elif [[ -f "$HOME/.profile" ]]; then
        SHELL_CONFIG="$HOME/.profile"
    fi

    if [[ -n "$SHELL_CONFIG" ]]; then
        # Add PATH export to shell config
        echo "" >> "$SHELL_CONFIG"
        echo "# Added by Vrooli Comment System CLI installer" >> "$SHELL_CONFIG"
        echo "export PATH=\"\$PATH:$VROOLI_BIN_DIR\"" >> "$SHELL_CONFIG"
        
        log_info "Added PATH export to $SHELL_CONFIG"
        log_info "Please run 'source $SHELL_CONFIG' or restart your terminal"
    else
        log_error "Could not determine shell configuration file"
        log_info "Please manually add $VROOLI_BIN_DIR to your PATH"
    fi
    
    # Temporarily add to current session PATH
    export PATH="$PATH:$VROOLI_BIN_DIR"
fi

# Test installation
log_info "Testing installation..."
if command -v comment-system >/dev/null 2>&1; then
    log_success "Comment System CLI installed successfully!"
    echo
    comment-system version
    echo
    log_info "Try: comment-system help"
else
    log_error "Installation verification failed"
    log_info "Please ensure $VROOLI_BIN_DIR is in your PATH and try again"
    exit 1
fi

# Create default configuration
CONFIG_DIR="$HOME/.vrooli/comment-system"
CONFIG_FILE="$CONFIG_DIR/config.yaml"

if [[ ! -f "$CONFIG_FILE" ]]; then
    log_info "Creating default configuration at $CONFIG_FILE"
    mkdir -p "$CONFIG_DIR"
    
    cat > "$CONFIG_FILE" << 'EOF'
# Comment System CLI Configuration
# This file contains default settings for the comment-system CLI

# API endpoint for the Comment System service
api_url: http://localhost:8080

# Default output format (json, table, tree)
default_format: table

# Default page size for listing comments
default_page_size: 20

# Authentication settings
auth:
  # Session token (set programmatically)
  token: ""
  
  # Auto-refresh token (future feature)
  auto_refresh: false

# Display settings
display:
  # Show colors in terminal output
  colors: true
  
  # Show timestamps in human-readable format
  human_dates: true
  
  # Maximum content length to display in table format
  max_content_length: 50

# Moderation settings (for admin users)
moderation:
  # Require confirmation for destructive actions
  confirm_delete: true
  
  # Show moderated content by default
  show_moderated: false

EOF
    
    log_success "Created default configuration file"
fi

log_success "Comment System CLI installation complete!"
echo
echo "Next steps:"
echo "1. Start the Comment System API server"
echo "2. Run 'comment-system status' to check connectivity"  
echo "3. Run 'comment-system help' to see all available commands"