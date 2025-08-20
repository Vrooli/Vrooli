#!/usr/bin/env bash
set -euo pipefail

# Resolve symlinks to find the actual script location
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    KEYCLOAK_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    KEYCLOAK_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
KEYCLOAK_CLI_DIR="$(cd "$(dirname "$KEYCLOAK_CLI_SCRIPT")" && pwd)"

# Main CLI handler
main() {
    local command="${1:-}"
    shift || true
    
    case "${command}" in
        start)
            source "${KEYCLOAK_CLI_DIR}/lib/common.sh"
            source "${KEYCLOAK_CLI_DIR}/../../../lib/utils/log.sh"
            source "${KEYCLOAK_CLI_DIR}/lib/install.sh"
            source "${KEYCLOAK_CLI_DIR}/lib/lifecycle.sh"
            keycloak::start "$@"
            ;;
        stop)
            source "${KEYCLOAK_CLI_DIR}/lib/common.sh"
            source "${KEYCLOAK_CLI_DIR}/../../../lib/utils/log.sh"
            source "${KEYCLOAK_CLI_DIR}/lib/lifecycle.sh"
            keycloak::stop "$@"
            ;;
        restart)
            source "${KEYCLOAK_CLI_DIR}/lib/common.sh"
            source "${KEYCLOAK_CLI_DIR}/../../../lib/utils/log.sh"
            source "${KEYCLOAK_CLI_DIR}/lib/install.sh"
            source "${KEYCLOAK_CLI_DIR}/lib/lifecycle.sh"
            keycloak::restart "$@"
            ;;
        status)
            source "${KEYCLOAK_CLI_DIR}/lib/common.sh"
            source "${KEYCLOAK_CLI_DIR}/../../../lib/utils/log.sh"
            source "${KEYCLOAK_CLI_DIR}/../../../lib/utils/format.sh"
            source "${KEYCLOAK_CLI_DIR}/lib/status.sh"
            keycloak::status "$@"
            ;;
        install)
            source "${KEYCLOAK_CLI_DIR}/lib/common.sh"
            source "${KEYCLOAK_CLI_DIR}/../../../lib/utils/log.sh"
            source "${KEYCLOAK_CLI_DIR}/lib/install.sh"
            keycloak::install "$@"
            ;;
        uninstall)
            source "${KEYCLOAK_CLI_DIR}/lib/common.sh"
            source "${KEYCLOAK_CLI_DIR}/../../../lib/utils/log.sh"
            source "${KEYCLOAK_CLI_DIR}/lib/install.sh"
            keycloak::uninstall "$@"
            ;;
        inject)
            source "${KEYCLOAK_CLI_DIR}/lib/common.sh"
            source "${KEYCLOAK_CLI_DIR}/../../../lib/utils/log.sh"
            source "${KEYCLOAK_CLI_DIR}/lib/inject.sh"
            keycloak::inject "$@"
            ;;
        list|list-injected)
            source "${KEYCLOAK_CLI_DIR}/lib/common.sh"
            source "${KEYCLOAK_CLI_DIR}/../../../lib/utils/log.sh"
            source "${KEYCLOAK_CLI_DIR}/lib/inject.sh"
            keycloak::list_injected "$@"
            ;;
        clear|clear-data)
            source "${KEYCLOAK_CLI_DIR}/lib/common.sh"
            source "${KEYCLOAK_CLI_DIR}/../../../lib/utils/log.sh"
            source "${KEYCLOAK_CLI_DIR}/lib/inject.sh"
            keycloak::clear_data "$@"
            ;;
        help|--help|-h|"")
            cat << EOF
Keycloak Resource CLI

Usage: resource-keycloak <command> [options]

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
    resource-keycloak start
    resource-keycloak status --format json
    resource-keycloak inject realm-config.json
    resource-keycloak list
    resource-keycloak clear

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