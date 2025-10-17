#!/usr/bin/env bash
# Vrooli Orchestrator CLI Installation Script

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
CLI_NAME="vrooli-orchestrator"
INSTALL_DIR="$HOME/.vrooli/bin"
CLI_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$CLI_NAME"

echo -e "${BLUE}🔧 Installing Vrooli Orchestrator CLI...${NC}"
echo

# Verify CLI file exists and is executable
if [[ ! -f "$CLI_PATH" ]]; then
    echo -e "${RED}❌ CLI file not found: $CLI_PATH${NC}"
    exit 1
fi

if [[ ! -x "$CLI_PATH" ]]; then
    echo -e "${YELLOW}⚠️  Making CLI executable...${NC}"
    chmod +x "$CLI_PATH"
fi

# Create install directory if it doesn't exist
if [[ ! -d "$INSTALL_DIR" ]]; then
    echo -e "${YELLOW}📁 Creating install directory: $INSTALL_DIR${NC}"
    mkdir -p "$INSTALL_DIR"
fi

# Remove existing symlink/file if it exists
if [[ -L "$INSTALL_DIR/$CLI_NAME" ]] || [[ -f "$INSTALL_DIR/$CLI_NAME" ]]; then
    echo -e "${YELLOW}🗑️  Removing existing installation...${NC}"
    rm -f "$INSTALL_DIR/$CLI_NAME"
fi

# Create symlink
echo -e "${BLUE}🔗 Creating symlink: $INSTALL_DIR/$CLI_NAME -> $CLI_PATH${NC}"
ln -s "$CLI_PATH" "$INSTALL_DIR/$CLI_NAME"

# Verify installation
if [[ -x "$INSTALL_DIR/$CLI_NAME" ]]; then
    echo -e "${GREEN}✅ CLI installed successfully!${NC}"
else
    echo -e "${RED}❌ Installation failed - CLI not executable${NC}"
    exit 1
fi

# Check if install directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo
    echo -e "${YELLOW}⚠️  Warning: $INSTALL_DIR is not in your PATH${NC}"
    echo -e "${YELLOW}   You may need to add it to your shell profile:${NC}"
    echo
    echo -e "   ${BLUE}echo 'export PATH=\"\$HOME/.vrooli/bin:\$PATH\"' >> ~/.bashrc${NC}"
    echo -e "   ${BLUE}echo 'export PATH=\"\$HOME/.vrooli/bin:\$PATH\"' >> ~/.zshrc${NC}"
    echo
    echo -e "${YELLOW}   Then restart your shell or run: source ~/.bashrc${NC}"
    echo
    echo -e "${YELLOW}   Alternative: Run directly as $INSTALL_DIR/$CLI_NAME${NC}"
else
    echo -e "${GREEN}✅ CLI is in your PATH and ready to use!${NC}"
fi

echo
echo -e "${BLUE}🚀 Quick Start:${NC}"
echo -e "   ${CLI_NAME} help          # Show help"
echo -e "   ${CLI_NAME} status        # Show current status"
echo -e "   ${CLI_NAME} list-profiles # Show available profiles"
echo -e "   ${CLI_NAME} activate developer-light # Activate a profile"
echo

# Test basic functionality
echo -e "${BLUE}🧪 Testing CLI...${NC}"
if "$INSTALL_DIR/$CLI_NAME" version >/dev/null 2>&1; then
    VERSION=$("$INSTALL_DIR/$CLI_NAME" version 2>/dev/null)
    echo -e "${GREEN}✅ CLI test passed: $VERSION${NC}"
else
    echo -e "${YELLOW}⚠️  CLI test failed (this is normal if the orchestrator isn't running yet)${NC}"
fi

echo
echo -e "${GREEN}🎉 Installation complete!${NC}"
echo -e "${BLUE}   Remember to run 'vrooli scenario run vrooli-orchestrator' to start the orchestrator service${NC}"