#!/bin/bash

# Recipe Book CLI Installation Script

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}Installing Recipe Book CLI...${NC}"

# Get the directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CLI_NAME="recipe-book"
CLI_PATH="${SCRIPT_DIR}/${CLI_NAME}"

# Make CLI executable
chmod +x "${CLI_PATH}"

# Create ~/.vrooli/bin if it doesn't exist
BIN_DIR="${HOME}/.vrooli/bin"
mkdir -p "${BIN_DIR}"

# Create symlink
ln -sf "${CLI_PATH}" "${BIN_DIR}/${CLI_NAME}"

# Check if ~/.vrooli/bin is in PATH
if [[ ":$PATH:" != *":${BIN_DIR}:"* ]]; then
    echo -e "${YELLOW}Adding ${BIN_DIR} to PATH...${NC}"
    
    # Determine which shell config to update
    if [ -f "${HOME}/.zshrc" ]; then
        SHELL_CONFIG="${HOME}/.zshrc"
    elif [ -f "${HOME}/.bashrc" ]; then
        SHELL_CONFIG="${HOME}/.bashrc"
    else
        SHELL_CONFIG="${HOME}/.profile"
    fi
    
    # Add to PATH if not already there
    if ! grep -q "export PATH=\"\${HOME}/.vrooli/bin:\${PATH}\"" "${SHELL_CONFIG}"; then
        echo "" >> "${SHELL_CONFIG}"
        echo "# Recipe Book CLI" >> "${SHELL_CONFIG}"
        echo "export PATH=\"\${HOME}/.vrooli/bin:\${PATH}\"" >> "${SHELL_CONFIG}"
        echo -e "${GREEN}✓ Added to ${SHELL_CONFIG}${NC}"
        echo -e "${YELLOW}Please run: source ${SHELL_CONFIG}${NC}"
    fi
fi

echo -e "${GREEN}✓ Recipe Book CLI installed successfully!${NC}"
echo ""
echo "Usage: recipe-book --help"
echo ""
echo "Examples:"
echo "  recipe-book list"
echo "  recipe-book search 'chocolate cake'"
echo "  recipe-book generate 'healthy dinner for two'"