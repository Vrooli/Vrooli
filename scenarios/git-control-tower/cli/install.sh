#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="git-control-tower"
CLI_SOURCE="${SCRIPT_DIR}/${CLI_NAME}"
INSTALL_BIN_DIR="${HOME}/.vrooli/bin"

if [[ ! -f "${CLI_SOURCE}" ]]; then
  echo "CLI binary not found at ${CLI_SOURCE}" >&2
  exit 1
fi

mkdir -p "${INSTALL_BIN_DIR}"
chmod +x "${CLI_SOURCE}"
ln -sf "${CLI_SOURCE}" "${INSTALL_BIN_DIR}/${CLI_NAME}"

echo "Installed ${CLI_NAME} CLI to ${INSTALL_BIN_DIR}/${CLI_NAME}"
