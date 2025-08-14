#!/usr/bin/env bash
# Install Web Scraper Manager CLI globally

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SCRIPT="$SCRIPT_DIR/web-scraper-manager"
INSTALL_DIR="/usr/local/bin"
CLI_NAME="web-scraper-manager"

# Check if script exists
if [[ ! -f "$CLI_SCRIPT" ]]; then
    echo "Error: CLI script not found at $CLI_SCRIPT"
    exit 1
fi

# Check if already installed
if [[ -f "$INSTALL_DIR/$CLI_NAME" ]]; then
    echo "Web Scraper Manager CLI is already installed"
    echo "Updating existing installation..."
    sudo rm -f "$INSTALL_DIR/$CLI_NAME"
fi

# Install CLI
echo "Installing Web Scraper Manager CLI to $INSTALL_DIR..."
sudo cp "$CLI_SCRIPT" "$INSTALL_DIR/$CLI_NAME"
sudo chmod +x "$INSTALL_DIR/$CLI_NAME"

echo "✅ Web Scraper Manager CLI installed successfully!"
echo "Usage: $CLI_NAME --help"

# Verify installation
if command -v "$CLI_NAME" &> /dev/null; then
    echo "✅ CLI is available in PATH"
    "$CLI_NAME" version
else
    echo "⚠️  CLI installed but not in PATH. You may need to restart your shell."
fi