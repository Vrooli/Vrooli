#\!/usr/bin/env bash
# Install App Monitor CLI globally

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SOURCE="$SCRIPT_DIR/app-monitor"
CLI_TARGET="/usr/local/bin/app-monitor"

echo "Installing App Monitor CLI..."

# Check if source exists
if [[ \! -f "$CLI_SOURCE" ]]; then
    echo "❌ CLI source file not found: $CLI_SOURCE"
    exit 1
fi

# Copy CLI to global location
sudo cp "$CLI_SOURCE" "$CLI_TARGET"
sudo chmod +x "$CLI_TARGET"

echo "✅ App Monitor CLI installed successfully at $CLI_TARGET"
echo "📖 Run 'app-monitor help' to see available commands"
EOF < /dev/null
