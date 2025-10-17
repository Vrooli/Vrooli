#!/bin/bash

# Install agent-dashboard CLI
set -e

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scenarios/agent-dashboard/cli"
CLI_NAME="agent-dashboard"
INSTALL_DIR="${HOME}/.local/bin"

# Create install directory if it doesn't exist
mkdir -p "${INSTALL_DIR}"

# Copy CLI script to install directory
cp "${SCRIPT_DIR}/${CLI_NAME}" "${INSTALL_DIR}/"
chmod +x "${INSTALL_DIR}/${CLI_NAME}"

# Check if install directory is in PATH
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    echo "WARNING: ${INSTALL_DIR} is not in your PATH"
    echo "Add the following to your shell profile:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
fi

echo "✅ ${CLI_NAME} installed successfully"
echo "Run '${CLI_NAME} --help' to get started"