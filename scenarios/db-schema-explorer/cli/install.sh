#!/bin/bash

# Database Schema Explorer CLI Installation Script

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Installation directory
INSTALL_DIR="${HOME}/.vrooli/bin"
CLI_NAME="db-schema-explorer"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo -e "${GREEN}Installing Database Schema Explorer CLI...${NC}"

# Create installation directory if it doesn't exist
if [ ! -d "$INSTALL_DIR" ]; then
    echo "Creating installation directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
fi

# Copy CLI script
echo "Installing CLI to $INSTALL_DIR/$CLI_NAME"
cp "$SCRIPT_DIR/$CLI_NAME" "$INSTALL_DIR/$CLI_NAME"

# Make executable
chmod +x "$INSTALL_DIR/$CLI_NAME"

# Check if install directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${YELLOW}Adding $INSTALL_DIR to PATH...${NC}"
    
    # Determine shell configuration file
    if [ -f "$HOME/.bashrc" ]; then
        SHELL_CONFIG="$HOME/.bashrc"
    elif [ -f "$HOME/.zshrc" ]; then
        SHELL_CONFIG="$HOME/.zshrc"
    else
        SHELL_CONFIG="$HOME/.profile"
    fi
    
    # Add to PATH if not already there
    if ! grep -q "export PATH=\"\$PATH:$INSTALL_DIR\"" "$SHELL_CONFIG" 2>/dev/null; then
        echo "" >> "$SHELL_CONFIG"
        echo "# Database Schema Explorer CLI" >> "$SHELL_CONFIG"
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_CONFIG"
        echo -e "${GREEN}✓ Added to PATH in $SHELL_CONFIG${NC}"
        echo -e "${YELLOW}  Run 'source $SHELL_CONFIG' to update current session${NC}"
    fi
fi

# Create command alias for convenience
if command -v db-schema-explorer &> /dev/null; then
    echo -e "${GREEN}✓ CLI already accessible via PATH${NC}"
else
    # Create temporary symlink for current session
    if [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
        ln -sf "$INSTALL_DIR/$CLI_NAME" "/usr/local/bin/$CLI_NAME" 2>/dev/null || true
    fi
fi

echo
echo -e "${GREEN}✓ Database Schema Explorer CLI installed successfully!${NC}"
echo
echo "Usage:"
echo "  db-schema-explorer --help    Show help information"
echo "  db-schema-explorer status    Check service status"
echo "  db-schema-explorer connect   Connect to a database"
echo "  db-schema-explorer query     Generate SQL from natural language"
echo
echo "Make sure the API is running:"
echo "  vrooli scenario run db-schema-explorer"
echo
echo "For immediate use in this session, run:"
echo -e "${YELLOW}  export PATH=\"\$PATH:$INSTALL_DIR\"${NC}"