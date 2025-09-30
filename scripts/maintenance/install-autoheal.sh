#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "${BASH_SOURCE[0]%/*}" && pwd)"
SRC_SCRIPT="$SCRIPT_DIR/vrooli-autoheal.sh"
TARGET_PATH="${VROOLI_AUTOHEAL_TARGET:-/usr/local/bin/vrooli-autoheal.sh}"
CRON_SCHEDULE="${VROOLI_AUTOHEAL_SCHEDULE:-@hourly}"
RESOURCES="${VROOLI_AUTOHEAL_RESOURCES:-postgres,redis,qdrant,ollama,searxng,browserless}"
SCENARIOS="${VROOLI_AUTOHEAL_SCENARIOS:-app-monitor,system-monitor,ecosystem-manager,maintenance-orchestrator,app-issue-tracker,vrooli-orchestrator,scenario-auditor,scenario-authenticator,web-console}"
GRACE="${VROOLI_AUTOHEAL_GRACE_SECONDS:-60}"
LOG_FILE="${VROOLI_AUTOHEAL_LOG_FILE:-/var/log/vrooli-autoheal.log}"
LOCK_FILE="${VROOLI_AUTOHEAL_LOCK_FILE:-/tmp/vrooli-autoheal.lock}"

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
  VROOLI_AUTOHEAL_LOG_FILE       Log file path (default: $LOG_FILE)
  VROOLI_AUTOHEAL_LOCK_FILE      Lock file path (default: $LOCK_FILE)
USAGE
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

    {
        cat "$tmp" 2>/dev/null || true
        echo '# Vrooli autoheal'
        echo "$CRON_SCHEDULE PATH=/usr/local/bin:/usr/bin:/bin VROOLI_AUTOHEAL_RESOURCES=$RESOURCES VROOLI_AUTOHEAL_SCENARIOS=$SCENARIOS VROOLI_AUTOHEAL_GRACE_SECONDS=$GRACE VROOLI_AUTOHEAL_LOG_FILE=$LOG_FILE VROOLI_AUTOHEAL_LOCK_FILE=$LOCK_FILE $TARGET_PATH"
    } | crontab -

    rm -f "$tmp"
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
