#!/usr/bin/env bash
set -euo pipefail

# n8n Workflow Automation Platform Setup and Management
# This script handles installation, configuration, and management of n8n using Docker

export DESCRIPTION="Install and manage n8n workflow automation platform using Docker"

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

# Source refactored library modules
# Core module contains most shared functionality
# shellcheck disable=SC1091
source "${N8N_SCRIPT_DIR}/lib/core.sh"
# shellcheck disable=SC1091
source "${N8N_SCRIPT_DIR}/lib/docker.sh"
# shellcheck disable=SC1091
source "${N8N_SCRIPT_DIR}/lib/health.sh"
# shellcheck disable=SC1091
source "${N8N_SCRIPT_DIR}/lib/recovery.sh"
# shellcheck disable=SC1091
source "${N8N_SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${N8N_SCRIPT_DIR}/lib/api.sh"
# shellcheck disable=SC1091
source "${N8N_SCRIPT_DIR}/lib/inject.sh"

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
        --options "install|uninstall|start|stop|restart|status|reset-password|logs|info|version|test|test-api-key|execute|api-setup|save-api-key|list-workflows|list-executions|inject|validate-injection|url|create-backup|list-backups|backup-info|recover|recover-api-key|auto-credentials|refresh-credentials|validate-credentials|list-discoverable|setup-redis-databases|activate-workflow|deactivate-workflow|activate-all|activate-pattern|delete-workflow|delete-all-workflows" \
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
        --name "workflow-id" \
        --desc "Workflow ID for activation/deactivation" \
        --type "value" \
        --default ""
    
    args::register \
        --name "pattern" \
        --desc "Pattern for matching workflow names (supports wildcards)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "data" \
        --desc "JSON data to pass to workflow (for webhook workflows)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "label" \
        --desc "Label for backup (default: auto)" \
        --type "value" \
        --default "auto"
    
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
    
    args::register \
        --name "injection-config-file" \
        --desc "Path to file containing JSON configuration for data injection" \
        --type "value" \
        --default ""
    
    args::register \
        --name "auto-credentials" \
        --desc "Enable/disable automatic credential creation" \
        --type "value" \
        --options "yes|no" \
        --default "yes"
    
    if args::is_asking_for_help "$@"; then
        n8n::usage
        exit 0
    fi
    
    args::parse "$@"
    
    ACTION=$(args::get "action")
    FORCE=$(args::get "force")
    LINES=$(args::get "lines")
    YES=$(args::get "yes")
    WEBHOOK_URL=$(args::get "webhook-url")
    WORKFLOW_ID=$(args::get "workflow-id")
    API_KEY=$(args::get "api-key")
    WORKFLOW_DATA=$(args::get "data")
    BASIC_AUTH=$(args::get "basic-auth")
    AUTH_USERNAME=$(args::get "username")
    AUTH_PASSWORD=$(args::get "password")
    DATABASE_TYPE=$(args::get "database")
    TUNNEL_ENABLED=$(args::get "tunnel")
    BUILD_IMAGE=$(args::get "build-image")
    INJECTION_CONFIG=$(args::get "injection-config")
    INJECTION_CONFIG_FILE=$(args::get "injection-config-file")
    VALIDATION_TYPE=$(args::get "validation-type")
    VALIDATION_FILE=$(args::get "validation-file")
    AUTO_CREATE_CREDENTIALS=$(args::get "auto-credentials")
    ACTIVATE_WORKFLOW_ID=$(args::get "workflow-id")
    ACTIVATION_PATTERN=$(args::get "pattern")
    export ACTION FORCE LINES YES WEBHOOK_URL WORKFLOW_ID API_KEY WORKFLOW_DATA BASIC_AUTH AUTH_USERNAME AUTH_PASSWORD DATABASE_TYPE TUNNEL_ENABLED BUILD_IMAGE INJECTION_CONFIG VALIDATION_TYPE VALIDATION_FILE AUTO_CREATE_CREDENTIALS ACTIVATE_WORKFLOW_ID ACTIVATION_PATTERN
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
        version)
            n8n::version
            ;;
        test)
            n8n::test
            ;;
        test-api-key)
            n8n::test_api_key
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
        list-workflows)
            n8n::list_workflows
            ;;
        list-executions)
            n8n::get_executions
            ;;
        inject)
            n8n::inject
            ;;
        validate-injection)
            n8n::validate_injection
            ;;
        url)
            n8n::get_urls
            ;;
        create-backup)
            n8n::create_backup "${LABEL:-auto}"
            ;;
        list-backups)
            backup::list "n8n"
            ;;
        backup-info)
            backup::info "n8n"
            ;;
        recover)
            n8n::recover
            ;;
        recover-api-key)
            n8n::recover_api_key
            ;;
        auto-credentials)
            n8n::auto_manage_credentials
            ;;
        setup-redis-databases)
            log::info "Setting up Redis multi-database credentials..."
            
            # Discover Redis resource
            local redis_resource
            redis_resource=$(n8n::discover_resources | jq -c '.[] | select(.name == "redis")')
            
            if [[ -z "$redis_resource" || "$redis_resource" == "null" ]]; then
                log::error "No Redis resource found. Ensure Redis is running and discoverable."
                exit 1
            fi
            
            # Create multi-database credentials
            if n8n::create_redis_database_credentials "$redis_resource"; then
                log::success "✅ Redis multi-database credentials setup complete"
                log::info "Now available: vrooli-redis-0 through vrooli-redis-15"
            else
                log::error "❌ Failed to setup Redis multi-database credentials"
                exit 1
            fi
            ;;
        refresh-credentials)
            log::info "Refreshing auto-credentials for all discovered resources..."
            AUTO_CREATE_CREDENTIALS=yes n8n::auto_manage_credentials
            ;;
        validate-credentials)
            n8n::validate_auto_credentials
            ;;
        list-discoverable)
            n8n::list_discoverable_resources
            ;;
        activate-workflow)
            if [[ -z "$ACTIVATE_WORKFLOW_ID" ]]; then
                log::error "Workflow ID is required for activation"
                echo "Usage: $0 --action activate-workflow --workflow-id <workflow_id>"
                echo ""
                echo "To find workflow IDs, run: $0 --action list-workflows"
                exit 1
            fi
            n8n::activate_workflow "$ACTIVATE_WORKFLOW_ID"
            ;;
        deactivate-workflow)
            if [[ -z "$ACTIVATE_WORKFLOW_ID" ]]; then
                log::error "Workflow ID is required for deactivation"
                echo "Usage: $0 --action deactivate-workflow --workflow-id <workflow_id>"
                echo ""
                echo "To find workflow IDs, run: $0 --action list-workflows"
                exit 1
            fi
            n8n::deactivate_workflow "$ACTIVATE_WORKFLOW_ID"
            ;;
        activate-all)
            n8n::activate_all_workflows
            ;;
        activate-pattern)
            if [[ -z "$ACTIVATION_PATTERN" ]]; then
                log::error "Pattern is required for pattern-based activation"
                echo "Usage: $0 --action activate-pattern --pattern <pattern>"
                echo ""
                echo "Examples:"
                echo "  $0 --action activate-pattern --pattern \"embedding*\""
                echo "  $0 --action activate-pattern --pattern \"*generator*\""
                echo "  $0 --action activate-pattern --pattern \"reasoning-chain\""
                exit 1
            fi
            n8n::activate_workflows_by_pattern "$ACTIVATION_PATTERN"
            ;;
        delete-workflow)
            if [[ -z "$WORKFLOW_ID" ]]; then
                log::error "Workflow ID is required for deletion"
                echo "Usage: $0 --action delete-workflow --workflow-id <id>"
                echo ""
                echo "Examples:"
                echo "  $0 --action delete-workflow --workflow-id wbu5QO9dVCD521pe"
                exit 1
            fi
            n8n::delete_workflow "$WORKFLOW_ID"
            ;;
        delete-all-workflows)
            n8n::delete_all_workflows
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