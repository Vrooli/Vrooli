#!/usr/bin/env bash
set -euo pipefail

# Qdrant Vector Database Management Script
# This script handles installation, configuration, and management of Qdrant

DESCRIPTION="Install and manage Qdrant vector database using Docker"

QDRANT_SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
QDRANT_LIB_DIR="${QDRANT_SCRIPT_DIR}/lib"

# shellcheck disable=SC1091
source "${QDRANT_SCRIPT_DIR}/../../../lib/utils/var.sh"

# Source common resources
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args-cli.sh"
# shellcheck disable=SC1091
source "${var_LIB_DIR}/runtimes/docker.sh"

# Source configuration
# shellcheck disable=SC1091
source "${QDRANT_SCRIPT_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${QDRANT_SCRIPT_DIR}/config/messages.sh"

# Export configuration
qdrant::export_config
qdrant::messages::init

# Source library modules
# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/core.sh"
# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/health.sh"
# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/collections.sh"
# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/snapshots.sh"
# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/recovery.sh"
# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/inject.sh"
# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/install.sh"

#######################################
# Parse command line arguments
#######################################
qdrant::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|info|test|diagnose|monitor|list-collections|create-collection|delete-collection|collection-info|backup|restore|list-backups|backup-info|index-stats|upgrade|inject|validate-injection" \
        --default "status"
    
    args::register \
        --name "collection" \
        --flag "c" \
        --desc "Collection name for collection operations" \
        --type "value" \
        --default ""
    
    args::register \
        --name "vector-size" \
        --flag "s" \
        --desc "Vector size for new collections" \
        --type "value" \
        --default "1536"
    
    args::register \
        --name "distance" \
        --flag "d" \
        --desc "Distance metric (Cosine, Dot, Euclid)" \
        --type "value" \
        --options "Cosine|Dot|Euclid" \
        --default "Cosine"
    
    args::register \
        --name "remove-data" \
        --desc "Remove all data when uninstalling" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force operation (e.g., delete non-empty collections)" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "snapshot-name" \
        --desc "Name for backup snapshot" \
        --type "value" \
        --default ""
    
    args::register \
        --name "collections" \
        --desc "Comma-separated list of collections for backup" \
        --type "value" \
        --default "all"
    
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
        qdrant::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export COLLECTION=$(args::get "collection")
    export VECTOR_SIZE=$(args::get "vector-size")
    export DISTANCE_METRIC=$(args::get "distance")
    export REMOVE_DATA=$(args::get "remove-data")
    export FORCE=$(args::get "force")
    export SNAPSHOT_NAME=$(args::get "snapshot-name")
    export COLLECTIONS_LIST=$(args::get "collections")
    export LOG_LINES=$(args::get "lines")
    export MONITOR_INTERVAL=$(args::get "interval")
    export INJECTION_CONFIG=$(args::get "injection-config")
    export YES=$(args::get "yes")
}

#######################################
# Display usage information
#######################################
#######################################
# Execute command only if Qdrant is running
# Arguments: command to execute
# Returns: command result or exits with error
#######################################
qdrant::require_running() {
    if qdrant::common::is_running; then
        "$@"
    else
        log::error "Qdrant is not running"
        exit 1
    fi
}

#######################################
# Display usage information
#######################################
qdrant::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  # Install Qdrant with default settings"
    echo "  $0 --action install"
    echo
    echo "  # Install with custom ports"
    echo "  QDRANT_CUSTOM_PORT=6433 QDRANT_CUSTOM_GRPC_PORT=6434 $0 --action install"
    echo
    echo "  # Install with API key authentication"
    echo "  QDRANT_CUSTOM_API_KEY=secure-api-key $0 --action install"
    echo
    echo "  # Check status"
    echo "  $0 --action status"
    echo
    echo "  # Create a new collection"
    echo "  $0 --action create-collection --collection my_vectors --vector-size 768 --distance Dot"
    echo
    echo "  # List all collections"
    echo "  $0 --action list-collections"
    echo
    echo "  # Get collection information"
    echo "  $0 --action collection-info --collection agent_memory"
    echo
    echo "  # Create backup of specific collections"
    echo "  $0 --action backup --collections \"agent_memory,code_embeddings\" --snapshot-name daily-backup"
    echo
    echo "  # Create backup of all collections"
    echo "  $0 --action backup --snapshot-name full-backup"
    echo
    echo "  # Monitor health continuously"
    echo "  $0 --action monitor --interval 10"
    echo
    echo "  # Uninstall and remove all data"
    echo "  $0 --action uninstall --remove-data yes"
    echo
    echo "  # Validate injection configuration"
    echo "  $0 --action validate-injection --injection-config '{\"collections\": [{\"name\": \"docs\", \"size\": 384}]}'"
    echo
    echo "  # Inject collections and vectors"
    echo "  $0 --action inject --injection-config '{\"collections\": [{\"name\": \"docs\", \"size\": 384}], \"vectors\": [{\"collection\": \"docs\", \"vectors\": \"data.json\"}]}'"
    echo
    echo "Environment Variables:"
    echo "  QDRANT_CUSTOM_PORT         - Custom REST API port (default: 6333)"
    echo "  QDRANT_CUSTOM_GRPC_PORT    - Custom gRPC port (default: 6334)"
    echo "  QDRANT_CUSTOM_API_KEY      - API key for authentication (default: none)"
}

