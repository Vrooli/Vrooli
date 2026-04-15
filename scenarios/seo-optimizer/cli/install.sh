#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
INSTALL_DIR="${HOME}/.vrooli/bin"

mkdir -p "${INSTALL_DIR}"
chmod +x "${SCRIPT_DIR}/seo-optimizer"
ln -sf "${SCRIPT_DIR}/seo-optimizer" "${INSTALL_DIR}/seo-optimizer"

echo "Installed seo-optimizer to ${INSTALL_DIR}/seo-optimizer"
