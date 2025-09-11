#!/bin/bash
# Install time-tools CLI command globally

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="time-tools"
CLI_PATH="$SCRIPT_DIR/$CLI_NAME"
INSTALL_DIR="$HOME/.local/bin"

# Create install directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Create or update symlink
if [[ -L "$INSTALL_DIR/$CLI_NAME" ]]; then
    rm "$INSTALL_DIR/$CLI_NAME"
fi

ln -s "$CLI_PATH" "$INSTALL_DIR/$CLI_NAME"

# Check if install directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "Warning: $INSTALL_DIR is not in your PATH"
    echo "Add the following to your shell configuration:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo "✓ $CLI_NAME CLI installed successfully"
echo "  Location: $INSTALL_DIR/$CLI_NAME"
echo "  Run 'time-tools help' to get started"