#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="kids-dashboard"
INSTALL_DIR="${HOME}/.vrooli/bin"

mkdir -p "$INSTALL_DIR"
chmod +x "$SCRIPT_DIR/$CLI_NAME"
ln -sf "$SCRIPT_DIR/$CLI_NAME" "$INSTALL_DIR/$CLI_NAME"

if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
  cat <<NOTICE
⚠️  $INSTALL_DIR is not in your PATH
Add the following to your shell profile and re-source it:
  export PATH="$INSTALL_DIR:\$PATH"
NOTICE
fi

echo "✓ Kids Dashboard CLI installed"
echo "Run 'kids-dashboard help' to explore commands"
