#!/bin/bash

# Fall Foliage Explorer CLI Installation Script

set -e

# Get the directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CLI_NAME="foliage-explorer"

# Default installation directory
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

echo "Installing Fall Foliage Explorer CLI..."

# Check if the CLI script exists
if [[ ! -f "$SCRIPT_DIR/$CLI_NAME" ]]; then
    echo "Error: CLI script not found at $SCRIPT_DIR/$CLI_NAME"
    exit 1
fi

# Create the installation directory if it doesn't exist
if [[ ! -d "$INSTALL_DIR" ]]; then
    echo "Creating installation directory: $INSTALL_DIR"
    sudo mkdir -p "$INSTALL_DIR"
fi

# Copy the CLI to the installation directory
echo "Installing $CLI_NAME to $INSTALL_DIR..."
sudo cp "$SCRIPT_DIR/$CLI_NAME" "$INSTALL_DIR/$CLI_NAME"
sudo chmod +x "$INSTALL_DIR/$CLI_NAME"

# Verify installation
if command -v "$CLI_NAME" &> /dev/null; then
    echo "✅ Fall Foliage Explorer CLI installed successfully!"
    echo "   Run '$CLI_NAME --help' to get started"
else
    echo "⚠️  CLI installed but not in PATH. Add $INSTALL_DIR to your PATH or use the full path: $INSTALL_DIR/$CLI_NAME"
fi