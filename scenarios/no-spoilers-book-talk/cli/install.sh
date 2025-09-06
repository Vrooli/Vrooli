#!/bin/bash

# No Spoilers Book Talk CLI Installation Script
# Installs the CLI command globally for easy access

set -e

CLI_NAME="no-spoilers-book-talk"
CLI_SOURCE="$(dirname "$0")/${CLI_NAME}"
INSTALL_DIR="${HOME}/.vrooli/bin"
INSTALL_PATH="${INSTALL_DIR}/${CLI_NAME}"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Verify source CLI exists
if [[ ! -f "$CLI_SOURCE" ]]; then
    log_error "CLI source not found: $CLI_SOURCE"
    exit 1
fi

# Create install directory
log_info "Creating install directory: $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"

# Copy CLI to install location
log_info "Installing CLI to: $INSTALL_PATH"
cp "$CLI_SOURCE" "$INSTALL_PATH"
chmod +x "$INSTALL_PATH"

# Check if install directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    log_warn "Install directory not in PATH: $INSTALL_DIR"
    
    # Determine shell config file
    SHELL_CONFIG=""
    if [[ -n "$BASH_VERSION" ]]; then
        if [[ -f "$HOME/.bashrc" ]]; then
            SHELL_CONFIG="$HOME/.bashrc"
        elif [[ -f "$HOME/.bash_profile" ]]; then
            SHELL_CONFIG="$HOME/.bash_profile"
        fi
    elif [[ -n "$ZSH_VERSION" ]]; then
        SHELL_CONFIG="$HOME/.zshrc"
    elif [[ "$SHELL" == *"zsh"* ]]; then
        SHELL_CONFIG="$HOME/.zshrc"
    elif [[ "$SHELL" == *"bash"* ]]; then
        if [[ -f "$HOME/.bashrc" ]]; then
            SHELL_CONFIG="$HOME/.bashrc"
        else
            SHELL_CONFIG="$HOME/.bash_profile"
        fi
    fi

    if [[ -n "$SHELL_CONFIG" ]]; then
        log_info "Adding $INSTALL_DIR to PATH in $SHELL_CONFIG"
        echo "" >> "$SHELL_CONFIG"
        echo "# Added by no-spoilers-book-talk installer" >> "$SHELL_CONFIG"
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_CONFIG"
        log_info "Please run 'source $SHELL_CONFIG' or restart your terminal"
    else
        log_warn "Could not determine shell config file"
        log_warn "Please manually add $INSTALL_DIR to your PATH"
    fi
    
    # Export for current session
    export PATH="$PATH:$INSTALL_DIR"
fi

# Test installation
log_info "Testing installation..."
if command -v "$CLI_NAME" >/dev/null 2>&1; then
    log_info "✅ Installation successful!"
    echo ""
    echo "CLI installed as: $CLI_NAME"
    echo "Try: $CLI_NAME --help"
    echo ""
    
    # Show version
    "$CLI_NAME" version
else
    log_error "❌ Installation failed - CLI not found in PATH"
    log_error "You may need to restart your terminal or run:"
    log_error "  export PATH=\"\$PATH:$INSTALL_DIR\""
    exit 1
fi

# Create completion if bash-completion is available
COMPLETION_DIR="$HOME/.bash_completion.d"
if [[ -d "$COMPLETION_DIR" ]] || [[ -d "/etc/bash_completion.d" ]]; then
    log_info "Setting up bash completion..."
    
    # Create basic completion script
    COMPLETION_SCRIPT="$COMPLETION_DIR/${CLI_NAME}"
    mkdir -p "$COMPLETION_DIR"
    
    cat > "$COMPLETION_SCRIPT" << 'EOF'
# Bash completion for no-spoilers-book-talk CLI

_no_spoilers_book_talk_completions() {
    local cur prev opts commands
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    
    # Available commands
    commands="help version status upload list show chat progress conversations"
    
    # Global options
    opts="--user-id --json --verbose --debug --help"
    
    case $prev in
        no-spoilers-book-talk)
            COMPREPLY=( $(compgen -W "$commands" -- $cur) )
            return 0
            ;;
        upload)
            # File completion for upload
            COMPREPLY=( $(compgen -f -- $cur) )
            return 0
            ;;
        show|chat|progress|conversations)
            # TODO: Complete with book IDs if API is available
            return 0
            ;;
        --user-id|--title|--author|--position|--notes)
            # Don't complete these, let user type
            return 0
            ;;
        *)
            COMPREPLY=( $(compgen -W "$opts" -- $cur) )
            return 0
            ;;
    esac
}

complete -F _no_spoilers_book_talk_completions no-spoilers-book-talk
EOF

    log_info "Bash completion installed to: $COMPLETION_SCRIPT"
fi

log_info ""
log_info "🎉 Installation complete!"
log_info ""
log_info "Next steps:"
log_info "1. Upload a book: $CLI_NAME upload /path/to/book.txt"
log_info "2. Check status: $CLI_NAME status"
log_info "3. Start reading: $CLI_NAME progress <book-id> --set-position 1"
log_info "4. Ask questions: $CLI_NAME chat <book-id> 'What do you think about...'"
log_info ""
log_info "For help: $CLI_NAME --help"