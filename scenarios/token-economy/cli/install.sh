#!/bin/bash

# Token Economy CLI Installation Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="token-economy"
INSTALL_DIR="$HOME/.vrooli/bin"

echo "Installing Token Economy CLI..."

# Create installation directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Create symlink
ln -sf "$SCRIPT_DIR/$CLI_NAME" "$INSTALL_DIR/$CLI_NAME"

# Check if ~/.vrooli/bin is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "⚠️  $INSTALL_DIR is not in your PATH"
    echo ""
    echo "Add the following line to your shell configuration file:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    echo ""
    
    # Try to detect shell and provide specific instructions
    if [ -n "$BASH_VERSION" ]; then
        echo "For bash, add to ~/.bashrc or ~/.bash_profile"
    elif [ -n "$ZSH_VERSION" ]; then
        echo "For zsh, add to ~/.zshrc"
    fi
fi

echo "✓ Token Economy CLI installed successfully!"
echo ""
echo "Usage: token-economy help"