#!/usr/bin/env bash
set -euo pipefail

# Ollama AI Resource Management
# This script orchestrates Ollama installation, configuration, and management using modular components

DESCRIPTION="Install and manage Ollama AI resource"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# Handle Ctrl+C gracefully
trap 'echo ""; log::info "Ollama operation interrupted by user. Exiting..."; exit 130' INT TERM

# Source common resources
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/args.sh"

# Source configuration modules
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/messages.sh"

# Export configuration and messages
ollama::export_config
ollama::export_messages  

# Source all library modules
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/install.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/models.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/api.sh"

#######################################
# Main orchestration function
# This function routes commands to the appropriate module functions
#######################################
ollama::main() {
    # Parse command line arguments
    ollama::parse_arguments "$@"
    
    # If no action specified, show usage
    if [[ -z "$ACTION" ]]; then
        log::error "No action specified"
        ollama::usage
        exit 1
    fi
    
    # Route to appropriate action
    case "$ACTION" in
        install)
            ollama::install
            ;;
        uninstall)
            ollama::uninstall
            ;;
        start)
            ollama::start
            ;;
        stop)
            ollama::stop
            ;;
        restart)
            ollama::restart
            ;;
        status)
            ollama::status
            ;;
        logs)
            ollama::logs
            ;;
        models)
            ollama::list_models
            ;;
        available)
            ollama::show_available_models
            ;;
        info)
            ollama::info
            ;;
        test)
            ollama::test
            ;;
        prompt)
            # For prompt action, we need to pass the parsed parameters
            if [[ -z "$PROMPT_TEXT" ]]; then
                log::error "No prompt text provided"
                log::info "Use: $0 --action prompt --text 'your prompt here'"
                exit 1
            fi
            
            # Send prompt with parsed parameters
            ollama::send_prompt "$PROMPT_TEXT" "$PROMPT_MODEL" "$PROMPT_TYPE"
            ;;
        url)
            ollama::get_urls
            ;;
        *)
            log::error "Unknown action: $ACTION"
            ollama::usage
            exit 1
            ;;
    esac
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    ollama::main "$@"
fi