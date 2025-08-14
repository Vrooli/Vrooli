#!/usr/bin/env bash
# Install script for Image Generation Pipeline CLI

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="image-generation-pipeline"
CLI_SCRIPT="image-generation-pipeline-cli.sh"

# Check if running as root or with sudo privileges
if [[ $EUID -eq 0 ]]; then
    INSTALL_DIR="/usr/local/bin"
    echo "🔧 Installing globally to $INSTALL_DIR (requires root)"
else
    INSTALL_DIR="$HOME/.local/bin"
    echo "🔧 Installing to user directory $INSTALL_DIR"
    
    # Create user bin directory if it doesn't exist
    mkdir -p "$INSTALL_DIR"
    
    # Add to PATH if not already there
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo "📝 Adding $INSTALL_DIR to PATH in ~/.bashrc"
        echo "export PATH=\"\$HOME/.local/bin:\$PATH\"" >> ~/.bashrc
        echo "⚠️  Please run 'source ~/.bashrc' or restart your terminal"
    fi
fi

# Copy and make executable
echo "📦 Installing $CLI_NAME to $INSTALL_DIR"
cp "$SCRIPT_DIR/$CLI_SCRIPT" "$INSTALL_DIR/$CLI_NAME"
chmod +x "$INSTALL_DIR/$CLI_NAME"

echo "✅ $CLI_NAME installed successfully!"
echo "🚀 Usage: $CLI_NAME --help"