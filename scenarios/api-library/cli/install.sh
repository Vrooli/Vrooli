#!/usr/bin/env bash

# API Library CLI Installation Script
# Installs the api-library CLI tool into the user's PATH

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="api-library"
CLI_PATH="$SCRIPT_DIR/$CLI_NAME"
INSTALL_DIR="$HOME/.vrooli/bin"
INSTALL_PATH="$INSTALL_DIR/$CLI_NAME"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo "Installing API Library CLI..."

# Create install directory if it doesn't exist
if [ ! -d "$INSTALL_DIR" ]; then
    echo "Creating directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
fi

# Create symlink
if [ -L "$INSTALL_PATH" ]; then
    echo -e "${YELLOW}Removing existing symlink...${NC}"
    rm "$INSTALL_PATH"
fi

echo "Creating symlink: $INSTALL_PATH -> $CLI_PATH"
ln -s "$CLI_PATH" "$INSTALL_PATH"

# Check if ~/.vrooli/bin is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${YELLOW}Adding $INSTALL_DIR to PATH...${NC}"
    
    # Detect shell and update appropriate config file
    SHELL_CONFIG=""
    if [ -n "$ZSH_VERSION" ]; then
        SHELL_CONFIG="$HOME/.zshrc"
    elif [ -n "$BASH_VERSION" ]; then
        if [ -f "$HOME/.bashrc" ]; then
            SHELL_CONFIG="$HOME/.bashrc"
        else
            SHELL_CONFIG="$HOME/.bash_profile"
        fi
    fi
    
    if [ -n "$SHELL_CONFIG" ]; then
        echo "" >> "$SHELL_CONFIG"
        echo "# Added by API Library CLI installer" >> "$SHELL_CONFIG"
        echo "export PATH=\"\$HOME/.vrooli/bin:\$PATH\"" >> "$SHELL_CONFIG"
        echo -e "${GREEN}Added PATH to $SHELL_CONFIG${NC}"
        echo -e "${YELLOW}Please run: source $SHELL_CONFIG${NC}"
    else
        echo -e "${YELLOW}Please add the following to your shell configuration:${NC}"
        echo "export PATH=\"\$HOME/.vrooli/bin:\$PATH\""
    fi
fi

# Create config directory
CONFIG_DIR="$HOME/.vrooli/api-library"
if [ ! -d "$CONFIG_DIR" ]; then
    echo "Creating config directory: $CONFIG_DIR"
    mkdir -p "$CONFIG_DIR"
fi

# Verify installation
if [ -x "$INSTALL_PATH" ]; then
    echo -e "${GREEN}✓ API Library CLI installed successfully!${NC}"
    echo ""
    echo "You can now use: api-library help"
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo -e "${YELLOW}Note: Don't forget to reload your shell or run the source command above${NC}"
    fi
else
    echo -e "${RED}✗ Installation failed${NC}"
    exit 1
fi