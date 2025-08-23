#!/usr/bin/env bash

set -euo pipefail

# Install the Workflow Scheduler CLI

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly CLI_NAME="scheduler-cli"
readonly CLI_SOURCE="${SCRIPT_DIR}/${CLI_NAME}"

# Use user's local bin directory or create symlink in project bin
readonly USER_BIN_DIR="${HOME}/.local/bin"
# Find the project root (generated app or scenario location)
if [[ "${SCRIPT_DIR}" == */generated-apps/* ]]; then
    # We're in a generated app
    readonly PROJECT_BIN_DIR="${SCRIPT_DIR}/../bin"
else
    # We're in the scenario source
    readonly PROJECT_BIN_DIR="${SCRIPT_DIR}/../../../../bin"
fi
readonly INSTALL_DIR="${USER_BIN_DIR}"
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
    mkdir -p "${INSTALL_DIR}"
fi

# Install CLI (create symlink instead of copy for easier updates)
echo "Installing ${CLI_NAME} to ${INSTALL_PATH}..."
if [[ -L "${INSTALL_PATH}" ]] || [[ -f "${INSTALL_PATH}" ]]; then
    echo "Removing existing installation..."
    rm -f "${INSTALL_PATH}"
fi
ln -s "${CLI_SOURCE}" "${INSTALL_PATH}"

# Also create a symlink in the project bin directory for integration with other scenarios
if [[ ! -d "${PROJECT_BIN_DIR}" ]]; then
    mkdir -p "${PROJECT_BIN_DIR}"
fi
PROJECT_INSTALL_PATH="${PROJECT_BIN_DIR}/${CLI_NAME}"
if [[ -L "${PROJECT_INSTALL_PATH}" ]] || [[ -f "${PROJECT_INSTALL_PATH}" ]]; then
    rm -f "${PROJECT_INSTALL_PATH}"
fi
ln -s "${CLI_SOURCE}" "${PROJECT_INSTALL_PATH}"

# Add to PATH if not already there
if [[ ":$PATH:" != *":${USER_BIN_DIR}:"* ]]; then
    echo -e "${YELLOW}Note: ${USER_BIN_DIR} is not in your PATH${NC}"
    echo "Add this line to your ~/.bashrc or ~/.profile:"
    echo "  export PATH=\"\${PATH}:${USER_BIN_DIR}\""
fi

# Verify installation
if [[ -L "${INSTALL_PATH}" ]] && [[ -x "${CLI_SOURCE}" ]]; then
    echo -e "${GREEN}✓ ${CLI_NAME} installed successfully${NC}"
    echo ""
    echo "You can now use the scheduler CLI with:"
    echo "  ${CLI_NAME} --help  (if ${USER_BIN_DIR} is in PATH)"
    echo "  ${INSTALL_PATH} --help  (full path)"
    echo ""
    echo "Quick start examples:"
    echo "  ${CLI_NAME} list                              # List all schedules"
    echo "  ${CLI_NAME} create --name \"Test\" --cron \"0 * * * *\" --url http://example.com/webhook"
    echo "  ${CLI_NAME} dashboard                          # View system statistics"
    echo ""
    echo "The CLI is also available for other scenarios at:"
    echo "  ${PROJECT_INSTALL_PATH}"
else
    echo -e "${RED}Installation verification failed${NC}"
    exit 1
fi

echo ""
echo "Installation complete!"