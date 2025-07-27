#!/usr/bin/env bash
set -euo pipefail

# Handle Ctrl+C gracefully
trap 'echo ""; log::info "Agent S2 installation interrupted by user. Exiting..."; exit 130' INT TERM

# Agent S2 Autonomous Computer Interaction Setup and Management
# This script handles installation, configuration, and management of Agent S2 using Docker

DESCRIPTION="Install and manage Agent S2 autonomous computer interaction service using Docker"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# Source common resources
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/args.sh"

# Source configuration
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/messages.sh"

# Export messages (config will be exported after parsing arguments)
agents2::export_messages

# Source all library modules
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/docker.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/install.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/api.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/usage.sh"

#######################################
# Parse command line arguments
#######################################
agents2::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|info|usage" \
        --default "install"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if Agent S2 appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "llm-provider" \
        --desc "LLM provider for Agent S2 (openai, anthropic, ollama)" \
        --type "value" \
        --options "openai|anthropic|ollama" \
        --default "anthropic"
    
    args::register \
        --name "llm-model" \
        --desc "LLM model to use" \
        --type "value" \
        --default "claude-3-7-sonnet-20250219"
    
    args::register \
        --name "enable-ai" \
        --desc "Enable AI capabilities (vs core automation only)" \
        --type "value" \
        --options "yes|no" \
        --default "yes"
    
    args::register \
        --name "enable-search" \
        --desc "Enable web search integration (requires Perplexica)" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "vnc-password" \
        --desc "Password for VNC access" \
        --type "value" \
        --default "agents2vnc"
    
    args::register \
        --name "enable-host-display" \
        --desc "Enable access to host display (security risk)" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "usage-type" \
        --desc "Type of usage example to run" \
        --type "value" \
        --options "screenshot|automation|planning|capabilities|all|help" \
        --default "help"
    
    if args::is_asking_for_help "$@"; then
        agents2::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export FORCE=$(args::get "force")
    export YES=$(args::get "yes")
    export LLM_PROVIDER=$(args::get "llm-provider")
    export LLM_MODEL=$(args::get "llm-model")
    export ARGS_LLM_PROVIDER=$(args::get "llm-provider")
    export ARGS_LLM_MODEL=$(args::get "llm-model")
    export VNC_PASSWORD=$(args::get "vnc-password")
    export ENABLE_HOST_DISPLAY=$(args::get "enable-host-display")
    export ENABLE_AI=$(args::get "enable-ai")
    export ARGS_ENABLE_AI=$(args::get "enable-ai")
    export USAGE_TYPE=$(args::get "usage-type")
}

#######################################
# Display usage information
#######################################
agents2::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  $0 --action install                              # Install Agent S2 with default settings"
    echo "  $0 --action install --llm-provider anthropic     # Install with Anthropic provider"
    echo "  $0 --action status                               # Check Agent S2 status"
    echo "  $0 --action logs                                 # View Agent S2 logs"
    echo "  $0 --action usage                                # Show usage examples"
    echo "  $0 --action usage --usage-type screenshot        # Test screenshot API"
    echo "  $0 --action usage --usage-type automation        # Test automation capabilities"
    echo "  $0 --action uninstall                           # Remove Agent S2"
    echo
    echo "Access Points:"
    echo "  API: http://localhost:4113"
    echo "  VNC: vnc://localhost:5900"
    echo
    echo "Security Notes:"
    echo "  - Agent S2 runs in a secure Docker container with isolated display"
    echo "  - VNC access is password-protected"
    echo "  - Host display access is disabled by default for security"
}

#######################################
# Main execution function
#######################################
agents2::main() {
    agents2::parse_arguments "$@"
    
    # Export configuration after parsing arguments
    agents2::export_config
    
    case "$ACTION" in
        "install")
            agents2::install_service
            ;;
        "uninstall")
            agents2::uninstall_service
            ;;
        "start")
            agents2::docker_start
            ;;
        "stop")
            agents2::docker_stop
            ;;
        "restart")
            agents2::docker_restart
            ;;
        "status")
            agents2::show_status
            ;;
        "logs")
            agents2::docker_logs
            ;;
        "info")
            agents2::show_info
            ;;
        "usage")
            agents2::run_usage_example "$USAGE_TYPE"
            ;;
        *)
            log::error "Unknown action: $ACTION"
            agents2::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    agents2::main "$@"
fi