#!/usr/bin/env bash

# Graph Studio CLI Installation Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="graph-studio"
INSTALL_DIR="${HOME}/.vrooli/bin"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo "Installing Graph Studio CLI..."

# Create installation directory
mkdir -p "$INSTALL_DIR"

# Create symlink
ln -sf "${SCRIPT_DIR}/${CLI_NAME}" "${INSTALL_DIR}/${CLI_NAME}"

# Check if directory is in PATH
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    echo -e "${YELLOW}Adding ${INSTALL_DIR} to PATH...${NC}"
    
    # Detect shell and update appropriate config file
    if [[ -n "$ZSH_VERSION" ]]; then
        SHELL_CONFIG="$HOME/.zshrc"
    elif [[ -n "$BASH_VERSION" ]]; then
        SHELL_CONFIG="$HOME/.bashrc"
    else
        SHELL_CONFIG="$HOME/.profile"
    fi
    
    # Add to PATH if not already there
    if ! grep -q "export PATH=\"\${PATH}:${INSTALL_DIR}\"" "$SHELL_CONFIG" 2>/dev/null; then
        echo "" >> "$SHELL_CONFIG"
        echo "# Graph Studio CLI" >> "$SHELL_CONFIG"
        echo "export PATH=\"\${PATH}:${INSTALL_DIR}\"" >> "$SHELL_CONFIG"
        echo -e "${GREEN}✓ Added to PATH in $SHELL_CONFIG${NC}"
        echo -e "${YELLOW}Please run: source $SHELL_CONFIG${NC}"
    fi
fi

# Make CLI executable
chmod +x "${SCRIPT_DIR}/${CLI_NAME}"

echo -e "${GREEN}✓ Graph Studio CLI installed successfully!${NC}"
echo ""
echo "Usage: graph-studio help"
echo ""
echo "If 'graph-studio' command is not found, please:"
echo "  1. Run: source ~/.bashrc (or ~/.zshrc)"
echo "  2. Or add ${INSTALL_DIR} to your PATH manually"