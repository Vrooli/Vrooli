#!/bin/bash

# Quiz Generator CLI Installation Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_NAME="quiz-generator"
INSTALL_DIR="$HOME/.vrooli/bin"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo "Installing Quiz Generator CLI..."

# Create install directory if it doesn't exist
if [ ! -d "$INSTALL_DIR" ]; then
    echo "Creating installation directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
fi

# Create symlink to the CLI
if [ -f "$SCRIPT_DIR/$BINARY_NAME" ]; then
    ln -sf "$SCRIPT_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    echo -e "${GREEN}✓ CLI installed to $INSTALL_DIR/$BINARY_NAME${NC}"
else
    echo -e "${RED}Error: CLI binary not found at $SCRIPT_DIR/$BINARY_NAME${NC}"
    exit 1
fi

# Check if ~/.vrooli/bin is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${YELLOW}Adding $INSTALL_DIR to PATH...${NC}"
    
    # Detect shell and update appropriate config file
    if [ -n "$ZSH_VERSION" ]; then
        SHELL_CONFIG="$HOME/.zshrc"
    elif [ -n "$BASH_VERSION" ]; then
        SHELL_CONFIG="$HOME/.bashrc"
    else
        SHELL_CONFIG="$HOME/.profile"
    fi
    
    # Add to PATH
    echo "" >> "$SHELL_CONFIG"
    echo "# Quiz Generator CLI" >> "$SHELL_CONFIG"
    echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_CONFIG"
    
    echo -e "${GREEN}✓ Added to PATH in $SHELL_CONFIG${NC}"
    echo -e "${YELLOW}Please run 'source $SHELL_CONFIG' or restart your terminal${NC}"
else
    echo -e "${GREEN}✓ $INSTALL_DIR already in PATH${NC}"
fi

# Create config directory
CONFIG_DIR="$HOME/.vrooli/quiz-generator"
if [ ! -d "$CONFIG_DIR" ]; then
    mkdir -p "$CONFIG_DIR"
    echo -e "${GREEN}✓ Created config directory: $CONFIG_DIR${NC}"
fi

# Create default config if it doesn't exist
CONFIG_FILE="$CONFIG_DIR/config.yaml"
if [ ! -f "$CONFIG_FILE" ]; then
    cat > "$CONFIG_FILE" << EOF
# Quiz Generator Configuration
api:
  base_url: http://localhost:3250/api/v1
  timeout: 30

defaults:
  question_count: 10
  difficulty: medium
  question_types:
    - mcq
    - true_false
    - short_answer

export:
  default_format: json
  output_directory: ~/quiz-exports
EOF
    echo -e "${GREEN}✓ Created default config: $CONFIG_FILE${NC}"
fi

echo ""
echo -e "${GREEN}Installation complete!${NC}"
echo ""
echo "Usage: quiz-generator <command> [options]"
echo "Run 'quiz-generator help' for more information"
echo ""

# Test installation
if command -v quiz-generator &> /dev/null; then
    echo -e "${GREEN}✓ CLI is accessible from command line${NC}"
else
    echo -e "${YELLOW}Note: You may need to reload your shell or add $INSTALL_DIR to PATH${NC}"
fi