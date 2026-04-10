#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLI_DIR="${APP_ROOT}/scenarios/app-monitor/cli"
source "${APP_ROOT}/scripts/lib/utils/cli-install.sh"

show_help() {
    cat <<'HELP'
App Monitor CLI installer

Usage:
  install.sh              Install app-monitor CLI into ~/.local/bin
  install.sh help         Show this help message
  install.sh version      Show installer version
HELP
}

case "${1:-}" in
    "")
        install_cli "$CLI_DIR/app-monitor" "app-monitor"
        ;;
    help|-h|--help)
        show_help
        ;;
    version|-v|--version)
        echo "app-monitor installer 1.0.0"
        ;;
    *)
        echo "Unknown command: $1" >&2
        echo "Run 'install.sh help' for usage." >&2
        exit 1
        ;;
esac
