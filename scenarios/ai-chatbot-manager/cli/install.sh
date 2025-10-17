#!/bin/bash

# AI Chatbot Manager CLI Installation Script
# Creates global CLI access for the ai-chatbot-manager command

set -e

SCRIPT_NAME="ai-chatbot-manager"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_PATH="${SCRIPT_DIR}/${SCRIPT_NAME}"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if script exists
if [[ ! -f "$SCRIPT_PATH" ]]; then
    log_error "CLI script not found at: $SCRIPT_PATH"
    exit 1
fi

# Make script executable
chmod +x "$SCRIPT_PATH"

# Create ~/.vrooli/bin directory if it doesn't exist
VROOLI_BIN_DIR="$HOME/.vrooli/bin"
mkdir -p "$VROOLI_BIN_DIR"

# Create symlink
SYMLINK_PATH="${VROOLI_BIN_DIR}/${SCRIPT_NAME}"

if [[ -L "$SYMLINK_PATH" ]] || [[ -f "$SYMLINK_PATH" ]]; then
    log_info "Removing existing CLI installation..."
    rm -f "$SYMLINK_PATH"
fi

ln -s "$SCRIPT_PATH" "$SYMLINK_PATH"
log_info "CLI installed to: $SYMLINK_PATH"

# Check if ~/.vrooli/bin is in PATH
if [[ ":$PATH:" != *":$VROOLI_BIN_DIR:"* ]]; then
    log_warning "~/.vrooli/bin is not in your PATH"
    log_info "To use the CLI from anywhere, add this to your shell profile:"
    echo ""
    echo "    export PATH=\"\$HOME/.vrooli/bin:\$PATH\""
    echo ""
    
    # Try to add to common shell profiles
    SHELL_PROFILES=(
        "$HOME/.bashrc"
        "$HOME/.zshrc" 
        "$HOME/.bash_profile"
        "$HOME/.profile"
    )
    
    PROFILE_UPDATED=false
    for profile in "${SHELL_PROFILES[@]}"; do
        if [[ -f "$profile" ]] && ! grep -q "/.vrooli/bin" "$profile"; then
            log_info "Adding PATH export to: $profile"
            echo "" >> "$profile"
            echo "# Added by AI Chatbot Manager CLI installer" >> "$profile"
            echo "export PATH=\"\$HOME/.vrooli/bin:\$PATH\"" >> "$profile"
            PROFILE_UPDATED=true
            break
        fi
    done
    
    if [[ "$PROFILE_UPDATED" == true ]]; then
        log_info "Please restart your shell or run: source $profile"
    else
        log_warning "Could not automatically update PATH. Please add manually."
    fi
else
    log_info "PATH is already configured correctly"
fi

# Test CLI installation
log_info "Testing CLI installation..."
if "$SYMLINK_PATH" version >/dev/null 2>&1; then
    log_info "✅ CLI installation successful!"
    log_info "You can now use: $SCRIPT_NAME --help"
else
    log_error "CLI installation test failed"
    exit 1
fi

# Show usage information
echo ""
echo "🚀 AI Chatbot Manager CLI installed successfully!"
echo ""
echo "Quick start:"
echo "  $SCRIPT_NAME status           # Check service status"
echo "  $SCRIPT_NAME create \"My Bot\" # Create a new chatbot"
echo "  $SCRIPT_NAME list             # List all chatbots"
echo "  $SCRIPT_NAME help             # Show all commands"
echo ""
echo "Make sure the API server is running:"
echo "  vrooli scenario run ai-chatbot-manager"