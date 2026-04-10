#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../.." && pwd)}"
RESOURCE_DIR="${APP_ROOT}/resources/vnc"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Session metadata directory
VNC_SESSIONS_DIR="${VNC_SESSIONS_DIR:-${HOME}/.vrooli/vnc/sessions}"

show_help() {
    cat <<'HELP'
Usage: resource-vnc <command> [options]

Commands:
  start     Start x11vnc + websockify on a display
  stop      Stop a VNC session
  status    Show running VNC sessions
  health    Check if x11vnc and websockify are available

Start options:
  --display DISPLAY   X display to serve (default :99)
  --vnc-port PORT     VNC port (default: auto-assign 5900-5999)
  --ws-port PORT      WebSocket port (default: auto-assign 6080-6180)

Examples:
  resource-vnc start --display :99
  resource-vnc stop abc123
  resource-vnc status
  resource-vnc health
HELP
}

# Generate a short session ID
generate_session_id() {
    head -c 6 /dev/urandom | base64 | tr '+/' '-_' | tr -d '='
}

# Find an available port in range
find_available_port() {
    local start="$1" end="$2"
    for port in $(seq "$start" "$end"); do
        if ! ss -tlnp 2>/dev/null | grep -q ":${port} " && \
           ! ss -tlnp 2>/dev/null | grep -q ":${port}\b"; then
            echo "$port"
            return 0
        fi
    done
    return 1
}

cmd_start() {
    local display=":99"
    local vnc_port=""
    local ws_port=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --display) display="$2"; shift 2 ;;
            --vnc-port) vnc_port="$2"; shift 2 ;;
            --ws-port) ws_port="$2"; shift 2 ;;
            *) log::error "Unknown option: $1"; return 1 ;;
        esac
    done

    # Auto-assign ports if not specified
    if [[ -z "$vnc_port" ]]; then
        vnc_port=$(find_available_port 5900 5999) || { log::error "No available VNC port in 5900-5999"; return 1; }
    fi
    if [[ -z "$ws_port" ]]; then
        ws_port=$(find_available_port 6080 6180) || { log::error "No available WebSocket port in 6080-6180"; return 1; }
    fi

    # Start x11vnc
    x11vnc -display "$display" -rfbport "$vnc_port" -nopw -shared -forever -noxdamage -bg -q 2>/dev/null
    local x11vnc_pid
    x11vnc_pid=$(pgrep -f "x11vnc.*-rfbport ${vnc_port}" | head -1)
    if [[ -z "$x11vnc_pid" ]]; then
        log::error "Failed to start x11vnc"
        return 1
    fi

    # Start websockify
    websockify --daemon "$ws_port" "localhost:${vnc_port}" >/dev/null 2>&1
    local ws_pid
    # Give websockify a moment to start
    sleep 0.5
    ws_pid=$(pgrep -f "websockify.*${ws_port}.*localhost:${vnc_port}" | head -1)
    if [[ -z "$ws_pid" ]]; then
        kill "$x11vnc_pid" 2>/dev/null || true
        log::error "Failed to start websockify"
        return 1
    fi

    # Save session metadata
    local session_id
    session_id=$(generate_session_id)
    mkdir -p "${VNC_SESSIONS_DIR}"
    cat > "${VNC_SESSIONS_DIR}/${session_id}.json" <<METADATA
{
  "session_id": "${session_id}",
  "display": "${display}",
  "vnc_port": ${vnc_port},
  "ws_port": ${ws_port},
  "x11vnc_pid": ${x11vnc_pid},
  "websockify_pid": ${ws_pid},
  "started_at": "$(date -Iseconds)"
}
METADATA

    # Output session ID to stdout (for programmatic use)
    echo "$session_id"
}

cmd_stop() {
    local session_id="$1"
    local meta_file="${VNC_SESSIONS_DIR}/${session_id}.json"

    if [[ ! -f "$meta_file" ]]; then
        log::error "Session not found: ${session_id}"
        return 1
    fi

    local x11vnc_pid ws_pid
    x11vnc_pid=$(jq -r '.x11vnc_pid' "$meta_file")
    ws_pid=$(jq -r '.websockify_pid' "$meta_file")

    # Kill websockify first, then x11vnc
    kill "$ws_pid" 2>/dev/null || true
    kill "$x11vnc_pid" 2>/dev/null || true

    rm -f "$meta_file"
    log::info "Stopped VNC session ${session_id}"
}

cmd_status() {
    local session_id="${1:-}"
    mkdir -p "${VNC_SESSIONS_DIR}"

    if [[ -n "$session_id" ]]; then
        local meta_file="${VNC_SESSIONS_DIR}/${session_id}.json"
        if [[ -f "$meta_file" ]]; then
            cat "$meta_file"
        else
            log::error "Session not found: ${session_id}"
            return 1
        fi
        return 0
    fi

    # List all sessions
    local count=0
    for f in "${VNC_SESSIONS_DIR}"/*.json; do
        [[ -f "$f" ]] || continue
        # Verify processes are still alive
        local x11vnc_pid ws_pid
        x11vnc_pid=$(jq -r '.x11vnc_pid' "$f")
        ws_pid=$(jq -r '.websockify_pid' "$f")
        if kill -0 "$x11vnc_pid" 2>/dev/null && kill -0 "$ws_pid" 2>/dev/null; then
            cat "$f"
            count=$((count + 1))
        else
            # Clean up stale session
            rm -f "$f"
        fi
    done

    if [[ "$count" -eq 0 ]]; then
        echo "No active VNC sessions"
    fi
}

cmd_health() {
    local ok=true
    if command -v x11vnc &>/dev/null; then
        echo "x11vnc: installed ($(x11vnc -version 2>&1 | head -1))"
    else
        echo "x11vnc: NOT INSTALLED"
        ok=false
    fi
    if command -v websockify &>/dev/null; then
        echo "websockify: installed"
    else
        echo "websockify: NOT INSTALLED"
        ok=false
    fi
    $ok
}

# Route commands
case "${1:-help}" in
    start)   shift; cmd_start "$@" ;;
    stop)    shift; cmd_stop "$@" ;;
    status)  shift; cmd_status "$@" ;;
    health)  shift; cmd_health ;;
    help|-h|--help) show_help ;;
    *)       log::error "Unknown command: $1"; show_help; exit 1 ;;
esac
