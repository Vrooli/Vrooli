#!/bin/bash

# Installation script for Prompt Injection Arena CLI
# Creates symlink in Vrooli bin directory

set -e

CLI_NAME="prompt-injection-arena"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_PATH="$SCRIPT_DIR/$CLI_NAME"

# Determine bin directory
VROOLI_BIN_DIR="${HOME}/.vrooli/bin"

# Create bin directory if it doesn't exist
mkdir -p "$VROOLI_BIN_DIR"

# Remove existing symlink if it exists
if [[ -L "$VROOLI_BIN_DIR/$CLI_NAME" ]]; then
    rm "$VROOLI_BIN_DIR/$CLI_NAME"
fi

# Create symlink
ln -s "$CLI_PATH" "$VROOLI_BIN_DIR/$CLI_NAME"

echo "✅ Prompt Injection Arena CLI installed successfully"
echo "📍 Location: $VROOLI_BIN_DIR/$CLI_NAME"

# Check if Vrooli bin is in PATH
if [[ ":$PATH:" != *":$VROOLI_BIN_DIR:"* ]]; then
    echo ""
    echo "⚠️  Add Vrooli bin directory to your PATH:"
    echo "   export PATH=\"\$PATH:$VROOLI_BIN_DIR\""
    echo ""
    echo "   Or add this line to your shell configuration file:"
    echo "   ~/.bashrc, ~/.zshrc, or ~/.profile"
else
    echo "✅ Vrooli bin directory is already in PATH"
fi

echo ""
echo "🚀 Test the installation:"
echo "   $CLI_NAME --help"
echo "   $CLI_NAME status"