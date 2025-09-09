#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_NAME="image-tools"
INSTALL_DIR="${HOME}/.vrooli/bin"

echo "Installing Image Tools CLI..."

# Create install directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Create symlink
ln -sf "$SCRIPT_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"

# Check if install directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "Adding $INSTALL_DIR to PATH..."
    
    # Determine shell config file
    if [[ -n "${ZSH_VERSION:-}" ]]; then
        SHELL_CONFIG="$HOME/.zshrc"
    elif [[ -n "${BASH_VERSION:-}" ]]; then
        SHELL_CONFIG="$HOME/.bashrc"
    else
        SHELL_CONFIG="$HOME/.profile"
    fi
    
    # Add to PATH if not already there
    if ! grep -q "export PATH.*\.vrooli/bin" "$SHELL_CONFIG" 2>/dev/null; then
        echo "" >> "$SHELL_CONFIG"
        echo "# Added by Image Tools CLI installer" >> "$SHELL_CONFIG"
        echo "export PATH=\"\$HOME/.vrooli/bin:\$PATH\"" >> "$SHELL_CONFIG"
        echo "✓ Added to $SHELL_CONFIG"
        echo ""
        echo "Please run: source $SHELL_CONFIG"
        echo "Or restart your terminal for PATH changes to take effect"
    fi
fi

echo "✓ Image Tools CLI installed successfully!"
echo ""
echo "Usage: image-tools help"