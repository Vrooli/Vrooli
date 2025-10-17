#!/bin/bash

# Install script for Funnel Builder CLI

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${HOME}/.vrooli/bin"
CLI_NAME="funnel-builder"

echo "Installing Funnel Builder CLI..."

# Create install directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Create symlink
ln -sf "${SCRIPT_DIR}/${CLI_NAME}" "${INSTALL_DIR}/${CLI_NAME}"

# Add to PATH if not already there
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo ""
    echo "Adding $INSTALL_DIR to PATH..."
    
    # Detect shell and add to appropriate config file
    if [ -n "$ZSH_VERSION" ]; then
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> ~/.zshrc
        echo "Added to ~/.zshrc"
    elif [ -n "$BASH_VERSION" ]; then
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> ~/.bashrc
        echo "Added to ~/.bashrc"
    else
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> ~/.profile
        echo "Added to ~/.profile"
    fi
    
    echo ""
    echo "Please restart your terminal or run:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo ""
echo "✓ Funnel Builder CLI installed successfully!"
echo ""
echo "You can now use: funnel-builder --help"