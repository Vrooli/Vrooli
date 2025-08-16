#!/bin/bash

# QR Code Generator CLI Installation Script

set -e

CLI_NAME="qr-generator"
CLI_PATH="$(dirname "$0")/$CLI_NAME"
INSTALL_DIR="/usr/local/bin"

echo "Installing QR Code Generator CLI..."

# Check if the CLI script exists
if [ ! -f "$CLI_PATH" ]; then
    echo "Error: CLI script not found at $CLI_PATH"
    exit 1
fi

# Make the CLI script executable
chmod +x "$CLI_PATH"

# Check if we have permission to install to /usr/local/bin
if [ -w "$INSTALL_DIR" ]; then
    # Direct installation
    cp "$CLI_PATH" "$INSTALL_DIR/$CLI_NAME"
    echo "✓ QR Code Generator CLI installed to $INSTALL_DIR/$CLI_NAME"
else
    # Try with sudo
    echo "Need sudo permission to install to $INSTALL_DIR"
    sudo cp "$CLI_PATH" "$INSTALL_DIR/$CLI_NAME"
    sudo chmod +x "$INSTALL_DIR/$CLI_NAME"
    echo "✓ QR Code Generator CLI installed to $INSTALL_DIR/$CLI_NAME"
fi

# Verify installation
if command -v "$CLI_NAME" &> /dev/null; then
    echo "✓ Installation successful! You can now use '$CLI_NAME' command."
    echo ""
    echo "Usage examples:"
    echo "  $CLI_NAME generate \"Hello World\""
    echo "  $CLI_NAME batch file.txt"
    echo "  $CLI_NAME --help"
else
    echo "⚠ Warning: Installation completed but '$CLI_NAME' command not found in PATH"
    echo "You may need to add $INSTALL_DIR to your PATH or restart your terminal"
fi