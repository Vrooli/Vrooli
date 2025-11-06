#!/usr/bin/env bash
# Install web-console CLI to system PATH

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="web-console"
CLI_SOURCE="$SCRIPT_DIR/$CLI_NAME"

# Determine installation directory
if [ -n "${VROOLI_CLI_DIR:-}" ]; then
    INSTALL_DIR="$VROOLI_CLI_DIR"
elif [ -d "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
elif [ -d "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
else
    echo "Error: No suitable installation directory found"
    exit 1
fi

# Create installation directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Install CLI
echo "Installing $CLI_NAME to $INSTALL_DIR..."
cp "$CLI_SOURCE" "$INSTALL_DIR/$CLI_NAME"
chmod +x "$INSTALL_DIR/$CLI_NAME"

# Verify installation
if command -v "$CLI_NAME" > /dev/null 2>&1; then
    echo "✓ $CLI_NAME installed successfully"
    echo "  Location: $(command -v $CLI_NAME)"
else
    echo "⚠ $CLI_NAME installed but not found in PATH"
    echo "  Add $INSTALL_DIR to your PATH:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
fi
