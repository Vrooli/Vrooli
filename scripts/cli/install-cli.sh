#!/usr/bin/env bash
################################################################################
# Vrooli CLI Installation Script
# 
# Installs the 'vrooli' command by creating a symlink in the user's PATH.
#
# Usage:
#   ./install-cli.sh [--uninstall]
#
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CLI_SCRIPT="$SCRIPT_DIR/vrooli"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print colored output
print_error() { echo -e "${RED}✗ $1${NC}"; }
print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_info() { echo -e "$1"; }

# Find appropriate installation directory
find_install_dir() {
    # Check common user bin directories in order of preference
    local dirs=(
        "$HOME/.local/bin"
        "$HOME/bin"
        "/usr/local/bin"
    )
    
    for dir in "${dirs[@]}"; do
        # Check if directory exists and is in PATH
        if [[ -d "$dir" ]] && echo "$PATH" | grep -q "$dir"; then
            # Check if we can write to it
            if [[ -w "$dir" ]]; then
                echo "$dir"
                return 0
            elif [[ "$dir" == "/usr/local/bin" ]]; then
                # For /usr/local/bin, we might need sudo
                echo "$dir"
                return 0
            fi
        fi
    done
    
    # Default to ~/.local/bin (will create if needed)
    echo "$HOME/.local/bin"
}

# Check if directory is in PATH
check_path() {
    local dir="$1"
    if echo "$PATH" | grep -q "$dir"; then
        return 0
    fi
    return 1
}

# Add directory to PATH in shell config
add_to_path() {
    local dir="$1"
    local shell_config=""
    
    # Determine shell configuration file
    if [[ -n "${BASH_VERSION:-}" ]]; then
        if [[ -f "$HOME/.bashrc" ]]; then
            shell_config="$HOME/.bashrc"
        elif [[ -f "$HOME/.bash_profile" ]]; then
            shell_config="$HOME/.bash_profile"
        fi
    elif [[ -n "${ZSH_VERSION:-}" ]]; then
        shell_config="$HOME/.zshrc"
    fi
    
    if [[ -n "$shell_config" ]]; then
        # Check if PATH export already exists
        if ! grep -q "export PATH=\".*$dir.*\"" "$shell_config" 2>/dev/null; then
            echo "" >> "$shell_config"
            echo "# Added by Vrooli CLI installer" >> "$shell_config"
            echo "export PATH=\"\$PATH:$dir\"" >> "$shell_config"
            print_success "Added $dir to PATH in $shell_config"
            print_info "   Run 'source $shell_config' or open a new terminal"
        fi
    else
        print_warning "Could not determine shell configuration file"
        print_info "   Add this to your shell configuration:"
        print_info "   export PATH=\"\$PATH:$dir\""
    fi
}

# Check if CLI is already installed and up-to-date
check_existing_installation() {
    local symlink_path="$1"
    
    # Check if symlink exists and points to the correct location
    if [[ -L "$symlink_path" ]]; then
        local current_target
        current_target=$(readlink -f "$symlink_path" 2>/dev/null)
        local expected_target
        expected_target=$(readlink -f "$CLI_SCRIPT" 2>/dev/null)
        
        if [[ "$current_target" == "$expected_target" ]]; then
            return 0  # Already installed and pointing to correct location
        fi
    fi
    
    return 1  # Not installed or pointing to wrong location
}

