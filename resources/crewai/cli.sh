#!/bin/bash
set -euo pipefail

# CrewAI CLI wrapper - resolve symlinks to get actual script location
REAL_PATH="$(readlink -f "${BASH_SOURCE[0]}")"
SCRIPT_DIR="$(dirname "${REAL_PATH}")"
LIB_DIR="${SCRIPT_DIR}/lib"

# Source utilities
source "${SCRIPT_DIR}/../../../lib/utils/var.sh"
source "${SCRIPT_DIR}/../../../lib/utils/log.sh"
source "${SCRIPT_DIR}/../../../lib/utils/format.sh"

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