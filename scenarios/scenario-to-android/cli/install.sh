#!/bin/bash

# Install script for scenario-to-android CLI
# Adds the CLI to PATH for easy access

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CLI_NAME="scenario-to-android"
CLI_PATH="${SCRIPT_DIR}/${CLI_NAME}"
INSTALL_DIR="${HOME}/.vrooli/bin"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

print_color() {
    local color=$1
    shift
    echo -e "${color}$*${NC}"
}

# Create install directory
mkdir -p "$INSTALL_DIR"

# Create symlink
if [ -f "$CLI_PATH" ]; then
    ln -sf "$CLI_PATH" "${INSTALL_DIR}/${CLI_NAME}"
    print_color "$GREEN" "✓ CLI installed to ${INSTALL_DIR}/${CLI_NAME}"
else
    print_color "$RED" "Error: CLI not found at $CLI_PATH"
    exit 1
fi

# Check if directory is in PATH
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    print_color "$YELLOW" "Adding ${INSTALL_DIR} to PATH..."

    # Determine shell config file
    SHELL_CONFIG=""
    if [ -n "$BASH_VERSION" ]; then
        SHELL_CONFIG="$HOME/.bashrc"
    elif [ -n "$ZSH_VERSION" ]; then
        SHELL_CONFIG="$HOME/.zshrc"
    else
        SHELL_CONFIG="$HOME/.profile"
    fi

    # Add to PATH
    {
        echo ""
        echo "# Added by scenario-to-android"
        echo "export PATH=\"\$PATH:${INSTALL_DIR}\""
    } >> "$SHELL_CONFIG"

    print_color "$GREEN" "✓ PATH updated in $SHELL_CONFIG"
    print_color "$YELLOW" "Run 'source $SHELL_CONFIG' or restart your terminal"
else
    print_color "$GREEN" "✓ ${INSTALL_DIR} already in PATH"
fi

# Verify installation
if command -v scenario-to-android &> /dev/null; then
    print_color "$GREEN" "✓ Installation successful!"
    print_color "$GREEN" "Run 'scenario-to-android help' to get started"
else
    print_color "$YELLOW" "Installation complete. Restart terminal to use 'scenario-to-android'"
fi