#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "${BASH_SOURCE[0]%/*}" && pwd)"
SRC_SCRIPT="$SCRIPT_DIR/vrooli-autoheal.sh"
SRC_MODULES_DIR="$SCRIPT_DIR/autoheal"
TARGET_DIR="${VROOLI_AUTOHEAL_BIN_DIR:-${HOME}/.vrooli/bin}"
TARGET_PATH="${TARGET_DIR}/vrooli-autoheal.sh"
TARGET_MODULES_DIR="${TARGET_DIR}/autoheal"
TARGET_WRAPPER="${TARGET_DIR}/vrooli-autoheal-wrapper.sh"
CRON_SCHEDULE="${VROOLI_AUTOHEAL_SCHEDULE:-@hourly}"

# Configuration defaults
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
LOG_FILE="${VROOLI_AUTOHEAL_LOG_FILE:-${HOME}/.vrooli/logs/vrooli-autoheal.log}"
LOCK_FILE="${VROOLI_AUTOHEAL_LOCK_FILE:-/tmp/vrooli-autoheal.lock}"

# New configuration for enhanced features
INFRASTRUCTURE="${VROOLI_AUTOHEAL_INFRASTRUCTURE:-true}"
SERVICES="${VROOLI_AUTOHEAL_SERVICES:-true}"
SYSTEM="${VROOLI_AUTOHEAL_SYSTEM:-true}"
CHECK_NETWORK="${VROOLI_AUTOHEAL_CHECK_NETWORK:-true}"
CHECK_DNS="${VROOLI_AUTOHEAL_CHECK_DNS:-true}"
CHECK_TIME_SYNC="${VROOLI_AUTOHEAL_CHECK_TIME_SYNC:-true}"
CHECK_CLOUDFLARED="${VROOLI_AUTOHEAL_CHECK_CLOUDFLARED:-true}"
CLOUDFLARED_SERVICE="${VROOLI_AUTOHEAL_CLOUDFLARED_SERVICE:-cloudflared}"
TUNNEL_URL="${VROOLI_AUTOHEAL_TUNNEL_URL:-}"
CLOUDFLARED_TEST_PORT="${VROOLI_AUTOHEAL_CLOUDFLARED_TEST_PORT:-21774}"
CHECK_DISPLAY="${VROOLI_AUTOHEAL_CHECK_DISPLAY:-true}"
DISPLAY_MANAGER="${VROOLI_AUTOHEAL_DISPLAY_MANAGER:-gdm}"
DISPLAY_RESTART="${VROOLI_AUTOHEAL_DISPLAY_RESTART:-false}"
CHECK_SYSTEMD_RESOLVED="${VROOLI_AUTOHEAL_CHECK_RESOLVED:-true}"
CHECK_DOCKER_DAEMON="${VROOLI_AUTOHEAL_CHECK_DOCKER_DAEMON:-true}"
CHECK_DISK="${VROOLI_AUTOHEAL_CHECK_DISK:-true}"
DISK_THRESHOLD="${VROOLI_AUTOHEAL_DISK_THRESHOLD:-85}"
DISK_PARTITIONS="${VROOLI_AUTOHEAL_DISK_PARTITIONS:-/ /home}"
CHECK_INODE="${VROOLI_AUTOHEAL_CHECK_INODE:-true}"
INODE_THRESHOLD="${VROOLI_AUTOHEAL_INODE_THRESHOLD:-85}"
CHECK_SWAP="${VROOLI_AUTOHEAL_CHECK_SWAP:-true}"
SWAP_THRESHOLD="${VROOLI_AUTOHEAL_SWAP_THRESHOLD:-50}"
CHECK_ZOMBIES="${VROOLI_AUTOHEAL_CHECK_ZOMBIES:-true}"
ZOMBIE_THRESHOLD="${VROOLI_AUTOHEAL_ZOMBIE_THRESHOLD:-20}"
ZOMBIE_CLEANUP="${VROOLI_AUTOHEAL_ZOMBIE_CLEANUP:-true}"
CHECK_PORTS="${VROOLI_AUTOHEAL_CHECK_PORTS:-true}"
PORT_THRESHOLD="${VROOLI_AUTOHEAL_PORT_THRESHOLD:-80}"
CHECK_CERTS="${VROOLI_AUTOHEAL_CHECK_CERTS:-false}"
CERT_WARNING_DAYS="${VROOLI_AUTOHEAL_CERT_WARNING_DAYS:-7}"
DNS_TEST="${VROOLI_AUTOHEAL_DNS_TEST:-google.com}"
PING_TEST="${VROOLI_AUTOHEAL_PING_TEST:-8.8.8.8}"

LOGROTATE_FILE="${VROOLI_AUTOHEAL_LOGROTATE_PATH:-${HOME}/.config/logrotate/vrooli-autoheal}"

