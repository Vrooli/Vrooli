#!/usr/bin/env bash

# Swarm Manager Installation Script
# Adds swarm-manager to the system PATH

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_PATH="${SCRIPT_DIR}/cli.sh"
INSTALL_DIR="${HOME}/.local/bin"
SYMLINK_NAME="swarm-manager"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Installing Swarm Manager...${NC}"

# Create .local/bin if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Create symlink
ln -sf "$CLI_PATH" "${INSTALL_DIR}/${SYMLINK_NAME}"

# Check if .local/bin is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo -e "${BLUE}Note: Add ${INSTALL_DIR} to your PATH:${NC}"
    echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc"
    echo "  source ~/.bashrc"
fi

echo -e "${GREEN}✓ Swarm Manager installed successfully${NC}"
echo ""
echo "Usage:"
echo "  swarm-manager start    # Start the service"
echo "  swarm-manager status   # Check status"
echo "  swarm-manager help     # Show all commands"