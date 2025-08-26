#!/bin/bash
set -euo pipefail

# Get the real script directory (following symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    RESOLVED_PATH="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="${APP_ROOT:-$(builtin cd "${RESOLVED_PATH%/*}/../.." && builtin pwd)}"
else
    APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
fi

ODOO_CLI_DIR="${APP_ROOT}/resources/odoo"

# Source library functions
source "$ODOO_CLI_DIR/lib/common.sh"
source "$ODOO_CLI_DIR/lib/install.sh"
source "$ODOO_CLI_DIR/lib/start.sh"
source "$ODOO_CLI_DIR/lib/stop.sh"
source "$ODOO_CLI_DIR/lib/status.sh"
source "$ODOO_CLI_DIR/lib/inject.sh"
source "$ODOO_CLI_DIR/lib/test.sh"

# Main command dispatcher
main() {
    local command="${1:-}"
    shift || true
    
    case "$command" in
        install)
            odoo_install "$@"
            ;;
        start)
            odoo_start "$@"
            ;;
        stop)
            odoo_stop "$@"
            ;;
        status)
            odoo_status "$@"
            ;;
        logs)
            odoo_logs "$@"
            ;;
        inject)
            odoo_inject "$@"
            ;;
        test)
            odoo_test "$@"
            ;;
        help|--help|-h)
            echo "Usage: $0 {install|start|stop|status|logs|inject|test|help}"
            echo ""
            echo "Commands:"
            echo "  install  - Install Odoo Community"
            echo "  start    - Start Odoo service"
            echo "  stop     - Stop Odoo service"
            echo "  status   - Check service status"
            echo "  logs     - View service logs"
            echo "  inject   - Install modules or import data"
            echo "  test     - Run integration tests"
            echo "  help     - Show this help message"
            exit 0
            ;;
        *)
            echo "Error: Unknown command '$command'"
            echo "Run '$0 help' for usage information"
            exit 1
            ;;
    esac
}

main "$@"
