#!/bin/bash
# Home Assistant Resource CLI

# Get script directory - resolve symlinks
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
HOME_ASSISTANT_CLI_DIR="${APP_ROOT}/resources/home-assistant"

# Source all library functions
source "${HOME_ASSISTANT_CLI_DIR}/lib/core.sh"
source "${HOME_ASSISTANT_CLI_DIR}/lib/health.sh"
source "${HOME_ASSISTANT_CLI_DIR}/lib/install.sh"
source "${HOME_ASSISTANT_CLI_DIR}/lib/status.sh"
source "${HOME_ASSISTANT_CLI_DIR}/lib/inject.sh"

#######################################
# Show help message
#######################################
home_assistant::help() {
    cat << EOF
Home Assistant Resource Management

Usage: $(basename "$0") <command> [options]

Commands:
    install         Install Home Assistant
    uninstall       Uninstall Home Assistant (requires --force)
    start           Start Home Assistant
    stop            Stop Home Assistant
    restart         Restart Home Assistant
    status          Show Home Assistant status
    logs            Show Home Assistant logs
    inject <file>   Inject automation/configuration file (.yaml, .json, .py)
    list            List injected files
    clear           Clear all injected files (requires --force)
    help            Show this help message

Global Options:
    --format <type> Output format: text, json (for status command)
    --fast          Skip expensive operations (for status command)
    --force         Force operation without confirmation
    --help, -h      Show help for specific command

Examples:
    $(basename "$0") install
    $(basename "$0") status
    $(basename "$0") inject automation.yaml
    $(basename "$0") inject script.py
    $(basename "$0") list
    $(basename "$0") logs --tail 50

Home Assistant Web UI:
    Once installed, access Home Assistant at http://localhost:8123
    
Integration Notes:
    - Automations: Place .yaml files in automations/ or use inject
    - Python Scripts: Place .py files in python_scripts/ or use inject
    - Packages: Complex configurations in packages/ directory
    - JSON configs are automatically converted to YAML when applicable
    
For more information, see the Home Assistant documentation:
    https://www.home-assistant.io/docs/
EOF
}

#######################################
# Main command handler
#######################################
main() {
    local command="${1:-}"
    shift || true
    
    case "$command" in
        install)
            home_assistant::install "$@"
            ;;
        uninstall)
            home_assistant::uninstall "$@"
            ;;
        start)
            home_assistant::init
            if docker::container_exists "$HOME_ASSISTANT_CONTAINER_NAME"; then
                log::info "Starting Home Assistant..."
                docker start "$HOME_ASSISTANT_CONTAINER_NAME"
                home_assistant::health::wait_for_healthy 60
            else
                log::error "Home Assistant is not installed. Run 'install' first."
                exit 1
            fi
            ;;
        stop)
            home_assistant::init
            if docker::is_running "$HOME_ASSISTANT_CONTAINER_NAME"; then
                log::info "Stopping Home Assistant..."
                docker stop "$HOME_ASSISTANT_CONTAINER_NAME"
                log::success "Home Assistant stopped"
            else
                log::warning "Home Assistant is not running"
            fi
            ;;
        restart)
            home_assistant::init
            if docker::container_exists "$HOME_ASSISTANT_CONTAINER_NAME"; then
                log::info "Restarting Home Assistant..."
                docker restart "$HOME_ASSISTANT_CONTAINER_NAME"
                home_assistant::health::wait_for_healthy 60
            else
                log::error "Home Assistant is not installed. Run 'install' first."
                exit 1
            fi
            ;;
        status)
            home_assistant::status "$@"
            ;;
        logs)
            home_assistant::init
            local tail_lines="50"
            
            # Parse options
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --tail)
                        tail_lines="$2"
                        shift 2
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            if docker::container_exists "$HOME_ASSISTANT_CONTAINER_NAME"; then
                docker logs "$HOME_ASSISTANT_CONTAINER_NAME" --tail "$tail_lines"
            else
                log::error "Home Assistant is not installed"
                exit 1
            fi
            ;;
        inject)
            home_assistant::inject "$@"
            ;;
        list)
            home_assistant::inject::list "$@"
            ;;
        clear)
            home_assistant::inject::clear "$@"
            ;;
        help|--help|-h)
            home_assistant::help
            ;;
        "")
            log::error "No command provided"
            home_assistant::help
            exit 1
            ;;
        *)
            log::error "Unknown command: $command"
            home_assistant::help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
