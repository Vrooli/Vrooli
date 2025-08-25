#!/usr/bin/env bash
# SimPy CLI - Thin wrapper around library functions

# Get the real script directory (follows symlinks)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    SIMPY_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${SIMPY_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
SIMPY_CLI_DIR="${APP_ROOT}/resources/simpy"

# Source library functions
# shellcheck disable=SC1091
source "${SIMPY_CLI_DIR}/lib/core.sh"
# shellcheck disable=SC1091
source "${SIMPY_CLI_DIR}/lib/install.sh"
# shellcheck disable=SC1091
source "${SIMPY_CLI_DIR}/lib/start.sh"
# shellcheck disable=SC1091
source "${SIMPY_CLI_DIR}/lib/stop.sh"
# shellcheck disable=SC1091
source "${SIMPY_CLI_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"

#######################################
# Show help message
#######################################
simpy::help() {
    cat << EOF
SimPy Resource CLI

Usage: $(basename "$0") <command> [options]

Commands:
    install                 Install SimPy and dependencies
    start                   Start SimPy service
    stop                    Stop SimPy service
    status [--json]         Show SimPy status
    run <script>            Run a simulation script
    list-examples           List available example simulations
    help                    Show this help message

Examples:
    $(basename "$0") install
    $(basename "$0") start
    $(basename "$0") status
    $(basename "$0") run examples/machine_shop.py
    $(basename "$0") stop

EOF
}

#######################################
# Main CLI entry point
#######################################
main() {
    local command="${1:-help}"
    shift
    
    case "$command" in
        install)
            simpy::install "$@"
            ;;
        start)
            simpy::start "$@"
            ;;
        stop)
            simpy::stop "$@"
            ;;
        status)
            simpy::status "$@"
            ;;
        run)
            if [[ -z "$1" ]]; then
                log::error "Usage: $(basename "$0") run <script>"
                exit 1
            fi
            simpy::run_simulation "$@"
            ;;
        list-examples)
            simpy::list_examples
            ;;
        help|--help|-h)
            simpy::help
            ;;
        *)
            log::error "Unknown command: $command"
            simpy::help
            exit 1
            ;;
    esac
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
