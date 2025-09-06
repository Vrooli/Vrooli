#!/bin/bash

# Scenario Surfer CLI Installation Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="scenario-surfer"
CLI_SCRIPT="$SCRIPT_DIR/$CLI_NAME"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}🌊 Installing Scenario Surfer CLI...${NC}"

# Create Vrooli bin directory if it doesn't exist
VROOLI_BIN="$HOME/.vrooli/bin"
if [ ! -d "$VROOLI_BIN" ]; then
    echo "Creating ~/.vrooli/bin directory..."
    mkdir -p "$VROOLI_BIN"
fi

# Create symlink
SYMLINK_PATH="$VROOLI_BIN/$CLI_NAME"
if [ -L "$SYMLINK_PATH" ] || [ -f "$SYMLINK_PATH" ]; then
    echo "Removing existing installation..."
    rm -f "$SYMLINK_PATH"
fi

echo "Creating symlink..."
ln -s "$CLI_SCRIPT" "$SYMLINK_PATH"

# Make sure the CLI script is executable
chmod +x "$CLI_SCRIPT"

# Check if ~/.vrooli/bin is in PATH
if [[ ":$PATH:" != *":$VROOLI_BIN:"* ]]; then
    echo -e "${YELLOW}⚠️  ~/.vrooli/bin is not in your PATH${NC}"
    echo "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo "export PATH=\"\$HOME/.vrooli/bin:\$PATH\""
    echo ""
    echo "Or run: export PATH=\"\$HOME/.vrooli/bin:\$PATH\""
    echo "Then reload your shell or run: source ~/.bashrc"
else
    echo -e "${GREEN}✓${NC} ~/.vrooli/bin is already in PATH"
fi

echo -e "${GREEN}✅ Scenario Surfer CLI installed successfully!${NC}"
echo ""
echo "Usage:"
echo "  scenario-surfer status      # Check service status"
echo "  scenario-surfer discover    # List available scenarios" 
echo "  scenario-surfer open        # Open browser interface"
echo "  scenario-surfer help        # Show help"
echo ""
echo "Test installation:"
echo "  $VROOLI_BIN/$CLI_NAME version"