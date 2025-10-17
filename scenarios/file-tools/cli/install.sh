#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="file-tools"

# Make CLI executable
chmod +x "${SCRIPT_DIR}/${CLI_NAME}"

# Create symlink in a directory that's likely in PATH
if [ -d "$HOME/.local/bin" ]; then
    ln -sf "${SCRIPT_DIR}/${CLI_NAME}" "$HOME/.local/bin/${CLI_NAME}"
    echo "✅ ${CLI_NAME} installed to ~/.local/bin/"
elif [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
    ln -sf "${SCRIPT_DIR}/${CLI_NAME}" "/usr/local/bin/${CLI_NAME}"
    echo "✅ ${CLI_NAME} installed to /usr/local/bin/"
else
    echo "⚠️  Could not install ${CLI_NAME} to PATH. Please add ${SCRIPT_DIR} to your PATH."
fi