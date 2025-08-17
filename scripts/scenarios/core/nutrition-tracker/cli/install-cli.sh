#!/bin/bash
set -e

# Install Nutrition Tracker CLI globally
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="nutrition-tracker"
CLI_PATH="$SCRIPT_DIR/$CLI_NAME"

# Ensure the CLI script is executable
chmod +x "$CLI_PATH"

# Create symlink in /usr/local/bin
if [ -L "/usr/local/bin/$CLI_NAME" ]; then
    echo "Removing existing symlink..."
    sudo rm "/usr/local/bin/$CLI_NAME"
fi

echo "Installing $CLI_NAME CLI..."
sudo ln -s "$CLI_PATH" "/usr/local/bin/$CLI_NAME"

echo "✓ $CLI_NAME CLI installed successfully"
echo "  You can now use: $CLI_NAME [command]"