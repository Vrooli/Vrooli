#!/usr/bin/env bash
set -euo pipefail

# Get the actual directory of this script (resolving symlinks)
SCRIPT_PATH="${BASH_SOURCE[0]}"
if [[ -L "$SCRIPT_PATH" ]]; then
    SCRIPT_PATH="$(readlink -f "$SCRIPT_PATH")"
fi
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    CLINE_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${CLINE_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
CLINE_CLI_DIR="${APP_ROOT}/resources/cline"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "$CLINE_CLI_DIR/lib/common.sh"
source "$CLINE_CLI_DIR/lib/status.sh"
source "$CLINE_CLI_DIR/lib/install.sh"
source "$CLINE_CLI_DIR/lib/start.sh"
source "$CLINE_CLI_DIR/lib/stop.sh"
source "$CLINE_CLI_DIR/lib/logs.sh"
source "$CLINE_CLI_DIR/lib/config.sh"
source "$CLINE_CLI_DIR/lib/inject.sh"

# Help function
show_help() {
    cat << HELP
Cline Resource CLI

Usage: resource-cline <command> [options]

Commands:
    status          Check Cline status
    install         Install Cline extension
    start           Start Cline service
    stop            Stop Cline service
    logs            Show Cline logs
    config          View/update Cline configuration
    inject [file]   Inject agent configuration
    help            Show this help message

Examples:
    resource-cline status
    resource-cline start
    resource-cline inject my-agent.json

HELP
}

# Main command router
main() {
    local cmd="${1:-help}"
    shift || true
    
    case "$cmd" in
        status)
            cline::status "$@"
            ;;
        install)
            cline::install "$@"
            ;;
        start)
            cline::start "$@"
            ;;
        stop)
            cline::stop "$@"
            ;;
        logs)
            cline::logs "$@"
            ;;
        config)
            cline::config "$@"
            ;;
        inject)
            cline::inject "$@"
            ;;
        help)
            show_help
            ;;
        *)
            echo "Error: Unknown command '$cmd'"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
