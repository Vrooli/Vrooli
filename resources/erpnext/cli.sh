#!/usr/bin/env bash
# ERPNext Resource CLI - Thin wrapper over lib functions

# Get the actual script location, resolving any symlinks
SCRIPT_PATH="$(readlink -f "${BASH_SOURCE[0]}")"
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
ERPNEXT_CLI_DIR="${APP_ROOT}/resources/erpnext"

# Source the main library
source "${ERPNEXT_CLI_DIR}/lib/main.sh" || {
    echo "[ERROR] Failed to source lib/main.sh" >&2
    exit 1
}

# Main function
main() {
    local command="${1:-}"
    shift || true
    
    case "$command" in
        start)      erpnext::start "$@" ;;
        stop)       erpnext::stop "$@" ;;
        restart)    erpnext::restart "$@" ;;
        status)     erpnext::status "$@" ;;
        install)    erpnext::install "$@" ;;
        uninstall)  erpnext::uninstall "$@" ;;
        inject)     erpnext::inject "$@" ;;
        list)       erpnext::list "$@" ;;
        help)       erpnext::help "$@" ;;
        *)
            echo "[ERROR] Unknown command: $command" >&2
            erpnext::help
            exit 1
            ;;
    esac
}

main "$@"
