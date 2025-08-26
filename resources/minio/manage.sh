#!/usr/bin/env bash

# ⚠️ DEPRECATION NOTICE: This script is deprecated as of v2.0 (January 2025)
# Please use cli.sh instead: resource-minio <command>
# This file will be removed in v3.0 (target: December 2025)
#
# Migration guide:
#   OLD: ./manage.sh --action <command>
#   NEW: ./cli.sh <command>  OR  resource-minio <command>

# Show deprecation warning
if [[ "${VROOLI_SUPPRESS_DEPRECATION:-}" != "true" ]]; then
    echo "⚠️  WARNING: manage.sh is deprecated. Please use cli.sh instead." >&2
    echo "   This script will be removed in v3.0 (December 2025)" >&2
    echo "   To suppress this warning: export VROOLI_SUPPRESS_DEPRECATION=true" >&2
    echo "" >&2
fi

set -euo pipefail

# MinIO Object Storage Management Script
# This script handles installation, configuration, and management of MinIO

DESCRIPTION="Install and manage MinIO S3-compatible object storage using Docker"

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
MINIO_SCRIPT_DIR="${APP_ROOT}/resources/minio"
MINIO_LIB_DIR="${MINIO_SCRIPT_DIR}/lib"

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source common resources using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args-cli.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"

# Source configuration
# shellcheck disable=SC1091
source "${MINIO_SCRIPT_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${MINIO_SCRIPT_DIR}/config/messages.sh"

# Export configuration
minio::export_config
minio::messages::init

# Source all library modules
# shellcheck disable=SC1091
source "${MINIO_LIB_DIR}/common.sh"
# shellcheck disable=SC1091
source "${MINIO_LIB_DIR}/docker.sh"
# shellcheck disable=SC1091
source "${MINIO_LIB_DIR}/api.sh"
# shellcheck disable=SC1091
source "${MINIO_LIB_DIR}/buckets.sh"
# shellcheck disable=SC1091
source "${MINIO_LIB_DIR}/status.sh"
# shellcheck disable=SC1091
source "${MINIO_LIB_DIR}/install.sh"

#######################################
# Parse command line arguments
#######################################
minio::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|info|test|diagnose|monitor|list-buckets|create-bucket|remove-bucket|show-credentials|reset-credentials|test-upload|upgrade|inject|validate-injection" \
        --default "status"
    
    args::register \
        --name "bucket" \
        --flag "b" \
        --desc "Bucket name for bucket operations" \
        --type "value" \
        --default ""
    
    args::register \
        --name "policy" \
        --flag "p" \
        --desc "Bucket policy (public, download, upload, none)" \
        --type "value" \
        --options "public|download|upload|none" \
        --default "none"
    
    args::register \
        --name "remove-data" \
        --desc "Remove all data when uninstalling" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force operation (e.g., remove non-empty buckets)" \
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
        --name "interval" \
        --flag "i" \
        --desc "Monitor interval in seconds" \
        --type "value" \
        --default "5"
    
    args::register \
        --name "injection-config" \
        --desc "JSON configuration for data injection" \
        --type "value" \
        --default ""
    
    if args::is_asking_for_help "$@"; then
        minio::usage
        exit 0
    fi
    
    args::parse "$@"
    
    ACTION=$(args::get "action")
    BUCKET=$(args::get "bucket")
    POLICY=$(args::get "policy")
    REMOVE_DATA=$(args::get "remove-data")
    FORCE=$(args::get "force")
    LOG_LINES=$(args::get "lines")
    MONITOR_INTERVAL=$(args::get "interval")
    INJECTION_CONFIG=$(args::get "injection-config")
    export ACTION BUCKET POLICY REMOVE_DATA FORCE LOG_LINES MONITOR_INTERVAL INJECTION_CONFIG
}

