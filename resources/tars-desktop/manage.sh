#!/bin/bash
set -euo pipefail

# Get the directory of this script
TARS_DESKTOP_MANAGE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source shared utilities  
source "$TARS_DESKTOP_MANAGE_DIR/../../../lib/utils/var.sh"
source "$TARS_DESKTOP_MANAGE_DIR/../../../lib/utils/format.sh"

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