#######################################
# Main execution
#######################################
main() {
    qdrant::parse_arguments "$@"
    
    case "$ACTION" in
        install)
            qdrant::install
            ;;
        uninstall)
            if [[ "$REMOVE_DATA" == "yes" ]]; then
                if [[ "$YES" != "yes" ]]; then
                    log::warn "This will permanently delete all Qdrant data!"
                    read -p "Are you sure? (y/N) " -n 1 -r
                    echo
                    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                        log::info "Uninstall cancelled"
                        exit 0
                    fi
                fi
                qdrant::uninstall false
            else
                qdrant::uninstall true
            fi
            ;;
        start)
            qdrant::docker::start
            ;;
        stop)
            qdrant::docker::stop
            ;;
        restart)
            qdrant::docker::restart
            ;;
        status)
            qdrant::status::check true
            ;;
        logs)
            qdrant::common::show_logs "$LOG_LINES"
            ;;
        info)
            qdrant::info
            ;;
        test)
            qdrant::test
            ;;
        diagnose)
            qdrant::diagnose
            ;;
        monitor)
            qdrant::status::monitor "$MONITOR_INTERVAL"
            ;;
        list-collections)
            qdrant::require_running qdrant::collections::list
            ;;
        create-collection)
            if [[ -z "${COLLECTION:-}" ]]; then
                log::error "Collection name is required"
                log::info "Usage: $0 --action create-collection --collection <name> [--vector-size <size>] [--distance <metric>]"
                exit 1
            fi
            qdrant::require_running qdrant::collections::create "$COLLECTION" "$VECTOR_SIZE" "$DISTANCE_METRIC"
            ;;
        delete-collection)
            if [[ -z "${COLLECTION:-}" ]]; then
                log::error "Collection name is required"
                log::info "Usage: $0 --action delete-collection --collection <name> [--force yes]"
                exit 1
            fi
            qdrant::require_running qdrant::collections::delete "$COLLECTION" "$FORCE"
            ;;
        collection-info)
            if [[ -z "${COLLECTION:-}" ]]; then
                log::error "Collection name is required"
                log::info "Usage: $0 --action collection-info --collection <name>"
                exit 1
            fi
            qdrant::require_running qdrant::collections::info "$COLLECTION"
            ;;
        backup)
            if [[ -n "${SNAPSHOT_NAME:-}" ]]; then
                qdrant::require_running qdrant::create_backup "$SNAPSHOT_NAME"
            else
                qdrant::require_running qdrant::create_backup
            fi
            ;;
        restore)
            if [[ -z "${SNAPSHOT_NAME:-}" ]]; then
                log::error "Backup name is required"
                log::info "Usage: $0 --action restore --snapshot-name <name>"
                exit 1
            fi
            qdrant::restore_backup "$SNAPSHOT_NAME"
            ;;
        list-backups)
            qdrant::list_backups
            ;;
        backup-info)
            if [[ -z "${SNAPSHOT_NAME:-}" ]]; then
                log::error "Backup name is required"
                log::info "Usage: $0 --action backup-info --snapshot-name <name>"
                exit 1
            fi
            qdrant::backup_info "$SNAPSHOT_NAME"
            ;;
        index-stats)
            if [[ -n "${COLLECTION:-}" ]]; then
                qdrant::require_running qdrant::collections::index_stats "$COLLECTION"
            else
                qdrant::require_running qdrant::collections::index_stats_all
            fi
            ;;
        upgrade)
            qdrant::install::upgrade
            ;;
        inject)
            if [[ -z "${INJECTION_CONFIG:-}" ]]; then
                log::error "Injection configuration is required"
                log::info "Usage: $0 --action inject --injection-config '{...}'"
                exit 1
            fi
            qdrant::require_running qdrant::inject
            ;;
        validate-injection)
            if [[ -z "${INJECTION_CONFIG:-}" ]]; then
                log::error "Injection configuration is required"
                log::info "Usage: $0 --action validate-injection --injection-config '{...}'"
                exit 1
            fi
            qdrant::validate_injection
            ;;
        *)
            log::error "Unknown action: $ACTION"
            qdrant::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi