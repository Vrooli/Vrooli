#!/bin/bash

# Install System Monitor CLI
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_BINARY="$SCRIPT_DIR/system-monitor"
INSTALL_DIR="/usr/local/bin"

echo "Installing System Monitor CLI..."

# Make the CLI executable
chmod +x "$CLI_BINARY"

# Create symlink in /usr/local/bin (requires sudo)
if [[ $EUID -eq 0 ]]; then
    # Running as root
    ln -sf "$CLI_BINARY" "$INSTALL_DIR/system-monitor"
    echo "System Monitor CLI installed to $INSTALL_DIR/system-monitor"
else
    # Not running as root, try with sudo
    if command -v sudo >/dev/null 2>&1; then
        sudo ln -sf "$CLI_BINARY" "$INSTALL_DIR/system-monitor"
        echo "System Monitor CLI installed to $INSTALL_DIR/system-monitor"
    else
        echo "Warning: Could not install to $INSTALL_DIR (no sudo available)"
        echo "CLI is available at: $CLI_BINARY"
        echo "Add this to your PATH or create an alias:"
        echo "  alias system-monitor='$CLI_BINARY'"
    fi
fi

echo "Installation complete!"
echo "Usage: system-monitor --help"