#!/bin/bash

# Algorithm Library CLI Installation Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="algorithm-library"
INSTALL_DIR="$HOME/.vrooli/bin"

echo "Installing Algorithm Library CLI..."

# Create installation directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Make CLI executable
chmod +x "$SCRIPT_DIR/$CLI_NAME"

# Create symlink
ln -sf "$SCRIPT_DIR/$CLI_NAME" "$INSTALL_DIR/$CLI_NAME"

# Check if ~/.vrooli/bin is in PATH
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo ""
    echo "Note: $INSTALL_DIR is not in your PATH"
    echo "Add the following to your shell configuration file:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo "✓ Algorithm Library CLI installed successfully"
echo "Run 'algorithm-library help' to get started"