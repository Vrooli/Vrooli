#!/bin/bash

# CLI Installation Script for scenario-to-extension
# This script installs the scenario-to-extension CLI into the user's PATH

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

# Configuration
CLI_NAME="scenario-to-extension"
CLI_BINARY="scenario-to-extension"
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLI_SOURCE="$SCENARIO_DIR/cli/$CLI_BINARY"
VROOLI_BIN_DIR="$HOME/.vrooli/bin"
CLI_SYMLINK="$VROOLI_BIN_DIR/$CLI_BINARY"

# Helper functions
log() {
    echo -e "${GREEN}[INSTALL]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check if CLI source exists
    if [[ ! -f "$CLI_SOURCE" ]]; then
        error "CLI binary not found at $CLI_SOURCE"
    fi
    
    # Check if CLI source is executable
    if [[ ! -x "$CLI_SOURCE" ]]; then
        log "Making CLI binary executable..."
        chmod +x "$CLI_SOURCE"
    fi
    
    # Check for required dependencies
    local missing_deps=()
    
    if ! command -v curl >/dev/null 2>&1; then
        missing_deps+=("curl")
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        missing_deps+=("jq")
    fi
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        error "Missing required dependencies: ${missing_deps[*]}. Please install them first."
    fi
    
    log "Prerequisites check passed"
}

# Create Vrooli bin directory
create_bin_directory() {
    log "Setting up Vrooli CLI directory..."
    
    if [[ ! -d "$VROOLI_BIN_DIR" ]]; then
        mkdir -p "$VROOLI_BIN_DIR"
        log "Created directory: $VROOLI_BIN_DIR"
    fi
}

# Install CLI binary
install_cli() {
    log "Installing $CLI_NAME CLI..."
    
    # Remove existing symlink if it exists
    if [[ -L "$CLI_SYMLINK" ]]; then
        rm "$CLI_SYMLINK"
        log "Removed existing symlink"
    elif [[ -f "$CLI_SYMLINK" ]]; then
        warn "Existing file at $CLI_SYMLINK will be overwritten"
        rm "$CLI_SYMLINK"
    fi
    
    # Create symlink
    ln -s "$CLI_SOURCE" "$CLI_SYMLINK"
    log "Created symlink: $CLI_SYMLINK -> $CLI_SOURCE"
    
    # Verify installation
    if [[ -x "$CLI_SYMLINK" ]]; then
        log "CLI installed successfully"
    else
        error "CLI installation failed - symlink not executable"
    fi
}

