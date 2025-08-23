#!/usr/bin/env bash
set -euo pipefail

# Get the actual directory of this script (resolving symlinks)
SCRIPT_PATH="${BASH_SOURCE[0]}"
if [[ -L "$SCRIPT_PATH" ]]; then
    SCRIPT_PATH="$(readlink -f "$SCRIPT_PATH")"
fi
CLINE_CLI_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)"

# Source utilities
source "$CLINE_CLI_DIR/../../../lib/utils/var.sh"
source "$CLINE_CLI_DIR/../../../lib/utils/format.sh"
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
