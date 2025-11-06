#!/bin/bash

set -euo pipefail

# Install script for Feature Voting CLI

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SCRIPT="feature-voting"
CLI_COMMAND="feature-request-voting"
CLI_ALIAS="feature-voting"
CLI_PATH="${SCRIPT_DIR}/${CLI_SCRIPT}"
INSTALL_DIR="${HOME}/.vrooli/bin"

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1" >&2
}

# Create install directory if it doesn't exist
if [ ! -d "$INSTALL_DIR" ]; then
    echo "Creating installation directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
fi

# Check if CLI script exists
if [ ! -f "$CLI_PATH" ]; then
    print_error "CLI script not found at $CLI_PATH"
    exit 1
fi

# Create symlink
if [ -L "${INSTALL_DIR}/${CLI_COMMAND}" ]; then
    echo "Removing existing symlink..."
    rm "${INSTALL_DIR}/${CLI_COMMAND}"
fi

ln -s "$CLI_PATH" "${INSTALL_DIR}/${CLI_COMMAND}"
print_success "Created symlink: ${INSTALL_DIR}/${CLI_COMMAND} -> $CLI_PATH"

# Maintain backward compatibility for legacy command name
ln -sf "$CLI_PATH" "${INSTALL_DIR}/${CLI_ALIAS}"
print_success "Created symlink: ${INSTALL_DIR}/${CLI_ALIAS} -> $CLI_PATH"

# Check if install directory is in PATH
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    echo ""
    echo "⚠️  ${INSTALL_DIR} is not in your PATH"
    echo ""
    echo "Add it to your PATH by adding this line to your shell config file:"
    echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
    echo ""
    echo "For bash, add to ~/.bashrc or ~/.bash_profile"
    echo "For zsh, add to ~/.zshrc"
    echo ""
    
    # Try to auto-add to common shell configs
    read -p "Would you like to automatically add this to your shell config? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        PATH_LINE="export PATH=\"\$PATH:${INSTALL_DIR}\""
        
        # Detect shell and update appropriate config
        if [ -n "${ZSH_VERSION:-}" ]; then
            CONFIG_FILE="${HOME}/.zshrc"
        elif [ -n "${BASH_VERSION:-}" ]; then
            CONFIG_FILE="${HOME}/.bashrc"
        else
            CONFIG_FILE="${HOME}/.profile"
        fi
        
        # Check if already in config
        if ! grep -q "${INSTALL_DIR}" "$CONFIG_FILE" 2>/dev/null; then
            echo "" >> "$CONFIG_FILE"
    echo "# Added by feature-request-voting install script" >> "$CONFIG_FILE"
            echo "$PATH_LINE" >> "$CONFIG_FILE"
            print_success "Added PATH update to $CONFIG_FILE"
            echo "Please run: source $CONFIG_FILE"
        else
            echo "PATH already contains ${INSTALL_DIR}"
        fi
    fi
else
    print_success "${INSTALL_DIR} is already in PATH"
fi

echo ""
print_success "Installation complete!"
echo ""
echo "You can now use the CLI with: ${CLI_COMMAND} --help"
echo "Legacy alias still available as: ${CLI_ALIAS} --help"

# Test the installation
if command -v "$CLI_COMMAND" &> /dev/null; then
    print_success "CLI is accessible from PATH"
else
    echo "Note: You may need to restart your terminal or run 'source' on your shell config"
fi
