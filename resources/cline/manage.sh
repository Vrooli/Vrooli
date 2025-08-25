#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
CLINE_MANAGE_DIR="${APP_ROOT}/resources/cline"

# Main router - delegates to lib scripts
main() {
    # Handle --action flag for vrooli resource compatibility
    if [[ "${1:-}" == "--action" ]]; then
        shift
    fi
    
    local cmd="${1:-help}"
    shift || true
    
    case "$cmd" in
        status|start|stop|install|logs|config|inject)
            "$CLINE_MANAGE_DIR/lib/${cmd}.sh" "$@"
            ;;
        help)
            "$CLINE_MANAGE_DIR/cli.sh" help
            ;;
        *)
            echo "Error: Unknown command '$cmd'"
            "$CLINE_MANAGE_DIR/cli.sh" help
            exit 1
            ;;
    esac
}

main "$@"