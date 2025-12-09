#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "${BASH_SOURCE[0]%/*}" && pwd)"
BIN_LOCAL="${SCRIPT_DIR}/resource-sqlite"
BIN_LOCAL_EMBED="${SCRIPT_DIR}/.bin/resource-sqlite"
BIN_HOME="${VROOLI_BIN:-${HOME}/.vrooli/bin}/resource-sqlite"

if [[ -x "$BIN_LOCAL" ]]; then
  exec "$BIN_LOCAL" "$@"
elif [[ -x "$BIN_LOCAL_EMBED" ]]; then
  exec "$BIN_LOCAL_EMBED" "$@"
elif [[ -x "$BIN_HOME" ]]; then
  exec "$BIN_HOME" "$@"
elif command -v resource-sqlite >/dev/null 2>&1; then
  exec resource-sqlite "$@"
else
  cat >&2 <<'EOF'
resource-sqlite binary not found.
Build and install with: ./install.sh   (or ./install.ps1 on Windows)
EOF
  exit 1
fi
