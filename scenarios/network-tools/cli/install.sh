#!/usr/bin/env bash
set -euo pipefail

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="network-tools"
CLI_PATH="$SCRIPT_DIR/$CLI_NAME"

# Simple CLI install - just create symlink
if [[ -f "$CLI_PATH" ]]; then
    chmod +x "$CLI_PATH"

    # Create symlink in home bin directory (most portable option)
    mkdir -p "$HOME/.local/bin"
    ln -sf "$CLI_PATH" "$HOME/.local/bin/$CLI_NAME" 2>/dev/null || true

    echo "✅ CLI installed: $CLI_NAME"
    echo "   Location: $HOME/.local/bin/$CLI_NAME"
    echo "   Make sure $HOME/.local/bin is in your PATH"
else
    echo "❌ Error: CLI script not found at $CLI_PATH"
    exit 1
fi