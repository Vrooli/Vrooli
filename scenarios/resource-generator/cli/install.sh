#!/bin/bash

# Install Resource Generator CLI command globally

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SCRIPT="$SCRIPT_DIR/resource-generator"
INSTALL_DIR="/usr/local/bin"

# Make CLI script executable
chmod +x "$CLI_SCRIPT"

# Create symlink in /usr/local/bin
if [[ -w "$INSTALL_DIR" ]]; then
    ln -sf "$CLI_SCRIPT" "$INSTALL_DIR/resource-generator"
    echo "✅ Resource Generator CLI installed successfully"
    echo "   You can now use: resource-generator --help"
else
    echo "⚠️  Cannot write to $INSTALL_DIR"
    echo "   Try: sudo $0"
    exit 1
fi