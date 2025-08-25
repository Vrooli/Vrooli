#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    REAL_PATH="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${REAL_PATH%/*}/../.." && builtin pwd)"
fi
SCRIPT_DIR="${APP_ROOT}/resources/crewai"
LIB_DIR="${SCRIPT_DIR}/lib"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Command handling
case "${1:-help}" in
    install)
        shift
        source "${LIB_DIR}/install.sh"
        ;;
    start)
        shift
        source "${LIB_DIR}/core.sh"
        start_crewai "$@"
        ;;
    stop)
        shift
        source "${LIB_DIR}/core.sh"
        stop_crewai "$@"
        ;;
    status)
        shift
        source "${LIB_DIR}/status.sh"
        main "$@"
        ;;
    inject)
        shift
        source "${LIB_DIR}/inject.sh"
        ;;
    list-crews)
        shift
        source "${LIB_DIR}/core.sh"
        list_crews "$@"
        ;;
    list-agents)
        shift
        source "${LIB_DIR}/core.sh"
        list_agents "$@"
        ;;
    help|--help|-h)
        echo "CrewAI Resource Manager"
        echo ""
        echo "Usage: resource-crewai <command> [options]"
        echo ""
        echo "Commands:"
        echo "  install          Install CrewAI framework"
        echo "  start           Start CrewAI service"
        echo "  stop            Stop CrewAI service"
        echo "  status          Check CrewAI status"
        echo "  inject          Inject crews/agents from a directory"
        echo "  list-crews      List available crews"
        echo "  list-agents     List available agents"
        echo "  help            Show this help message"
        ;;
    *)
        log::error "Unknown command: $1"
        echo "Run 'resource-crewai help' for usage"
        exit 1
        ;;
esac
