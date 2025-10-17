#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../.." && pwd)}"
API_PORT="${VROOLI_API_PORT:-8092}"
LOG_FILE="${VROOLI_API_RESTART_LOG:-/var/log/vrooli-api-restart.log}"
FALLBACK_LOG="${HOME}/.vrooli/logs/vrooli-api-restart.log"

prepare_log() {
    local target="$1"
    if mkdir -p "$(dirname "$target")" 2>/dev/null && touch "$target" 2>/dev/null; then
        LOG_FILE="$target"
        return 0
    fi
    return 1
}

if ! prepare_log "$LOG_FILE"; then
    if ! prepare_log "$FALLBACK_LOG"; then
        printf '%s\n' "ERROR: Unable to create log file at '$LOG_FILE' or fallback '$FALLBACK_LOG'" >&2
        exit 1
    fi
fi

log() {
    printf '%s :: %s\n' "$(date -Is)" "$1" | tee -a "$LOG_FILE" >/dev/null
}

log "Initiating Vrooli API restart on port $API_PORT"

if command -v lsof >/dev/null 2>&1; then
    mapfile -t pids < <( (lsof -t -i ":${API_PORT}" 2>/dev/null | sort -u) || printf '')
elif command -v fuser >/dev/null 2>&1; then
    mapfile -t pids < <( (fuser -n tcp "$API_PORT" 2>/dev/null | tr ' ' '\n' | grep -E '^[0-9]+$' | sort -u) || printf '')
else
    log "WARN: neither lsof nor fuser available; skipping targeted process kill"
    pids=()
fi

for pid in "${pids[@]}"; do
    if [[ -n "$pid" ]]; then
        log "Stopping process $pid bound to port $API_PORT"
        if ! kill "$pid" 2>/dev/null; then
            log "WARN: failed to send SIGTERM to $pid"
        fi
    fi
done

sleep 2

if (( ${#pids[@]} > 0 )); then
    for pid in "${pids[@]}"; do
        if [[ -n "$pid" && -d "/proc/$pid" ]]; then
            log "Process $pid still alive; sending SIGKILL"
            kill -9 "$pid" 2>/dev/null || true
        fi
    done
fi

log "Starting Vrooli API using api/start.sh"
nohup env VROOLI_API_PORT="$API_PORT" bash "$APP_ROOT/api/start.sh" >>"$LOG_FILE" 2>&1 &
api_pid=$!
disown "$api_pid" 2>/dev/null || true
log "API restart command dispatched (PID $api_pid)"
