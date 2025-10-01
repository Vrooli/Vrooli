#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "${BASH_SOURCE[0]%/*}" && pwd)"
SRC_SCRIPT="$SCRIPT_DIR/vrooli-autoheal.sh"
TARGET_PATH="${VROOLI_AUTOHEAL_TARGET:-/usr/local/bin/vrooli-autoheal.sh}"
CRON_SCHEDULE="${VROOLI_AUTOHEAL_SCHEDULE:-@hourly}"
RESOURCES="${VROOLI_AUTOHEAL_RESOURCES:-postgres,redis,qdrant,ollama,searxng,browserless}"
SCENARIOS="${VROOLI_AUTOHEAL_SCENARIOS:-app-monitor,system-monitor,ecosystem-manager,maintenance-orchestrator,app-issue-tracker,vrooli-orchestrator,scenario-auditor,scenario-authenticator,web-console}"
GRACE="${VROOLI_AUTOHEAL_GRACE_SECONDS:-60}"
VERIFY_DELAY="${VROOLI_AUTOHEAL_VERIFY_DELAY:-30}"
CMD_TIMEOUT="${VROOLI_AUTOHEAL_CMD_TIMEOUT:-120}"
API_PORT="${VROOLI_API_PORT:-8092}"
API_URL_DEFAULT="http://127.0.0.1:${API_PORT}/health"
API_URL="${VROOLI_AUTOHEAL_API_URL:-$API_URL_DEFAULT}"
API_TIMEOUT="${VROOLI_AUTOHEAL_API_TIMEOUT:-5}"
API_RECOVERY="${VROOLI_AUTOHEAL_API_RECOVERY:-}"
LOG_FILE="${VROOLI_AUTOHEAL_LOG_FILE:-/var/log/vrooli-autoheal.log}"
LOCK_FILE="${VROOLI_AUTOHEAL_LOCK_FILE:-/tmp/vrooli-autoheal.lock}"
LOGROTATE_FILE="${VROOLI_AUTOHEAL_LOGROTATE_PATH:-/etc/logrotate.d/vrooli-autoheal}"

usage() {
    cat <<USAGE
Install or remove the Vrooli autoheal watchdog.

Usage: ${0##*/} [--remove]

Environment variables:
  VROOLI_AUTOHEAL_TARGET         Target path for installed script (default: $TARGET_PATH)
  VROOLI_AUTOHEAL_SCHEDULE       Cron expression (default: $CRON_SCHEDULE)
  VROOLI_AUTOHEAL_RESOURCES      Comma-separated resources to enforce
  VROOLI_AUTOHEAL_SCENARIOS      Comma-separated scenarios to enforce
  VROOLI_AUTOHEAL_GRACE_SECONDS  Seconds to wait before checks begin (default: $GRACE)
  VROOLI_AUTOHEAL_VERIFY_DELAY   Seconds to wait after restart before verification (default: $VERIFY_DELAY)
  VROOLI_AUTOHEAL_CMD_TIMEOUT    Seconds before CLI commands time out (default: $CMD_TIMEOUT)
  VROOLI_AUTOHEAL_API_URL        API health endpoint (default: $API_URL)
  VROOLI_AUTOHEAL_API_TIMEOUT    Seconds before API health check times out (default: $API_TIMEOUT)
  VROOLI_AUTOHEAL_API_RECOVERY   Command to run when API health fails (default: none)
  VROOLI_AUTOHEAL_LOG_FILE       Log file path (default: $LOG_FILE)
  VROOLI_AUTOHEAL_LOCK_FILE      Lock file path (default: $LOCK_FILE)
  VROOLI_AUTOHEAL_LOGROTATE_PATH Logrotate config path (default: $LOGROTATE_FILE)
USAGE
}

install_logrotate() {
    local config_dir
    config_dir="$(dirname "$LOGROTATE_FILE")"

    if [[ ! -d "$config_dir" ]]; then
        echo "Skipping logrotate install (directory $config_dir not found)"
        return
    fi

    if [[ $EUID -ne 0 && ! -w "$LOGROTATE_FILE" && ! -w "$config_dir" ]]; then
        echo "Insufficient permissions to create $LOGROTATE_FILE; skipping logrotate config"
        return
    fi

    cat >"$LOGROTATE_FILE" <<EOF
$LOG_FILE {
    weekly
    rotate 6
    compress
    missingok
    notifempty
    copytruncate
}
EOF
    echo "Logrotate configuration installed at $LOGROTATE_FILE"
}

remove_job() {
    local tmp
    tmp="$(mktemp)"
    if crontab -l >/dev/null 2>&1; then
        crontab -l | grep -v '# Vrooli autoheal' | grep -v 'vrooli-autoheal.sh' >"$tmp" || true
        crontab "$tmp" || true
    fi
    rm -f "$tmp"
    if [[ -f "$TARGET_PATH" ]]; then
        echo "Removing $TARGET_PATH"
        rm -f "$TARGET_PATH"
    fi
    if [[ -f "$LOGROTATE_FILE" ]]; then
        if [[ $EUID -ne 0 && ! -w "$LOGROTATE_FILE" ]]; then
            echo "Logrotate config present at $LOGROTATE_FILE (insufficient permissions to remove)"
        else
            rm -f "$LOGROTATE_FILE"
            echo "Removed $LOGROTATE_FILE"
        fi
    fi
    echo "Autoheal cron job removed"
}

install_job() {
    if [[ ! -f "$SRC_SCRIPT" ]]; then
        echo "Source script not found: $SRC_SCRIPT" >&2
        exit 1
    fi

    if [[ $EUID -ne 0 && ! -w "$(dirname "$TARGET_PATH")" ]]; then
        echo "Target directory $(dirname "$TARGET_PATH") requires elevated permissions" >&2
        exit 1
    fi

    install -m 0755 "$SRC_SCRIPT" "$TARGET_PATH"
    mkdir -p "$(dirname "$LOG_FILE")"
    touch "$LOG_FILE"

    local tmp
    tmp="$(mktemp)"
    if crontab -l >/dev/null 2>&1; then
        crontab -l | grep -v '# Vrooli autoheal' | grep -v 'vrooli-autoheal.sh' >"$tmp" || true
    fi

    resources_clean="$(printf %s "$RESOURCES" | tr -d '\n' | tr -d '[:space:]')"
    scenarios_clean="$(printf %s "$SCENARIOS" | tr -d '\n' | tr -d '[:space:]')"
    grace_escaped="$(printf %q "$GRACE")"
    verify_escaped="$(printf %q "$VERIFY_DELAY")"
    cmd_timeout_escaped="$(printf %q "$CMD_TIMEOUT")"
    api_url_escaped="$(printf %q "$API_URL")"
    api_timeout_escaped="$(printf %q "$API_TIMEOUT")"
    api_recovery_escaped="$(printf %q "$API_RECOVERY")"
    log_file_escaped="$(printf %q "$LOG_FILE")"
    lock_file_escaped="$(printf %q "$LOCK_FILE")"
    target_path_escaped="$(printf %q "$TARGET_PATH")"

    {
        cat "$tmp" 2>/dev/null || true
        echo '# Vrooli autoheal'
        printf '%s PATH=/usr/local/bin:/usr/bin:/bin ' "$CRON_SCHEDULE"
        printf 'VROOLI_AUTOHEAL_RESOURCES=%s ' "$resources_clean"
        printf 'VROOLI_AUTOHEAL_SCENARIOS=%s ' "$scenarios_clean"
        printf 'VROOLI_AUTOHEAL_GRACE_SECONDS=%s ' "$grace_escaped"
        printf 'VROOLI_AUTOHEAL_VERIFY_DELAY=%s ' "$verify_escaped"
        printf 'VROOLI_AUTOHEAL_CMD_TIMEOUT=%s ' "$cmd_timeout_escaped"
        printf 'VROOLI_AUTOHEAL_API_URL=%s ' "$api_url_escaped"
        printf 'VROOLI_AUTOHEAL_API_TIMEOUT=%s ' "$api_timeout_escaped"
        printf 'VROOLI_AUTOHEAL_API_RECOVERY=%s ' "$api_recovery_escaped"
        printf 'VROOLI_AUTOHEAL_LOG_FILE=%s ' "$log_file_escaped"
        printf 'VROOLI_AUTOHEAL_LOCK_FILE=%s ' "$lock_file_escaped"
        printf '%s' "$target_path_escaped"
        printf '\n'
    } | crontab -

    rm -f "$tmp"
    install_logrotate
    echo "Autoheal installed: schedule $CRON_SCHEDULE"
}

case "${1:-}" in
    --remove)
        remove_job
        ;;
    --help|-h)
        usage
        ;;
    "")
        install_job
        ;;
    *)
        usage >&2
        exit 1
        ;;
esac
