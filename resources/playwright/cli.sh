#!/usr/bin/env bash
set -euo pipefail

CMD="${1:-}"
PORT="${PLAYWRIGHT_DRIVER_PORT:-39400}"
HOST="${PLAYWRIGHT_DRIVER_HOST:-127.0.0.1}"
PID_FILE="${PLAYWRIGHT_PID_FILE:-/tmp/vrooli-playwright-driver.pid}"
LOG_FILE="${PLAYWRIGHT_LOG_FILE:-/tmp/vrooli-playwright-driver.log}"

start() {
  if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    echo "playwright driver already running (pid $(cat "$PID_FILE"))"
    return 0
  fi
  node driver/server.js >/dev/null 2>>"$LOG_FILE" &
  echo $! >"$PID_FILE"
  echo "started playwright driver on ${HOST}:${PORT} (pid $(cat "$PID_FILE"))"
}

stop() {
  if [ -f "$PID_FILE" ]; then
    kill "$(cat "$PID_FILE")" 2>/dev/null || true
    rm -f "$PID_FILE"
    echo "stopped playwright driver"
  else
    echo "playwright driver not running"
  fi
}

status() {
  if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    echo "running (pid $(cat "$PID_FILE"))"
  else
    echo "stopped"
  fi
}

case "$CMD" in
  start) start ;;
  stop) stop ;;
  restart) stop; start ;;
  status) status ;;
  *)
    echo "usage: $0 {start|stop|restart|status}"
    exit 1
    ;;
esac
