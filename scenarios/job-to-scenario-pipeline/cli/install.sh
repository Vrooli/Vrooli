#!/bin/bash

# Job-to-Scenario Pipeline CLI Installer

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="job-to-scenario"
TARGET_DIR="${HOME}/.local/bin"

# Create target directory if it doesn't exist
mkdir -p "$TARGET_DIR"

# Create symlink
ln -sf "${SCRIPT_DIR}/${CLI_NAME}" "${TARGET_DIR}/${CLI_NAME}"

# Check if target directory is in PATH
if [[ ":$PATH:" != *":${TARGET_DIR}:"* ]]; then
    echo "Adding ${TARGET_DIR} to PATH"
    
    # Add to appropriate shell config
    if [ -f "${HOME}/.bashrc" ]; then
        echo "export PATH=\"\${PATH}:${TARGET_DIR}\"" >> "${HOME}/.bashrc"
    fi
    
    if [ -f "${HOME}/.zshrc" ]; then
        echo "export PATH=\"\${PATH}:${TARGET_DIR}\"" >> "${HOME}/.zshrc"
    fi
    
    echo "Please restart your shell or run: export PATH=\"\${PATH}:${TARGET_DIR}\""
fi

echo "✓ Job-to-Scenario Pipeline CLI installed"
echo "Usage: job-to-scenario --help"