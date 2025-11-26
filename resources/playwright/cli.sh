#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CMD="${1:-}"
shift || true

PORT="${PLAYWRIGHT_DRIVER_PORT:-39400}"
HOST="${PLAYWRIGHT_DRIVER_HOST:-127.0.0.1}"
PID_FILE="${PLAYWRIGHT_PID_FILE:-/tmp/vrooli-playwright-driver.pid}"
LOG_FILE="${PLAYWRIGHT_LOG_FILE:-/tmp/vrooli-playwright-driver.log}"

usage() {
  cat <<EOF
usage: $0 {install|start|stop|restart|status|health|logs|info|help} [options]

Commands:
  install   Install npm dependencies (playwright-core)
  start     Start the driver (respects PLAYWRIGHT_DRIVER_PORT/HOST)
  stop      Stop the driver (PID-file based)
  restart   Restart the driver
  status    Show status (supports --json / --format json / --verbose)
  health    Return 0/1 based on /health endpoint
  logs      Tail driver logs (uses PLAYWRIGHT_LOG_FILE)
  info      Show runtime metadata from config/runtime.json
EOF
}

install() {
  (cd "$ROOT" && npm install >/dev/null 2>&1)
}

info() {
  local runtime="${ROOT}/config/runtime.json"
  if command -v jq >/dev/null 2>&1; then
    jq '.' "$runtime"
  else
    cat "$runtime"
  fi
}

start() {
  if [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    echo "playwright driver already running (pid $(cat "$PID_FILE"))"
    return 0
  fi
  PLAYWRIGHT_DRIVER_PORT="$PORT" PLAYWRIGHT_DRIVER_HOST="$HOST" \
    node "$ROOT/driver/server.js" >>"$LOG_FILE" 2>&1 &
  echo $! >"$PID_FILE"
  echo "started playwright driver on ${HOST}:${PORT} (pid $(cat "$PID_FILE"))"
}

stop() {
  if [[ -f "$PID_FILE" ]]; then
    kill "$(cat "$PID_FILE")" 2>/dev/null || true
    rm -f "$PID_FILE"
    echo "stopped playwright driver"
  else
    echo "playwright driver not running"
  fi
}

health_raw() {
  local url="http://${HOST}:${PORT}/health"
  curl -fsS "$url" >/dev/null 2>&1
}

health() {
  if health_raw; then
    echo "healthy"
    return 0
  fi
  echo "unhealthy"
  return 1
}

status() {
  local format="text"
  local verbose="false"
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --json) format="json"; shift ;;
      --format) shift; format="${1:-text}"; shift || true ;;
      --verbose|-v) verbose="true"; shift ;;
      *) shift ;;
    esac
  done

  local running="false"
  local healthy="false"
  local pid=""
  if [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    running="true"
    pid="$(cat "$PID_FILE")"
    if health_raw; then
      healthy="true"
    fi
  fi

  if [[ "$format" == "json" ]]; then
    local status_value
    if [[ "$healthy" == "true" ]]; then
      status_value="healthy"
    elif [[ "$running" == "true" ]]; then
      status_value="starting"
    else
      status_value="stopped"
    fi

    if command -v jq >/dev/null 2>&1; then
      jq -n \
        --arg status "$status_value" \
        --argjson running "$running" \
        --argjson healthy "$healthy" \
        --arg host "$HOST" \
        --arg port "$PORT" \
        --arg pid "$pid" \
        '{
          status: $status,
          running: $running,
          healthy: $healthy,
          host: $host,
          port: ($port|tonumber? // $port),
          pid: ($pid | select(. != "") // null)
        }'
    else
      # Minimal JSON fallback without jq
      echo "{"
      echo "  \"status\": \"${status_value}\","
      echo "  \"running\": ${running},"
      echo "  \"healthy\": ${healthy},"
      echo "  \"host\": \"${HOST}\","
      echo "  \"port\": ${PORT},"
      echo "  \"pid\": \"${pid}\""
      echo "}"
    fi
  else
    if [[ "$healthy" == "true" ]]; then
      echo "playwright driver: healthy (pid ${pid}) on ${HOST}:${PORT}"
    elif [[ "$running" == "true" ]]; then
      echo "playwright driver: running but health check failed (pid ${pid}) on ${HOST}:${PORT}"
    else
      echo "playwright driver: stopped"
    fi
    if [[ "$verbose" == "true" && -f "$LOG_FILE" ]]; then
      echo "--- recent log lines ---"
      tail -n 10 "$LOG_FILE" || true
    fi
  fi

  if [[ "$healthy" == "true" ]]; then
    return 0
  elif [[ "$running" == "true" ]]; then
    return 1
  else
    return 2
  fi
}

logs() {
  if [[ -f "$LOG_FILE" ]]; then
    tail -n 100 "$LOG_FILE"
  else
    echo "no log file at $LOG_FILE"
  fi
}

case "$CMD" in
  install) install ;;
  start) start ;;
  stop) stop ;;
  restart) stop; start ;;
  status) status "$@" ;;
  health) health ;;
  logs) logs ;;
  info) info ;;
  help|-h|--help|"") usage ;;
  *) echo "unknown command: $CMD"; usage; exit 1 ;;
esac
