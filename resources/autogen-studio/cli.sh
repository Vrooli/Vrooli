#!/bin/bash

# AutoGen Studio CLI
# Multi-agent conversation framework for complex task orchestration

set -euo pipefail

# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    RESOURCE_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${RESOURCE_CLI_SCRIPT%/*}/../.." && builtin pwd)"
else
    APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
fi
SCRIPT_DIR="${APP_ROOT}/resources/autogen-studio"

# Source core functions
source "${SCRIPT_DIR}/lib/core.sh"

# Main CLI handler
main() {
    local command="${1:-}"
    shift || true
    
    case "${command}" in
        install)
            autogen_install "$@"
            ;;
        start)
            autogen_start "$@"
            ;;
        stop)
            autogen_stop "$@"
            ;;
        restart)
            autogen_stop "$@" && autogen_start "$@"
            ;;
        status)
            # Parse format flag
            local format="text"
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --format)
                        format="${2:-text}"
                        shift 2
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            autogen_status "${format}"
            ;;
        create-agent)
            autogen_create_agent "$@"
            ;;
        create-skill)
            autogen_create_skill "$@"
            ;;
        list-agents)
            autogen_list_agents "$@"
            ;;
        list-skills)
            autogen_list_skills "$@"
            ;;
        inject)
            autogen_inject "$@"
            ;;
        help|--help|-h)
            cat << EOF
AutoGen Studio Resource CLI

Usage: $(basename "$0") [COMMAND] [OPTIONS]

Commands:
    install              Install AutoGen Studio
    start                Start AutoGen Studio service
    stop                 Stop AutoGen Studio service
    restart              Restart AutoGen Studio service
    status [--format]    Show service status (text/json)
    create-agent NAME    Create a new agent
    create-skill NAME    Create a new skill
    list-agents          List available agents
    list-skills          List available skills
    inject FILE          Inject agent or skill from file
    help                 Show this help message

Examples:
    $(basename "$0") install
    $(basename "$0") start
    $(basename "$0") status --format json
    $(basename "$0") create-agent researcher assistant "You are a research assistant"
    $(basename "$0") create-skill data_processor

AutoGen Studio UI will be available at: http://localhost:8085
EOF
            ;;
        *)
            if [[ -z "${command}" ]]; then
                echo "ERROR:" "No command provided. Use 'help' for usage information."
            else
                echo "ERROR:" "Unknown command: ${command}"
            fi
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
