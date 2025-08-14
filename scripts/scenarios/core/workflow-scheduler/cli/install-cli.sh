#!/usr/bin/env bash

set -euo pipefail

# Install the Workflow Scheduler CLI

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly CLI_NAME="scheduler-cli"
readonly CLI_SOURCE="${SCRIPT_DIR}/${CLI_NAME}"
readonly INSTALL_DIR="/usr/local/bin"
readonly INSTALL_PATH="${INSTALL_DIR}/${CLI_NAME}"

# Color codes
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m'

echo "Installing Workflow Scheduler CLI..."

# Check if CLI source exists
if [[ ! -f "${CLI_SOURCE}" ]]; then
    echo -e "${RED}Error: CLI source not found at ${CLI_SOURCE}${NC}"
    exit 1
fi

# Make CLI executable
chmod +x "${CLI_SOURCE}"

# Create install directory if it doesn't exist
if [[ ! -d "${INSTALL_DIR}" ]]; then
    echo "Creating install directory: ${INSTALL_DIR}"
    sudo mkdir -p "${INSTALL_DIR}"
fi

# Install CLI
echo "Installing ${CLI_NAME} to ${INSTALL_PATH}..."
sudo cp "${CLI_SOURCE}" "${INSTALL_PATH}"
sudo chmod +x "${INSTALL_PATH}"

# Verify installation
if command -v "${CLI_NAME}" &> /dev/null; then
    echo -e "${GREEN}✓ ${CLI_NAME} installed successfully${NC}"
    echo ""
    echo "You can now use the scheduler CLI with:"
    echo "  ${CLI_NAME} --help"
    echo ""
    echo "Quick start examples:"
    echo "  ${CLI_NAME} list                              # List all schedules"
    echo "  ${CLI_NAME} create --name \"Test\" --cron \"0 * * * *\" --url http://example.com/webhook"
    echo "  ${CLI_NAME} dashboard                          # View system statistics"
else
    echo -e "${YELLOW}Warning: ${CLI_NAME} installed but not in PATH${NC}"
    echo "You may need to add ${INSTALL_DIR} to your PATH"
    echo "Or use the full path: ${INSTALL_PATH}"
fi

echo ""
echo "Installation complete!"