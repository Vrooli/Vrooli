#!/bin/bash

# Install news-bias CLI command globally
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="news-bias"
CLI_PATH="$SCRIPT_DIR/$CLI_NAME"

# Check if CLI exists
if [ ! -f "$CLI_PATH" ]; then
    echo "Error: CLI script not found at $CLI_PATH"
    exit 1
fi

# Create symlink in user's local bin (doesn't require sudo)
LOCAL_BIN="$HOME/.local/bin"
mkdir -p "$LOCAL_BIN"

# Remove old symlink if it exists
[ -L "$LOCAL_BIN/$CLI_NAME" ] && rm "$LOCAL_BIN/$CLI_NAME"

# Create new symlink
ln -s "$CLI_PATH" "$LOCAL_BIN/$CLI_NAME"

# Add to PATH if not already there
if ! echo "$PATH" | grep -q "$LOCAL_BIN"; then
    echo "" >> "$HOME/.bashrc"
    echo "# Add local bin to PATH for news-bias CLI" >> "$HOME/.bashrc"
    echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"
    echo ""
    echo "📰 CLI installed! Please run: source ~/.bashrc"
    echo "   Or start a new terminal session"
else
    echo "📰 CLI installed successfully!"
fi

echo ""
echo "Usage: news-bias help"