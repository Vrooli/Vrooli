#!/usr/bin/env bash

# Install script for scenario-to-mcp CLI

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="scenario-to-mcp"
CLI_PATH="$SCRIPT_DIR/$CLI_NAME"
VROOLI_BIN="$HOME/.vrooli/bin"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo "Installing scenario-to-mcp CLI..."

# Create Vrooli bin directory if it doesn't exist
if [[ ! -d "$VROOLI_BIN" ]]; then
    echo "Creating $VROOLI_BIN directory..."
    mkdir -p "$VROOLI_BIN"
fi

# Make CLI executable
chmod +x "$CLI_PATH"

# Create symlink
if [[ -L "$VROOLI_BIN/$CLI_NAME" ]]; then
    echo "Removing existing symlink..."
    rm "$VROOLI_BIN/$CLI_NAME"
fi

ln -s "$CLI_PATH" "$VROOLI_BIN/$CLI_NAME"
echo -e "${GREEN}✓${NC} Created symlink: $VROOLI_BIN/$CLI_NAME"

# Check if Vrooli bin is in PATH
if [[ ":$PATH:" != *":$VROOLI_BIN:"* ]]; then
    echo -e "${YELLOW}⚠${NC} $VROOLI_BIN is not in your PATH"
    echo ""
    echo "Add this to your shell configuration file (.bashrc, .zshrc, etc.):"
    echo ""
    echo "  export PATH=\"\$HOME/.vrooli/bin:\$PATH\""
    echo ""
    echo "Then reload your shell or run: source ~/.bashrc"
else
    echo -e "${GREEN}✓${NC} $VROOLI_BIN is already in PATH"
fi

echo ""
echo -e "${GREEN}Installation complete!${NC}"
echo ""
echo "You can now use: scenario-to-mcp --help"