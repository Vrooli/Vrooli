#\!/bin/bash
# Install Smart File Photo Manager CLI globally

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SCRIPT="$SCRIPT_DIR/smart-file-photo-manager"
INSTALL_PATH="/usr/local/bin/smart-file-photo-manager"

# Check if running as root or with sudo
if [ "$EUID" -eq 0 ]; then
    echo "Installing Smart File Photo Manager CLI to $INSTALL_PATH..."
    cp "$CLI_SCRIPT" "$INSTALL_PATH"
    chmod +x "$INSTALL_PATH"
    echo "✅ Smart File Photo Manager CLI installed successfully\!"
    echo "You can now run: smart-file-photo-manager --help"
else
    echo "Installing Smart File Photo Manager CLI to $INSTALL_PATH (requires sudo)..."
    sudo cp "$CLI_SCRIPT" "$INSTALL_PATH"
    sudo chmod +x "$INSTALL_PATH"
    echo "✅ Smart File Photo Manager CLI installed successfully\!"
    echo "You can now run: smart-file-photo-manager --help"
fi
