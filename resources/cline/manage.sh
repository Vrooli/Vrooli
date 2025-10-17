#!/usr/bin/env bash

# ⚠️ DEPRECATION NOTICE: This script is deprecated as of v2.0 (January 2025)
# Please use cli.sh instead: resource-cline <command>
# This file will be removed in v3.0 (target: December 2025)
#
# Migration guide:
#   OLD: ./manage.sh --action <command>
#   NEW: ./cli.sh <command>  OR  resource-cline <command>

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