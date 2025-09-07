#!/bin/bash

# Email Triage CLI Installation Script
# Installs the email-triage CLI tool into the Vrooli CLI ecosystem

set -euo pipefail

CLI_NAME="email-triage"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_BINARY="${SCRIPT_DIR}/${CLI_NAME}"
VROOLI_BIN_DIR="${HOME}/.vrooli/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Check if CLI binary exists
check_binary() {
    if [[ ! -f "$CLI_BINARY" ]]; then
        print_error "CLI binary not found at: $CLI_BINARY"
        exit 1
    fi
    
    if [[ ! -x "$CLI_BINARY" ]]; then
        print_error "CLI binary is not executable: $CLI_BINARY"
        exit 1
    fi
    
    print_status "CLI binary found and is executable"
}

# Create Vrooli bin directory if it doesn't exist
create_bin_dir() {
    if [[ ! -d "$VROOLI_BIN_DIR" ]]; then
        print_status "Creating Vrooli bin directory: $VROOLI_BIN_DIR"
        mkdir -p "$VROOLI_BIN_DIR"
    else
        print_status "Vrooli bin directory exists: $VROOLI_BIN_DIR"
    fi
}

# Install CLI binary
install_binary() {
    local target_binary="${VROOLI_BIN_DIR}/${CLI_NAME}"
    
    # Remove existing installation if present
    if [[ -f "$target_binary" ]] || [[ -L "$target_binary" ]]; then
        print_warning "Removing existing installation: $target_binary"
        rm -f "$target_binary"
    fi
    
    # Create symlink to the CLI binary
    print_status "Creating symlink: $target_binary -> $CLI_BINARY"
    ln -s "$CLI_BINARY" "$target_binary"
    
    # Verify installation
    if [[ -x "$target_binary" ]]; then
        print_success "CLI binary installed successfully"
    else
        print_error "Failed to install CLI binary"
        exit 1
    fi
}

# Update PATH if necessary
update_path() {
    local shell_rc_file=""
    
    # Detect shell and RC file
    if [[ -n "${BASH_VERSION:-}" ]]; then
        shell_rc_file="${HOME}/.bashrc"
    elif [[ -n "${ZSH_VERSION:-}" ]]; then
        shell_rc_file="${HOME}/.zshrc"
    else
        # Try to detect from SHELL environment variable
        case "${SHELL:-}" in
            */bash)
                shell_rc_file="${HOME}/.bashrc"
                ;;
            */zsh)
                shell_rc_file="${HOME}/.zshrc"
                ;;
            */fish)
                shell_rc_file="${HOME}/.config/fish/config.fish"
                ;;
            *)
                print_warning "Unknown shell, PATH may need manual configuration"
                return 0
                ;;
        esac
    fi
    
    # Check if PATH already contains Vrooli bin directory
    if echo "$PATH" | grep -q "$VROOLI_BIN_DIR"; then
        print_status "PATH already contains Vrooli bin directory"
        return 0
    fi
    
    # Add to PATH in shell RC file
    if [[ -n "$shell_rc_file" ]]; then
        print_status "Adding Vrooli bin directory to PATH in: $shell_rc_file"
        
        # Create backup of RC file
        if [[ -f "$shell_rc_file" ]]; then
            cp "$shell_rc_file" "${shell_rc_file}.backup.$(date +%Y%m%d_%H%M%S)"
        fi
        
        # Add PATH export
        echo "" >> "$shell_rc_file"
        echo "# Added by email-triage CLI installation" >> "$shell_rc_file"
        echo "export PATH=\"\$HOME/.vrooli/bin:\$PATH\"" >> "$shell_rc_file"
        
        print_success "PATH updated in $shell_rc_file"
        print_warning "Please run 'source $shell_rc_file' or restart your terminal"
    fi
}

# Create configuration directory and default config
create_config() {
    local config_dir="${HOME}/.vrooli/${CLI_NAME}"
    local config_file="${config_dir}/config.yaml"
    
    if [[ ! -d "$config_dir" ]]; then
        print_status "Creating configuration directory: $config_dir"
        mkdir -p "$config_dir"
    fi
    
    if [[ ! -f "$config_file" ]]; then
        print_status "Creating default configuration: $config_file"
        cat > "$config_file" << EOF
# Email Triage CLI Configuration
api_url: http://localhost:3200
auth_token: ""
default_account: ""
verbose: false

# API endpoints can be customized
endpoints:
  health: /health
  accounts: /api/v1/accounts
  rules: /api/v1/rules
  search: /api/v1/emails/search
  sync: /api/v1/emails/sync

# Default options
defaults:
  search_limit: 20
  json_output: false
  timeout: 30

# Logging
log_level: info
log_file: "${config_dir}/email-triage.log"
EOF
        print_success "Default configuration created"
    else
        print_status "Configuration file already exists: $config_file"
    fi
}

# Test CLI installation
test_installation() {
    print_status "Testing CLI installation..."
    
    local cli_path="${VROOLI_BIN_DIR}/${CLI_NAME}"
    
    # Test basic execution
    if "$cli_path" version > /dev/null 2>&1; then
        print_success "CLI installation test passed"
        
        # Show version information
        print_status "Installed CLI version:"
        "$cli_path" version
    else
        print_error "CLI installation test failed"
        exit 1
    fi
}

# Main installation function
main() {
    echo "Email Triage CLI Installation"
    echo "============================="
    echo
    
    print_status "Starting CLI installation process..."
    
    # Run installation steps
    check_binary
    create_bin_dir
    install_binary
    update_path
    create_config
    test_installation
    
    echo
    print_success "Email Triage CLI installation completed successfully!"
    echo
    echo "Next steps:"
    echo "  1. Restart your terminal or run: source ~/.bashrc (or ~/.zshrc)"
    echo "  2. Test the installation: email-triage --help"
    echo "  3. Check system status: email-triage status"
    echo "  4. Configure authentication if needed"
    echo
    echo "Documentation:"
    echo "  - CLI Reference: email-triage --help"
    echo "  - Configuration: ~/.vrooli/email-triage/config.yaml"
    echo "  - Project README: $(dirname "$SCRIPT_DIR")/README.md"
    echo
}

# Run installation if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi