#!/bin/bash

set -e

# Install Stream of Consciousness Analyzer CLI

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="soc-analyzer"
CLI_PATH="${SCRIPT_DIR}/${CLI_NAME}"
INSTALL_DIR="/usr/local/bin"

echo "Installing Stream of Consciousness Analyzer CLI..."

# Check if CLI script exists
if [ ! -f "${CLI_PATH}" ]; then
    echo "Error: CLI script not found at ${CLI_PATH}"
    exit 1
fi

# Make CLI executable
chmod +x "${CLI_PATH}"

# Create symlink in install directory
if [ -w "${INSTALL_DIR}" ]; then
    ln -sf "${CLI_PATH}" "${INSTALL_DIR}/${CLI_NAME}"
    echo "✓ CLI installed to ${INSTALL_DIR}/${CLI_NAME}"
else
    # Try with sudo
    echo "Need sudo access to install to ${INSTALL_DIR}"
    sudo ln -sf "${CLI_PATH}" "${INSTALL_DIR}/${CLI_NAME}"
    echo "✓ CLI installed to ${INSTALL_DIR}/${CLI_NAME} (with sudo)"
fi

# Verify installation
if command -v ${CLI_NAME} &> /dev/null; then
    echo "✓ Installation successful! Run '${CLI_NAME} help' to get started."
else
    echo "⚠ Warning: ${CLI_NAME} command not found in PATH"
    echo "  You may need to add ${INSTALL_DIR} to your PATH"
    echo "  Or run the CLI directly: ${CLI_PATH}"
fi