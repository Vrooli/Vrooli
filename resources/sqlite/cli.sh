#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "${BASH_SOURCE[0]%/*}" && pwd)"
APP_ROOT="${APP_ROOT:-$(cd "${SCRIPT_DIR}/../.." && pwd)}"
BIN_LOCAL="${SCRIPT_DIR}/resource-sqlite"
BIN_LOCAL_EMBED="${SCRIPT_DIR}/.bin/resource-sqlite"
BIN_HOME="${VROOLI_BIN:-${HOME}/.vrooli/bin}/resource-sqlite"

delegate_standard_command() {
  local action="$1"
  shift || true

  if [[ -n "${VROOLI_CLI_BIN:-}" ]]; then
    (
      cd "${APP_ROOT}"
      "${VROOLI_CLI_BIN}" resource "${action}" "sqlite" "$@"
    )
    return
  fi

  if command -v vrooli >/dev/null 2>&1; then
    (
      cd "${APP_ROOT}"
      vrooli resource "${action}" "sqlite" "$@"
    )
    return
  fi

  if [[ -f "${APP_ROOT}/go.mod" ]] && command -v go >/dev/null 2>&1; then
    (
      cd "${APP_ROOT}"
      go run ./cmd/vrooli resource "${action}" "sqlite" "$@"
    )
    return
  fi

  echo "Native Vrooli CLI is unavailable for delegated resource command" >&2
  exit 1
}

if [[ $# -gt 0 ]]; then
  case "$1" in
    status)
      shift
      delegate_standard_command status "$@"
      ;;
    logs)
      shift
      delegate_standard_command logs "$@"
      ;;
    manage)
      subcommand="${2:-}"
      case "${subcommand}" in
        install|uninstall|start|stop|restart)
          shift 2
          delegate_standard_command "${subcommand}" "$@"
          ;;
      esac
      ;;
  esac
fi

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
