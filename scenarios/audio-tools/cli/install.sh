#!/usr/bin/env bash
set -euo pipefail

# Install the audio-tools CLI
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="audio-tools"
CLI_PATH="$SCRIPT_DIR/$CLI_NAME"

# Create symlink in ~/.local/bin
mkdir -p ~/.local/bin
ln -sf "$CLI_PATH" ~/.local/bin/$CLI_NAME

# Ensure ~/.local/bin is in PATH
if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    echo "NOTE: Add ~/.local/bin to your PATH to use the $CLI_NAME command"
    echo "Run: export PATH=\"\$HOME/.local/bin:\$PATH\""
fi

echo "✓ $CLI_NAME CLI installed successfully"