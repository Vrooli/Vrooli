#!/bin/bash

# Text Tools CLI Installation Script

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="text-tools"
INSTALL_DIR="${HOME}/.vrooli/bin"

echo "Installing Text Tools CLI..."

# Create installation directory
mkdir -p "$INSTALL_DIR"

# Create symlink
ln -sf "$SCRIPT_DIR/$CLI_NAME" "$INSTALL_DIR/$CLI_NAME"

# Add to PATH if not already there
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "Adding $INSTALL_DIR to PATH..."
    
    # Add to appropriate shell config
    if [[ -f "$HOME/.bashrc" ]]; then
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$HOME/.bashrc"
        echo "Added to ~/.bashrc"
    fi
    
    if [[ -f "$HOME/.zshrc" ]]; then
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$HOME/.zshrc"
        echo "Added to ~/.zshrc"
    fi
    
    echo "Please run 'source ~/.bashrc' or 'source ~/.zshrc' to update your PATH"
fi

echo "Text Tools CLI installed successfully!"
echo "Run 'text-tools help' to get started"