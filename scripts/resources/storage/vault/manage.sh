#!/usr/bin/env bash
set -euo pipefail

# Vault Secret Management Script
# This script handles installation, configuration, and management of HashiCorp Vault

DESCRIPTION="Install and manage HashiCorp Vault secret management service using Docker"

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
vault::export_config
vault::messages::init

# Source all library modules
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/docker.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/api.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/install.sh"

#######################################
# Parse command line arguments
#######################################
vault::parse_arguments() {
    # Check for help before args system processes it
    for arg in "$@"; do
        if [[ "$arg" == "--help" || "$arg" == "-help" || "$arg" == "help" ]]; then
            vault::show_help
            exit 0
        fi
    done
    
    args::reset
    
    args::register_help
    args::register_yes
    
    # Action argument
    args::register \
        --name "action" \
        --desc "Action to perform" \
        --type "value" \
        --default "status"
    
    # Secret management arguments
    args::register \
        --name "path" \
        --desc "Secret path for operations" \
        --type "value"
    
    args::register \
        --name "value" \
        --desc "Secret value to store" \
        --type "value"
    
    args::register \
        --name "key" \
        --desc "Secret key name (default: value)" \
        --type "value" \
        --default "value"
    
    args::register \
        --name "format" \
        --desc "Output format (json, raw, list)" \
        --type "value" \
        --options "json|raw|list" \
        --default "raw"
    
    # Migration arguments
    args::register \
        --name "env-file" \
        --desc "Environment file to migrate" \
        --type "value"
    
    args::register \
        --name "vault-prefix" \
        --desc "Vault path prefix for migration" \
        --type "value"
    
    # Mode configuration
    args::register \
        --name "mode" \
        --desc "Vault mode (dev, prod)" \
        --type "value" \
        --options "dev|prod" \
        --default "$VAULT_MODE"
    
    # Monitoring arguments
    args::register \
        --name "interval" \
        --desc "Monitoring interval in seconds" \
        --type "value" \
        --default "30"
    
    # Log arguments
    args::register \
        --name "lines" \
        --desc "Number of log lines to show" \
        --type "value" \
        --default "50"
    
    args::register \
        --name "follow" \
        --desc "Follow log output" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    # Data removal flag
    args::register \
        --name "remove-data" \
        --desc "Remove data directories during uninstall" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    # Backup/restore arguments
    args::register \
        --name "backup-file" \
        --desc "Backup file path" \
        --type "value"
    
    # Parse all arguments
    args::parse "$@"
    
    # Export parsed values
    export ACTION=$(args::get "action")
    export SECRET_PATH=$(args::get "path")
    export SECRET_VALUE=$(args::get "value")
    export SECRET_KEY=$(args::get "key")
    export OUTPUT_FORMAT=$(args::get "format")
    export ENV_FILE=$(args::get "env-file")
    export VAULT_PREFIX=$(args::get "vault-prefix")
    export VAULT_MODE=$(args::get "mode")
    export MONITOR_INTERVAL=$(args::get "interval")
    export LOG_LINES=$(args::get "lines")
    export FOLLOW_LOGS=$(args::get "follow")
    export VAULT_REMOVE_DATA=$(args::get "remove-data")
    export BACKUP_FILE=$(args::get "backup-file")
    
    # Re-export configuration with updated mode
    vault::export_config
}

#######################################
# Main execution function
#######################################
vault::main() {
    # Set up error handling
    trap vault::cleanup EXIT
    
    case "$ACTION" in
        "install")
            vault::install
            ;;
        "uninstall")
            vault::uninstall
            ;;
        "start")
            vault::docker::start_container
            ;;
        "stop")
            vault::docker::stop_container
            ;;
        "restart")
            vault::docker::restart_container
            ;;
        "status")
            vault::show_status
            ;;
        "auth-info")
            vault::show_auth_info
            ;;
        "test-functional")
            vault::test_functional
            ;;
        "logs")
            if [[ "$FOLLOW_LOGS" == "yes" ]]; then
                vault::docker::show_logs "$LOG_LINES" "follow"
            else
                vault::docker::show_logs "$LOG_LINES"
            fi
            ;;
        "init-dev")
            vault::init_dev
            ;;
        "init-prod")
            vault::init_prod
            ;;
        "unseal")
            vault::unseal
            ;;
        "put-secret")
            if [[ -z "$SECRET_PATH" ]] || [[ -z "$SECRET_VALUE" ]]; then
                log::error "Both --path and --value are required for put-secret"
                exit 1
            fi
            vault::put_secret "$SECRET_PATH" "$SECRET_VALUE" "$SECRET_KEY"
            ;;
        "get-secret")
            if [[ -z "$SECRET_PATH" ]]; then
                log::error "--path is required for get-secret"
                exit 1
            fi
            vault::get_secret "$SECRET_PATH" "$SECRET_KEY" "$OUTPUT_FORMAT"
            ;;
        "list-secrets")
            if [[ -z "$SECRET_PATH" ]]; then
                log::error "--path is required for list-secrets"
                exit 1
            fi
            vault::list_secrets "$SECRET_PATH" "$OUTPUT_FORMAT"
            ;;
        "delete-secret")
            if [[ -z "$SECRET_PATH" ]]; then
                log::error "--path is required for delete-secret"
                exit 1
            fi
            vault::delete_secret "$SECRET_PATH"
            ;;
        "migrate-env")
            if [[ -z "$ENV_FILE" ]] || [[ -z "$VAULT_PREFIX" ]]; then
                log::error "Both --env-file and --vault-prefix are required for migrate-env"
                exit 1
            fi
            vault::migrate_env_file "$ENV_FILE" "$VAULT_PREFIX"
            ;;
        "backup")
            vault::docker::backup "$BACKUP_FILE"
            ;;
        "restore")
            if [[ -z "$BACKUP_FILE" ]]; then
                log::error "--backup-file is required for restore"
                exit 1
            fi
            vault::docker::restore "$BACKUP_FILE"
            ;;
        "diagnose")
            vault::diagnose
            ;;
        "monitor")
            vault::monitor "$MONITOR_INTERVAL"
            ;;
        "help")
            vault::show_help
            ;;
        *)
            log::error "Unknown action: $ACTION"
            echo
            vault::show_help
            exit 1
            ;;
    esac
}

#######################################
# Script entry point
#######################################
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Only run if not being sourced
    if ! vault::parse_arguments "$@"; then
        exit 1
    fi
    
    vault::main
fi