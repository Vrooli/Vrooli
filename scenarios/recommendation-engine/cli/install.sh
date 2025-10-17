#!/bin/bash

# Recommendation Engine CLI Installation Script
# Creates symlink to make the CLI available globally

set -euo pipefail

CLI_NAME="recommendation-engine"
CLI_SOURCE_PATH="$(realpath "$(dirname "$0")/${CLI_NAME}")"
INSTALL_DIR="${HOME}/.vrooli/bin"
CLI_INSTALL_PATH="${INSTALL_DIR}/${CLI_NAME}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

main() {
    log_info "Installing Recommendation Engine CLI..."
    
    # Verify source CLI exists
    if [[ ! -f "${CLI_SOURCE_PATH}" ]]; then
        log_error "CLI source not found: ${CLI_SOURCE_PATH}"
        exit 1
    fi
    
    # Create install directory
    if [[ ! -d "${INSTALL_DIR}" ]]; then
        log_info "Creating CLI install directory: ${INSTALL_DIR}"
        mkdir -p "${INSTALL_DIR}"
    fi
    
    # Remove existing installation
    if [[ -L "${CLI_INSTALL_PATH}" || -f "${CLI_INSTALL_PATH}" ]]; then
        log_info "Removing existing installation"
        rm -f "${CLI_INSTALL_PATH}"
    fi
    
    # Create symlink
    log_info "Creating symlink: ${CLI_INSTALL_PATH} -> ${CLI_SOURCE_PATH}"
    ln -sf "${CLI_SOURCE_PATH}" "${CLI_INSTALL_PATH}"
    
    # Make sure it's executable
    chmod +x "${CLI_SOURCE_PATH}"
    
    # Check if install directory is in PATH
    if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
        log_warn "Install directory is not in PATH: ${INSTALL_DIR}"
        log_info "Add this line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo "export PATH=\"\$PATH:${INSTALL_DIR}\""
        log_info "Or run: echo 'export PATH=\"\$PATH:${INSTALL_DIR}\"' >> ~/.bashrc"
    fi
    
    # Test installation
    if [[ -x "${CLI_INSTALL_PATH}" ]]; then
        log_success "CLI installed successfully!"
        log_info "Test the installation with: ${CLI_NAME} --help"
        
        # Show version if API is running
        if "${CLI_INSTALL_PATH}" version --json >/dev/null 2>&1; then
            log_info "CLI is working and can connect to the API"
        else
            log_warn "CLI installed but cannot connect to API (this is normal if the scenario is not running)"
            log_info "Start the scenario with: vrooli scenario run recommendation-engine"
        fi
    else
        log_error "Installation failed - CLI is not executable"
        exit 1
    fi
    
    log_success "Installation complete! 🚀"
    echo ""
    echo "Available commands:"
    echo "  ${CLI_NAME} status      - Check system status"
    echo "  ${CLI_NAME} ingest      - Ingest data from scenarios" 
    echo "  ${CLI_NAME} recommend   - Get personalized recommendations"
    echo "  ${CLI_NAME} similar     - Find similar items"
    echo "  ${CLI_NAME} help        - Show detailed help"
    echo ""
    echo "Documentation: See README.md in the scenario directory"
}

main "$@"