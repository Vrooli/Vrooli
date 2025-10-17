#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="financial-calculators-hub"
INSTALL_DIR="${HOME}/.vrooli/bin"

echo "Installing Financial Calculators Hub CLI..."

mkdir -p "$INSTALL_DIR"

chmod +x "${SCRIPT_DIR}/${CLI_NAME}"

ln -sf "${SCRIPT_DIR}/${CLI_NAME}" "${INSTALL_DIR}/${CLI_NAME}"

if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo ""
    echo "Adding $INSTALL_DIR to PATH..."
    
    if [[ -f "$HOME/.bashrc" ]]; then
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$HOME/.bashrc"
        echo "Added to ~/.bashrc"
    fi
    
    if [[ -f "$HOME/.zshrc" ]]; then
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$HOME/.zshrc"
        echo "Added to ~/.zshrc"
    fi
    
    echo ""
    echo "Please run: source ~/.bashrc (or ~/.zshrc)"
    echo "Or start a new terminal session"
fi

echo ""
echo "✅ CLI installed successfully!"
echo "Run: ${CLI_NAME} --help"