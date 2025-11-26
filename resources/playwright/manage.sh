#!/usr/bin/env bash
set -euo pipefail

CMD="${1:-}"
shift || true
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PID_FILE="${PLAYWRIGHT_PID_FILE:-/tmp/vrooli-playwright-driver.pid}"

usage() {
  echo "usage: $0 {install|start|stop|restart|status|health|logs|info|help}"
}

health() {
  local port="${PLAYWRIGHT_DRIVER_PORT:-39400}"
  curl -fsS "http://127.0.0.1:${port}/health" >/dev/null 2>&1
}

install() {
  (cd "$ROOT" && npm install)
}

info() {
  if command -v jq >/dev/null 2>&1; then
    jq '.' "$ROOT/config/runtime.json"
  else
    cat "$ROOT/config/runtime.json"
  fi
}

start() {
  "$ROOT/cli.sh" start
}

stop() {
  "$ROOT/cli.sh" stop
}

status() {
  "$ROOT/cli.sh" status "$@"
}

case "$CMD" in
  install) install ;;
  start) start ;;
  stop) stop ;;
  restart) stop; start ;;
  status) status "$@" ;;
  health) health && echo "healthy" || { echo "unhealthy"; exit 1; } ;;
  info) info ;;
  logs) "$ROOT/cli.sh" logs ;;
  help|-h|--help|"") usage ;;
  *)
    usage
    exit 1
    ;;
esac
