#!/usr/bin/env bash
set -euo pipefail

CMD="${1:-}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

usage() {
  echo "usage: $0 {install|start|stop|restart|status|health|logs|info|help}"
}

case "$CMD" in
  install|uninstall|start|stop|restart|status|health|logs|info|test|content|manage)
    shift || true
    exec "$ROOT/cli.sh" "$CMD" "$@"
    ;;
  help|-h|--help|"") usage ;;
  *)
    usage
    exit 1
    ;;
esac
