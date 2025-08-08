#!/usr/bin/env bash
set -euo pipefail

# Redis Resource Management Script
# This script handles installation, configuration, and management of Redis

DESCRIPTION="Install and manage Redis in-memory data structure store using Docker"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# Source common resources
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../app/utils/args.sh"

# Source configuration
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/messages.sh"

# Export configuration
redis::export_config
redis::messages::init

# Source all library modules
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/docker.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/backup.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/install.sh"

#######################################
# Parse command line arguments
#######################################
redis::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|monitor|backup|restore|list-backups|delete-backup|cleanup-backups|flush|config|upgrade|create-client|destroy-client|list-clients|cli|benchmark|diagnose" \
        --default "status"
    
    args::register \
        --name "backup-name" \
        --flag "b" \
        --desc "Backup name for backup operations" \
        --type "value" \
        --default ""
    
    args::register \
        --name "client-id" \
        --flag "c" \
        --desc "Client ID for multi-tenant operations" \
        --type "value" \
        --default ""
    
    args::register \
        --name "database" \
        --flag "d" \
        --desc "Database number (0-15)" \
        --type "value" \
        --default "0"
    
    args::register \
        --name "remove-data" \
        --desc "Remove all data when uninstalling" \
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
        --name "days" \
        --desc "Number of days for cleanup operations" \
        --type "value" \
        --default "30"
    
    args::register \
        --name "memory" \
        --flag "m" \
        --desc "Max memory setting" \
        --type "value" \
        --default ""
    
    args::register \
        --name "persistence" \
        --flag "p" \
        --desc "Persistence mode" \
        --type "value" \
        --options "rdb|aof|both|none" \
        --default ""
    
    # Parse arguments
    args::parse "$@"
    
    # Export parsed arguments
    export ACTION=$(args::get "action")
    export BACKUP_NAME=$(args::get "backup-name")
    export CLIENT_ID=$(args::get "client-id")
    export DATABASE=$(args::get "database")
    export REMOVE_DATA=$(args::get "remove-data")
    export LOG_LINES=$(args::get "lines")
    export MONITOR_INTERVAL=$(args::get "interval")
    export CLEANUP_DAYS=$(args::get "days")
    export MEMORY_SETTING=$(args::get "memory")
    export PERSISTENCE_SETTING=$(args::get "persistence")
    export YES=$(args::get "yes")
}

#######################################
# Execute Redis CLI command
#######################################
redis::execute_cli() {
    shift  # Remove 'cli' action
    redis::docker::exec_cli "$@"
}

#######################################
# Flush Redis database(s)
#######################################
redis::flush_database() {
    local database="$DATABASE"
    
    if ! redis::common::is_running; then
        log::error "Redis is not running"
        return 1
    fi
    
    if [[ "$database" == "all" ]]; then
        log::warn "${MSG_DATABASE_FLUSH_ALL}"
        if [[ "$YES" != "yes" ]]; then
            echo -n "${MSG_DATABASE_FLUSH_CONFIRM}"
            read -r confirm
            if [[ "$confirm" != "yes" ]]; then
                log::info "Flush cancelled"
                return 0
            fi
        fi
        
        redis::docker::exec_cli FLUSHALL
        log::success "All databases flushed"
    else
        log::warn "${MSG_DATABASE_FLUSH}${database}"
        if [[ "$YES" != "yes" ]]; then
            echo -n "${MSG_DATABASE_FLUSH_CONFIRM}"
            read -r confirm
            if [[ "$confirm" != "yes" ]]; then
                log::info "Flush cancelled"
                return 0
            fi
        fi
        
        redis::docker::exec_cli -n "$database" FLUSHDB
        log::success "Database ${database} flushed"
    fi
}

#######################################
# Update Redis configuration
#######################################
redis::update_config() {
    local memory="$MEMORY_SETTING"
    local persistence="$PERSISTENCE_SETTING"
    local updated=false
    
    if [[ -n "$memory" ]]; then
        REDIS_MAX_MEMORY="$memory"
        export REDIS_MAX_MEMORY
        updated=true
        log::info "Updated max memory to: ${memory}"
    fi
    
    if [[ -n "$persistence" ]]; then
        REDIS_PERSISTENCE="$persistence"
        export REDIS_PERSISTENCE
        updated=true
        log::info "Updated persistence mode to: ${persistence}"
    fi
    
    if [[ "$updated" == true ]]; then
        log::info "${MSG_CONFIG_UPDATE}"
        
        # Regenerate configuration
        redis::common::generate_config
        
        # Restart Redis to apply changes
        redis::docker::restart
        
        log::success "${MSG_CONFIG_SUCCESS}"
    else
        log::info "Current configuration:"
        redis::status::show_config_summary
    fi
}

#######################################
# Run Redis benchmark
#######################################
redis::run_benchmark() {
    if ! redis::common::is_running; then
        log::error "Redis is not running"
        return 1
    fi
    
    log::info "${MSG_BENCHMARK_START}"
    
    # Run basic benchmark
    docker exec "${REDIS_CONTAINER_NAME}" redis-benchmark \
        -p "${REDIS_INTERNAL_PORT}" \
        -n 10000 \
        -c 50 \
        -t set,get,incr,lpush,rpush,lpop,rpop,sadd,hset,spop,lrange,mset \
        --csv
    
    log::success "${MSG_BENCHMARK_COMPLETE}"
}

#######################################
# Diagnose Redis issues
#######################################
redis::diagnose() {
    log::info "🔍 Redis Diagnostics"
    echo "===================="
    
    # Container status
    echo "🐳 Container Status:"
    if redis::common::container_exists; then
        echo "   Container exists: ✅"
        local status
        status=$(docker inspect --format='{{.State.Status}}' "${REDIS_CONTAINER_NAME}" 2>/dev/null)
        echo "   Container status: ${status}"
        
        if redis::common::is_running; then
            echo "   Container running: ✅"
            
            if redis::common::is_healthy; then
                echo "   Health check: ✅"
            else
                echo "   Health check: ❌"
                echo "   Health details:"
                docker inspect --format='{{json .State.Health}}' "${REDIS_CONTAINER_NAME}" | jq .
            fi
        else
            echo "   Container running: ❌"
        fi
    else
        echo "   Container exists: ❌"
    fi
    
    echo
    echo "🌐 Network Status:"
    echo "   Port ${REDIS_PORT} available: $(redis::common::is_port_available "$REDIS_PORT" && echo "✅" || echo "❌")"
    
    if netstat -tuln 2>/dev/null | grep -q ":${REDIS_PORT} "; then
        echo "   Port ${REDIS_PORT} in use by:"
        netstat -tuln | grep ":${REDIS_PORT} "
    fi
    
    echo
    echo "📁 File System:"
    echo "   Data directory: ${REDIS_DATA_DIR}"
    echo "   Data dir exists: $(test -d "${REDIS_DATA_DIR}" && echo "✅" || echo "❌")"
    echo "   Data dir writable: $(test -w "${REDIS_DATA_DIR}" && echo "✅" || echo "❌")"
    
    if [[ -d "${REDIS_DATA_DIR}" ]]; then
        local data_size
        data_size=$(du -sh "${REDIS_DATA_DIR}" | cut -f1)
        echo "   Data size: ${data_size}"
    fi
    
    echo
    echo "📊 Redis Metrics (if running):"
    if redis::common::is_running && redis::common::ping; then
        local info
        info=$(redis::common::get_info)
        
        local memory_used
        memory_used=$(echo "$info" | grep "used_memory_human:" | cut -d: -f2 | tr -d '\r' | xargs)
        echo "   Memory used: ${memory_used:-N/A}"
        
        local clients
        clients=$(echo "$info" | grep "connected_clients:" | cut -d: -f2 | tr -d '\r' | xargs)
        echo "   Connected clients: ${clients:-N/A}"
        
        local commands
        commands=$(echo "$info" | grep "total_commands_processed:" | cut -d: -f2 | tr -d '\r' | xargs)
        echo "   Commands processed: ${commands:-N/A}"
    else
        echo "   Redis not responding to ping"
    fi
    
    echo
    echo "🔧 Configuration:"
    echo "   Image: ${REDIS_IMAGE}"
    echo "   Port: ${REDIS_PORT}"
    echo "   Max memory: ${REDIS_MAX_MEMORY}"
    echo "   Persistence: ${REDIS_PERSISTENCE}"
    echo "   Databases: ${REDIS_DATABASES}"
    
    echo
    echo "📝 Recent Logs:"
    if redis::common::container_exists; then
        docker logs --tail 10 "${REDIS_CONTAINER_NAME}" 2>/dev/null || echo "   No logs available"
    else
        echo "   Container not found"
    fi
}

#######################################
# Main function
#######################################
redis::main() {
    redis::parse_arguments "$@"
    
    local action="$ACTION"
    
    case "$action" in
        "install")
            redis::install::main
            ;;
        "uninstall")
            redis::install::uninstall "$REMOVE_DATA"
            ;;
        "start")
            redis::docker::start
            ;;
        "stop")
            redis::docker::stop
            ;;
        "restart")
            redis::docker::restart
            ;;
        "status")
            redis::status::show
            ;;
        "logs")
            redis::docker::show_logs "$LOG_LINES"
            ;;
        "monitor")
            redis::status::monitor "$MONITOR_INTERVAL"
            ;;
        "backup")
            redis::backup::create "$BACKUP_NAME"
            ;;
        "restore")
            redis::backup::restore "$BACKUP_NAME"
            ;;
        "list-backups")
            redis::backup::list
            ;;
        "delete-backup")
            redis::backup::delete "$BACKUP_NAME"
            ;;
        "cleanup-backups")
            redis::backup::cleanup "$CLEANUP_DAYS"
            ;;
        "flush")
            redis::flush_database
            ;;
        "config")
            redis::update_config
            ;;
        "upgrade")
            redis::install::upgrade
            ;;
        "create-client")
            redis::docker::create_client_instance "$CLIENT_ID"
            ;;
        "destroy-client")
            redis::docker::destroy_client_instance "$CLIENT_ID"
            ;;
        "list-clients")
            redis::status::list_clients
            ;;
        "cli")
            redis::execute_cli "$@"
            ;;
        "benchmark")
            redis::run_benchmark
            ;;
        "diagnose")
            redis::diagnose
            ;;
        *)
            log::error "${MSG_ERROR_INVALID_ACTION}${action}"
            args::help
            exit 1
            ;;
    esac
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    redis::main "$@"
fi