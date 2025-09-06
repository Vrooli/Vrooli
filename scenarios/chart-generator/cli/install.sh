#!/bin/bash

# Chart Generator CLI Installation Script

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_info() { echo -e "${BLUE}ℹ${NC} $1"; }
print_success() { echo -e "${GREEN}✓${NC} $1"; }
print_warning() { echo -e "${YELLOW}⚠${NC} $1"; }
print_error() { echo -e "${RED}✗${NC} $1"; }

# Configuration
CLI_NAME="chart-generator"
CLI_SOURCE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$CLI_NAME"
INSTALL_DIR="${HOME}/.vrooli/bin"
INSTALL_PATH="${INSTALL_DIR}/${CLI_NAME}"

main() {
    print_info "Installing Chart Generator CLI..."

    # Check if source CLI exists
    if [[ ! -f "$CLI_SOURCE" ]]; then
        print_error "CLI source not found: $CLI_SOURCE"
        exit 1
    fi

    # Create install directory
    if [[ ! -d "$INSTALL_DIR" ]]; then
        print_info "Creating install directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR"
    fi

    # Copy CLI to install location
    print_info "Installing CLI to: $INSTALL_PATH"
    cp "$CLI_SOURCE" "$INSTALL_PATH"
    chmod +x "$INSTALL_PATH"

    # Check if install directory is in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        print_warning "Install directory not in PATH: $INSTALL_DIR"
        print_info "Add this to your shell profile (.bashrc, .zshrc, etc.):"
        echo "export PATH=\"\$PATH:$INSTALL_DIR\""
        echo
    fi

    # Verify installation
    if [[ -x "$INSTALL_PATH" ]]; then
        print_success "Chart Generator CLI installed successfully!"
        print_info "Run 'chart-generator help' to get started"
        
        # Test CLI
        if "$INSTALL_PATH" version >/dev/null 2>&1; then
            print_success "CLI test passed"
        else
            print_warning "CLI installed but test failed - check dependencies"
        fi
    else
        print_error "Installation failed - CLI not executable"
        exit 1
    fi

    # Check dependencies
    print_info "Checking dependencies..."
    
    local missing_deps=()
    if ! command -v curl >/dev/null 2>&1; then
        missing_deps+=("curl")
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        missing_deps+=("jq")
    fi
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        print_warning "Missing dependencies: ${missing_deps[*]}"
        print_info "Install with your package manager:"
        print_info "  Ubuntu/Debian: sudo apt install curl jq"
        print_info "  macOS: brew install curl jq"
        print_info "  CentOS/RHEL: sudo yum install curl jq"
    else
        print_success "All dependencies available"
    fi

    echo
    print_info "🎨 Chart Generator CLI is ready!"
    print_info "Next steps:"
    print_info "  1. Start the service: vrooli scenario run chart-generator"
    print_info "  2. Check status: chart-generator status"
    print_info "  3. Generate your first chart: chart-generator generate bar --data sample.json"
    print_info "  4. Get help: chart-generator help"
}

main "$@"