#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="${SCRIPT_DIR}/lib"

# Source core library
source "${LIB_DIR}/core.sh"

# Main CLI handler
main() {
    local command="${1:-help}"
    shift || true

    case "$command" in
        help)
            parlant_help
            ;;
        info)
            parlant_info "$@"
            ;;
        manage)
            parlant_manage "$@"
            ;;
        test)
            parlant_test "$@"
            ;;
        content)
            parlant_content "$@"
            ;;
        status)
            parlant_status "$@"
            ;;
        logs)
            parlant_logs "$@"
            ;;
        credentials)
            parlant_credentials "$@"
            ;;
        *)
            echo "Error: Unknown command '$command'"
            echo "Run 'vrooli resource parlant help' for usage"
            exit 1
            ;;
    esac
}

main "$@"