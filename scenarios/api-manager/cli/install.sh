#!/bin/bash

set -e

# API Manager CLI Installation Script

CLI_NAME="api-manager"
BUILD_DIR="$(dirname "$0")"
INSTALL_DIR="${VROOLI_CLI_DIR:-$HOME/.local/bin}"

echo "Installing API Manager CLI..."

# Create install directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Build the CLI
echo "Building CLI..."
cd "$BUILD_DIR"
go build -o "$CLI_NAME" .

# Install to the target directory
echo "Installing to $INSTALL_DIR..."
mv "$CLI_NAME" "$INSTALL_DIR/"

# Make executable
chmod +x "$INSTALL_DIR/$CLI_NAME"

# Check if install directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "⚠️  Warning: $INSTALL_DIR is not in your PATH."
    echo "   Add this to your shell profile (.bashrc, .zshrc, etc.):"
    echo "   export PATH=\"$INSTALL_DIR:\$PATH\""
    echo ""
fi

echo "✅ API Manager CLI installed successfully!"
echo ""
echo "Usage:"
echo "  $CLI_NAME help              # Show help"
echo "  $CLI_NAME health            # Check API manager status"
echo "  $CLI_NAME list scenarios    # List all scenarios"
echo "  $CLI_NAME scan <scenario>   # Scan a scenario for issues"
echo "  $CLI_NAME vulnerabilities  # Show all vulnerabilities"
echo ""
echo "Environment variables:"
echo "  API_MANAGER_URL  # API manager base URL (default: http://localhost:8421)"