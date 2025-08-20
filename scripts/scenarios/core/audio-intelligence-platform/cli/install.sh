#!/usr/bin/env bash
# Install audio intelligence platform CLI globally
set -euo pipefail

# Script directory
CLI_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
CLI_SCRIPT="$CLI_DIR/audio-intelligence-platform-cli.sh"

# Target installation path  
INSTALL_PATH="/usr/local/bin/audio-intelligence-platform"

# Check if CLI script exists
if [[ ! -f "$CLI_SCRIPT" ]]; then
    echo "❌ CLI script not found: $CLI_SCRIPT" >&2
    exit 1
fi

# Check if we have permission to install
if [[ ! -w "/usr/local/bin" ]]; then
    echo "❌ Permission denied: Cannot write to /usr/local/bin" >&2
    echo "Try running with sudo or ensure /usr/local/bin is writable" >&2
    exit 1
fi

# Install the CLI
echo "📦 Installing audio-intelligence-platform CLI to $INSTALL_PATH..."

# Copy the script and make it executable
cp "$CLI_SCRIPT" "$INSTALL_PATH"
chmod +x "$INSTALL_PATH"

# Verify installation
if [[ -x "$INSTALL_PATH" ]]; then
    echo "✅ CLI installed successfully!"
    echo "📍 Location: $INSTALL_PATH"
    echo "🔧 Test with: audio-intelligence-platform help"
    
    # Show version info
    if command -v audio-intelligence-platform >/dev/null 2>&1; then
        echo ""
        echo "🎯 Installation verified:"
        which audio-intelligence-platform
    fi
else
    echo "❌ Installation failed: CLI not executable at $INSTALL_PATH" >&2
    exit 1
fi