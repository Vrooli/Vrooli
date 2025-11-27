#!/usr/bin/env bash
# Playwright resource CLI (universal v2.0 contract, non-Docker)
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
ROOT="${APP_ROOT}/resources/playwright"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${ROOT}/config/defaults.sh"

PORT="${PLAYWRIGHT_DRIVER_PORT}"
HOST="${PLAYWRIGHT_DRIVER_HOST}"
STATE_DIR="${PLAYWRIGHT_STATE_DIR:-${TMPDIR:-/tmp}}"
PID_FILE="${PLAYWRIGHT_PID_FILE:-${STATE_DIR%/}/vrooli-playwright-driver.pid}"
LOG_FILE="${PLAYWRIGHT_LOG_FILE:-${STATE_DIR%/}/vrooli-playwright-driver.log}"
PORT_FILE="${PLAYWRIGHT_PORT_FILE:-${STATE_DIR%/}/vrooli-playwright-driver.port}"

pw::resolve_port() {
  if [[ -f "$PORT_FILE" ]]; then
    local p
    p=$(cat "$PORT_FILE" 2>/dev/null | tr -dc '0-9')
    if [[ -n "$p" ]]; then
      echo "$p"
      return
    fi
  fi
  if [[ "$PORT" != "0" ]]; then
    echo "$PORT"
    return
  fi
  # fallback to default if port not discovered
  echo "39400"
  return 1
}

pw::install() {
  (cd "$ROOT" && npm install >/dev/null 2>&1)
}

pw::uninstall() {
  rm -rf "$ROOT/node_modules"
  echo "uninstalled playwright driver (node_modules removed)"
}

pw::start() {
  if [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    echo "playwright driver already running (pid $(cat "$PID_FILE"))"
    return 0
  fi
  rm -f "$PORT_FILE"
  PLAYWRIGHT_DRIVER_PORT="$PORT" PLAYWRIGHT_DRIVER_HOST="$HOST" PLAYWRIGHT_PORT_FILE="$PORT_FILE" \
    node "$ROOT/driver/server.js" >>"$LOG_FILE" 2>&1 &
  echo $! >"$PID_FILE"
  sleep 0.2
  local effective_port rc=0
  effective_port="$(pw::resolve_port)" || rc=$?
  if [[ "$PORT" == "0" && $rc -ne 0 ]]; then
    echo "warning: PLAYWRIGHT_DRIVER_PORT=0 but could not read chosen port from ${PORT_FILE}; falling back to 39400 in status"
    effective_port="39400"
  fi
  echo "started playwright driver on ${HOST}:${effective_port} (pid $(cat "$PID_FILE"))"
}

pw::stop() {
  if [[ -f "$PID_FILE" ]]; then
    kill "$(cat "$PID_FILE")" 2>/dev/null || true
    rm -f "$PID_FILE"
    rm -f "$PORT_FILE"
    echo "stopped playwright driver"
  else
    echo "playwright driver not running"
  fi
}

pw::health_raw() {
  local eff_port
  eff_port="$(pw::resolve_port)"
  local url="http://${HOST}:${eff_port}/health"
  curl -fsS "$url" >/dev/null 2>&1
}

pw::health_cmd() {
  if pw::health_raw; then
    echo "healthy"
    return 0
  fi
  echo "unhealthy"
  return 1
}

pw::status() {
  local format="text"
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --json|--format) format="json"; shift ;;
      *) shift ;;
    esac
  done

  local running="false" healthy="false" pid=""
  local eff_port
  eff_port="$(pw::resolve_port)"
  if [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    running="true"; pid="$(cat "$PID_FILE")"
    if pw::health_raw; then healthy="true"; fi
  fi

  local status_value
  if [[ "$healthy" == "true" ]]; then status_value="healthy"
  elif [[ "$running" == "true" ]]; then status_value="starting"
  else status_value="stopped"; fi

  if [[ "$format" == "json" ]]; then
    echo "{\"status\":\"$status_value\",\"running\":$running,\"healthy\":$healthy,\"host\":\"$HOST\",\"port\":${eff_port},\"pid\":${pid:+\"$pid\"}}"
  else
    if [[ "$healthy" == "true" ]]; then
      echo "playwright driver: healthy (pid ${pid}) on ${HOST}:${eff_port}"
    elif [[ "$running" == "true" ]]; then
      echo "playwright driver: running but health check failed (pid ${pid}) on ${HOST}:${eff_port}"
    else
      echo "playwright driver: stopped"
    fi
  fi

  if [[ "$healthy" == "true" ]]; then return 0
  elif [[ "$running" == "true" ]]; then return 1
  else return 2; fi
}

