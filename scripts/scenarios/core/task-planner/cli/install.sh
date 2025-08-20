#!/bin/bash

# Task Planner CLI Installation Script
# Installs the task-planner CLI globally

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SCRIPT="${SCRIPT_DIR}/task-planner-cli.sh"

# Installation directory
INSTALL_DIR="/usr/local/bin"
CLI_NAME="task-planner"

# Check if script exists
if [[ ! -f "$CLI_SCRIPT" ]]; then
    echo "Error: CLI script not found at $CLI_SCRIPT" >&2
    exit 1
fi

# Create symlink to install CLI globally
echo "Installing task-planner CLI..."

# Remove existing installation if it exists
if [[ -L "${INSTALL_DIR}/${CLI_NAME}" ]]; then
    echo "Removing existing installation..."
    sudo rm -f "${INSTALL_DIR}/${CLI_NAME}"
fi

# Create new symlink
sudo ln -sf "$CLI_SCRIPT" "${INSTALL_DIR}/${CLI_NAME}"

# Make CLI script executable
chmod +x "$CLI_SCRIPT"

echo "✅ Task Planner CLI installed successfully!"
echo "Usage: $CLI_NAME --help"

# Test installation
if command -v "$CLI_NAME" &> /dev/null; then
    echo "✅ CLI is available in PATH"
else
    echo "⚠️  CLI may not be available in PATH. You may need to restart your shell."
fi