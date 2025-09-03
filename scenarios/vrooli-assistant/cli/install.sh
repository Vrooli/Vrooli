#!/bin/bash

# Vrooli Assistant CLI Installation Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="vrooli-assistant"
INSTALL_DIR="$HOME/.vrooli/bin"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}Installing Vrooli Assistant CLI...${NC}"

# Create install directory
mkdir -p "$INSTALL_DIR"

# Copy CLI to install directory
cp "$SCRIPT_DIR/$CLI_NAME" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/$CLI_NAME"

# Create symlink in system path if possible
if [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
    ln -sf "$INSTALL_DIR/$CLI_NAME" "~/.local/bin/$CLI_NAME"
    echo -e "${GREEN}✓ Created symlink in /usr/local/bin${NC}"
elif [ -d "$HOME/.local/bin" ]; then
    mkdir -p "$HOME/.local/bin"
    ln -sf "$INSTALL_DIR/$CLI_NAME" "$HOME/.local/bin/$CLI_NAME"
    echo -e "${GREEN}✓ Created symlink in ~/.local/bin${NC}"
fi

# Check if PATH needs updating
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo -e "${YELLOW}Adding $INSTALL_DIR to PATH...${NC}"
    
    # Determine shell config file
    if [ -n "$ZSH_VERSION" ]; then
        SHELL_RC="$HOME/.zshrc"
    elif [ -n "$BASH_VERSION" ]; then
        SHELL_RC="$HOME/.bashrc"
    else
        SHELL_RC="$HOME/.profile"
    fi
    
    # Add to PATH
    echo "" >> "$SHELL_RC"
    echo "# Vrooli CLI tools" >> "$SHELL_RC"
    echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_RC"
    
    echo -e "${GREEN}✓ Added to PATH in $SHELL_RC${NC}"
    echo -e "${YELLOW}Please run: source $SHELL_RC${NC}"
fi

# Install Electron app dependencies
echo -e "${BLUE}Installing Electron app dependencies...${NC}"
cd "$SCRIPT_DIR/../ui/electron"
npm install

# Create desktop entry for Linux
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    DESKTOP_FILE="$HOME/.local/share/applications/vrooli-assistant.desktop"
    mkdir -p "$(dirname "$DESKTOP_FILE")"
    
    cat > "$DESKTOP_FILE" << EOF
[Desktop Entry]
Name=Vrooli Assistant
Comment=Real-time issue capture and agent spawning
Exec=$INSTALL_DIR/$CLI_NAME start
Icon=$SCRIPT_DIR/../ui/electron/assets/icon.png
Terminal=false
Type=Application
Categories=Development;
EOF
    
    echo -e "${GREEN}✓ Created desktop entry${NC}"
fi

echo -e "${GREEN}✓ Vrooli Assistant CLI installed successfully!${NC}"
echo ""
echo "Usage:"
echo "  $CLI_NAME start --daemon    # Start in background"
echo "  $CLI_NAME status            # Check status"
echo "  $CLI_NAME help              # Show all commands"
echo ""
echo "Global Hotkey: Cmd+Shift+Space (Mac) / Ctrl+Shift+Space (Linux/Win)"