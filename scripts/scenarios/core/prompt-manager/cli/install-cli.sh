#!/usr/bin/env bash
# Install script for Prompt Manager CLI

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="prompt-manager"

# Color codes
RED='\033[1;31m'
GREEN='\033[1;32m'
YELLOW='\033[1;33m'
BLUE='\033[1;34m'
NC='\033[0m' # No Color

info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

warn() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

error() {
    echo -e "${RED}❌ Error: $1${NC}" >&2
    exit 1
}

# Check for required dependencies
check_dependencies() {
    info "Checking dependencies..."
    
    local missing_deps=()
    
    if ! command -v curl >/dev/null 2>&1; then
        missing_deps+=("curl")
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        missing_deps+=("jq")
    fi
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        error "Missing required dependencies: ${missing_deps[*]}"
        echo "Please install them and try again."
        echo ""
        echo "On Ubuntu/Debian: sudo apt-get install ${missing_deps[*]}"
        echo "On macOS: brew install ${missing_deps[*]}"
        echo "On CentOS/RHEL: sudo yum install ${missing_deps[*]}"
        exit 1
    fi
    
    success "All dependencies are available"
}

# Install CLI to system PATH
install_cli() {
    local source_file="$SCRIPT_DIR/$CLI_NAME"
    local target_dir=""
    
    # Determine installation directory
    if [[ -w "/usr/local/bin" ]]; then
        target_dir="/usr/local/bin"
    elif [[ -w "$HOME/.local/bin" ]]; then
        target_dir="$HOME/.local/bin"
        # Ensure ~/.local/bin is in PATH
        if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
            warn "Adding $HOME/.local/bin to PATH"
            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"
            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.zshrc" 2>/dev/null || true
        fi
    elif [[ -w "$HOME/bin" ]]; then
        target_dir="$HOME/bin"
        # Ensure ~/bin is in PATH
        if [[ ":$PATH:" != *":$HOME/bin:"* ]]; then
            warn "Adding $HOME/bin to PATH"
            echo 'export PATH="$HOME/bin:$PATH"' >> "$HOME/.bashrc"
            echo 'export PATH="$HOME/bin:$PATH"' >> "$HOME/.zshrc" 2>/dev/null || true
        fi
    else
        mkdir -p "$HOME/.local/bin"
        target_dir="$HOME/.local/bin"
        echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"
        echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.zshrc" 2>/dev/null || true
    fi
    
    info "Installing $CLI_NAME to $target_dir"
    
    cp "$source_file" "$target_dir/$CLI_NAME"
    chmod +x "$target_dir/$CLI_NAME"
    
    success "$CLI_NAME installed successfully!"
    info "Location: $target_dir/$CLI_NAME"
    
    # Test installation
    if command -v "$CLI_NAME" >/dev/null 2>&1; then
        success "CLI is available in PATH"
        info "Try running: $CLI_NAME help"
    else
        warn "CLI may not be immediately available in current shell"
        info "You may need to restart your shell or run: source ~/.bashrc"
        info "Or run directly: $target_dir/$CLI_NAME help"
    fi
}

# Create desktop entry (Linux only)
create_desktop_entry() {
    if [[ "$OSTYPE" == "linux-gnu"* ]] && command -v desktop-file-install >/dev/null 2>&1; then
        local desktop_dir="$HOME/.local/share/applications"
        mkdir -p "$desktop_dir"
        
        cat > "$desktop_dir/prompt-manager.desktop" << EOF
[Desktop Entry]
Name=Prompt Manager
Comment=Personal prompt storage and management tool
Exec=x-terminal-emulator -e prompt-manager
Icon=text-editor
Terminal=true
Type=Application
Categories=Development;Office;Utility;
Keywords=prompt;ai;development;productivity;
EOF
        
        success "Desktop entry created"
    fi
}

# Setup shell completions (basic)
setup_completions() {
    local completion_dir="$HOME/.local/share/bash-completion/completions"
    
    if [[ "$SHELL" == *"bash"* ]] && [[ -d "$HOME/.local/share" ]]; then
        mkdir -p "$completion_dir"
        
        cat > "$completion_dir/prompt-manager" << 'EOF'
_prompt_manager_completions()
{
    local cur prev commands
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    
    commands="status campaigns add list search show use quick favorite help version"
    
    case ${prev} in
        prompt-manager)
            COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
            return 0
            ;;
        campaigns)
            COMPREPLY=( $(compgen -W "list create" -- ${cur}) )
            return 0
            ;;
        list)
            COMPREPLY=( $(compgen -W "favorites recent" -- ${cur}) )
            return 0
            ;;
        *)
            ;;
    esac
}

complete -F _prompt_manager_completions prompt-manager
EOF
        
        success "Bash completions installed"
        info "Completions will be available in new shell sessions"
    fi
}

# Main installation process
main() {
    echo -e "${BLUE}"
    echo "🚀 Prompt Manager CLI Installation"
    echo "=================================="
    echo -e "${NC}"
    
    check_dependencies
    install_cli
    create_desktop_entry
    setup_completions
    
    echo ""
    echo -e "${GREEN}🎉 Installation completed successfully!${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Start the Prompt Manager API server"
    echo "2. Run: $CLI_NAME help"
    echo "3. Create your first campaign: $CLI_NAME campaigns create \"My Campaign\""
    echo "4. Add your first prompt: $CLI_NAME add \"My First Prompt\""
    echo ""
    echo "For more information, visit: https://github.com/Vrooli/Vrooli"
}

# Run installation
main "$@"