#######################################
# Display usage information
#######################################
minio::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  # Install MinIO with default settings"
    echo "  $0 --action install"
    echo
    echo "  # Install with custom ports"
    echo "  MINIO_CUSTOM_PORT=9100 MINIO_CUSTOM_CONSOLE_PORT=9101 $0 --action install"
    echo
    echo "  # Install with custom credentials"
    echo "  MINIO_CUSTOM_ROOT_USER=admin MINIO_CUSTOM_ROOT_PASSWORD=secretpass $0 --action install"
    echo
    echo "  # Check status"
    echo "  $0 --action status"
    echo
    echo "  # Create a new bucket"
    echo "  $0 --action create-bucket --bucket my-bucket --policy download"
    echo
    echo "  # Show access credentials"
    echo "  $0 --action show-credentials"
    echo
    echo "  # Monitor health continuously"
    echo "  $0 --action monitor --interval 10"
    echo
    echo "  # Uninstall and remove all data"
    echo "  $0 --action uninstall --remove-data yes"
    echo
    echo "Environment Variables:"
    echo "  MINIO_CUSTOM_PORT          - Custom API port (default: 9000)"
    echo "  MINIO_CUSTOM_CONSOLE_PORT  - Custom console port (default: 9001)"
    echo "  MINIO_CUSTOM_ROOT_USER     - Custom root username"
    echo "  MINIO_CUSTOM_ROOT_PASSWORD - Custom root password"
}

#######################################
# Main execution
#######################################
main() {
    minio::parse_arguments "$@"
    
    case "$ACTION" in
        install)
            minio::install
            ;;
        uninstall)
            if [[ "$REMOVE_DATA" == "yes" ]]; then
                if [[ "$YES" != "yes" ]]; then
                    log::warn "This will permanently delete all MinIO data!"
                    read -p "Are you sure? (y/N) " -n 1 -r
                    echo
                    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                        log::info "Uninstall cancelled"
                        exit 0
                    fi
                fi
                minio::uninstall false
            else
                minio::uninstall true
            fi
            ;;
        start)
            minio::docker::start
            ;;
        stop)
            minio::docker::stop
            ;;
        restart)
            minio::docker::restart
            ;;
        status)
            minio::status::check true
            ;;
        logs)
            minio::common::show_logs "$LOG_LINES"
            ;;
        info)
            minio::info
            ;;
        test)
            minio::test
            ;;
        diagnose)
            minio::status::diagnose
            ;;
        monitor)
            minio::status::monitor "$MONITOR_INTERVAL"
            ;;
        list-buckets)
            if minio::common::is_running; then
                minio::buckets::show_stats
            else
                log::error "MinIO is not running"
                exit 1
            fi
            ;;
        create-bucket)
            if [[ -z "${BUCKET:-}" ]]; then
                log::error "Bucket name is required"
                log::info "Usage: $0 --action create-bucket --bucket <name> [--policy <type>]"
                exit 1
            fi
            if minio::common::is_running; then
                minio::buckets::create_custom "$BUCKET" "$POLICY"
            else
                log::error "MinIO is not running"
                exit 1
            fi
            ;;
        remove-bucket)
            if [[ -z "${BUCKET:-}" ]]; then
                log::error "Bucket name is required"
                log::info "Usage: $0 --action remove-bucket --bucket <name> [--force yes]"
                exit 1
            fi
            if minio::common::is_running; then
                minio::buckets::remove "$BUCKET" "$FORCE"
            else
                log::error "MinIO is not running"
                exit 1
            fi
            ;;
        show-credentials)
            minio::status::show_credentials
            ;;
        reset-credentials)
            if [[ "$YES" != "yes" ]]; then
                log::warn "This will reset your MinIO access credentials!"
                log::warn "You will need to update any applications using the current credentials."
                read -p "Continue? (y/N) " -n 1 -r
                echo
                if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                    log::info "Credential reset cancelled"
                    exit 0
                fi
            fi
            minio::install::reset_credentials
            ;;
        test-upload)
            if minio::common::is_running; then
                minio::buckets::test_upload
            else
                log::error "MinIO is not running"
                exit 1
            fi
            ;;
        upgrade)
            minio::install::upgrade
            ;;
        inject)
            if [[ -z "$INJECTION_CONFIG" ]]; then
                log::error "Injection configuration required for inject action"
                log::info "Use: --injection-config 'JSON_CONFIG'"
                exit 1
            fi
            "${MINIO_SCRIPT_DIR}/inject.sh" --inject "$INJECTION_CONFIG"
            ;;
        validate-injection)
            if [[ -z "$INJECTION_CONFIG" ]]; then
                log::error "Injection configuration required for validate-injection action"
                log::info "Use: --injection-config 'JSON_CONFIG'"
                exit 1
            fi
            "${MINIO_SCRIPT_DIR}/inject.sh" --validate "$INJECTION_CONFIG"
            ;;
        *)
            log::error "Unknown action: $ACTION"
            minio::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi