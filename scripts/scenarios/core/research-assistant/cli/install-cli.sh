#!/usr/bin/env bash
# Install Research Assistant CLI globally

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="research-assistant"
CLI_SCRIPT="$SCRIPT_DIR/$CLI_NAME"

# Colors
GREEN='\033[1;32m'
BLUE='\033[1;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}🔍 Installing Research Assistant CLI...${NC}"

# Check if script exists
if [[ ! -f "$CLI_SCRIPT" ]]; then
    echo "❌ CLI script not found: $CLI_SCRIPT"
    exit 1
fi

# Make sure it's executable
chmod +x "$CLI_SCRIPT"

# Install to system path
INSTALL_DIR="/usr/local/bin"
INSTALL_PATH="$INSTALL_DIR/$CLI_NAME"

if [[ -w "$INSTALL_DIR" ]]; then
    # Can write directly
    cp "$CLI_SCRIPT" "$INSTALL_PATH"
    echo -e "${GREEN}✅ CLI installed to $INSTALL_PATH${NC}"
else
    # Need sudo
    echo -e "${YELLOW}🔐 Installing to system directory (requires sudo)...${NC}"
    sudo cp "$CLI_SCRIPT" "$INSTALL_PATH"
    echo -e "${GREEN}✅ CLI installed to $INSTALL_PATH${NC}"
fi

# Verify installation
if command -v "$CLI_NAME" &> /dev/null; then
    echo -e "${GREEN}✅ Installation successful!${NC}"
    echo -e "${BLUE}🚀 Try: $CLI_NAME help${NC}"
else
    echo "❌ Installation failed - command not found in PATH"
    echo "You may need to add $INSTALL_DIR to your PATH"
    exit 1
fi