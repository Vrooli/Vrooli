#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SOURCE="${SCRIPT_DIR}/competitor-monitor"
INSTALL_DIR="${HOME}/.vrooli/bin"
PRIMARY_LINK="${INSTALL_DIR}/competitor-monitor"
ALIAS_LINK="${INSTALL_DIR}/competitor-change-monitor"

mkdir -p "${INSTALL_DIR}"

if [[ ! -f "${CLI_SOURCE}" ]]; then
  echo "[setup] CLI script ${CLI_SOURCE} not found; skipping install" >&2
  exit 0
fi

chmod +x "${CLI_SOURCE}"
ln -sf "${CLI_SOURCE}" "${PRIMARY_LINK}"
ln -sf "${CLI_SOURCE}" "${ALIAS_LINK}"

echo "competitor-monitor CLI installed to ${PRIMARY_LINK}"
echo "Alias available via ${ALIAS_LINK}"
