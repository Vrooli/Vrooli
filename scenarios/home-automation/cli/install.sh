#!/bin/bash

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Installing home-automation CLI...${NC}"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="home-automation"
CLI_PATH="$SCRIPT_DIR/$CLI_NAME"

# Check if CLI binary exists
if [ ! -f "$CLI_PATH" ]; then
    echo -e "${RED}Error: CLI binary not found at $CLI_PATH${NC}"
    exit 1
fi

# Make sure it's executable
chmod +x "$CLI_PATH"

# Install to user's local bin
INSTALL_DIR="$HOME/.local/bin"

# Create install directory if it doesn't exist
if [ ! -d "$INSTALL_DIR" ]; then
    echo -e "${YELLOW}Creating $INSTALL_DIR directory...${NC}"
    mkdir -p "$INSTALL_DIR"
fi

# Create symlink or copy
if [ -L "$INSTALL_DIR/$CLI_NAME" ]; then
    echo -e "${YELLOW}Removing existing symlink...${NC}"
    rm "$INSTALL_DIR/$CLI_NAME"
fi

echo -e "${GREEN}Installing $CLI_NAME to $INSTALL_DIR...${NC}"
ln -sf "$CLI_PATH" "$INSTALL_DIR/$CLI_NAME"

# Check if ~/.local/bin is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${YELLOW}Warning: $INSTALL_DIR is not in your PATH${NC}"
    echo -e "${YELLOW}Add this line to your ~/.bashrc or ~/.zshrc:${NC}"
    echo -e "${YELLOW}  export PATH=\"\$HOME/.local/bin:\$PATH\"${NC}"
    echo ""
    echo -e "${YELLOW}Then run: source ~/.bashrc (or source ~/.zshrc)${NC}"
else
    echo -e "${GREEN}✅ $INSTALL_DIR is already in your PATH${NC}"
fi

# Verify installation
if command -v "$CLI_NAME" >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Installation successful!${NC}"
    echo ""
    echo "Try running: $CLI_NAME --help"
else
    echo -e "${YELLOW}⚠️  Installation completed, but $CLI_NAME is not yet in PATH${NC}"
    echo "You may need to restart your shell or update your PATH as shown above."
fi
