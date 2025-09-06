#!/bin/bash

# Smart Shopping Assistant CLI Installation Script
# Version: 1.0.0

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Configuration
CLI_NAME="smart-shopping-assistant"
INSTALL_DIR="${HOME}/.vrooli/bin"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_FILE="${SCRIPT_DIR}/${CLI_NAME}"

echo -e "${GREEN}Installing Smart Shopping Assistant CLI...${NC}"

# Check if source file exists
if [ ! -f "$SOURCE_FILE" ]; then
    echo -e "${RED}Error: CLI script not found at ${SOURCE_FILE}${NC}"
    exit 1
fi

# Create install directory if it doesn't exist
if [ ! -d "$INSTALL_DIR" ]; then
    echo "Creating installation directory: ${INSTALL_DIR}"
    mkdir -p "$INSTALL_DIR"
fi

# Create symlink
echo "Creating symlink..."
ln -sf "$SOURCE_FILE" "${INSTALL_DIR}/${CLI_NAME}"

# Check if install directory is in PATH
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    echo -e "${YELLOW}Adding ${INSTALL_DIR} to PATH...${NC}"
    
    # Determine shell config file
    if [ -n "${BASH_VERSION:-}" ]; then
        SHELL_CONFIG="${HOME}/.bashrc"
    elif [ -n "${ZSH_VERSION:-}" ]; then
        SHELL_CONFIG="${HOME}/.zshrc"
    else
        SHELL_CONFIG="${HOME}/.profile"
    fi
    
    # Add to PATH if not already there
    if ! grep -q "export PATH=\"\${PATH}:${INSTALL_DIR}\"" "$SHELL_CONFIG" 2>/dev/null; then
        echo "" >> "$SHELL_CONFIG"
        echo "# Added by Smart Shopping Assistant" >> "$SHELL_CONFIG"
        echo "export PATH=\"\${PATH}:${INSTALL_DIR}\"" >> "$SHELL_CONFIG"
        echo -e "${GREEN}✓ Added to PATH in ${SHELL_CONFIG}${NC}"
        echo -e "${YELLOW}  Run 'source ${SHELL_CONFIG}' or restart your terminal${NC}"
    fi
fi

# Verify installation
if [ -L "${INSTALL_DIR}/${CLI_NAME}" ]; then
    echo -e "${GREEN}✓ Smart Shopping Assistant CLI installed successfully!${NC}"
    echo ""
    echo "Usage: ${CLI_NAME} --help"
    echo ""
    echo "Examples:"
    echo "  ${CLI_NAME} research \"laptop under 1000\" --alternatives"
    echo "  ${CLI_NAME} track \"https://amazon.com/product\" --target-price 50"
    echo "  ${CLI_NAME} analyze-patterns --timeframe 90d"
else
    echo -e "${RED}✗ Installation failed${NC}"
    exit 1
fi