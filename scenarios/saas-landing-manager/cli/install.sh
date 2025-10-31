#!/bin/bash

# SaaS Landing Manager CLI Install Script
set -euo pipefail

# Configuration
CLI_NAME="saas-landing-manager"
INSTALL_DIR="$HOME/.vrooli/bin"
CLI_BINARY="$CLI_NAME"

echo "Installing SaaS Landing Manager CLI..."

# Create install directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Build the CLI binary
echo "Building CLI binary..."
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    exit 1
fi

# Build the binary
go mod download
go build -o "$CLI_BINARY" .

# Ensure the local binary is executable for scenario tests
chmod +x "$CLI_BINARY"

# Copy binary to install directory (preserve local copy for tests)
echo "Installing to $INSTALL_DIR/$CLI_BINARY..."
cp "$CLI_BINARY" "$INSTALL_DIR/$CLI_BINARY"
chmod +x "$INSTALL_DIR/$CLI_BINARY"

# Add to PATH if not already there
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "Adding $INSTALL_DIR to PATH..."
    
    # Add to bashrc
    if [ -f "$HOME/.bashrc" ]; then
        echo 'export PATH="$HOME/.vrooli/bin:$PATH"' >> "$HOME/.bashrc"
    fi
    
    # Add to zshrc if it exists
    if [ -f "$HOME/.zshrc" ]; then
        echo 'export PATH="$HOME/.vrooli/bin:$PATH"' >> "$HOME/.zshrc"
    fi
    
    # Add to current session
    export PATH="$INSTALL_DIR:$PATH"
fi

# Verify installation
if command -v "$CLI_NAME" &> /dev/null; then
    echo "✓ Installation successful!"
    echo "Run '$CLI_NAME help' to get started."
    
    # Show version
    "$CLI_NAME" version
else
    echo "✓ Binary installed to $INSTALL_DIR/$CLI_BINARY"
    echo "Please restart your shell or run: export PATH=\"$INSTALL_DIR:\$PATH\""
    echo "Then run '$CLI_NAME help' to get started."
fi

echo ""
echo "SaaS Landing Manager CLI is now installed!"
echo ""
echo "Quick start:"
echo "  $CLI_NAME status              # Check service health"
echo "  $CLI_NAME scan --force        # Scan for SaaS scenarios"
echo "  $CLI_NAME template list       # List available templates"
echo "  $CLI_NAME help                # Show all commands"
echo ""
