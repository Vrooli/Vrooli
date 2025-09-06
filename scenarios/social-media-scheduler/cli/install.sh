#!/bin/bash

# Social Media Scheduler CLI Installation Script
# This script installs the CLI command globally

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

CLI_NAME="social-media-scheduler"
INSTALL_DIR="/usr/local/bin"
SOURCE_FILE="$(dirname "$0")/$CLI_NAME"

log() {
    echo -e "${GREEN}[INSTALL]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Check if running as root for global installation
check_permissions() {
    if [ "$EUID" -ne 0 ]; then
        error "This script must be run with sudo for global installation"
        info "Run: sudo $0"
        exit 1
    fi
}

# Check dependencies
check_dependencies() {
    local missing_deps=()
    
    if ! command -v curl >/dev/null 2>&1; then
        missing_deps+=("curl")
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        missing_deps+=("jq")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        error "Missing required dependencies: ${missing_deps[*]}"
        info "Please install the missing dependencies first:"
        
        # Provide installation instructions for common package managers
        if command -v apt-get >/dev/null 2>&1; then
            info "Ubuntu/Debian: apt-get update && apt-get install -y ${missing_deps[*]}"
        elif command -v yum >/dev/null 2>&1; then
            info "RHEL/CentOS: yum install -y ${missing_deps[*]}"
        elif command -v brew >/dev/null 2>&1; then
            info "macOS: brew install ${missing_deps[*]}"
        fi
        
        exit 1
    fi
}

# Install CLI
install_cli() {
    log "Installing Social Media Scheduler CLI..."
    
    if [ ! -f "$SOURCE_FILE" ]; then
        error "CLI source file not found: $SOURCE_FILE"
        exit 1
    fi
    
    # Copy the CLI script to the installation directory
    cp "$SOURCE_FILE" "$INSTALL_DIR/$CLI_NAME"
    
    # Make it executable
    chmod +x "$INSTALL_DIR/$CLI_NAME"
    
    # Verify installation
    if [ -x "$INSTALL_DIR/$CLI_NAME" ]; then
        log "CLI installed successfully to $INSTALL_DIR/$CLI_NAME"
    else
        error "Failed to install CLI"
        exit 1
    fi
}

# Create completion script (bash completion)
install_completion() {
    local completion_dir="/etc/bash_completion.d"
    local completion_file="$completion_dir/$CLI_NAME"
    
    if [ -d "$completion_dir" ]; then
        log "Installing bash completion..."
        
        cat > "$completion_file" << 'EOF'
_social_media_scheduler() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Main commands
    opts="login register logout whoami schedule list status platforms accounts health version help"
    
    case "${prev}" in
        social-media-scheduler)
            COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
            return 0
            ;;
        schedule)
            if [ ${COMP_CWORD} -eq 4 ]; then
                # Platform suggestions
                COMPREPLY=( $(compgen -W "twitter instagram linkedin facebook twitter,linkedin twitter,instagram linkedin,facebook twitter,instagram,linkedin twitter,instagram,linkedin,facebook" -- ${cur}) )
            fi
            return 0
            ;;
        *)
            ;;
    esac
}

complete -F _social_media_scheduler social-media-scheduler
EOF
        
        chmod 644 "$completion_file"
        log "Bash completion installed to $completion_file"
    else
        warn "Bash completion directory not found, skipping completion installation"
    fi
}

# Test installation
test_installation() {
    log "Testing installation..."
    
    if command -v "$CLI_NAME" >/dev/null 2>&1; then
        log "CLI command is available in PATH"
        
        # Test version command
        if "$CLI_NAME" version >/dev/null 2>&1; then
            log "CLI is working correctly"
        else
            warn "CLI installed but may have issues"
        fi
    else
        error "CLI command not found in PATH"
        info "You may need to reload your shell or add $INSTALL_DIR to your PATH"
        exit 1
    fi
}

# Main installation process
main() {
    echo ""
    log "Social Media Scheduler CLI Installation"
    echo ""
    
    # Perform installation steps
    check_permissions
    check_dependencies
    install_cli
    install_completion
    test_installation
    
    echo ""
    log "Installation completed successfully!"
    echo ""
    info "Usage examples:"
    echo "  $CLI_NAME help                    # Show help"
    echo "  $CLI_NAME health                  # Check system health" 
    echo "  $CLI_NAME login user@example.com # Login to your account"
    echo "  $CLI_NAME platforms               # List supported platforms"
    echo ""
    info "For OAuth social media connections, use the web UI at:"
    info "  http://localhost:\${UI_PORT:-38000}"
    echo ""
    warn "Note: Make sure the API server is running before using CLI commands"
    echo ""
}

# Handle command line arguments
case "${1:-install}" in
    "install"|"")
        main
        ;;
    "uninstall")
        log "Uninstalling Social Media Scheduler CLI..."
        rm -f "$INSTALL_DIR/$CLI_NAME"
        rm -f "/etc/bash_completion.d/$CLI_NAME"
        log "CLI uninstalled successfully"
        ;;
    "help")
        echo "Social Media Scheduler CLI Installer"
        echo ""
        echo "Usage:"
        echo "  sudo $0 [install]    # Install CLI globally (default)"
        echo "  sudo $0 uninstall    # Uninstall CLI"
        echo "  $0 help              # Show this help"
        echo ""
        echo "Requirements:"
        echo "  - curl (for API communication)"
        echo "  - jq (for JSON processing)"
        echo "  - sudo access (for global installation)"
        ;;
    *)
        error "Unknown option: $1"
        echo "Run '$0 help' for usage information"
        exit 1
        ;;
esac