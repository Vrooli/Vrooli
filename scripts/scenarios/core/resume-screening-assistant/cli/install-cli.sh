#!/bin/bash
# Install script for Resume Screening Assistant CLI

set -euo pipefail

CLI_NAME="resume-screening-assistant"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SCRIPT="$SCRIPT_DIR/$CLI_NAME"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Installing $CLI_NAME CLI...${NC}"

# Check if the CLI script exists
if [[ ! -f "$CLI_SCRIPT" ]]; then
    echo "Error: CLI script not found at $CLI_SCRIPT"
    exit 1
fi

# Make script executable
chmod +x "$CLI_SCRIPT"

# Create symlink in /usr/local/bin for global access
if [[ -w /usr/local/bin ]]; then
    ln -sf "$CLI_SCRIPT" "/usr/local/bin/$CLI_NAME"
    echo -e "${GREEN}✅ $CLI_NAME installed globally at /usr/local/bin/$CLI_NAME${NC}"
elif sudo -n true 2>/dev/null; then
    sudo ln -sf "$CLI_SCRIPT" "/usr/local/bin/$CLI_NAME"
    echo -e "${GREEN}✅ $CLI_NAME installed globally at /usr/local/bin/$CLI_NAME (with sudo)${NC}"
else
    echo "Warning: Cannot install globally (no write access to /usr/local/bin)"
    echo "You can still use the CLI from: $CLI_SCRIPT"
    echo "Or add $SCRIPT_DIR to your PATH"
    exit 0
fi

echo
echo "Installation complete! Test with:"
echo "  $CLI_NAME --help"
echo "  $CLI_NAME status"