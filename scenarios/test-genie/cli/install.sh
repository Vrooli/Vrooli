#!/bin/bash
set -e

# Test Genie CLI Installation Script
echo "Installing Test Genie CLI..."

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SOURCE="$SCRIPT_DIR/test-genie"

# Create a temporary file for the CLI binary
TEMP_CLI=$(mktemp)
cp "$CLI_SOURCE" "$TEMP_CLI"
chmod +x "$TEMP_CLI"

# Determine installation directory
if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
    # Windows
    INSTALL_DIR="$HOME/bin"
    CLI_NAME="test-genie.exe"
else
    # Unix-like systems
    if [ -w "/usr/local/bin" ]; then
        INSTALL_DIR="/usr/local/bin"
    elif [ -w "$HOME/.local/bin" ]; then
        INSTALL_DIR="$HOME/.local/bin"
    else
        INSTALL_DIR="$HOME/bin"
    fi
    CLI_NAME="test-genie"
fi

# Create install directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Install the CLI
cp "$TEMP_CLI" "$INSTALL_DIR/$CLI_NAME"
chmod +x "$INSTALL_DIR/$CLI_NAME"

# Clean up
rm "$TEMP_CLI"

echo "✓ Test Genie CLI installed successfully to $INSTALL_DIR/$CLI_NAME"

# Check if directory is in PATH
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo "⚠ Warning: $INSTALL_DIR is not in your PATH"
    echo "Add the following to your shell profile (.bashrc, .zshrc, etc.):"
    echo "export PATH=\"$INSTALL_DIR:\$PATH\""
fi

echo ""
echo "Usage:"
echo "  test-genie help                    Show help"
echo "  test-genie generate <scenario>     Generate test suite"
echo "  test-genie execute <suite_id>      Execute test suite"
echo "  test-genie vault <scenario>        Create test vault"
echo "  test-genie coverage <scenario>     Analyze coverage"
echo ""
echo "Run 'test-genie --version' to verify installation."