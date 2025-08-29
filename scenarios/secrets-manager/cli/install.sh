#!/usr/bin/env bash
# Secrets Manager CLI Installation Script
# Installs the secrets-manager command globally via symlink

set -euo pipefail

CLI_NAME="secrets-manager"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SCRIPT="${SCRIPT_DIR}/${CLI_NAME}"

# Color codes for dark chrome aesthetic
RED='\033[1;31m'
MATRIX_GREEN='\033[1;92m'
SILVER='\033[0;37m'
CYAN='\033[1;36m'
NC='\033[0m' # No Color

echo -e "${SILVER}🔐 Installing Secrets Manager CLI...${NC}"

# Ensure CLI script exists and is executable
if [[ ! -f "$CLI_SCRIPT" ]]; then
    echo -e "${RED}❌ CLI script not found at: $CLI_SCRIPT${NC}" >&2
    exit 1
fi

if [[ ! -x "$CLI_SCRIPT" ]]; then
    echo -e "${SILVER}   Making CLI script executable...${NC}"
    chmod +x "$CLI_SCRIPT"
fi

# Create CLI directory if it doesn't exist
CLI_DIR="$HOME/.vrooli/bin"
if [[ ! -d "$CLI_DIR" ]]; then
    echo -e "${SILVER}   Creating CLI directory: $CLI_DIR${NC}"
    mkdir -p "$CLI_DIR"
fi

# Create symlink
SYMLINK_TARGET="${CLI_DIR}/${CLI_NAME}"
if [[ -L "$SYMLINK_TARGET" ]]; then
    echo -e "${SILVER}   Removing existing symlink...${NC}"
    rm "$SYMLINK_TARGET"
fi

echo -e "${SILVER}   Creating symlink: $SYMLINK_TARGET -> $CLI_SCRIPT${NC}"
ln -s "$CLI_SCRIPT" "$SYMLINK_TARGET"

# Check if CLI directory is in PATH
if [[ ":$PATH:" != *":$CLI_DIR:"* ]]; then
    echo -e "${CYAN}   Adding $CLI_DIR to PATH...${NC}"
    
    # Determine shell configuration file
    SHELL_CONFIG=""
    if [[ -n "${BASH_VERSION:-}" ]]; then
        if [[ -f "$HOME/.bashrc" ]]; then
            SHELL_CONFIG="$HOME/.bashrc"
        elif [[ -f "$HOME/.bash_profile" ]]; then
            SHELL_CONFIG="$HOME/.bash_profile"
        fi
    elif [[ -n "${ZSH_VERSION:-}" ]]; then
        SHELL_CONFIG="$HOME/.zshrc"
    fi
    
    if [[ -n "$SHELL_CONFIG" ]]; then
        # Add PATH export if not already present
        if ! grep -q "export PATH.*\.vrooli/bin" "$SHELL_CONFIG" 2>/dev/null; then
            echo "" >> "$SHELL_CONFIG"
            echo "# Vrooli CLI tools" >> "$SHELL_CONFIG"
            echo 'export PATH="$HOME/.vrooli/bin:$PATH"' >> "$SHELL_CONFIG"
            echo -e "${CYAN}   Updated $SHELL_CONFIG${NC}"
            echo -e "${CYAN}   Please run: source $SHELL_CONFIG${NC}"
        fi
    else
        echo -e "${CYAN}   Please add $CLI_DIR to your PATH manually${NC}"
    fi
fi

# Verify installation
if command -v "$CLI_NAME" >/dev/null 2>&1; then
    echo -e "${MATRIX_GREEN}✅ Secrets Manager CLI installed successfully!${NC}"
    echo -e "${SILVER}   Try: $CLI_NAME --help${NC}"
else
    echo -e "${CYAN}⚠️  Installation complete but CLI not in current PATH${NC}"
    echo -e "${SILVER}   Try: $CLI_DIR/$CLI_NAME --help${NC}"
    echo -e "${SILVER}   Or source your shell config to update PATH${NC}"
fi

echo -e "${SILVER}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${MATRIX_GREEN}🔐 SECURITY VAULT ACCESS TERMINAL READY${NC}"
echo -e "${SILVER}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"