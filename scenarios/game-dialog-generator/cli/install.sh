#!/bin/bash

# Game Dialog Generator CLI Installation Script 🌿🎮
# Jungle Platformer Adventure Theme

set -e

# Colors for themed output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}" >&2
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_header() {
    echo -e "${GREEN}🌿 Installing Game Dialog Generator CLI 🎮${NC}"
    echo -e "${GREEN}Jungle Platformer Adventure Theme${NC}"
    echo ""
}

main() {
    print_header
    
    # Check if we're in the right directory
    if [ ! -f "game-dialog-generator" ]; then
        print_error "CLI binary not found in current directory"
        print_info "Make sure you're running this from the cli/ directory"
        exit 1
    fi
    
    # Make CLI executable
    chmod +x game-dialog-generator
    print_success "Made CLI executable"
    
    # Create ~/.local/bin if it doesn't exist
    mkdir -p "$HOME/.local/bin"
    print_success "Created ~/.local/bin directory"
    
    # Create symlink in ~/.local/bin
    if [ -L "$HOME/.local/bin/game-dialog-generator" ]; then
        rm "$HOME/.local/bin/game-dialog-generator"
        print_info "Removed existing symlink"
    fi
    
    ln -sf "$(pwd)/game-dialog-generator" "$HOME/.local/bin/game-dialog-generator"
    print_success "Created symlink in ~/.local/bin"
    
    # Check if ~/.local/bin is in PATH
    if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
        print_info "Adding ~/.local/bin to PATH"
        
        # Add to various shell profiles
        for profile in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
            if [ -f "$profile" ]; then
                if ! grep -q 'export PATH="$HOME/.local/bin:$PATH"' "$profile"; then
                    echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$profile"
                    print_success "Updated $profile"
                fi
            fi
        done
        
        # For current session
        export PATH="$HOME/.local/bin:$PATH"
        print_success "Updated PATH for current session"
    else
        print_success "PATH already includes ~/.local/bin"
    fi
    
    # Check dependencies
    print_info "Checking dependencies..."
    
    if ! command -v jq >/dev/null 2>&1; then
        print_error "jq is required but not installed"
        print_info "Install with: sudo apt-get install jq (Ubuntu/Debian) or brew install jq (macOS)"
        exit 1
    fi
    print_success "jq is available"
    
    if ! command -v curl >/dev/null 2>&1; then
        print_error "curl is required but not installed"
        print_info "Install with: sudo apt-get install curl (Ubuntu/Debian)"
        exit 1
    fi
    print_success "curl is available"
    
    echo ""
    print_success "🌿 Game Dialog Generator CLI installed successfully! 🎮"
    echo ""
    print_info "Test the installation:"
    echo "  game-dialog-generator --help"
    echo "  game-dialog-generator status"
    echo ""
    print_info "Start creating jungle characters:"
    echo "  game-dialog-generator character-create \"Your Hero\" --interactive"
    echo ""
    print_info "Note: You may need to restart your terminal or run:"
    echo "  source ~/.bashrc  (or ~/.zshrc)"
    echo ""
    print_success "Welcome to the jungle adventure! 🌿"
}

main "$@"