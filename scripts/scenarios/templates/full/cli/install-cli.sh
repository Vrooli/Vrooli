#!/bin/bash
# Install CLI_NAME_PLACEHOLDER CLI tool globally

set -euo pipefail

readonly CLI_NAME="CLI_NAME_PLACEHOLDER"
readonly INSTALL_DIR="/usr/local/bin"
readonly SOURCE_FILE="cli.sh"

echo "Installing $CLI_NAME CLI..."

# Check if source file exists
if [[ ! -f "$SOURCE_FILE" ]]; then
    echo "Error: $SOURCE_FILE not found in current directory" >&2
    exit 1
fi

# Make executable
chmod +x "$SOURCE_FILE"

# Copy to install directory (may require sudo)
if [[ -w "$INSTALL_DIR" ]]; then
    cp "$SOURCE_FILE" "$INSTALL_DIR/$CLI_NAME"
else
    echo "Note: Installation requires sudo privileges"
    sudo cp "$SOURCE_FILE" "$INSTALL_DIR/$CLI_NAME"
fi

# Verify installation
if command -v "$CLI_NAME" &> /dev/null; then
    echo "✓ $CLI_NAME installed successfully to $INSTALL_DIR/$CLI_NAME"
    echo "  Run '$CLI_NAME help' to get started"
else
    echo "⚠ Installation completed but $CLI_NAME not found in PATH"
    echo "  You may need to add $INSTALL_DIR to your PATH"
fi