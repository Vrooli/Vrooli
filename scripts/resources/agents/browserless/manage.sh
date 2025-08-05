#!/usr/bin/env bash
set -euo pipefail

# Handle Ctrl+C gracefully
trap 'echo ""; log::info "Browserless installation interrupted by user. Exiting..."; exit 130' INT TERM

# Browserless Chrome Service Setup and Management
# This script handles installation, configuration, and management of Browserless.io using Docker

DESCRIPTION="Install and manage Browserless headless Chrome service using Docker"

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

# Export configuration
browserless::export_config
browserless::export_messages

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
browserless::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|info|test|usage" \
        --default "install"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if Browserless appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "max-browsers" \
        --desc "Maximum concurrent browser instances" \
        --type "value" \
        --default "5"
    
    args::register \
        --name "headless" \
        --desc "Run browsers in headless mode" \
        --type "value" \
        --options "yes|no" \
        --default "yes"
    
    args::register \
        --name "timeout" \
        --desc "Browser timeout in milliseconds" \
        --type "value" \
        --default "30000"
    
    args::register \
        --name "usage-type" \
        --desc "Type of usage example to run" \
        --type "value" \
        --options "screenshot|pdf|scrape|pressure|function|all|help" \
        --default "help"
    
    args::register \
        --name "url" \
        --desc "URL to use for usage examples" \
        --type "value" \
        --default "https://example.com"
    
    args::register \
        --name "output" \
        --desc "Output file for usage examples" \
        --type "value" \
        --default ""
    
    if args::is_asking_for_help "$@"; then
        browserless::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export FORCE=$(args::get "force")
    export YES=$(args::get "yes")
    export MAX_BROWSERS=$(args::get "max-browsers")
    export HEADLESS=$(args::get "headless")
    export TIMEOUT=$(args::get "timeout")
    export USAGE_TYPE=$(args::get "usage-type")
    export URL=$(args::get "url")
    export OUTPUT=$(args::get "output")
}

#######################################
# Display usage information
#######################################
browserless::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  $0 --action install                              # Install Browserless with default settings"
    echo "  $0 --action install --max-browsers 10           # Install with 10 max browsers"
    echo "  $0 --action install --headless no                # Install with headed browsers"
    echo "  $0 --action status                               # Check Browserless status"
    echo "  $0 --action logs                                 # View Browserless logs"
    echo "  $0 --action usage                                # Show usage examples"
    echo "  $0 --action usage --usage-type screenshot        # Test screenshot API"
    echo "  $0 --action usage --usage-type all              # Run all usage examples"
    echo "  $0 --action uninstall                           # Remove Browserless"
}

#######################################
# Main execution function
#######################################
browserless::main() {
    browserless::parse_arguments "$@"
    
    case "$ACTION" in
        "install")
            browserless::install_service
            ;;
        "uninstall")
            browserless::uninstall_service
            ;;
        "start")
            browserless::docker_start
            ;;
        "stop")
            browserless::docker_stop
            ;;
        "restart")
            browserless::docker_restart
            ;;
        "status")
            browserless::show_status
            ;;
        "logs")
            browserless::docker_logs
            ;;
        "info")
            browserless::show_info
            ;;
        "test")
            browserless::test_all_apis "${URL:-https://example.com}"
            ;;
        "usage")
            browserless::run_usage_example "$USAGE_TYPE"
            ;;
        *)
            log::error "Unknown action: $ACTION"
            browserless::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    browserless::main "$@"
fi