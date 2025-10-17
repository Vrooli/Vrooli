#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="knowledge-observatory"
INSTALL_DIR="$HOME/.vrooli/bin"

echo "🔭 Installing Knowledge Observatory CLI..."

mkdir -p "$INSTALL_DIR"

if [[ ! -f "$SCRIPT_DIR/$CLI_NAME" ]]; then
    echo "Error: CLI script not found at $SCRIPT_DIR/$CLI_NAME" >&2
    exit 1
fi

cp "$SCRIPT_DIR/$CLI_NAME" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/$CLI_NAME"

if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo ""
    echo "⚠️  $INSTALL_DIR is not in your PATH"
    echo "Add the following line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo ""
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    echo ""
fi

if command -v "$CLI_NAME" &> /dev/null; then
    echo "✅ Knowledge Observatory CLI installed successfully!"
    echo "   Run '$CLI_NAME help' to get started"
else
    echo "✅ Knowledge Observatory CLI installed to $INSTALL_DIR/$CLI_NAME"
    echo "   Restart your shell or update your PATH to use it"
fi