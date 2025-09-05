#!/bin/bash

# Visited Tracker CLI Installation Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="visited-tracker"
VROOLI_BIN_DIR="${HOME}/.vrooli/bin"

# Ensure .vrooli/bin directory exists
mkdir -p "$VROOLI_BIN_DIR"

# Create symlink to CLI
CLI_SOURCE="${SCRIPT_DIR}/${CLI_NAME}"
CLI_TARGET="${VROOLI_BIN_DIR}/${CLI_NAME}"

if [[ -L "$CLI_TARGET" ]]; then
    echo "Removing existing symlink..."
    rm "$CLI_TARGET"
fi

echo "Installing ${CLI_NAME} CLI..."
ln -s "$CLI_SOURCE" "$CLI_TARGET"

# Make sure CLI is executable
chmod +x "$CLI_SOURCE"

# Check if .vrooli/bin is in PATH
if [[ ":$PATH:" != *":$VROOLI_BIN_DIR:"* ]]; then
    echo
    echo "⚠️  ${VROOLI_BIN_DIR} is not in your PATH"
    echo "Add this line to your shell profile (.bashrc, .zshrc, etc.):"
    echo "export PATH=\"\$PATH:$VROOLI_BIN_DIR\""
    echo
fi

echo "✅ ${CLI_NAME} CLI installed successfully"
echo "Usage: ${CLI_NAME} --help"