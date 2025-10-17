#!/bin/bash

# ⚠️ DEPRECATION NOTICE: This script is deprecated as of v2.0 (January 2025)
# Please use cli.sh instead: resource-tars-desktop <command>
# This file will be removed in v3.0 (target: December 2025)
#
# Migration guide:
#   OLD: ./manage.sh --action <command>
#   NEW: ./cli.sh <command>  OR  resource-tars-desktop <command>

# Show deprecation warning
if [[ "${VROOLI_SUPPRESS_DEPRECATION:-}" != "true" ]]; then
    echo "⚠️  WARNING: manage.sh is deprecated. Please use cli.sh instead." >&2
    echo "   This script will be removed in v3.0 (December 2025)" >&2
    echo "   To suppress this warning: export VROOLI_SUPPRESS_DEPRECATION=true" >&2
    echo "" >&2
fi

set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
TARS_DESKTOP_MANAGE_DIR="${APP_ROOT}/resources/tars-desktop"

# Source shared utilities  
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Source resource functions directly
[[ -f "$TARS_DESKTOP_MANAGE_DIR/lib/core.sh" ]] && source "$TARS_DESKTOP_MANAGE_DIR/lib/core.sh"
[[ -f "$TARS_DESKTOP_MANAGE_DIR/lib/config.sh" ]] && source "$TARS_DESKTOP_MANAGE_DIR/lib/config.sh"
[[ -f "$TARS_DESKTOP_MANAGE_DIR/lib/install.sh" ]] && source "$TARS_DESKTOP_MANAGE_DIR/lib/install.sh"
[[ -f "$TARS_DESKTOP_MANAGE_DIR/lib/start.sh" ]] && source "$TARS_DESKTOP_MANAGE_DIR/lib/start.sh"
[[ -f "$TARS_DESKTOP_MANAGE_DIR/lib/status.sh" ]] && source "$TARS_DESKTOP_MANAGE_DIR/lib/status.sh"
[[ -f "$TARS_DESKTOP_MANAGE_DIR/lib/inject.sh" ]] && source "$TARS_DESKTOP_MANAGE_DIR/lib/inject.sh"

# Parse command line arguments
ACTION=""
FORMAT="text"

while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --format)
            FORMAT="$2"
            shift 2
            ;;
        install|start|stop|status|inject|help)
            ACTION="$1"
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Main execution
case "$ACTION" in
    install)
        tars_desktop_install
        ;;
    start)
        tars_desktop_start
        ;;
    stop)
        tars_desktop_stop
        ;;
    status)
        tars_desktop_status "$FORMAT"
        ;;
    inject)
        shift
        tars_desktop_inject "$@"
        ;;
    help|"")
        echo "Usage: $0 [--action] {install|start|stop|status|inject|help} [--format {text|json}]"
        echo ""
        echo "Actions:"
        echo "  install  - Install TARS-desktop UI automation agent"
        echo "  start    - Start TARS-desktop service"
        echo "  stop     - Stop TARS-desktop service"
        echo "  status   - Check TARS-desktop status"
        echo "  inject   - Inject UI automation scripts"
        echo "  help     - Show this help message"
        echo ""
        echo "Options:"
        echo "  --format {text|json} - Output format for status (default: text)"
        exit 0
        ;;
    *)
        echo "Error: Unknown action: $ACTION"
        echo "Run '$0 help' for usage information"
        exit 1
        ;;
esac