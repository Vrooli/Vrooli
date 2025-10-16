#!/usr/bin/env bash
#
# Install PRD Control Tower CLI to user's bin directory
#

set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="prd-control-tower"
INSTALL_DIR="${HOME}/.local/bin"

# Create install directory if it doesn't exist
mkdir -p "${INSTALL_DIR}"

# Copy CLI script
cp "${SCRIPT_DIR}/${CLI_NAME}" "${INSTALL_DIR}/${CLI_NAME}"
chmod +x "${INSTALL_DIR}/${CLI_NAME}"

echo "✓ Installed ${CLI_NAME} to ${INSTALL_DIR}/${CLI_NAME}"
echo ""
echo "Make sure ${INSTALL_DIR} is in your PATH:"
echo "  export PATH=\"\${HOME}/.local/bin:\${PATH}\""
echo ""
echo "Try it out:"
echo "  ${CLI_NAME} help"
