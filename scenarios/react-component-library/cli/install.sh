#!/usr/bin/env bash

# React Component Library CLI Installation Script
# Version: 1.0.0

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✅${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}⚠️${NC} $1"
}

log_error() {
    echo -e "${RED}❌${NC} $1"
}

log_step() {
    echo -e "${GREEN}🔧${NC} $1"
}

# Configuration
CLI_NAME="react-component-library"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SCRIPT="${SCRIPT_DIR}/${CLI_NAME}"
INSTALL_DIR="${HOME}/.vrooli/bin"
INSTALLED_CLI="${INSTALL_DIR}/${CLI_NAME}"

main() {
    log_step "Installing React Component Library CLI..."
    
    # Check if CLI script exists
    if [[ ! -f "$CLI_SCRIPT" ]]; then
        log_error "CLI script not found: $CLI_SCRIPT"
        exit 1
    fi
    
    # Create install directory if it doesn't exist
    if [[ ! -d "$INSTALL_DIR" ]]; then
        log_step "Creating install directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR"
    fi
    
    # Copy CLI script to install directory
    log_step "Installing CLI binary..."
    cp "$CLI_SCRIPT" "$INSTALLED_CLI"
    chmod +x "$INSTALLED_CLI"
    
    # Check if ~/.vrooli/bin is in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        log_warn "~/.vrooli/bin is not in your PATH"
        
        # Detect shell and add to appropriate config file
        local shell_config=""
        if [[ -n "$ZSH_VERSION" ]]; then
            shell_config="$HOME/.zshrc"
        elif [[ -n "$BASH_VERSION" ]]; then
            if [[ -f "$HOME/.bashrc" ]]; then
                shell_config="$HOME/.bashrc"
            elif [[ -f "$HOME/.bash_profile" ]]; then
                shell_config="$HOME/.bash_profile"
            fi
        fi
        
        if [[ -n "$shell_config" ]]; then
            log_step "Adding ~/.vrooli/bin to PATH in $shell_config"
            echo "" >> "$shell_config"
            echo "# Added by React Component Library CLI installer" >> "$shell_config"
            echo 'export PATH="$HOME/.vrooli/bin:$PATH"' >> "$shell_config"
            log_info "Please restart your terminal or run: source $shell_config"
        else
            log_warn "Could not detect shell configuration file"
            log_info "Please add the following to your shell configuration:"
            echo '  export PATH="$HOME/.vrooli/bin:$PATH"'
        fi
    else
        log_success "~/.vrooli/bin is already in PATH"
    fi
    
    # Verify installation
    log_step "Verifying installation..."
    if [[ -x "$INSTALLED_CLI" ]]; then
        log_success "CLI installed successfully!"
        log_info "Installed at: $INSTALLED_CLI"
        
        # Test the CLI
        if command -v "$CLI_NAME" >/dev/null 2>&1; then
            log_success "CLI is accessible via command: $CLI_NAME"
            log_info "Try running: $CLI_NAME --help"
        else
            log_warn "CLI installed but not in PATH. You may need to restart your terminal."
            log_info "Or run directly: $INSTALLED_CLI --help"
        fi
    else
        log_error "Installation failed - CLI not found or not executable"
        exit 1
    fi
    
    echo
    log_success "React Component Library CLI installation complete!"
    echo
    echo "Available commands:"
    echo "  $CLI_NAME help       - Show help"
    echo "  $CLI_NAME status     - Check status"
    echo "  $CLI_NAME create     - Create component"
    echo "  $CLI_NAME search     - Search components"
    echo "  $CLI_NAME generate   - AI generate component"
    echo "  $CLI_NAME test       - Test components"
    echo
    echo "Web interface: http://localhost:31012"
    echo "API documentation: http://localhost:8090/api/docs"
}

main "$@"