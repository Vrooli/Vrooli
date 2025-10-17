#!/bin/bash

# Install script for Scalable App Cookbook CLI
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="scalable-app-cookbook"
CLI_PATH="${SCRIPT_DIR}/${CLI_NAME}"

# Ensure the CLI binary is executable
chmod +x "$CLI_PATH"

# Create ~/.vrooli/bin directory if it doesn't exist
mkdir -p ~/.vrooli/bin

# Create symlink in ~/.vrooli/bin
ln -sf "$CLI_PATH" ~/.vrooli/bin/$CLI_NAME

# Add ~/.vrooli/bin to PATH if not already present
if [[ ":$PATH:" != *":$HOME/.vrooli/bin:"* ]]; then
    echo "Adding ~/.vrooli/bin to PATH..."
    
    # Add to shell profile
    if [[ "$SHELL" =~ bash ]]; then
        echo 'export PATH="$HOME/.vrooli/bin:$PATH"' >> ~/.bashrc
        echo "Added to ~/.bashrc"
    elif [[ "$SHELL" =~ zsh ]]; then
        echo 'export PATH="$HOME/.vrooli/bin:$PATH"' >> ~/.zshrc
        echo "Added to ~/.zshrc"
    fi
    
    # Add to current session
    export PATH="$HOME/.vrooli/bin:$PATH"
fi

echo "✓ Scalable App Cookbook CLI installed successfully!"
echo "  Command: $CLI_NAME"
echo "  Location: ~/.vrooli/bin/$CLI_NAME"
echo "  Try: $CLI_NAME help"

# Verify installation
if command -v $CLI_NAME >/dev/null 2>&1; then
    echo "✓ Installation verified - CLI is available in PATH"
else
    echo "⚠ CLI installed but not in PATH. You may need to restart your shell."
    echo "  Or run: source ~/.bashrc (or ~/.zshrc)"
fi