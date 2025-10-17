#!/bin/bash

# Vrooli Bridge CLI Installation Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="vrooli-bridge"
CLI_SCRIPT="$SCRIPT_DIR/$CLI_NAME"

# Ensure ~/.vrooli/bin exists
VROOLI_BIN_DIR="$HOME/.vrooli/bin"
mkdir -p "$VROOLI_BIN_DIR"

# Copy CLI script
cp "$CLI_SCRIPT" "$VROOLI_BIN_DIR/"
chmod +x "$VROOLI_BIN_DIR/$CLI_NAME"

echo "✓ Installed $CLI_NAME to $VROOLI_BIN_DIR"

# Check if ~/.vrooli/bin is in PATH
if [[ ":$PATH:" != *":$VROOLI_BIN_DIR:"* ]]; then
    echo "⚠ Warning: $VROOLI_BIN_DIR is not in your PATH"
    echo ""
    echo "Add this line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo "  export PATH=\"\$PATH:$VROOLI_BIN_DIR\""
    echo ""
    echo "Then restart your shell or run:"
    echo "  source ~/.bashrc  # (or ~/.zshrc)"
    echo ""
fi

echo "✓ Installation complete!"
echo ""
echo "Usage:"
echo "  $CLI_NAME help"
echo "  $CLI_NAME scan"
echo "  $CLI_NAME list"