# Install the CLI
install_cli() {
    local force="${1:-false}"
    
    # Check if CLI script exists
    if [[ ! -f "$CLI_SCRIPT" ]]; then
        print_error "CLI script not found: $CLI_SCRIPT"
        exit 1
    fi
    
    # Find installation directory
    local install_dir
    install_dir=$(find_install_dir)
    local symlink_path="$install_dir/vrooli"
    
    # Check if already installed and up-to-date (unless force is specified)
    if [[ "$force" != "true" ]] && check_existing_installation "$symlink_path"; then
        print_success "✅ Vrooli CLI is already installed and up-to-date!"
        
        # Check if the command is available in PATH
        if command -v vrooli >/dev/null 2>&1; then
            print_info "   Location: $symlink_path -> $CLI_SCRIPT"
            
            # Verify VROOLI_ROOT is set
            if [[ -n "${VROOLI_ROOT:-}" ]]; then
                print_info "   VROOLI_ROOT: $VROOLI_ROOT"
            else
                print_warning "   VROOLI_ROOT not set in current session"
            fi
            
            echo ""
            echo "The 'vrooli' command is ready to use!"
            echo "Use --force to reinstall anyway"
            return 0
        else
            # CLI is installed but not in PATH yet
            if ! check_path "$install_dir"; then
                print_warning "$install_dir is not in your PATH"
                add_to_path "$install_dir"
                echo ""
                echo "To start using the CLI:"
                echo "  source ~/.bashrc (or ~/.zshrc)"
                echo "  Or open a new terminal"
            fi
            return 0
        fi
    fi
    
    print_info "🚀 Installing Vrooli CLI..."
    echo ""
    print_info "Installation directory: $install_dir"
    
    # Create directory if it doesn't exist
    if [[ ! -d "$install_dir" ]]; then
        print_info "Creating directory: $install_dir"
        mkdir -p "$install_dir"
    fi
    
    # Check if we need sudo for installation
    local use_sudo=false
    if [[ "$install_dir" == "/usr/local/bin" ]] && [[ ! -w "$install_dir" ]]; then
        use_sudo=true
        print_warning "Installation to $install_dir requires sudo privileges"
    fi
    
    # Remove existing symlink if it exists
    if [[ -L "$symlink_path" ]] || [[ -f "$symlink_path" ]]; then
        print_info "Removing existing installation..."
        if [[ "$use_sudo" == "true" ]]; then
            sudo rm -f "$symlink_path"
        else
            rm -f "$symlink_path"
        fi
    fi
    
    # Create symlink
    print_info "Creating symlink..."
    if [[ "$use_sudo" == "true" ]]; then
        sudo ln -s "$CLI_SCRIPT" "$symlink_path"
    else
        ln -s "$CLI_SCRIPT" "$symlink_path"
    fi
    
    # Verify installation
    if [[ -L "$symlink_path" ]]; then
        print_success "Symlink created: $symlink_path -> $CLI_SCRIPT"
    else
        print_error "Failed to create symlink"
        exit 1
    fi
    
    # Check if directory is in PATH
    if ! check_path "$install_dir"; then
        print_warning "$install_dir is not in your PATH"
        add_to_path "$install_dir"
    fi
    
    # Set VROOLI_ROOT environment variable
    local shell_config=""
    if [[ -n "${BASH_VERSION:-}" ]]; then
        shell_config="${HOME}/.bashrc"
    elif [[ -n "${ZSH_VERSION:-}" ]]; then
        shell_config="${HOME}/.zshrc"
    fi
    
    if [[ -n "$shell_config" ]] && [[ -f "$shell_config" ]]; then
        if ! grep -q "export VROOLI_ROOT=" "$shell_config" 2>/dev/null; then
            echo "" >> "$shell_config"
            echo "# Vrooli CLI root directory" >> "$shell_config"
            echo "export VROOLI_ROOT=\"$VROOLI_ROOT\"" >> "$shell_config"
            print_success "Added VROOLI_ROOT to $shell_config"
        fi
    fi
    
    echo ""
    print_success "✅ Vrooli CLI installed successfully!"
    echo ""
    
    # Test if command works
    if command -v vrooli >/dev/null 2>&1; then
        print_success "The 'vrooli' command is ready to use!"
        echo ""
        echo "Try these commands:"
        echo "  vrooli --help           # Show help"
        echo "  vrooli --version        # Show version"
        echo "  vrooli app list         # List generated apps"
    else
        print_warning "The 'vrooli' command is not yet available in this session"
        echo ""
        echo "To start using it:"
        if [[ -n "$shell_config" ]]; then
            echo "  source $shell_config"
        else
            echo "  export PATH=\"\$PATH:$install_dir\""
        fi
        echo "  Or open a new terminal"
    fi
}

# Uninstall the CLI
uninstall_cli() {
    print_info "🗑️  Uninstalling Vrooli CLI..."
    echo ""
    
    local removed=false
    
    # Check common locations
    local locations=(
        "$HOME/.local/bin/vrooli"
        "$HOME/bin/vrooli"
        "/usr/local/bin/vrooli"
    )
    
    for location in "${locations[@]}"; do
        if [[ -L "$location" ]] || [[ -f "$location" ]]; then
            print_info "Removing: $location"
            if [[ "$location" == "/usr/local/bin/vrooli" ]] && [[ ! -w "$(dirname "$location")" ]]; then
                sudo rm -f "$location"
            else
                rm -f "$location"
            fi
            removed=true
        fi
    done
    
    if [[ "$removed" == "true" ]]; then
        print_success "✅ Vrooli CLI uninstalled"
        echo ""
        echo "Note: Environment variables (VROOLI_ROOT) and PATH modifications"
        echo "were not removed from your shell configuration files."
        echo "You can remove them manually if desired."
    else
        print_warning "No Vrooli CLI installation found"
    fi
}

# Main execution
main() {
    case "${1:-}" in
        --uninstall|-u)
            uninstall_cli
            ;;
        --force|-f)
            install_cli true
            ;;
        --help|-h)
            cat << EOF
Vrooli CLI Installation Script

USAGE:
    $0 [OPTIONS]

OPTIONS:
    --force, -f         Force reinstallation even if already up-to-date
    --uninstall, -u     Uninstall the Vrooli CLI
    --help, -h          Show this help message

DESCRIPTION:
    This script installs the 'vrooli' command by creating a symlink
    in a directory that's in your PATH.
    
    Features:
    • Checks if CLI is already installed and up-to-date
    • Skips installation if no changes are needed
    • Finds appropriate directory in PATH automatically
    • Creates symlink to the CLI script
    • Adds directory to PATH if needed
    • Sets VROOLI_ROOT environment variable
    
    The script will skip installation if:
    • The symlink already exists
    • It points to the correct CLI script
    • No updates are needed
    
    Use --force to reinstall regardless of current state.

EOF
            ;;
        "")
            install_cli false
            ;;
        *)
            print_error "Unknown option: $1"
            echo "Run '$0 --help' for usage information"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"