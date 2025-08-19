#!/bin/bash

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source shared utilities
source "${SCRIPT_DIR}/../../../lib/utils/var.sh"
source "${SCRIPT_DIR}/../../../lib/utils/log.sh"
source "${SCRIPT_DIR}/../../../lib/utils/format.sh"

# Source resource functions
source "${SCRIPT_DIR}/lib/install.sh"
source "${SCRIPT_DIR}/lib/status.sh"
source "${SCRIPT_DIR}/lib/inject.sh"

# Main function
main() {
    local action="${1:-status}"
    shift || true
    
    case "$action" in
        install)
            crewai_install "$@"
            ;;
        uninstall)
            crewai_uninstall "$@"
            ;;
        start)
            crewai_start "$@"
            ;;
        stop)
            crewai_stop "$@"
            ;;
        status)
            crewai_status "$@"
            ;;
        inject)
            crewai_inject "$@"
            ;;
        help|--help|-h)
            cat << EOF
CrewAI Resource Manager

Usage: $0 <command> [options]

Commands:
  install      Install CrewAI framework
  uninstall    Uninstall CrewAI
  start        Start CrewAI service
  stop         Stop CrewAI service
  status       Show CrewAI status
  inject       Inject crew configuration
  help         Show this help message

Options:
  --verbose    Show detailed output
  --format     Output format (json|text) for status

Examples:
  $0 install
  $0 status --format json
  $0 inject /path/to/crew.yaml
EOF
            ;;
        *)
            log::error "Unknown command: $action"
            echo "Run '$0 help' for usage information"
            exit 1
            ;;
    esac
}

main "$@"