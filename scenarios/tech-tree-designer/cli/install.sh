#!/bin/bash

# Tech Tree Designer CLI Installation Script
# Installs the tech-tree-designer command globally for strategic intelligence access

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
CLI_NAME="tech-tree-designer"
INSTALL_DIR="$HOME/.vrooli/bin"
CLI_SOURCE="$(dirname "$0")/$CLI_NAME"

echo -e "${CYAN}🌟 Installing Tech Tree Designer CLI...${NC}"

# Create install directory if it doesn't exist
if [[ ! -d "$INSTALL_DIR" ]]; then
    echo -e "${YELLOW}Creating install directory: $INSTALL_DIR${NC}"
    mkdir -p "$INSTALL_DIR"
fi

# Copy CLI script to install directory
echo -e "${YELLOW}Installing CLI command...${NC}"
cp "$CLI_SOURCE" "$INSTALL_DIR/$CLI_NAME"
chmod +x "$INSTALL_DIR/$CLI_NAME"

# Check if install directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${YELLOW}Adding $INSTALL_DIR to PATH...${NC}"
    
    # Determine shell profile file
    if [[ -n "$BASH_VERSION" ]]; then
        PROFILE_FILE="$HOME/.bashrc"
    elif [[ -n "$ZSH_VERSION" ]]; then
        PROFILE_FILE="$HOME/.zshrc"
    else
        PROFILE_FILE="$HOME/.profile"
    fi
    
    # Add PATH export if not already present
    if ! grep -q "$INSTALL_DIR" "$PROFILE_FILE" 2>/dev/null; then
        echo "" >> "$PROFILE_FILE"
        echo "# Added by Tech Tree Designer CLI installer" >> "$PROFILE_FILE"
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$PROFILE_FILE"
        echo -e "${GREEN}Added to $PROFILE_FILE${NC}"
        echo -e "${YELLOW}Please run: source $PROFILE_FILE${NC}"
        echo -e "${YELLOW}Or restart your terminal to use the CLI${NC}"
    fi
fi

# Test installation
if [[ ":$PATH:" == *":$INSTALL_DIR:"* ]] || [[ -x "$INSTALL_DIR/$CLI_NAME" ]]; then
    echo -e "${GREEN}✅ Tech Tree Designer CLI installed successfully!${NC}"
    echo ""
    echo -e "${CYAN}🎯 Strategic Intelligence Command Interface Ready${NC}"
    echo ""
    echo "Usage examples:"
    echo "  tech-tree-designer status --verbose"
    echo "  tech-tree-designer analyze --resources 8 --timeline 6"  
    echo "  tech-tree-designer recommend --priority software,healthcare"
    echo "  tech-tree-designer milestones"
    echo ""
    echo "Get help:"
    echo "  tech-tree-designer help"
    echo ""
    echo -e "${GREEN}🚀 Ready to guide Vrooli's evolution toward superintelligence!${NC}"
else
    echo -e "${YELLOW}⚠️  CLI installed but not in PATH${NC}"
    echo "You can run it directly with: $INSTALL_DIR/$CLI_NAME"
fi