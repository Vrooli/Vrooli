#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    HAYSTACK_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${HAYSTACK_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
HAYSTACK_CLI_DIR="${APP_ROOT}/resources/haystack"

# Source the lib functions
source "${HAYSTACK_CLI_DIR}/lib/lifecycle.sh"
source "${HAYSTACK_CLI_DIR}/lib/inject.sh"

# Main CLI handler
main() {
    local command="${1:-}"
    shift || true
    
    case "${command}" in
        start)
            haystack::start "$@"
            ;;
        stop)
            haystack::stop "$@"
            ;;
        restart)
            haystack::restart "$@"
            ;;
        status)
            haystack::status "$@"
            ;;
        install)
            haystack::install "$@"
            ;;
        uninstall)
            haystack::uninstall "$@"
            ;;
        inject)
            haystack::inject "$@"
            ;;
        list|list-injected)
            haystack::list_injected "$@"
            ;;
        clear|clear-data)
            haystack::clear_data "$@"
            ;;
        help|--help|-h|"")
            cat << EOF
Haystack Resource CLI

Usage: $0 <command> [options]

Commands:
    start           Start Haystack service
    stop            Stop Haystack service
    restart         Restart Haystack service
    status          Check Haystack status
    install         Install Haystack
    uninstall       Uninstall Haystack
    inject <file>   Inject data into Haystack (.json, .txt, .md, .py)
    list            List injected data statistics
    clear           Clear all data from Haystack
    help            Show this help message

Examples:
    $0 start
    $0 status --format json
    $0 inject documents.json
    $0 inject script.py
    $0 clear
EOF
            ;;
        *)
            echo "Unknown command: ${command}"
            echo "Run '$0 help' for usage information"
            exit 1
            ;;
    esac
}

main "$@"
