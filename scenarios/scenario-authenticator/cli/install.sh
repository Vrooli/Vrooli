#!/bin/bash
# Scenario Authenticator CLI Installation Script

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Ensure the CLI is executable
chmod +x scenario-authenticator

# Create ~/.local/bin if it doesn't exist
mkdir -p ~/.local/bin

# Create symlink
ln -sf "$(pwd)/scenario-authenticator" ~/.local/bin/scenario-authenticator

# Check if ~/.local/bin is in PATH
if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    echo -e "${YELLOW}⚠ Warning: ~/.local/bin is not in your PATH${NC}"
    echo "Add the following to your ~/.bashrc or ~/.zshrc:"
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
else
    echo -e "${GREEN}✓ scenario-authenticator installed successfully${NC}"
    echo "You can now use: scenario-authenticator --help"
fi