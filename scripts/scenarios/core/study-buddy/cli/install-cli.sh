#!/usr/bin/env bash

# Install Study Buddy CLI
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="study-buddy"
CLI_PATH="$SCRIPT_DIR/$CLI_NAME"

# Make CLI executable
chmod +x "$CLI_PATH"

# Create symbolic link in user's local bin if it exists
if [ -d "$HOME/.local/bin" ]; then
    ln -sf "$CLI_PATH" "$HOME/.local/bin/$CLI_NAME"
    echo "✅ Study Buddy CLI installed to ~/.local/bin/$CLI_NAME"
elif [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
    ln -sf "$CLI_PATH" "/usr/local/bin/$CLI_NAME"
    echo "✅ Study Buddy CLI installed to /usr/local/bin/$CLI_NAME"
else
    echo "📚 Study Buddy CLI is ready at: $CLI_PATH"
    echo "   Add it to your PATH or run directly with: $CLI_PATH"
fi

echo ""
echo "Usage: study-buddy help"