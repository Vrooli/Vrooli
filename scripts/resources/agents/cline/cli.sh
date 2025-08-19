#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
CLINE_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$CLINE_CLI_DIR/../../../lib/utils/var.sh"
source "$CLINE_CLI_DIR/../../../lib/utils/format.sh"
source "$CLINE_CLI_DIR/lib/common.sh"

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
            "$CLINE_CLI_DIR/lib/status.sh" "$@"
            ;;
        install)
            "$CLINE_CLI_DIR/lib/install.sh" "$@"
            ;;
        start)
            "$CLINE_CLI_DIR/lib/start.sh" "$@"
            ;;
        stop)
            "$CLINE_CLI_DIR/lib/stop.sh" "$@"
            ;;
        logs)
            "$CLINE_CLI_DIR/lib/logs.sh" "$@"
            ;;
        config)
            "$CLINE_CLI_DIR/lib/config.sh" "$@"
            ;;
        inject)
            "$CLINE_CLI_DIR/lib/inject.sh" "$@"
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
