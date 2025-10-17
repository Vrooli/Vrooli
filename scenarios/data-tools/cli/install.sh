#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="data-tools"

echo "Installing ${CLI_NAME} CLI..."

if [[ ! -f "${SCRIPT_DIR}/${CLI_NAME}" ]]; then
    echo "❌ CLI script not found: ${SCRIPT_DIR}/${CLI_NAME}"
    exit 1
fi

chmod +x "${SCRIPT_DIR}/${CLI_NAME}"

INSTALL_DIR="${HOME}/.local/bin"
mkdir -p "${INSTALL_DIR}"

cp "${SCRIPT_DIR}/${CLI_NAME}" "${INSTALL_DIR}/${CLI_NAME}"

if ! command -v "${CLI_NAME}" &>/dev/null; then
    if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
        echo "⚠️  ${INSTALL_DIR} is not in your PATH"
        echo "   Add it with: export PATH=\"\$PATH:${INSTALL_DIR}\""
    fi
fi

echo "✅ ${CLI_NAME} CLI installed to ${INSTALL_DIR}"