#!/bin/bash

# Symbol Search CLI Installer
# Installs the symbol-search command globally for CLI access

set -e

# Configuration
CLI_NAME="symbol-search"
VROOLI_BIN_DIR="$HOME/.vrooli/bin"
CLI_SOURCE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$CLI_NAME"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create .vrooli/bin directory if it doesn't exist
if [ ! -d "$VROOLI_BIN_DIR" ]; then
    log_info "Creating $VROOLI_BIN_DIR directory..."
    mkdir -p "$VROOLI_BIN_DIR"
fi

# Check if CLI source exists
if [ ! -f "$CLI_SOURCE" ]; then
    log_error "CLI source not found at $CLI_SOURCE"
    exit 1
fi

# Check if CLI source is executable
if [ ! -x "$CLI_SOURCE" ]; then
    log_info "Making CLI source executable..."
    chmod +x "$CLI_SOURCE"
fi

# Create symlink in .vrooli/bin
CLI_TARGET="$VROOLI_BIN_DIR/$CLI_NAME"

if [ -L "$CLI_TARGET" ]; then
    log_info "Removing existing symlink..."
    rm "$CLI_TARGET"
elif [ -f "$CLI_TARGET" ]; then
    log_warning "File exists at $CLI_TARGET, backing up..."
    mv "$CLI_TARGET" "$CLI_TARGET.backup.$(date +%s)"
fi

log_info "Creating symlink: $CLI_TARGET -> $CLI_SOURCE"
ln -s "$CLI_SOURCE" "$CLI_TARGET"

# Check if .vrooli/bin is in PATH
if [[ ":$PATH:" != *":$VROOLI_BIN_DIR:"* ]]; then
    log_warning "$VROOLI_BIN_DIR is not in your PATH"
    
    # Detect shell and add to appropriate rc file
    SHELL_RC=""
    if [ -n "$ZSH_VERSION" ] || [[ "$SHELL" == *"zsh"* ]]; then
        SHELL_RC="$HOME/.zshrc"
    elif [ -n "$BASH_VERSION" ] || [[ "$SHELL" == *"bash"* ]]; then
        if [ -f "$HOME/.bashrc" ]; then
            SHELL_RC="$HOME/.bashrc"
        elif [ -f "$HOME/.bash_profile" ]; then
            SHELL_RC="$HOME/.bash_profile"
        fi
    fi
    
    if [ -n "$SHELL_RC" ] && [ -f "$SHELL_RC" ]; then
        log_info "Adding $VROOLI_BIN_DIR to PATH in $SHELL_RC"
        
        # Check if PATH update already exists
        if ! grep -q "\.vrooli/bin" "$SHELL_RC"; then
            echo "" >> "$SHELL_RC"
            echo "# Added by Symbol Search CLI installer" >> "$SHELL_RC"
            echo 'export PATH="$HOME/.vrooli/bin:$PATH"' >> "$SHELL_RC"
            log_success "Added PATH update to $SHELL_RC"
            log_info "Run 'source $SHELL_RC' or restart your terminal to use the CLI"
        else
            log_info "PATH already contains .vrooli/bin reference in $SHELL_RC"
        fi
    else
        log_warning "Could not detect shell configuration file"
        log_info "Manually add the following to your shell's rc file:"
        echo 'export PATH="$HOME/.vrooli/bin:$PATH"'
    fi
else
    log_success "$VROOLI_BIN_DIR is already in your PATH"
fi

# Test the installation
log_info "Testing CLI installation..."

# Add .vrooli/bin to current PATH for testing
export PATH="$VROOLI_BIN_DIR:$PATH"

if command -v "$CLI_NAME" >/dev/null 2>&1; then
    log_success "CLI installed successfully!"
    
    # Check dependencies
    log_info "Checking dependencies..."
    
    missing_deps=()
    if ! command -v curl >/dev/null 2>&1; then
        missing_deps+=("curl")
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        missing_deps+=("jq")
    fi
    
    if [ ${#missing_deps[@]} -eq 0 ]; then
        log_success "All dependencies are installed"
        
        # Show version info
        log_info "Testing CLI command..."
        "$CLI_NAME" version
        
        echo ""
        log_success "Installation complete!"
        echo ""
        echo "Usage:"
        echo "  $CLI_NAME help                    - Show help"
        echo "  $CLI_NAME status                  - Check service status"  
        echo "  $CLI_NAME search heart            - Search for symbols"
        echo "  $CLI_NAME character U+1F600      - Get character details"
        echo ""
    else
        log_warning "Missing dependencies: ${missing_deps[*]}"
        log_info "Install missing dependencies with your package manager:"
        for dep in "${missing_deps[@]}"; do
            echo "  - $dep"
        done
    fi
    
else
    log_error "CLI installation failed - command not found in PATH"
    log_info "Try running: export PATH=\"$VROOLI_BIN_DIR:\$PATH\""
    exit 1
fi