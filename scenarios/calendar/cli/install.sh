#!/bin/bash
# Calendar CLI Installation Script
# Creates symlink to calendar CLI in ~/.vrooli/bin/ and ensures it's in PATH

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_BINARY="$SCRIPT_DIR/calendar"
VROOLI_BIN_DIR="$HOME/.vrooli/bin"

echo "Installing Calendar CLI..."

# Ensure .vrooli/bin directory exists
mkdir -p "$VROOLI_BIN_DIR"

# Make binary executable
chmod +x "$CLI_BINARY"

# Create symlink
ln -sf "$CLI_BINARY" "$VROOLI_BIN_DIR/calendar"

echo "Calendar CLI installed to $VROOLI_BIN_DIR/calendar"

# Check if ~/.vrooli/bin is in PATH
if [[ ":$PATH:" != *":$VROOLI_BIN_DIR:"* ]]; then
    echo ""
    echo "⚠️  ~/.vrooli/bin is not in your PATH"
    echo "Add this to your shell profile (.bashrc, .zshrc, etc.):"
    echo ""
    echo "export PATH=\"\$HOME/.vrooli/bin:\$PATH\""
    echo ""
    echo "Then reload your shell or run: source ~/.bashrc"
else
    echo "✅ ~/.vrooli/bin is already in your PATH"
fi

echo ""
echo "Installation complete! Try running:"
echo "  calendar help"
echo "  calendar status"