pw::logs() {
  if [[ -f "$LOG_FILE" ]]; then
    tail -n 100 "$LOG_FILE"
  else
    echo "no log file at $LOG_FILE"
  fi
}

pw::info() {
  local runtime="$ROOT/config/runtime.json"
  if command -v jq >/dev/null 2>&1; then jq '.' "$runtime"; else cat "$runtime"; fi
}

pw::env() {
  local eff_port
  eff_port="$(pw::resolve_port)"
  echo "ENGINE=playwright"
  echo "PLAYWRIGHT_DRIVER_URL=http://${HOST}:${eff_port}"
  echo "PLAYWRIGHT_DRIVER_HOST=${HOST}"
  echo "PLAYWRIGHT_DRIVER_PORT=${eff_port}"
}

pw::test_smoke() {
  local started_here=false
  if [[ ! -f "$PID_FILE" ]] || ! kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    pw::start
    started_here=true
    sleep 1
  fi
  if pw::health_raw; then
    echo "smoke: healthy"
    $started_here && pw::stop || true
    return 0
  fi
  echo "smoke: unhealthy"
  $started_here && pw::stop || true
  return 1
}

pw::content_stub() {
  echo "content commands not supported for resource-playwright"
  return 0
}

cli::init "playwright" "Playwright driver resource" "v2"

# Flat commands (v1 compat)
cli::register_command "status" "Show resource status" "pw::status"
cli::register_command "logs" "Show recent logs" "pw::logs"
cli::register_command "health" "Health check" "pw::health_cmd"
cli::register_command "start" "Start the resource" "pw::start" "modifies-system"
cli::register_command "stop" "Stop the resource" "pw::stop" "modifies-system"
cli::register_command "restart" "Restart the resource" "pw::restart" "modifies-system"
cli::register_command "install" "Install dependencies" "pw::install" "modifies-system"
cli::register_command "uninstall" "Remove dependencies" "pw::uninstall" "modifies-system"

# Override v2 handlers
CLI_COMMAND_HANDLERS["manage::install"]="pw::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="pw::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="pw::start"
CLI_COMMAND_HANDLERS["manage::stop"]="pw::stop"
pw::restart() { pw::stop; pw::start; }
CLI_COMMAND_HANDLERS["manage::restart"]="pw::restart"
CLI_COMMAND_HANDLERS["manage::status"]="pw::status"
CLI_COMMAND_HANDLERS["manage::health"]="pw::health_cmd"
CLI_COMMAND_HANDLERS["manage::logs"]="pw::logs"
CLI_COMMAND_HANDLERS["manage::env"]="pw::env"

CLI_COMMAND_HANDLERS["start"]="pw::start"
CLI_COMMAND_HANDLERS["stop"]="pw::stop"
CLI_COMMAND_HANDLERS["restart"]="pw::restart"
CLI_COMMAND_HANDLERS["install"]="pw::install"
CLI_COMMAND_HANDLERS["uninstall"]="pw::uninstall"
CLI_COMMAND_HANDLERS["env"]="pw::env"
CLI_COMMAND_HANDLERS["test::smoke"]="pw::test_smoke"
CLI_COMMAND_HANDLERS["test::integration"]="pw::test_smoke"
CLI_COMMAND_HANDLERS["test::unit"]="pw::test_smoke"
CLI_COMMAND_HANDLERS["test::all"]="pw::test_smoke"

CLI_COMMAND_HANDLERS["content::add"]="pw::content_stub"
CLI_COMMAND_HANDLERS["content::list"]="pw::content_stub"
CLI_COMMAND_HANDLERS["content::get"]="pw::content_stub"
CLI_COMMAND_HANDLERS["content::remove"]="pw::content_stub"
CLI_COMMAND_HANDLERS["content::execute"]="pw::content_stub"

CLI_COMMAND_HANDLERS["info"]="pw::info"
CLI_COMMAND_HANDLERS["status"]="pw::status"
CLI_COMMAND_HANDLERS["logs"]="pw::logs"
CLI_COMMAND_HANDLERS["health"]="pw::health_cmd"

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cli::dispatch "$@"
fi
