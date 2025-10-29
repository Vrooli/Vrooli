#!/bin/bash
# Install Picker Wheel CLI
# Version: 1.0.0

set -euo pipefail

# Validate required environment variables
if [ -z "${HOME:-}" ]; then
    echo "Error: HOME environment variable is not set" >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="picker-wheel"
CLI_PATH="${SCRIPT_DIR}/${CLI_NAME}"
INSTALL_DIR="${HOME}/.vrooli/bin"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "Installing Picker Wheel CLI..."

# Create install directory if it doesn't exist
if [ ! -d "$INSTALL_DIR" ]; then
    echo "Creating directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
fi

# Validate CLI_PATH exists
if [ -z "${CLI_PATH:-}" ] || [ ! -f "$CLI_PATH" ]; then
    echo "Error: CLI binary not found at $CLI_PATH" >&2
    exit 1
fi

# Make CLI executable
chmod +x "$CLI_PATH"

# Create symlink
ln -sf "$CLI_PATH" "$INSTALL_DIR/$CLI_NAME"

# Check if ~/.vrooli/bin is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo -e "${GREEN}Installation complete!${NC}"
    echo ""
    echo "To use the CLI, add the following to your shell configuration:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    echo ""
    echo "Or run this command to add it now:"
    echo "  echo 'export PATH=\"\$PATH:$INSTALL_DIR\"' >> ~/.bashrc && source ~/.bashrc"
else
    echo -e "${GREEN}✓${NC} CLI installed successfully!"
    echo "Run 'picker-wheel help' to get started"
fi

# Test installation
if command -v "$CLI_NAME" &> /dev/null; then
    echo -e "${GREEN}✓${NC} CLI is accessible from PATH"
else
    echo "CLI installed to: $INSTALL_DIR/$CLI_NAME"
    echo "You may need to restart your terminal or source your shell config"
fi
