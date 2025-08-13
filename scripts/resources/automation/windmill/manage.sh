#!/usr/bin/env bash
set -euo pipefail

# Windmill Workflow Automation Platform Setup and Management
# This script handles installation, configuration, and management of Windmill using Docker Compose

export DESCRIPTION="Install and manage Windmill developer-centric workflow automation platform"

# Source var.sh first with relative path
# shellcheck disable=SC1091
WINDMILL_SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "$(dirname "$(dirname "$(dirname "${WINDMILL_SCRIPT_DIR}")")")/lib/utils/var.sh"

# Source common resources using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args-cli.sh"

# Source configuration using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/config/defaults.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/config/messages.sh"

# Export configuration
windmill::export_config

# Source all library modules using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/lib/state.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/lib/common.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/lib/docker.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/lib/database.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/lib/api.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/lib/workers.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/lib/status.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/lib/install.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/lib/apps.sh"

#######################################
# Parse command line arguments
#######################################
windmill::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|info|scale-workers|restart-workers|api-setup|save-api-key|backup|restore|list-apps|prepare-app|deploy-app|check-app-api|inject|validate-injection" \
        --default "install"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if Windmill appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "workers" \
        --flag "w" \
        --desc "Number of worker containers" \
        --type "value" \
        --default "${WINDMILL_WORKER_REPLICAS}"
    
    args::register \
        --name "external-db" \
        --desc "Use external PostgreSQL database" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "db-url" \
        --desc "External database URL (postgresql://user:pass@host:port/db)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "no-lsp" \
        --desc "Disable Language Server Protocol support" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "enable-multiplayer" \
        --desc "Enable multiplayer support (Enterprise Edition)" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "no-native-worker" \
        --desc "Disable native worker (for system commands)" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "memory-limit" \
        --desc "Memory limit per worker (e.g., 2048M, 4G)" \
        --type "value" \
        --default "${WINDMILL_WORKER_MEMORY_LIMIT}"
    
    args::register \
        --name "superadmin-email" \
        --desc "Super admin email address" \
        --type "value" \
        --default "${WINDMILL_SUPERADMIN_EMAIL}"
    
    args::register \
        --name "superadmin-password" \
        --desc "Super admin password (default: changeme)" \
        --type "value" \
        --default "${WINDMILL_SUPERADMIN_PASSWORD}"
    
    args::register \
        --name "service" \
        --desc "Specific service for logs/restart (server|worker|db|lsp|all)" \
        --type "value" \
        --default "all"
    
    args::register \
        --name "api-key" \
        --desc "Windmill API key to save" \
        --type "value" \
        --default ""
    
    args::register \
        --name "follow" \
        --desc "Follow log output (for logs action)" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "backup-path" \
        --desc "Path for backup/restore operations" \
        --type "value" \
        --default "${WINDMILL_BACKUP_DIR}"
    
    args::register \
        --name "app-name" \
        --desc "Name of the app to prepare/deploy" \
        --type "value" \
        --default ""
    
    args::register \
        --name "workspace" \
        --desc "Workspace to deploy app to (default: demo)" \
        --type "value" \
        --default "demo"
    
    args::register \
        --name "output-dir" \
        --desc "Output directory for prepared app files" \
        --type "value" \
        --default "${HOME}/windmill-apps"
    
    args::register \
        --name "injection-config" \
        --desc "JSON configuration for data injection" \
        --type "value" \
        --default ""
    
    if args::is_asking_for_help "$@"; then
        windmill::usage
        exit 0
    fi
    
    args::parse "$@"
    
    ACTION=$(args::get "action")
    FORCE=$(args::get "force")
    YES=$(args::get "yes")
    WORKER_COUNT=$(args::get "workers")
    EXTERNAL_DB=$(args::get "external-db")
    DB_URL=$(args::get "db-url")
    DISABLE_LSP=$(args::get "no-lsp")
    ENABLE_MULTIPLAYER=$(args::get "enable-multiplayer")
    DISABLE_NATIVE_WORKER=$(args::get "no-native-worker")
    WORKER_MEMORY_LIMIT=$(args::get "memory-limit")
    SUPERADMIN_EMAIL=$(args::get "superadmin-email")
    SUPERADMIN_PASSWORD=$(args::get "superadmin-password")
    SERVICE_NAME=$(args::get "service")
    FOLLOW_LOGS=$(args::get "follow")
    BACKUP_PATH=$(args::get "backup-path")
    APP_NAME=$(args::get "app-name")
    OUTPUT_DIR=$(args::get "output-dir")
    WORKSPACE=$(args::get "workspace")
    API_KEY=$(args::get "api-key")
    INJECTION_CONFIG=$(args::get "injection-config")
    
    export ACTION FORCE YES WORKER_COUNT EXTERNAL_DB DB_URL
    export DISABLE_LSP ENABLE_MULTIPLAYER DISABLE_NATIVE_WORKER
    export WORKER_MEMORY_LIMIT SUPERADMIN_EMAIL SUPERADMIN_PASSWORD
    export SERVICE_NAME FOLLOW_LOGS BACKUP_PATH APP_NAME
    export OUTPUT_DIR WORKSPACE API_KEY INJECTION_CONFIG
}

#######################################
# Main execution function
#######################################
windmill::main() {
    windmill::parse_arguments "$@"
    
    # Validate configuration
    if ! windmill::validate_config; then
        log::error "Configuration validation failed"
        exit 1
    fi
    
    case "$ACTION" in
        "install")
            windmill::install
            ;;
        "uninstall")
            windmill::uninstall
            ;;
        "start")
            windmill::start
            ;;
        "stop")
            windmill::stop
            ;;
        "restart")
            windmill::restart
            ;;
        "status")
            windmill::status
            ;;
        "logs")
            windmill::logs "$SERVICE_NAME" "$FOLLOW_LOGS"
            ;;
        "info")
            windmill::info
            ;;
        "scale-workers")
            windmill::scale_workers "$WORKER_COUNT"
            ;;
        "restart-workers")
            windmill::restart_workers
            ;;
        "api-setup")
            windmill::show_api_setup_instructions
            ;;
        "save-api-key")
            windmill::save_api_key
            ;;
        "backup")
            windmill::backup "$BACKUP_PATH"
            ;;
        "restore")
            windmill::restore "$BACKUP_PATH"
            ;;
        "list-apps")
            windmill::list_apps
            ;;
        "prepare-app")
            windmill::prepare_app "$APP_NAME" "$OUTPUT_DIR"
            ;;
        "deploy-app")
            windmill::deploy_app "$APP_NAME" "$WORKSPACE"
            ;;
        "check-app-api")
            windmill::check_app_api
            ;;
        "inject")
            if [[ -z "$INJECTION_CONFIG" ]]; then
                log::error "Injection configuration required for inject action"
                log::info "Use: --injection-config 'JSON_CONFIG'"
                exit 1
            fi
            "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/inject.sh" --inject "$INJECTION_CONFIG"
            ;;
        "validate-injection")
            if [[ -z "$INJECTION_CONFIG" ]]; then
                log::error "Injection configuration required for validate-injection action"
                log::info "Use: --injection-config 'JSON_CONFIG'"
                exit 1
            fi
            "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/inject.sh" --validate "$INJECTION_CONFIG"
            ;;
        *)
            log::error "Unknown action: $ACTION"
            windmill::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    windmill::main "$@"
fi