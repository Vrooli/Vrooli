#!/usr/bin/env bash
# Install the idea-generator CLI globally

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SOURCE="$SCRIPT_DIR/idea-generator"
CLI_TARGET="/usr/local/bin/idea-generator"

# Check if running as root or with sudo
if [[ $EUID -eq 0 ]]; then
    echo "Installing idea-generator CLI to $CLI_TARGET..."
    cp "$CLI_SOURCE" "$CLI_TARGET"
    chmod +x "$CLI_TARGET"
    echo "✅ idea-generator CLI installed successfully"
    echo "Usage: idea-generator --help"
else
    echo "Installing idea-generator CLI to $CLI_TARGET (requires sudo)..."
    sudo cp "$CLI_SOURCE" "$CLI_TARGET"
    sudo chmod +x "$CLI_TARGET"
    echo "✅ idea-generator CLI installed successfully"
    echo "Usage: idea-generator --help"
fi