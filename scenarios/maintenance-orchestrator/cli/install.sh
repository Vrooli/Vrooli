#!/bin/bash

set -euo pipefail

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="maintenance-orchestrator"
INSTALL_DIR="$HOME/.vrooli/bin"

echo "=== Installing Maintenance Orchestrator CLI ==="

# Create install directory if it doesn't exist
if [ ! -d "$INSTALL_DIR" ]; then
    echo "Creating installation directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
fi

# Copy CLI script
echo "Installing CLI to $INSTALL_DIR/$CLI_NAME"
cp "$SCRIPT_DIR/$CLI_NAME" "$INSTALL_DIR/$CLI_NAME"
chmod +x "$INSTALL_DIR/$CLI_NAME"

# Check if install directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${YELLOW}Warning: $INSTALL_DIR is not in your PATH${NC}"
    echo ""
    echo "Add the following to your shell configuration file:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    echo ""
    
    # Try to add to common shell configs
    for rc in "$HOME/.bashrc" "$HOME/.zshrc"; do
        if [ -f "$rc" ]; then
            read -p "Add to $rc? (y/N) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                echo "" >> "$rc"
                echo "# Added by maintenance-orchestrator installer" >> "$rc"
                echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$rc"
                echo -e "${GREEN}✅ Added to $rc${NC}"
                echo "Run 'source $rc' or restart your shell to update PATH"
            fi
        fi
    done
else
    echo -e "${GREEN}✅ $INSTALL_DIR is already in PATH${NC}"
fi

echo ""
echo -e "${GREEN}=== Installation Complete ===${NC}"
echo ""
echo "You can now use the maintenance-orchestrator CLI:"
echo "  $CLI_NAME help"
echo "  $CLI_NAME status"
echo "  $CLI_NAME list"
echo ""

# Test the installation
if command -v "$CLI_NAME" &> /dev/null; then
    echo -e "${GREEN}✅ CLI is accessible from PATH${NC}"
else
    echo -e "${YELLOW}Note: You may need to restart your shell or run 'source ~/.bashrc'${NC}"
fi