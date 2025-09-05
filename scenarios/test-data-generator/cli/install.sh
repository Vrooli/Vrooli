#!/bin/bash

# Test Data Generator CLI Installation Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SCRIPT="$SCRIPT_DIR/test-data-generator"
TARGET_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_color() {
    local color=$1
    shift
    echo -e "${color}$*${NC}"
}

print_error() {
    print_color $RED "❌ Error: $*"
}

print_success() {
    print_color $GREEN "✅ $*"
}

print_info() {
    print_color $BLUE "ℹ️  $*"
}

print_warning() {
    print_color $YELLOW "⚠️  $*"
}

# Check if script exists
if [ ! -f "$CLI_SCRIPT" ]; then
    print_error "CLI script not found at $CLI_SCRIPT"
    exit 1
fi

# Check dependencies
check_dependencies() {
    local missing=()
    
    if ! command -v curl &> /dev/null; then
        missing+=("curl")
    fi
    
    if ! command -v jq &> /dev/null; then
        print_warning "jq is not installed - JSON output will be less formatted"
        print_info "Install with: sudo apt install jq  # or  brew install jq"
    fi
    
    if [ ${#missing[@]} -ne 0 ]; then
        print_error "Missing required dependencies: ${missing[*]}"
        print_info "Please install the missing dependencies and try again"
        exit 1
    fi
}

# Install CLI
install_cli() {
    print_info "Installing Test Data Generator CLI..."
    
    # Check if we need sudo
    if [ ! -w "$TARGET_DIR" ]; then
        print_info "Installing to $TARGET_DIR requires sudo permissions"
        sudo cp "$CLI_SCRIPT" "$TARGET_DIR/"
        sudo chmod +x "$TARGET_DIR/test-data-generator"
    else
        cp "$CLI_SCRIPT" "$TARGET_DIR/"
        chmod +x "$TARGET_DIR/test-data-generator"
    fi
    
    print_success "CLI installed successfully!"
    print_info "You can now use: test-data-generator --help"
}

# Uninstall CLI
uninstall_cli() {
    print_info "Uninstalling Test Data Generator CLI..."
    
    if [ -f "$TARGET_DIR/test-data-generator" ]; then
        if [ ! -w "$TARGET_DIR" ]; then
            sudo rm "$TARGET_DIR/test-data-generator"
        else
            rm "$TARGET_DIR/test-data-generator"
        fi
        print_success "CLI uninstalled successfully!"
    else
        print_warning "CLI was not installed in $TARGET_DIR"
    fi
}

# Show help
show_help() {
    cat << EOF
Test Data Generator CLI Installation

USAGE:
    install.sh [command]

COMMANDS:
    install      Install the CLI (default)
    uninstall    Remove the CLI
    check        Check dependencies
    help         Show this help

The CLI will be installed to: $TARGET_DIR/test-data-generator

REQUIREMENTS:
    - curl (required)
    - jq (optional, for better JSON formatting)

EOF
}

# Main
main() {
    case "${1:-install}" in
        install)
            check_dependencies
            install_cli
            ;;
        uninstall)
            uninstall_cli
            ;;
        check)
            check_dependencies
            print_success "All required dependencies are available"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "Unknown command: $1"
            echo
            show_help
            exit 1
            ;;
    esac
}

main "$@"