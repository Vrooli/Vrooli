#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
INSTALL_DIR="${HOME}/.vrooli/bin"

mkdir -p "${INSTALL_DIR}"
chmod +x "${SCRIPT_DIR}/product-manager-agent"
ln -sf "${SCRIPT_DIR}/product-manager-agent" "${INSTALL_DIR}/product-manager-agent"

echo "Installed product-manager-agent to ${INSTALL_DIR}/product-manager-agent"
