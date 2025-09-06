#!/bin/bash

# Browser Automation Studio CLI Installation Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${HOME}/.vrooli/bin"
CLI_NAME="browser-automation-studio"

echo "Installing Browser Automation Studio CLI..."

# Create installation directory
mkdir -p "$INSTALL_DIR"

# Create symlink
ln -sf "$SCRIPT_DIR/$CLI_NAME" "$INSTALL_DIR/$CLI_NAME"

# Add to PATH if not already there
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo "" >> ~/.bashrc
    echo "# Browser Automation Studio CLI" >> ~/.bashrc
    echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> ~/.bashrc
    echo ""
    echo "Added $INSTALL_DIR to PATH in ~/.bashrc"
    echo "Please run: source ~/.bashrc"
fi

echo "✓ Browser Automation Studio CLI installed successfully!"
echo ""
echo "Usage: browser-automation-studio help"