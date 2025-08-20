#!/bin/bash
# Pushover resource CLI

# Get script directory (resolve symlinks)
PUSHOVER_CLI_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"

# Source lib functions
source "${PUSHOVER_CLI_DIR}/lib/core.sh"
source "${PUSHOVER_CLI_DIR}/lib/status.sh"
source "${PUSHOVER_CLI_DIR}/lib/install.sh"
source "${PUSHOVER_CLI_DIR}/lib/start.sh"
source "${PUSHOVER_CLI_DIR}/lib/inject.sh"
source "${PUSHOVER_CLI_DIR}/lib/configure.sh"

# CLI main function
pushover::cli() {
    local command="${1:-}"
    shift || true
    
    case "$command" in
        status)
            pushover::status "$@"
            ;;
        install)
            pushover::install "$@"
            ;;
        uninstall)
            pushover::uninstall "$@"
            ;;
        configure)
            pushover::configure "$@"
            ;;
        clear-credentials)
            pushover::clear_credentials "$@"
            ;;
        start)
            pushover::start "$@"
            ;;
        stop)
            pushover::stop "$@"
            ;;
        inject)
            pushover::inject "$@"
            ;;
        send)
            pushover::send "$@"
            ;;
        health|health-check)
            pushover::health_check "$@"
            ;;
        help|--help|-h|"")
            pushover::show_help
            ;;
        *)
            echo "Unknown command: $command"
            pushover::show_help
            exit 1
            ;;
    esac
}

# Show help
pushover::show_help() {
    cat <<EOF
Pushover Resource CLI

Usage: resource-pushover <command> [options]

Commands:
    status          Check Pushover status
    install         Install Pushover support
    uninstall       Remove Pushover support
    configure       Configure API credentials
    clear-credentials  Clear stored credentials
    start           Activate Pushover service
    stop            Deactivate Pushover service
    inject          Inject notification templates
    send            Send a notification
    health          Check service health
    help            Show this help message

Examples:
    resource-pushover status
    resource-pushover install
    resource-pushover configure
    resource-pushover send -m "Hello from Vrooli!"
    
For more information, see docs/README.md
EOF
}

# Run CLI
pushover::cli "$@"