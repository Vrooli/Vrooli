#!/usr/bin/env bash
set -euo pipefail

# Notification Hub CLI Installation Script
# Installs the notification-hub CLI command to user's local bin

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="notification-hub"
INSTALL_DIR="${HOME}/.local/bin"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}ℹ️  $1${NC}"
}

log_warn() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Create install directory if it doesn't exist
if [ ! -d "${INSTALL_DIR}" ]; then
    log_info "Creating ${INSTALL_DIR}"
    mkdir -p "${INSTALL_DIR}"
fi

# Check if CLI binary exists
if [ ! -f "${SCRIPT_DIR}/${CLI_NAME}" ]; then
    log_error "CLI binary not found at ${SCRIPT_DIR}/${CLI_NAME}"
    exit 1
fi

# Make the CLI executable
chmod +x "${SCRIPT_DIR}/${CLI_NAME}"

# Create symlink
TARGET="${INSTALL_DIR}/${CLI_NAME}"
if [ -L "${TARGET}" ] || [ -f "${TARGET}" ]; then
    log_warn "Removing existing ${CLI_NAME} installation"
    rm -f "${TARGET}"
fi

ln -sf "${SCRIPT_DIR}/${CLI_NAME}" "${TARGET}"
log_info "Installed ${CLI_NAME} to ${TARGET}"

# Check if install dir is in PATH
if [[ ":${PATH}:" != *":${INSTALL_DIR}:"* ]]; then
    log_warn "${INSTALL_DIR} is not in your PATH"
    log_info "Add this line to your ~/.bashrc or ~/.zshrc:"
    echo "    export PATH=\"\${HOME}/.local/bin:\${PATH}\""
else
    log_info "✅ Installation complete! Run '${CLI_NAME} --help' to get started"
fi
