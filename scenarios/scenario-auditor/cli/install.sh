#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="scenario-auditor"
VROOLI_BIN_DIR="${HOME}/.vrooli/bin"

mkdir -p "$VROOLI_BIN_DIR"
CLI_SOURCE="${SCRIPT_DIR}/${CLI_NAME}"
CLI_TARGET="${VROOLI_BIN_DIR}/${CLI_NAME}"

if [[ -L "$CLI_TARGET" || -f "$CLI_TARGET" ]]; then
    rm -f "$CLI_TARGET"
fi

echo "Installing ${CLI_NAME} CLI to ${VROOLI_BIN_DIR}"
ln -s "$CLI_SOURCE" "$CLI_TARGET"
chmod +x "$CLI_SOURCE"

if [[ ":$PATH:" != *":$VROOLI_BIN_DIR:"* ]]; then
    cat <<PATHWARN
⚠️  ${VROOLI_BIN_DIR} is not in your PATH.
   Add this to your shell profile (e.g. ~/.bashrc or ~/.zshrc):
       export PATH=\"\$PATH:${VROOLI_BIN_DIR}\"
PATHWARN
fi

echo "✅ ${CLI_NAME} CLI installed successfully"
