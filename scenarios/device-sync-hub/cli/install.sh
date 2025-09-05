#!/usr/bin/env bash

# Device Sync Hub CLI Installation Script
# Installs the CLI to ~/.vrooli/bin and updates PATH if needed

set -euo pipefail

CLI_NAME="device-sync-hub"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SOURCE="$SCRIPT_DIR/$CLI_NAME"
VROOLI_BIN_DIR="$HOME/.vrooli/bin"
CLI_TARGET="$VROOLI_BIN_DIR/$CLI_NAME"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
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
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_bold() {
    echo -e "${BOLD}$1${NC}"
}

# Check if CLI source exists
if [[ ! -f "$CLI_SOURCE" ]]; then
    log_error "CLI source not found: $CLI_SOURCE"
    exit 1
fi

# Create Vrooli bin directory if it doesn't exist
if [[ ! -d "$VROOLI_BIN_DIR" ]]; then
    log_info "Creating Vrooli bin directory: $VROOLI_BIN_DIR"
    mkdir -p "$VROOLI_BIN_DIR"
fi

# Copy CLI to target location
log_info "Installing $CLI_NAME to $CLI_TARGET"
cp "$CLI_SOURCE" "$CLI_TARGET"
chmod +x "$CLI_TARGET"

# Check if ~/.vrooli/bin is in PATH
if [[ ":$PATH:" != *":$VROOLI_BIN_DIR:"* ]]; then
    log_warning "~/.vrooli/bin is not in your PATH"
    
    # Determine shell and profile file
    local shell_name
    shell_name=$(basename "$SHELL")
    
    local profile_files=()
    case "$shell_name" in
        bash)
            profile_files=("$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.profile")
            ;;
        zsh)
            profile_files=("$HOME/.zshrc" "$HOME/.zprofile")
            ;;
        fish)
            profile_files=("$HOME/.config/fish/config.fish")
            ;;
        *)
            profile_files=("$HOME/.profile")
            ;;
    esac
    
    # Find the first existing profile file
    local profile_file=""
    for file in "${profile_files[@]}"; do
        if [[ -f "$file" ]]; then
            profile_file="$file"
            break
        fi
    done
    
    # If no profile file exists, create .profile
    if [[ -z "$profile_file" ]]; then
        profile_file="$HOME/.profile"
        touch "$profile_file"
    fi
    
    # Check if PATH export already exists
    if ! grep -q "\.vrooli/bin" "$profile_file" 2>/dev/null; then
        echo '' >> "$profile_file"
        echo '# Added by Device Sync Hub CLI installer' >> "$profile_file"
        echo 'export PATH="$HOME/.vrooli/bin:$PATH"' >> "$profile_file"
        
        log_success "Added ~/.vrooli/bin to PATH in $profile_file"
        log_info "Please run 'source $profile_file' or restart your terminal to update PATH"
    else
        log_info "PATH already includes ~/.vrooli/bin in $profile_file"
    fi
fi

# Verify installation
if [[ -x "$CLI_TARGET" ]]; then
    log_success "CLI installed successfully to $CLI_TARGET"
    
    # Test the CLI
    if "$CLI_TARGET" version >/dev/null 2>&1; then
        log_success "CLI is working correctly"
        
        echo ""
        log_bold "Installation Complete!"
        echo "Run '$CLI_NAME help' to get started"
        echo ""
        
        # Show quick start instructions
        log_info "Quick Start:"
        echo "  1. $CLI_NAME login          # Authenticate with the service"
        echo "  2. $CLI_NAME status         # Check service status"
        echo "  3. $CLI_NAME upload file.txt # Upload a file"
        echo "  4. $CLI_NAME list           # List all synced items"
        
        # Show PATH warning if needed
        if [[ ":$PATH:" != *":$VROOLI_BIN_DIR:"* ]]; then
            echo ""
            log_warning "Note: You may need to restart your terminal or run:"
            echo "  source $profile_file"
            echo "  to use the '$CLI_NAME' command from anywhere."
        fi
    else
        log_error "CLI installation succeeded but CLI is not working correctly"
        exit 1
    fi
else
    log_error "CLI installation failed"
    exit 1
fi

echo ""
log_info "For more information, visit the Device Sync Hub documentation"