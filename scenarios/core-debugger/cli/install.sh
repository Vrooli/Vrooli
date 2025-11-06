#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SOURCE="${SCRIPT_DIR}/core-debugger"
INSTALL_DIR="${HOME}/.vrooli/bin"
TARGET_LINK="${INSTALL_DIR}/core-debugger"

mkdir -p "${INSTALL_DIR}"

if [[ ! -f "${CLI_SOURCE}" ]]; then
  echo "[setup] CLI script ${CLI_SOURCE} not found; skipping install" >&2
  exit 0
fi

chmod +x "${CLI_SOURCE}"
ln -sf "${CLI_SOURCE}" "${TARGET_LINK}"

echo "core-debugger CLI installed to ${TARGET_LINK}"
