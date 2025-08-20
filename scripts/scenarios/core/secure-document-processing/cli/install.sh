#!/bin/bash
# Install script for Secure Document Processing CLI

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="secure-document-processing"
CLI_SOURCE="$SCRIPT_DIR/$CLI_NAME"
INSTALL_DIR="/usr/local/bin"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root or with sudo
if [[ $EUID -ne 0 ]]; then
    print_error "This script requires root privileges. Please run with sudo."
    exit 1
fi

# Check if CLI exists
if [[ ! -f "$CLI_SOURCE" ]]; then
    print_error "CLI binary not found at $CLI_SOURCE"
    exit 1
fi

# Install CLI
print_status "Installing $CLI_NAME CLI..."

# Copy CLI to install directory
cp "$CLI_SOURCE" "$INSTALL_DIR/$CLI_NAME"
chmod +x "$INSTALL_DIR/$CLI_NAME"

# Verify installation
if command -v "$CLI_NAME" >/dev/null 2>&1; then
    print_status "✓ $CLI_NAME CLI installed successfully"
    print_status "Run '$CLI_NAME --help' to get started"
else
    print_error "Installation failed - CLI not found in PATH"
    exit 1
fi

print_status "Installation complete!"
echo ""
echo "You can now use the CLI with:"
echo "  $CLI_NAME status       # Check service status"
echo "  $CLI_NAME documents    # List documents"
echo "  $CLI_NAME jobs         # List processing jobs"
echo "  $CLI_NAME workflows    # List workflows"
echo "  $CLI_NAME --help       # Show help"