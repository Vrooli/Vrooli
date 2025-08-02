#!/usr/bin/env bash
set -euo pipefail

# MinIO Object Storage Management Script
# This script handles installation, configuration, and management of MinIO

DESCRIPTION="Install and manage MinIO S3-compatible object storage using Docker"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# Source common resources
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/args.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/docker.sh"

# Source configuration
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/messages.sh"

# Export configuration
minio::export_config
minio::messages::init

# Source all library modules
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/docker.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/api.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/buckets.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/install.sh"

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
        --options "install|uninstall|start|stop|restart|status|logs|diagnose|monitor|list-buckets|create-bucket|remove-bucket|show-credentials|reset-credentials|test-upload|upgrade" \
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
    
    if args::is_asking_for_help "$@"; then
        minio::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export BUCKET=$(args::get "bucket")
    export POLICY=$(args::get "policy")
    export REMOVE_DATA=$(args::get "remove-data")
    export FORCE=$(args::get "force")
    export LOG_LINES=$(args::get "lines")
    export MONITOR_INTERVAL=$(args::get "interval")
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