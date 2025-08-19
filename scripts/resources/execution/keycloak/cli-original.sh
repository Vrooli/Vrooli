#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
KEYCLOAK_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source the lib functions (inject.sh will be sourced on demand to avoid circular deps)
source "${KEYCLOAK_CLI_DIR}/lib/lifecycle.sh"

# Main CLI handler
main() {
    local command="${1:-}"
    shift || true
    
    case "${command}" in
        start)
            keycloak::start "$@"
            ;;
        stop)
            keycloak::stop "$@"
            ;;
        restart)
            keycloak::restart "$@"
            ;;
        status)
            keycloak::status "$@"
            ;;
        install)
            source "${KEYCLOAK_CLI_DIR}/lib/install.sh"
            keycloak::install "$@"
            ;;
        uninstall)
            source "${KEYCLOAK_CLI_DIR}/lib/install.sh"
            keycloak::uninstall "$@"
            ;;
        inject)
            source "${KEYCLOAK_CLI_DIR}/lib/inject.sh"
            keycloak::inject "$@"
            ;;
        list|list-injected)
            source "${KEYCLOAK_CLI_DIR}/lib/inject.sh"
            keycloak::list_injected "$@"
            ;;
        clear|clear-data)
            source "${KEYCLOAK_CLI_DIR}/lib/inject.sh"
            keycloak::clear_data "$@"
            ;;
        help|--help|-h|"")
            cat << EOF
Keycloak Resource CLI

Usage: $0 <command> [options]

Commands:
    start           Start Keycloak service
    stop            Stop Keycloak service
    restart         Restart Keycloak service
    status          Check Keycloak status
    install         Install Keycloak
    uninstall       Uninstall Keycloak
    inject <file>   Inject realm configuration into Keycloak (.json)
    list            List realm and user statistics
    clear           Clear all data from Keycloak
    help            Show this help message

Examples:
    $0 start
    $0 status --format json
    $0 inject realm-config.json
    $0 list
    $0 clear

Keycloak will be available at: http://localhost:8070
Admin Console: http://localhost:8070/admin
Default credentials: admin/admin
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