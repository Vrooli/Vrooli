#!/bin/bash

# Ecosystem Manager CLI Installation Script
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SCRIPT="$SCRIPT_DIR/ecosystem-manager"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

echo "🔧 Installing Ecosystem Manager CLI..."

# Check if CLI script exists
if [ ! -f "$CLI_SCRIPT" ]; then
    echo "❌ Error: CLI script not found at $CLI_SCRIPT" >&2
    exit 1
fi

# Check if install directory exists and is writable
if [ ! -d "$INSTALL_DIR" ]; then
    echo "📁 Creating install directory: $INSTALL_DIR"
    sudo mkdir -p "$INSTALL_DIR"
fi

if [ ! -w "$INSTALL_DIR" ]; then
    echo "🔐 Installing with sudo (admin privileges required)"
    USE_SUDO=true
else
    USE_SUDO=false
fi

# Install CLI script
INSTALL_PATH="$INSTALL_DIR/ecosystem-manager"

if [ "$USE_SUDO" = true ]; then
    sudo cp "$CLI_SCRIPT" "$INSTALL_PATH"
    sudo chmod +x "$INSTALL_PATH"
else
    cp "$CLI_SCRIPT" "$INSTALL_PATH"
    chmod +x "$INSTALL_PATH"
fi

echo "✅ CLI installed successfully at: $INSTALL_PATH"

# Verify installation
if command -v ecosystem-manager >/dev/null 2>&1; then
    echo "🎉 Installation verified! You can now use:"
    echo ""
    echo "  ecosystem-manager --help"
    echo "  ecosystem-manager add resource matrix-synapse --category communication"
    echo "  ecosystem-manager improve scenario system-monitor" 
    echo "  ecosystem-manager list --status pending"
    echo ""
else
    echo "⚠️  Warning: 'ecosystem-manager' command not found in PATH"
    echo "   You may need to:"
    echo "   1. Add $INSTALL_DIR to your PATH"
    echo "   2. Restart your terminal"
    echo "   3. Or use the full path: $INSTALL_PATH"
fi

# Check for dependencies
echo "🔍 Checking dependencies..."

if ! command -v curl >/dev/null 2>&1; then
    echo "  ⚠️  Warning: curl not found - required for API calls"
fi

if ! command -v jq >/dev/null 2>&1; then
    echo "  ⚠️  Warning: jq not found - JSON formatting will be limited"
    echo "     Install with: sudo apt-get install jq  (Ubuntu/Debian)"
    echo "                   brew install jq         (macOS)"
fi

echo ""
echo "🚀 Ready to use Ecosystem Manager CLI!"
echo "   Start the service: vrooli scenario ecosystem-manager develop"
echo "   Then run: ecosystem-manager --help"