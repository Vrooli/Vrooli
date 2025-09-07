#!/bin/bash

# Code Smell CLI Installation Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"
INSTALL_DIR="${HOME}/.vrooli/bin"
CLI_NAME="code-smell"

echo "Installing Code Smell CLI..."

# Create installation directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Install Node.js dependencies
echo "Installing dependencies..."
cd "$SCENARIO_DIR"
if [ ! -d "node_modules" ]; then
    npm install --quiet \
        commander \
        axios \
        chalk \
        cli-table3 \
        js-yaml \
        chokidar
fi

# Create symlink to CLI
echo "Creating CLI symlink..."
ln -sf "$SCRIPT_DIR/$CLI_NAME" "$INSTALL_DIR/$CLI_NAME"

# Add to PATH if not already there
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "Adding $INSTALL_DIR to PATH..."
    
    # Detect shell and update appropriate config file
    if [ -n "$ZSH_VERSION" ]; then
        SHELL_CONFIG="$HOME/.zshrc"
    elif [ -n "$BASH_VERSION" ]; then
        SHELL_CONFIG="$HOME/.bashrc"
    else
        SHELL_CONFIG="$HOME/.profile"
    fi
    
    echo "" >> "$SHELL_CONFIG"
    echo "# Added by Code Smell CLI installer" >> "$SHELL_CONFIG"
    echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_CONFIG"
    
    echo "Please run: source $SHELL_CONFIG"
    echo "Or restart your terminal for PATH changes to take effect"
fi

echo ""
echo "✓ Code Smell CLI installed successfully!"
echo ""
echo "Usage:"
echo "  $CLI_NAME status         - Check operational status"
echo "  $CLI_NAME analyze <path> - Analyze files for code smells"
echo "  $CLI_NAME rules list     - List all detection rules"
echo "  $CLI_NAME help           - Show all commands"
echo ""

# Make sure the CLI is executable
chmod +x "$SCRIPT_DIR/$CLI_NAME"

exit 0