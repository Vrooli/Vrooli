#!/usr/bin/env bash
set -euo pipefail

# n8n Workflow Automation Platform Setup and Management
# This script handles installation, configuration, and management of n8n using Docker

DESCRIPTION="Install and manage n8n workflow automation platform using Docker"

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
n8n::export_config

# Source all library modules
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/docker.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/database.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/password.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/api.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/install.sh"

#######################################
# Parse command line arguments
#######################################
n8n::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|reset-password|logs|info|test|execute|api-setup|save-api-key|inject|validate-injection|url" \
        --default "install"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if n8n appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "lines" \
        --flag "n" \
        --desc "Number of log lines to show" \
        --type "value" \
        --default "50"
    
    args::register \
        --name "webhook-url" \
        --desc "External webhook URL (default: auto-detect)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "workflow-id" \
        --desc "Workflow ID for execution" \
        --type "value" \
        --default ""
    
    args::register \
        --name "api-key" \
        --desc "n8n API key to save" \
        --type "value" \
        --default ""
    
    args::register \
        --name "data" \
        --desc "JSON data to pass to workflow (for webhook workflows)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "basic-auth" \
        --desc "Enable basic authentication" \
        --type "value" \
        --options "yes|no" \
        --default "yes"
    
    args::register \
        --name "username" \
        --desc "Basic auth username (default: admin)" \
        --type "value" \
        --default "admin"
    
    args::register \
        --name "password" \
        --desc "Basic auth password (default: auto-generated)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "database" \
        --desc "Database type (sqlite or postgres)" \
        --type "value" \
        --options "sqlite|postgres" \
        --default "sqlite"
    
    args::register \
        --name "tunnel" \
        --desc "Enable tunnel for webhook testing (dev only)" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "build-image" \
        --desc "Build custom n8n image with host access" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "injection-config" \
        --desc "JSON configuration for data injection" \
        --type "value" \
        --default ""
    
    if args::is_asking_for_help "$@"; then
        n8n::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export FORCE=$(args::get "force")
    export LINES=$(args::get "lines")
    export YES=$(args::get "yes")
    export WEBHOOK_URL=$(args::get "webhook-url")
    export WORKFLOW_ID=$(args::get "workflow-id")
    export API_KEY=$(args::get "api-key")
    export WORKFLOW_DATA=$(args::get "data")
    export BASIC_AUTH=$(args::get "basic-auth")
    export AUTH_USERNAME=$(args::get "username")
    export AUTH_PASSWORD=$(args::get "password")
    export DATABASE_TYPE=$(args::get "database")
    export TUNNEL_ENABLED=$(args::get "tunnel")
    export BUILD_IMAGE=$(args::get "build-image")
    export INJECTION_CONFIG=$(args::get "injection-config")
}

#######################################
# Main execution function
#######################################

n8n::main() {
    n8n::parse_arguments "$@"
    
    case "$ACTION" in
        install)
            n8n::install
            ;;
        uninstall)
            n8n::uninstall
            ;;
        start)
            n8n::start
            ;;
        stop)
            n8n::stop
            ;;
        restart)
            n8n::restart
            ;;
        status)
            n8n::status
            ;;
        reset-password)
            n8n::reset_password
            ;;
        logs)
            n8n::logs
            ;;
        info)
            n8n::info
            ;;
        test)
            n8n::test
            ;;
        execute)
            n8n::execute
            ;;
        api-setup)
            n8n::api_setup
            ;;
        save-api-key)
            n8n::save_api_key
            ;;
        inject)
            if [[ -z "$INJECTION_CONFIG" ]]; then
                log::error "Injection configuration required for inject action"
                log::info "Use: --injection-config 'JSON_CONFIG'"
                exit 1
            fi
            "${SCRIPT_DIR}/inject.sh" --inject "$INJECTION_CONFIG"
            ;;
        validate-injection)
            if [[ -z "$INJECTION_CONFIG" ]]; then
                log::error "Injection configuration required for validate-injection action"
                log::info "Use: --injection-config 'JSON_CONFIG'"
                exit 1
            fi
            "${SCRIPT_DIR}/inject.sh" --validate "$INJECTION_CONFIG"
            ;;
        url)
            n8n::get_urls
            ;;
        *)
            log::error "Unknown action: $ACTION"
            n8n::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    n8n::main "$@"
fi