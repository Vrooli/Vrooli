#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Installing Scenario Auditor CLI...${NC}"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed. Please install Go first.${NC}"
    exit 1
fi

# Build the CLI
echo -e "${YELLOW}Building CLI binary...${NC}"
go mod download
go build -o scenario-auditor .

# Determine installation directory
if [ -n "$VROOLI_CLI_DIR" ]; then
    INSTALL_DIR="$VROOLI_CLI_DIR"
elif [ -d "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
elif [ -d "$HOME/bin" ]; then
    INSTALL_DIR="$HOME/bin"
else
    INSTALL_DIR="/usr/local/bin"
fi

# Check if we need sudo for installation
if [ ! -w "$INSTALL_DIR" ]; then
    echo -e "${YELLOW}Installing to $INSTALL_DIR (requires sudo)...${NC}"
    sudo cp scenario-auditor "$INSTALL_DIR/"
    sudo chmod +x "$INSTALL_DIR/scenario-auditor"
else
    echo -e "${YELLOW}Installing to $INSTALL_DIR...${NC}"
    cp scenario-auditor "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/scenario-auditor"
fi

# Verify installation
if command -v scenario-auditor &> /dev/null; then
    echo -e "${GREEN}✅ Scenario Auditor CLI installed successfully!${NC}"
    echo -e "${GREEN}Try: scenario-auditor help${NC}"
else
    echo -e "${YELLOW}⚠️  CLI installed but not in PATH. Add $INSTALL_DIR to your PATH.${NC}"
fi

# Clean up
rm -f scenario-auditor