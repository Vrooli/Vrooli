#!/bin/bash

# Elo Swipe CLI Installation Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_NAME="elo-swipe"
INSTALL_DIR="$HOME/.vrooli/bin"

echo "Installing Elo Swipe CLI..."

# Create installation directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Build the binary if it doesn't exist
if [ ! -f "$SCRIPT_DIR/$BINARY_NAME" ]; then
    echo "Building $BINARY_NAME..."
    cd "$SCRIPT_DIR"
    go build -o "$BINARY_NAME" main.go
fi

# Copy binary to installation directory
cp "$SCRIPT_DIR/$BINARY_NAME" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

# Add to PATH if not already there
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo ""
    echo "Adding $INSTALL_DIR to PATH..."
    
    # Determine shell config file
    if [ -n "$ZSH_VERSION" ]; then
        SHELL_CONFIG="$HOME/.zshrc"
    elif [ -n "$BASH_VERSION" ]; then
        SHELL_CONFIG="$HOME/.bashrc"
    else
        SHELL_CONFIG="$HOME/.profile"
    fi
    
    # Add to shell config
    echo "" >> "$SHELL_CONFIG"
    echo "# Elo Swipe CLI" >> "$SHELL_CONFIG"
    echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_CONFIG"
    
    echo "Added to $SHELL_CONFIG"
    echo "Please run: source $SHELL_CONFIG"
    echo "Or restart your terminal for changes to take effect"
fi

echo ""
echo "✅ Elo Swipe CLI installed successfully!"
echo ""
echo "Usage: elo-swipe --help"