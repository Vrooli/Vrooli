#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="${SCRIPT_DIR}/lib"
APP_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source agent management first
if [[ -f "${APP_ROOT}/resources/parlant/config/agents.conf" ]]; then
    source "${APP_ROOT}/resources/parlant/config/agents.conf"
    source "${APP_ROOT}/scripts/resources/agents/agent-manager.sh"
fi

# Source core library only (agents.sh no longer needed)
for lib in core; do
    lib_file="${LIB_DIR}/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        source "$lib_file" 2>/dev/null || true
    fi
done

# Create wrapper for agents command that delegates to manager
parlant::agents::command() {
    if type -t agent_manager::load_config &>/dev/null; then
        "${APP_ROOT}/scripts/resources/agents/agent-manager.sh" --config="parlant" "$@"
    else
        echo "Agent management not available"
        exit 1
    fi
}
export -f parlant::agents::command

# Main CLI handler
main() {
    local command="${1:-help}"
    shift || true

    case "$command" in
        help)
            parlant_help
            ;;
        info)
            parlant_info "$@"
            ;;
        manage)
            parlant_manage "$@"
            ;;
        test)
            parlant_test "$@"
            ;;
        content)
            parlant_content "$@"
            ;;
        status)
            parlant_status "$@"
            ;;
        logs)
            parlant_logs "$@"
            ;;
        credentials)
            parlant_credentials "$@"
            ;;
        agents)
            parlant::agents::command "$@"
            ;;
        *)
            echo "Error: Unknown command '$command'"
            echo "Run 'vrooli resource parlant help' for usage"
            exit 1
            ;;
    esac
}

main "$@"