usage() {
    cat <<USAGE
Install or remove the Vrooli autoheal watchdog (modular edition).

Usage: ${0##*/} [--remove|--help]

Environment variables:
  VROOLI_AUTOHEAL_BIN_DIR           Target directory (default: ~/.vrooli/bin)
  VROOLI_AUTOHEAL_SCHEDULE          Cron expression (default: @hourly)
  VROOLI_AUTOHEAL_RESOURCES         Comma-separated resources
  VROOLI_AUTOHEAL_SCENARIOS         Comma-separated scenarios
  VROOLI_AUTOHEAL_LOG_FILE          Log file path (default: ~/.vrooli/logs/vrooli-autoheal.log)

  # Feature Flags
  VROOLI_AUTOHEAL_INFRASTRUCTURE    Enable infrastructure checks (default: true)
  VROOLI_AUTOHEAL_SERVICES          Enable service checks (default: true)
  VROOLI_AUTOHEAL_SYSTEM            Enable system checks (default: true)

  # Infrastructure
  VROOLI_AUTOHEAL_CHECK_NETWORK     Enable network check (default: true)
  VROOLI_AUTOHEAL_CHECK_DNS         Enable DNS check (default: true)
  VROOLI_AUTOHEAL_CHECK_TIME_SYNC   Enable time sync check (default: true)

  # Services
  VROOLI_AUTOHEAL_CHECK_CLOUDFLARED Enable cloudflared check (default: true)
  VROOLI_AUTOHEAL_TUNNEL_URL        Cloudflare tunnel URL for end-to-end test
  VROOLI_AUTOHEAL_CHECK_DISPLAY     Enable display manager check (default: true)
  VROOLI_AUTOHEAL_DISPLAY_RESTART   Auto-restart display manager (default: false)

  # System
  VROOLI_AUTOHEAL_DISK_THRESHOLD    Disk usage threshold % (default: 85)
  VROOLI_AUTOHEAL_SWAP_THRESHOLD    Swap usage threshold % (default: 50)
  VROOLI_AUTOHEAL_ZOMBIE_THRESHOLD  Zombie process count threshold (default: 20)

See documentation for complete configuration reference.
USAGE
}

install_logrotate() {
    local config_dir
    config_dir="$(dirname "$LOGROTATE_FILE")"

    mkdir -p "$config_dir" 2>/dev/null || true

    if [[ ! -w "$config_dir" ]]; then
        echo "Skipping logrotate install (directory $config_dir not writable)"
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
        crontab -l | grep -v '# Vrooli autoheal' | grep -v 'vrooli-autoheal' >"$tmp" || true
        crontab "$tmp" || true
    fi
    rm -f "$tmp"

    if [[ -f "$TARGET_PATH" ]]; then
        echo "Removing $TARGET_PATH"
        rm -f "$TARGET_PATH"
    fi

    if [[ -f "$TARGET_WRAPPER" ]]; then
        echo "Removing $TARGET_WRAPPER"
        rm -f "$TARGET_WRAPPER"
    fi

    if [[ -d "$TARGET_MODULES_DIR" ]]; then
        echo "Removing $TARGET_MODULES_DIR"
        rm -rf "$TARGET_MODULES_DIR"
    fi

    if [[ -f "$LOGROTATE_FILE" ]]; then
        rm -f "$LOGROTATE_FILE"
        echo "Removed $LOGROTATE_FILE"
    fi

    echo "Autoheal cron job removed"
}

install_job() {
    if [[ ! -f "$SRC_SCRIPT" ]]; then
        echo "Source script not found: $SRC_SCRIPT" >&2
        exit 1
    fi

    if [[ ! -d "$SRC_MODULES_DIR" ]]; then
        echo "Source modules directory not found: $SRC_MODULES_DIR" >&2
        exit 1
    fi

    # Create target directory
    mkdir -p "$TARGET_DIR"

    # Install main script
    install -m 0755 "$SRC_SCRIPT" "$TARGET_PATH"
    echo "Installed main script: $TARGET_PATH"

    # Install modules directory
    if [[ -d "$TARGET_MODULES_DIR" ]]; then
        rm -rf "$TARGET_MODULES_DIR"
    fi
    cp -r "$SRC_MODULES_DIR" "$TARGET_MODULES_DIR"
    chmod -R +x "$TARGET_MODULES_DIR"
    echo "Installed modules: $TARGET_MODULES_DIR"

    # Create wrapper script with all environment variables
    cat >"$TARGET_WRAPPER" <<'WRAPPER_EOF'
#!/usr/bin/env bash
# Vrooli Autoheal Wrapper - Sets environment and calls main script
export PATH="/usr/local/bin:/usr/bin:/bin:${PATH}"
WRAPPER_EOF

    # Add environment variables to wrapper
    cat >>"$TARGET_WRAPPER" <<WRAPPER_VARS
export VROOLI_AUTOHEAL_RESOURCES="$RESOURCES"
export VROOLI_AUTOHEAL_SCENARIOS="$SCENARIOS"
export VROOLI_AUTOHEAL_GRACE_SECONDS="$GRACE"
export VROOLI_AUTOHEAL_VERIFY_DELAY="$VERIFY_DELAY"
export VROOLI_AUTOHEAL_CMD_TIMEOUT="$CMD_TIMEOUT"
export VROOLI_AUTOHEAL_API_URL="$API_URL"
export VROOLI_AUTOHEAL_API_TIMEOUT="$API_TIMEOUT"
export VROOLI_AUTOHEAL_API_RECOVERY="$API_RECOVERY"
export VROOLI_AUTOHEAL_LOG_FILE="$LOG_FILE"
export VROOLI_AUTOHEAL_LOCK_FILE="$LOCK_FILE"
export VROOLI_AUTOHEAL_INFRASTRUCTURE="$INFRASTRUCTURE"
export VROOLI_AUTOHEAL_SERVICES="$SERVICES"
export VROOLI_AUTOHEAL_SYSTEM="$SYSTEM"
export VROOLI_AUTOHEAL_CHECK_NETWORK="$CHECK_NETWORK"
export VROOLI_AUTOHEAL_CHECK_DNS="$CHECK_DNS"
export VROOLI_AUTOHEAL_CHECK_TIME_SYNC="$CHECK_TIME_SYNC"
export VROOLI_AUTOHEAL_CHECK_CLOUDFLARED="$CHECK_CLOUDFLARED"
export VROOLI_AUTOHEAL_CLOUDFLARED_SERVICE="$CLOUDFLARED_SERVICE"
export VROOLI_AUTOHEAL_TUNNEL_URL="$TUNNEL_URL"
export VROOLI_AUTOHEAL_CLOUDFLARED_TEST_PORT="$CLOUDFLARED_TEST_PORT"
export VROOLI_AUTOHEAL_CHECK_DISPLAY="$CHECK_DISPLAY"
export VROOLI_AUTOHEAL_DISPLAY_MANAGER="$DISPLAY_MANAGER"
export VROOLI_AUTOHEAL_DISPLAY_RESTART="$DISPLAY_RESTART"
export VROOLI_AUTOHEAL_CHECK_RESOLVED="$CHECK_SYSTEMD_RESOLVED"
export VROOLI_AUTOHEAL_CHECK_DOCKER_DAEMON="$CHECK_DOCKER_DAEMON"
export VROOLI_AUTOHEAL_CHECK_DISK="$CHECK_DISK"
export VROOLI_AUTOHEAL_DISK_THRESHOLD="$DISK_THRESHOLD"
export VROOLI_AUTOHEAL_DISK_PARTITIONS="$DISK_PARTITIONS"
export VROOLI_AUTOHEAL_CHECK_INODE="$CHECK_INODE"
export VROOLI_AUTOHEAL_INODE_THRESHOLD="$INODE_THRESHOLD"
export VROOLI_AUTOHEAL_CHECK_SWAP="$CHECK_SWAP"
export VROOLI_AUTOHEAL_SWAP_THRESHOLD="$SWAP_THRESHOLD"
export VROOLI_AUTOHEAL_CHECK_ZOMBIES="$CHECK_ZOMBIES"
export VROOLI_AUTOHEAL_ZOMBIE_THRESHOLD="$ZOMBIE_THRESHOLD"
export VROOLI_AUTOHEAL_ZOMBIE_CLEANUP="$ZOMBIE_CLEANUP"
export VROOLI_AUTOHEAL_CHECK_PORTS="$CHECK_PORTS"
export VROOLI_AUTOHEAL_PORT_THRESHOLD="$PORT_THRESHOLD"
export VROOLI_AUTOHEAL_CHECK_CERTS="$CHECK_CERTS"
export VROOLI_AUTOHEAL_CERT_WARNING_DAYS="$CERT_WARNING_DAYS"
export VROOLI_AUTOHEAL_DNS_TEST="$DNS_TEST"
export VROOLI_AUTOHEAL_PING_TEST="$PING_TEST"

exec "$TARGET_PATH"
WRAPPER_VARS

    chmod +x "$TARGET_WRAPPER"
    echo "Installed wrapper: $TARGET_WRAPPER"

    # Ensure log directory exists
    mkdir -p "$(dirname "$LOG_FILE")"
    touch "$LOG_FILE" || true

    # Build cron entry (simple, just calls wrapper)
    local tmp
    tmp="$(mktemp)"
    if crontab -l >/dev/null 2>&1; then
        crontab -l | grep -v '# Vrooli autoheal' | grep -v 'vrooli-autoheal' >"$tmp" || true
    fi

    {
        cat "$tmp" 2>/dev/null || true
        echo '# Vrooli autoheal'
        printf '%s %s\n' "$CRON_SCHEDULE" "$TARGET_WRAPPER"
    } | crontab -

    rm -f "$tmp"
    install_logrotate
    echo ""
    echo "✅ Autoheal installed successfully!"
    echo "   Schedule: $CRON_SCHEDULE"
    echo "   Wrapper: $TARGET_WRAPPER"
    echo "   Main script: $TARGET_PATH"
    echo "   Modules: $TARGET_MODULES_DIR"
    echo "   Log file: $LOG_FILE"
    echo ""
    echo "Test manually: $TARGET_WRAPPER"
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
