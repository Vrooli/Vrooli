#!/bin/bash

# Install Morning Vision Walk CLI
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="vision-walk"
INSTALL_DIR="/usr/local/bin"

echo "Installing Morning Vision Walk CLI..."

# Check if running with sufficient permissions
if [ ! -w "$INSTALL_DIR" ]; then
    echo "Installing to user bin directory instead..."
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

# Copy CLI script
cp "$SCRIPT_DIR/$CLI_NAME" "$INSTALL_DIR/$CLI_NAME"
chmod +x "$INSTALL_DIR/$CLI_NAME"

echo "✓ Morning Vision Walk CLI installed to $INSTALL_DIR/$CLI_NAME"
echo ""
echo "Usage: $CLI_NAME help"

# Add to PATH if needed
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "Note: Add $INSTALL_DIR to your PATH if not already present:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi