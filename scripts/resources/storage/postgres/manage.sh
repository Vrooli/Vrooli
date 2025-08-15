#!/usr/bin/env bash
set -euo pipefail

# PostgreSQL Storage Resource Management Script
# This script handles installation, configuration, and management of PostgreSQL instances

DESCRIPTION="Install and manage PostgreSQL database instances for client isolation using Docker"

POSTGRES_SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
POSTGRES_LIB_DIR="${POSTGRES_SCRIPT_DIR}/lib"

# Clear any source guards that may have been inherited from parent process
unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED 2>/dev/null || true

# Source var.sh first for directory variables
# shellcheck disable=SC1091
source "${POSTGRES_SCRIPT_DIR}/../../../lib/utils/var.sh"

# Source common resources using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args-cli.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"

# Source configuration
# shellcheck disable=SC1091
source "${POSTGRES_SCRIPT_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${POSTGRES_SCRIPT_DIR}/config/messages.sh"

# Export configuration
postgres::export_config
postgres::messages::init

# Source all library modules
# shellcheck disable=SC1091
source "${POSTGRES_LIB_DIR}/common.sh"
# shellcheck disable=SC1091
source "${POSTGRES_LIB_DIR}/docker.sh"
# shellcheck disable=SC1091
source "${POSTGRES_LIB_DIR}/instance.sh"
# shellcheck disable=SC1091
source "${POSTGRES_LIB_DIR}/status.sh"
# shellcheck disable=SC1091
source "${POSTGRES_LIB_DIR}/install.sh"
# shellcheck disable=SC1091
source "${POSTGRES_LIB_DIR}/database.sh"
# shellcheck disable=SC1091
source "${POSTGRES_LIB_DIR}/backup.sh"
# shellcheck disable=SC1091
source "${POSTGRES_LIB_DIR}/migration.sh"
# shellcheck disable=SC1091
source "${POSTGRES_LIB_DIR}/multi_instance.sh"
# shellcheck disable=SC1091
source "${POSTGRES_LIB_DIR}/network.sh"
# shellcheck disable=SC1091
source "${POSTGRES_LIB_DIR}/gui.sh"

#######################################
# Parse command line arguments
#######################################
postgres::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|create|destroy|start|stop|restart|status|list|logs|connect|diagnose|monitor|upgrade|backup|restore|migrate|migrate-init|migrate-status|migrate-rollback|migrate-list|seed|create-db|drop-db|create-user|drop-user|db-stats|list-backups|delete-backup|verify-backup|cleanup-backups|multi-start|multi-stop|multi-restart|multi-status|multi-migrate|multi-backup|multi-health|gui|gui-stop|gui-status|gui-list|network-update|network-migrate-all|inject|validate-injection" \
        --default "status"
    
    args::register \
        --name "instance" \
        --flag "i" \
        --desc "Instance name (use 'all' for operations on all instances)" \
        --type "value" \
        --default "main"
    
    args::register \
        --name "port" \
        --flag "p" \
        --desc "Port number for new instance (auto-assigned if not specified)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "gui-port" \
        --desc "Port number for GUI (auto-assigned if not specified)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "template" \
        --flag "t" \
        --desc "Configuration template (development, production, testing, minimal, real-estate, ecommerce, saas)" \
        --type "value" \
        --options "development|production|testing|minimal" \
        --default "development"
    
    args::register \
        --name "networks" \
        --desc "Additional Docker networks to join (comma-separated)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "format" \
        --desc "Output format for connection info (default, n8n, node-red, json)" \
        --type "value" \
        --options "default|n8n|node-red|json" \
        --default "default"
    
    
    args::register \
        --name "lines" \
        --flag "n" \
        --desc "Number of log lines to show" \
        --type "value" \
        --default "50"
    
    args::register \
        --name "interval" \
        --desc "Monitor interval in seconds" \
        --type "value" \
        --default "5"
    
    args::register \
        --name "remove-data" \
        --desc "Remove all data when uninstalling" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "backup-name" \
        --flag "b" \
        --desc "Name for backup (defaults to timestamp)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "backup-type" \
        --desc "Type of backup (full, schema, data)" \
        --type "value" \
        --options "full|schema|data" \
        --default "full"
    
    args::register \
        --name "database" \
        --flag "d" \
        --desc "Database name for operations" \
        --type "value" \
        --default ""
    
    args::register \
        --name "username" \
        --flag "u" \
        --desc "Username for user operations" \
        --type "value" \
        --default ""
    
    args::register \
        --name "password" \
        --desc "Password for user creation" \
        --type "value" \
        --default ""
    
    args::register \
        --name "migrations-dir" \
        --flag "m" \
        --desc "Directory containing migration files" \
        --type "value" \
        --default ""
    
    args::register \
        --name "seed-path" \
        --flag "s" \
        --desc "Path to seed file or directory" \
        --type "value" \
        --default ""
    
    args::register \
        --name "retention-days" \
        --desc "Backup retention period in days" \
        --type "value" \
        --default "$POSTGRES_BACKUP_RETENTION_DAYS"
    
    args::register \
        --name "injection-config" \
        --desc "JSON configuration for data injection" \
        --type "value" \
        --default ""
    
    if args::is_asking_for_help "$@"; then
        postgres::usage "$@"
        exit 0
    fi
    
    args::parse "$@"
    
    ACTION=$(args::get "action")
    INSTANCE_NAME=$(args::get "instance")
    PORT=$(args::get "port")
    GUI_PORT=$(args::get "gui-port")
    TEMPLATE=$(args::get "template")
    NETWORKS=$(args::get "networks")
    FORMAT=$(args::get "format")
    FORCE=$(args::get "yes")
    LOG_LINES=$(args::get "lines")
    MONITOR_INTERVAL=$(args::get "interval")
    REMOVE_DATA=$(args::get "remove-data")
    BACKUP_NAME=$(args::get "backup-name")
    BACKUP_TYPE=$(args::get "backup-type")
    DATABASE=$(args::get "database")
    USERNAME=$(args::get "username")
    PASSWORD=$(args::get "password")
    MIGRATIONS_DIR=$(args::get "migrations-dir")
    SEED_PATH=$(args::get "seed-path")
    RETENTION_DAYS=$(args::get "retention-days")
    INJECTION_CONFIG=$(args::get "injection-config")
    export ACTION INSTANCE_NAME PORT GUI_PORT TEMPLATE NETWORKS FORMAT FORCE LOG_LINES MONITOR_INTERVAL REMOVE_DATA BACKUP_NAME BACKUP_TYPE DATABASE USERNAME PASSWORD MIGRATIONS_DIR SEED_PATH RETENTION_DAYS INJECTION_CONFIG
}

#######################################
# Display usage information
#######################################
postgres::usage() {
    cat << EOF
PostgreSQL Storage Resource Manager

DESCRIPTION:
    $DESCRIPTION

USAGE:
    $0 [OPTIONS]

OPTIONS:
$(args::usage "$@")

ACTIONS:
    Resource Management:
        install         Install PostgreSQL resource (pulls Docker image, creates directories)
        uninstall       Uninstall PostgreSQL resource (destroys all instances)
        upgrade         Upgrade PostgreSQL Docker image
    
    Instance Management:
        create          Create a new PostgreSQL instance
        destroy         Destroy a PostgreSQL instance
        start           Start PostgreSQL instance(s)
        stop            Stop PostgreSQL instance(s)
        restart         Restart PostgreSQL instance(s)
        status          Show status of PostgreSQL resource or specific instance
        list            List all PostgreSQL instances with status
        logs            Show container logs for an instance
        connect         Get connection string for an instance
    
    Database Operations:
        create-db       Create a new database in an instance
        drop-db         Drop a database from an instance
        create-user     Create a new user in an instance
        drop-user       Drop a user from an instance
        migrate         Run schema migrations on an instance
        migrate-init    Initialize migration system for an instance
        migrate-status  Show migration status and history
        migrate-rollback Rollback specific migration
        migrate-list    List available migrations in directory
        seed            Seed database with initial data
        db-stats        Show database statistics
    
    Backup & Restore:
        backup          Create backup of an instance
        restore         Restore instance from backup
        list-backups    List available backups
        delete-backup   Delete a specific backup
        verify-backup   Verify backup integrity
        cleanup-backups Clean up old backups
    
    Multi-Instance Operations:
        multi-start     Start multiple instances
        multi-stop      Stop multiple instances
        multi-restart   Restart multiple instances
        multi-status    Show status of multiple instances
        multi-migrate   Run migrations on multiple instances
        multi-backup    Backup multiple instances
        multi-health    Health check multiple instances
    
    Monitoring & Diagnostics:
        diagnose        Run diagnostic checks
        monitor         Monitor instances continuously

EXAMPLES:
    # Install PostgreSQL resource
    $0 --action install

    # Create a new instance for a client
    $0 --action create --instance real-estate --template production

    # Create instance on specific port
    $0 --action create --instance testing --port 5440 --template testing

    # List all instances
    $0 --action list

    # Start all instances
    $0 --action start --instance all

    # Get connection details
    $0 --action connect --instance real-estate

    # Monitor instance health
    $0 --action monitor

    # Show logs for an instance
    $0 --action logs --instance real-estate --lines 100

    # Destroy instance (with confirmation)
    $0 --action destroy --instance testing

    # Force destroy without confirmation
    $0 --action destroy --instance testing --force yes

    # Database operations
    $0 --action create-db --instance real-estate --database myapp
    $0 --action migrate --instance real-estate --migrations-dir ./migrations
    $0 --action seed --instance real-estate --seed-path ./seeds/initial.sql
    
    # Backup and restore
    $0 --action backup --instance real-estate --backup-name production-snapshot
    $0 --action restore --instance real-estate --backup-name production-snapshot
    $0 --action list-backups --instance real-estate
    $0 --action cleanup-backups --retention-days 30
    
    # Enhanced migrations
    $0 --action migrate-init --instance real-estate
    $0 --action migrate-status --instance real-estate
    $0 --action migrate-rollback --instance real-estate --backup-name 20240130_001
    
    # Multi-instance operations
    $0 --action multi-start --instance all
    $0 --action multi-migrate --instance "client1,client2" --migrations-dir ./migrations
    $0 --action multi-status --instance all

TEMPLATES:
    development     Default template with verbose logging (default)
    production      Optimized for production workloads
    testing         Fast, ephemeral settings for testing
    minimal         Minimal resource usage
    real-estate     Optimized for real estate applications and lead management
    ecommerce       Configured for ecommerce with transaction workloads
    saas            Multi-tenant SaaS with row-level security and analytics

NOTES:
    - Instances are isolated PostgreSQL databases in separate Docker containers
    - Each instance gets its own port in the range $POSTGRES_INSTANCE_PORT_RANGE_START-$POSTGRES_INSTANCE_PORT_RANGE_END
    - Maximum $POSTGRES_MAX_INSTANCES instances can run simultaneously
    - Use instance names that describe your client or use case (e.g., 'real-estate', 'ecommerce')

EOF
}

#######################################
# Show container logs
# Arguments:
#   $1 - instance name
#   $2 - number of lines (optional)
#######################################
postgres::show_logs() {
    local instance_name="${1:-main}"
    local lines="${2:-50}"
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    
    if ! postgres::common::container_exists "$instance_name"; then
        log::error "${MSG_INSTANCE_NOT_FOUND}: $instance_name"
        return 1
    fi
    
    log::info "Showing last $lines lines from PostgreSQL instance '$instance_name':"
    log::info "================================"
    
    docker logs --tail "$lines" "$container_name" 2>&1 || {
        log::error "Failed to retrieve logs"
        return 1
    }
}

#######################################
# Get connection info with retry logic
#######################################
postgres::get_connection_info_with_retry() {
    local instance_name="$1"
    local max_attempts=3
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        # Try to get all required config values
        local port
        local password
        local user
        local database
        local networks
        port=$(postgres::common::get_instance_config "$instance_name" "port" 2>/dev/null)
        password=$(postgres::common::get_instance_config "$instance_name" "password" 2>/dev/null)
        user=$(postgres::common::get_instance_config "$instance_name" "user" 2>/dev/null)
        database=$(postgres::common::get_instance_config "$instance_name" "database" 2>/dev/null)
        networks=$(postgres::common::get_instance_config "$instance_name" "networks" 2>/dev/null)
        
        # Check if we got the essential values
        if [[ -n "$port" && -n "$password" ]]; then
            # Export values so the calling function can use them
            export CONN_PORT="$port"
            export CONN_PASSWORD="$password"
            export CONN_USER="${user:-vrooli}"
            export CONN_DATABASE="${database:-vrooli_client}"
            export CONN_NETWORKS="$networks"
            return 0
        fi
        
        log::debug "Connection attempt $attempt failed, retrying..."
        sleep 1
        ((attempt++))
    done
    
    log::error "Failed to get connection info after $max_attempts attempts"
    return 1
}

#######################################
# Show connection information
#######################################
postgres::show_connection() {
    local instance_name="${1:-main}"
    local format="${2:-default}"
    
    if ! postgres::common::container_exists "$instance_name"; then
        log::error "${MSG_INSTANCE_NOT_FOUND}: $instance_name"
        return 1
    fi
    
    # Use retry logic to get connection info
    if ! postgres::get_connection_info_with_retry "$instance_name"; then
        return 1
    fi
    
    # Use the exported values from the retry function
    local port="$CONN_PORT"
    local password="$CONN_PASSWORD"
    local user="$CONN_USER"
    local database="$CONN_DATABASE"
    local networks="$CONN_NETWORKS"
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    
    # Determine host based on network connectivity
    local host="localhost"
    if [[ -n "$networks" ]]; then
        # If connected to resource networks, use container name for internal access
        case "$format" in
            "n8n"|"node-red"|"json")
                host="$container_name"
                ;;
        esac
    fi
    
    case "$format" in
        "n8n")
            # n8n-friendly JSON format
            cat << EOF
{
  "credentials": {
    "postgres": {
      "host": "$host",
      "port": 5432,
      "database": "$database",
      "user": "$user",
      "password": "$password",
      "ssl": false
    }
  }
}

Note: Use port 5432 when connecting from within Docker network.
External connections should use port $port on localhost.
EOF
            ;;
        
        "node-red")
            # Node-RED configuration format
            cat << EOF
PostgreSQL Configuration for Node-RED:

Host: $host
Port: 5432 (internal) / $port (external)
Database: $database
Username: $user
Password: $password

For node-red-contrib-postgresql:
{
  "host": "$host",
  "port": 5432,
  "database": "$database"
}
EOF
            ;;
        
        "json")
            # Generic JSON format
            cat << EOF
{
  "host": "$host",
  "port_internal": 5432,
  "port_external": $port,
  "database": "$database",
  "username": "$user",
  "password": "$password",
  "connection_string": "postgresql://$user:$password@$host:5432/$database",
  "connection_string_external": "postgresql://$user:$password@localhost:$port/$database",
  "container_name": "$container_name",
  "networks": "$networks"
}
EOF
            ;;
        
        *)
            # Default human-readable format
            log::info "PostgreSQL Instance Connection Details: $instance_name"
            log::info "=================================================="
            log::info "Host: localhost"
            log::info "Port: $port"
            log::info "Database: $database"
            log::info "Username: $user"
            log::info "Password: $password"
            
            if [[ -n "$networks" ]]; then
                log::info ""
                log::info "Docker Networks: $networks"
                log::info "Internal hostname: $container_name"
            fi
            
            log::info ""
            local conn_string
            conn_string=$(postgres::instance::get_connection_string "$instance_name")
            if [[ $? -eq 0 && -n "$conn_string" ]]; then
                log::info "Connection String:"
                log::info "$conn_string"
            else
                log::warn "Unable to generate connection string for instance: $instance_name"
            fi
            ;;
    esac
    log::info ""
    
    if postgres::common::is_running "$instance_name"; then
        log::success "Instance is running and ready for connections"
    else
        log::warn "Instance is stopped - start it first with:"
        log::info "$0 --action start --instance $instance_name"
    fi
    
    log::info ""
    log::info "Example psql connection:"
    log::info "psql -h localhost -p $port -U $user -d $database"
}

#######################################
# Upgrade PostgreSQL Docker image
#######################################
postgres::upgrade() {
    log::info "Upgrading PostgreSQL Docker image..."
    
    # Pull latest image
    if ! postgres::docker::pull_image; then
        log::error "Failed to pull latest PostgreSQL image"
        return 1
    fi
    
    local instances
    mapfile -t instances < <(postgres::common::list_instances)
    if [[ ${#instances[@]} -eq 0 ]]; then
        log::success "PostgreSQL image upgraded (no instances to restart)"
        return 0
    fi
    
    log::info "Restarting instances to use new image..."
    local restarted=0
    local failed=0
    
    for instance in "${instances[@]}"; do
        if postgres::common::is_running "$instance"; then
            log::info "Restarting instance: $instance"
            if postgres::docker::restart "$instance"; then
                ((restarted++))
            else
                ((failed++))
            fi
        fi
    done
    
    if [[ $failed -eq 0 ]]; then
        log::success "PostgreSQL upgrade completed successfully"
        log::info "Restarted $restarted instance(s)"
    else
        log::error "Upgrade completed with issues: $failed instance(s) failed to restart"
        return 1
    fi
}

#######################################
# Main execution
#######################################
postgres::main() {
    postgres::parse_arguments "$@"
    
    case "$ACTION" in
        "install")
            postgres::install
            ;;
        "uninstall")
            postgres::uninstall
            ;;
        "create")
            postgres::instance::create "$INSTANCE_NAME" "$PORT" "$TEMPLATE" "$NETWORKS"
            ;;
        "destroy")
            postgres::instance::destroy "$INSTANCE_NAME" "$FORCE"
            ;;
        "start")
            postgres::instance::start "$INSTANCE_NAME"
            ;;
        "stop")
            postgres::instance::stop "$INSTANCE_NAME"
            ;;
        "restart")
            postgres::instance::restart "$INSTANCE_NAME"
            ;;
        "status")
            if [[ "$INSTANCE_NAME" == "main" ]] || [[ "$INSTANCE_NAME" == "all" ]]; then
                postgres::status::check true
            else
                postgres::status::show_instance "$INSTANCE_NAME"
            fi
            ;;
        "list")
            postgres::instance::list
            ;;
        "logs")
            postgres::show_logs "$INSTANCE_NAME" "$LOG_LINES"
            ;;
        "connect")
            postgres::show_connection "$INSTANCE_NAME" "$FORMAT"
            ;;
        "diagnose")
            postgres::status::diagnose
            ;;
        "monitor")
            postgres::status::monitor "$MONITOR_INTERVAL"
            ;;
        "upgrade")
            postgres::upgrade
            ;;
        "backup")
            postgres::backup::create "$INSTANCE_NAME" "$BACKUP_NAME" "$BACKUP_TYPE"
            ;;
        "restore")
            if [[ -z "$BACKUP_NAME" ]]; then
                log::error "Backup name is required for restore"
                log::info "Usage: $0 --action restore --instance <name> --backup-name <backup>"
                exit 1
            fi
            postgres::backup::restore "$INSTANCE_NAME" "$BACKUP_NAME" "$BACKUP_TYPE" "$FORCE"
            ;;
        "migrate")
            if [[ -z "$MIGRATIONS_DIR" ]]; then
                log::error "Migrations directory is required"
                log::info "Usage: $0 --action migrate --instance <name> --migrations-dir <path>"
                exit 1
            fi
            postgres::migration::run "$INSTANCE_NAME" "$MIGRATIONS_DIR" "$DATABASE"
            ;;
        "seed")
            if [[ -z "$SEED_PATH" ]]; then
                log::error "Seed path is required"
                log::info "Usage: $0 --action seed --instance <name> --seed-path <path>"
                exit 1
            fi
            postgres::database::seed "$INSTANCE_NAME" "$SEED_PATH" "$DATABASE"
            ;;
        "create-db")
            if [[ -z "$DATABASE" ]]; then
                log::error "Database name is required"
                log::info "Usage: $0 --action create-db --instance <name> --database <db_name>"
                exit 1
            fi
            postgres::database::create "$INSTANCE_NAME" "$DATABASE" "$USERNAME"
            ;;
        "drop-db")
            if [[ -z "$DATABASE" ]]; then
                log::error "Database name is required"
                log::info "Usage: $0 --action drop-db --instance <name> --database <db_name>"
                exit 1
            fi
            postgres::database::drop "$INSTANCE_NAME" "$DATABASE" "$FORCE"
            ;;
        "create-user")
            if [[ -z "$USERNAME" || -z "$PASSWORD" ]]; then
                log::error "Username and password are required"
                log::info "Usage: $0 --action create-user --instance <name> --username <user> --password <pass>"
                exit 1
            fi
            postgres::database::create_user "$INSTANCE_NAME" "$USERNAME" "$PASSWORD"
            ;;
        "drop-user")
            if [[ -z "$USERNAME" ]]; then
                log::error "Username is required"
                log::info "Usage: $0 --action drop-user --instance <name> --username <user>"
                exit 1
            fi
            postgres::database::drop_user "$INSTANCE_NAME" "$USERNAME" "$FORCE"
            ;;
        "db-stats")
            postgres::database::stats "$INSTANCE_NAME" "$DATABASE"
            ;;
        "list-backups")
            postgres::backup::list "$INSTANCE_NAME"
            ;;
        "delete-backup")
            if [[ -z "$BACKUP_NAME" ]]; then
                log::error "Backup name is required"
                log::info "Usage: $0 --action delete-backup --instance <name> --backup-name <backup>"
                exit 1
            fi
            postgres::backup::delete "$INSTANCE_NAME" "$BACKUP_NAME" "$FORCE"
            ;;
        "verify-backup")
            if [[ -z "$BACKUP_NAME" ]]; then
                log::error "Backup name is required"
                log::info "Usage: $0 --action verify-backup --instance <name> --backup-name <backup>"
                exit 1
            fi
            postgres::backup::verify "$INSTANCE_NAME" "$BACKUP_NAME"
            ;;
        "cleanup-backups")
            postgres::backup::cleanup "$INSTANCE_NAME" "$RETENTION_DAYS"
            ;;
        "migrate-init")
            postgres::migration::init "$INSTANCE_NAME" "$DATABASE"
            ;;
        "migrate-status")
            postgres::migration::status "$INSTANCE_NAME" "$DATABASE"
            ;;
        "migrate-rollback")
            if [[ -z "$BACKUP_NAME" ]]; then
                log::error "Migration version is required for rollback"
                log::info "Usage: $0 --action migrate-rollback --instance <name> --backup-name <version>"
                exit 1
            fi
            postgres::migration::rollback "$INSTANCE_NAME" "$BACKUP_NAME" "$DATABASE"
            ;;
        "migrate-list")
            if [[ -z "$MIGRATIONS_DIR" ]]; then
                log::error "Migrations directory is required"
                log::info "Usage: $0 --action migrate-list --migrations-dir <path>"
                exit 1
            fi
            postgres::migration::list_available "$MIGRATIONS_DIR"
            ;;
        "multi-start")
            local pattern="${INSTANCE_NAME}"
            if [[ "$pattern" == "main" ]]; then
                pattern="all"
            fi
            postgres::multi::start "$pattern"
            ;;
        "multi-stop")
            local pattern="${INSTANCE_NAME}"
            if [[ "$pattern" == "main" ]]; then
                pattern="all"
            fi
            postgres::multi::stop "$pattern"
            ;;
        "multi-restart")
            local pattern="${INSTANCE_NAME}"
            if [[ "$pattern" == "main" ]]; then
                pattern="all"
            fi
            postgres::multi::restart "$pattern"
            ;;
        "multi-status")
            local pattern="${INSTANCE_NAME}"
            if [[ "$pattern" == "main" ]]; then
                pattern="all"
            fi
            postgres::multi::status "$pattern"
            ;;
        "multi-migrate")
            if [[ -z "$MIGRATIONS_DIR" ]]; then
                log::error "Migrations directory is required"
                log::info "Usage: $0 --action multi-migrate --instance <pattern> --migrations-dir <path>"
                exit 1
            fi
            local pattern="${INSTANCE_NAME}"
            if [[ "$pattern" == "main" ]]; then
                pattern="all"
            fi
            postgres::multi::migrate "$pattern" "$MIGRATIONS_DIR" "$DATABASE"
            ;;
        "multi-backup")
            local pattern="${INSTANCE_NAME}"
            if [[ "$pattern" == "main" ]]; then
                pattern="all"
            fi
            postgres::multi::backup "$pattern" "$BACKUP_NAME" "$BACKUP_TYPE"
            ;;
        "multi-health")
            local pattern="${INSTANCE_NAME}"
            if [[ "$pattern" == "main" ]]; then
                pattern="all"
            fi
            postgres::multi::health_check "$pattern"
            ;;
        "gui")
            postgres::gui::start "$INSTANCE_NAME" "$GUI_PORT"
            ;;
        "gui-stop")
            postgres::gui::stop "$INSTANCE_NAME"
            ;;
        "gui-status")
            postgres::gui::show_status "$INSTANCE_NAME"
            ;;
        "gui-list")
            postgres::gui::list
            ;;
        "network-update")
            postgres::network::update_instance "$INSTANCE_NAME"
            ;;
        "network-migrate-all")
            postgres::network::migrate_all_instances
            ;;
        "inject")
            if [[ -z "$INJECTION_CONFIG" ]]; then
                log::error "Injection configuration required for inject action"
                log::info "Use: --injection-config 'JSON_CONFIG'"
                exit 1
            fi
            "${POSTGRES_SCRIPT_DIR}/inject.sh" --inject "$INJECTION_CONFIG"
            ;;
        "validate-injection")
            if [[ -z "$INJECTION_CONFIG" ]]; then
                log::error "Injection configuration required for validate-injection action"
                log::info "Use: --injection-config 'JSON_CONFIG'"
                exit 1
            fi
            "${POSTGRES_SCRIPT_DIR}/inject.sh" --validate "$INJECTION_CONFIG"
            ;;
        *)
            log::error "Unknown action: $ACTION"
            postgres::usage "$@"
            exit 1
            ;;
    esac
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    postgres::main "$@"
fi