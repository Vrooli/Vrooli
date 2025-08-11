#!/usr/bin/env bash
set -euo pipefail

# n8n Workflow Automation Platform Setup and Management
# This script handles installation, configuration, and management of n8n using Docker

DESCRIPTION="Install and manage n8n workflow automation platform using Docker"

N8N_SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "$(dirname "$(dirname "$(dirname "${N8N_SCRIPT_DIR}")")")/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args-cli.sh"

# Source configuration
# shellcheck disable=SC1091
source "${N8N_SCRIPT_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${N8N_SCRIPT_DIR}/config/messages.sh"

# Export configuration
n8n::export_config

# Source all library modules
# shellcheck disable=SC1091
source "${N8N_SCRIPT_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${N8N_SCRIPT_DIR}/lib/docker.sh"
# shellcheck disable=SC1091
source "${N8N_SCRIPT_DIR}/lib/database.sh"
# shellcheck disable=SC1091
source "${N8N_SCRIPT_DIR}/lib/password.sh"
# shellcheck disable=SC1091
source "${N8N_SCRIPT_DIR}/lib/api.sh"
# shellcheck disable=SC1091
source "${N8N_SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${N8N_SCRIPT_DIR}/lib/install.sh"

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
        --name "validation-type" \
        --desc "Type of content being validated (workflow, credential, etc.)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "validation-file" \
        --desc "Path to file containing content to validate" \
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
    export VALIDATION_TYPE=$(args::get "validation-type")
    export VALIDATION_FILE=$(args::get "validation-file")
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
            n8n::inject_data "$INJECTION_CONFIG"
            ;;
        validate-injection)
            # Support both legacy and new validation interfaces
            if [[ -n "$VALIDATION_TYPE" && -n "$VALIDATION_FILE" ]]; then
                # New interface: type + file
                if [[ ! -f "$VALIDATION_FILE" ]]; then
                    log::error "Validation file not found: $VALIDATION_FILE"
                    exit 1
                fi
                
                local content
                content=$(cat "$VALIDATION_FILE")
                
                # Call inject.sh with type and content
                "${SCRIPT_DIR}/lib/inject.sh" \
                    --validate \
                    --type "$VALIDATION_TYPE" \
                    --content "$content"
            elif [[ -n "$INJECTION_CONFIG" ]]; then
                # Legacy interface: full config JSON
                n8n::validate_injection "$INJECTION_CONFIG"
            else
                log::error "Required: --validation-type TYPE --validation-file PATH"
                log::info "   or legacy: --injection-config 'JSON_CONFIG'"
                exit 1
            fi
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