# Update PATH
update_path() {
    log "Configuring PATH..."
    
    # Check if Vrooli bin directory is already in PATH
    if [[ ":$PATH:" == *":$VROOLI_BIN_DIR:"* ]]; then
        log "PATH already includes $VROOLI_BIN_DIR"
        return 0
    fi
    
    # Add to various shell configurations
    local shell_configs=(
        "$HOME/.bashrc"
        "$HOME/.bash_profile"
        "$HOME/.zshrc"
        "$HOME/.profile"
    )
    
    local path_export="export PATH=\"\$PATH:$VROOLI_BIN_DIR\""
    local path_comment="# Added by scenario-to-extension CLI installer"
    local added_to_files=()
    
    for config_file in "${shell_configs[@]}"; do
        if [[ -f "$config_file" ]]; then
            # Check if path is already in this config file
            if grep -q "$VROOLI_BIN_DIR" "$config_file" 2>/dev/null; then
                log "PATH already configured in $config_file"
                continue
            fi
            
            # Add to this config file
            echo "" >> "$config_file"
            echo "$path_comment" >> "$config_file"
            echo "$path_export" >> "$config_file"
            added_to_files+=("$config_file")
        fi
    done
    
    if [[ ${#added_to_files[@]} -gt 0 ]]; then
        log "Added PATH configuration to: ${added_to_files[*]}"
        warn "You may need to restart your shell or run 'source ~/.bashrc' (or equivalent)"
    else
        warn "No shell configuration files found to update"
        log "Please add the following to your shell configuration:"
        echo "    $path_export"
    fi
}

# Test installation
test_installation() {
    log "Testing installation..."
    
    # Test direct execution
    if "$CLI_SYMLINK" version >/dev/null 2>&1; then
        log "Direct execution test: PASSED"
    else
        error "Direct execution test: FAILED"
    fi
    
    # Test PATH execution (in a new shell context)
    if bash -c "export PATH=\"\$PATH:$VROOLI_BIN_DIR\"; $CLI_BINARY version" >/dev/null 2>&1; then
        log "PATH execution test: PASSED"
    else
        warn "PATH execution test: FAILED (PATH may not be updated yet)"
    fi
    
    # Show version information
    local version_info
    version_info=$("$CLI_SYMLINK" version 2>/dev/null || echo "Version information unavailable")
    log "Installation test completed"
}

# Show installation summary
show_summary() {
    echo
    echo -e "${BOLD}Installation Summary${NC}"
    echo -e "CLI Name:        $CLI_NAME"
    echo -e "Binary Location: $CLI_SYMLINK"
    echo -e "Source Location: $CLI_SOURCE"
    echo -e "Bin Directory:   $VROOLI_BIN_DIR"
    echo
    echo -e "${BOLD}Usage Examples:${NC}"
    echo -e "  $CLI_BINARY help                              Show help information"
    echo -e "  $CLI_BINARY status                            Check system status"
    echo -e "  $CLI_BINARY generate my-scenario              Generate extension"
    echo -e "  $CLI_BINARY test ./platforms/extension        Test extension"
    echo
    echo -e "${BOLD}Quick Start:${NC}"
    echo -e "  1. Run '${CYAN}$CLI_BINARY status${NC}' to verify the service is running"
    echo -e "  2. Use '${CYAN}$CLI_BINARY templates${NC}' to see available templates"
    echo -e "  3. Generate your first extension with '${CYAN}$CLI_BINARY generate my-scenario${NC}'"
    echo
    echo -e "${BOLD}Documentation:${NC}"
    echo -e "  For more information, run: ${CYAN}$CLI_BINARY help${NC}"
    echo -e "  Project repository: https://github.com/vrooli/vrooli"
    echo
}

# Main installation process
main() {
    echo
    echo -e "${BOLD}scenario-to-extension CLI Installer${NC}"
    echo -e "Installing CLI for Browser Extension Generator"
    echo
    
    check_prerequisites
    create_bin_directory
    install_cli
    update_path
    test_installation
    
    echo
    echo -e "${GREEN}✓ Installation completed successfully!${NC}"
    
    show_summary
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        cat << EOF
scenario-to-extension CLI Installer

USAGE:
    install.sh [OPTIONS]

OPTIONS:
    --help, -h     Show this help message
    --uninstall    Uninstall the CLI
    --force        Force reinstallation even if already installed

DESCRIPTION:
    This script installs the scenario-to-extension CLI by creating a symlink
    in ~/.vrooli/bin/ and updating your PATH configuration.

    The CLI provides commands for generating, building, and testing browser
    extensions from Vrooli scenarios.

EXAMPLES:
    ./install.sh                Install the CLI
    ./install.sh --uninstall    Remove the CLI
    ./install.sh --force        Force reinstall

EOF
        exit 0
        ;;
    --uninstall)
        log "Uninstalling $CLI_NAME CLI..."
        
        # Remove symlink
        if [[ -L "$CLI_SYMLINK" ]]; then
            rm "$CLI_SYMLINK"
            log "Removed CLI symlink"
        else
            log "CLI symlink not found"
        fi
        
        # Note: We don't remove PATH entries as they may be used by other scenarios
        warn "PATH entries were not removed (may be used by other scenarios)"
        
        echo -e "${GREEN}✓ Uninstallation completed${NC}"
        exit 0
        ;;
    --force)
        log "Force installation mode enabled"
        ;;
    "")
        # Default installation
        ;;
    *)
        error "Unknown option: $1. Use --help for usage information."
        ;;
esac

# Check if already installed (unless force mode)
if [[ "$1" != "--force" ]] && [[ -x "$CLI_SYMLINK" ]]; then
    warn "CLI already installed at $CLI_SYMLINK"
    echo "Use --force to reinstall, or --uninstall to remove"
    
    # Test existing installation
    if "$CLI_SYMLINK" version >/dev/null 2>&1; then
        log "Existing installation is functional"
        show_summary
        exit 0
    else
        warn "Existing installation appears broken, proceeding with reinstall..."
    fi
fi

# Run main installation
main