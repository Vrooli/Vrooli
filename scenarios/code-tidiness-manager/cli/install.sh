#!/usr/bin/env bash
set -euo pipefail

CLI_NAME="code-tidiness-manager"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_PATH="${SCRIPT_DIR}/${CLI_NAME}"
INSTALL_DIR="${HOME}/.vrooli/bin"

mkdir -p "${INSTALL_DIR}"

if [[ -f "${CLI_PATH}" ]]; then
  chmod +x "${CLI_PATH}"
  ln -sf "${CLI_PATH}" "${INSTALL_DIR}/${CLI_NAME}"
  echo "${CLI_NAME} CLI installed at ${INSTALL_DIR}/${CLI_NAME}"
else
  echo "[setup] CLI script ${CLI_PATH} not found; skipping install" >&2
fi
