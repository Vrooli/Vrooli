#!/usr/bin/env bash
################################################################################
# Redis Resource CLI
# 
# Lightweight CLI interface for Redis that delegates to existing lib functions.
#
# Usage:
#   resource-redis <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory
REDIS_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$REDIS_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$REDIS_CLI_DIR"
export REDIS_SCRIPT_DIR="$REDIS_CLI_DIR"  # For compatibility with existing libs

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}" 2>/dev/null || true

# Source the CLI template
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/resources/cli/resource-cli-template.sh"

# Source Redis configuration
# shellcheck disable=SC1091
source "${REDIS_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
redis::export_config 2>/dev/null || true

# Source Redis libraries
for lib in common docker status backup install; do
    lib_file="${REDIS_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize with resource name
resource_cli::init "redis"

################################################################################
# Delegate to existing Redis functions
################################################################################

# Validate Redis configuration
resource_cli::validate() {
    if command -v redis::common::ping &>/dev/null; then
        redis::common::ping
    elif command -v redis::common::is_healthy &>/dev/null; then
        redis::common::is_healthy
    else
        # Basic validation
        log::header "Validating Redis"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "redis" || {
            log::error "Redis container not running"
            return 1
        }
        log::success "Redis is running"
    fi
}

# Show Redis status
resource_cli::status() {
    if command -v redis::status::show &>/dev/null; then
        redis::status::show
    else
        # Basic status
        log::header "Redis Status"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "redis"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=redis" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start Redis
resource_cli::start() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start Redis"
        return 0
    fi
    
    if command -v redis::docker::start &>/dev/null; then
        redis::docker::start
    else
        docker start redis || log::error "Failed to start Redis"
    fi
}

# Stop Redis
resource_cli::stop() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would stop Redis"
        return 0
    fi
    
    if command -v redis::docker::stop &>/dev/null; then
        redis::docker::stop
    else
        docker stop redis || log::error "Failed to stop Redis"
    fi
}

# Install Redis
resource_cli::install() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would install Redis"
        return 0
    fi
    
    if command -v redis::install::main &>/dev/null; then
        redis::install::main
    else
        log::error "redis::install::main not available"
        return 1
    fi
}

# Uninstall Redis
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Redis and all its data. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would uninstall Redis"
        return 0
    fi
    
    if command -v redis::install::uninstall &>/dev/null; then
        redis::install::uninstall true
    else
        docker stop redis 2>/dev/null || true
        docker rm redis 2>/dev/null || true
        log::success "Redis uninstalled"
    fi
}

# Inject data into Redis (using redis-cli exec)
resource_cli::inject() {
    local command="${1:-}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ -z "$command" ]]; then
        log::error "Redis command required for injection"
        echo "Usage: resource-redis inject '<redis-command>'"
        echo "Example: resource-redis inject 'SET mykey myvalue'"
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would execute: $command"
        return 0
    fi
    
    # Use existing Redis CLI execution function
    if command -v redis::docker::exec_cli &>/dev/null; then
        redis::docker::exec_cli "$command"
    else
        # Fallback to direct docker exec
        if [[ -n "${REDIS_PASSWORD:-}" ]]; then
            docker exec redis redis-cli -a "$REDIS_PASSWORD" $command
        else
            docker exec redis redis-cli $command
        fi
    fi
}

################################################################################
# Redis-specific commands (if functions exist)
################################################################################

# Monitor Redis in real-time
redis_monitor() {
    local interval="${1:-5}"
    
    if command -v redis::status::monitor &>/dev/null; then
        redis::status::monitor "$interval"
    else
        log::error "Redis monitoring not available"
        return 1
    fi
}

# Create backup
redis_backup() {
    local backup_name="${1:-}"
    
    if command -v redis::backup::create &>/dev/null; then
        redis::backup::create "$backup_name"
    else
        log::error "Redis backup not available"
        return 1
    fi
}

# Restore backup
redis_restore() {
    local backup_name="${1:-}"
    
    if [[ -z "$backup_name" ]]; then
        log::error "Backup name required for restore"
        return 1
    fi
    
    if command -v redis::backup::restore &>/dev/null; then
        redis::backup::restore "$backup_name"
    else
        log::error "Redis restore not available"
        return 1
    fi
}

# List backups
redis_list_backups() {
    if command -v redis::backup::list &>/dev/null; then
        redis::backup::list
    else
        log::error "Redis backup listing not available"
        return 1
    fi
}

# Show logs
redis_logs() {
    local lines="${1:-50}"
    
    if command -v redis::docker::show_logs &>/dev/null; then
        redis::docker::show_logs "$lines"
    else
        docker logs --tail "$lines" redis
    fi
}

# Flush all data
redis_flush() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will delete ALL Redis data. Use --force to confirm."
        return 1
    fi
    
    log::warning "Flushing all Redis data..."
    resource_cli::inject "FLUSHALL"
}

# Get credentials for n8n integration
resource_cli::credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    # Parse arguments
    credentials::parse_args "$@"
    local parse_result=$?
    if [[ $parse_result -eq 2 ]]; then
        credentials::show_help "redis"
        return 0
    elif [[ $parse_result -ne 0 ]]; then
        return 1
    fi
    
    # Get resource status
    local status
    status=$(credentials::get_resource_status "$REDIS_CONTAINER_NAME")
    
    # Build connections array for Redis databases (0-15)
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        local connections=()
        
        # Add connection for each Redis database (default is 16 databases: 0-15)
        local db_count="${REDIS_DATABASES:-16}"
        for ((db=0; db<db_count; db++)); do
            local connection_obj
            connection_obj=$(jq -n \
                --arg host "localhost" \
                --argjson port "${REDIS_PORT}" \
                --argjson database "$db" \
                '{
                    host: $host,
                    port: $port,
                    database: $database,
                    ssl: false
                }')
            
            local auth_obj="{}"
            if [[ -n "${REDIS_PASSWORD:-}" ]]; then
                auth_obj=$(jq -n --arg password "$REDIS_PASSWORD" '{password: $password}')
            fi
            
            local metadata_obj
            metadata_obj=$(jq -n \
                --arg description "Redis database $db" \
                --argjson capabilities '["cache", "pub_sub", "key_value"]' \
                '{
                    description: $description,
                    capabilities: $capabilities
                }')
            
            local connection
            connection=$(credentials::build_connection \
                "db-$db" \
                "Redis Database $db" \
                "redis" \
                "$connection_obj" \
                "$auth_obj" \
                "$metadata_obj")
            
            connections+=("$connection")
        done
        
        # Convert connections array to JSON
        connections_array=$(printf '%s\n' "${connections[@]}" | jq -s '.')
    fi
    
    # Build and validate response
    local response
    response=$(credentials::build_response "redis" "$status" "$connections_array")
    
    if credentials::validate_json "$response"; then
        credentials::format_output "$response"
    else
        log::error "Invalid credentials JSON generated"
        return 1
    fi
}

# Show help
resource_cli::show_help() {
    cat << EOF
🚀 Redis Resource CLI

USAGE:
    resource-redis <command> [options]

CORE COMMANDS:
    inject <command>        Execute Redis command (e.g., 'SET key value')
    validate                Validate Redis configuration
    status                  Show Redis status
    start                   Start Redis container
    stop                    Stop Redis container
    install                 Install Redis
    uninstall               Uninstall Redis (requires --force)
    credentials             Get connection credentials for n8n integration
    
REDIS COMMANDS:
    monitor [interval]      Monitor Redis in real-time (default: 5s)
    backup [name]           Create Redis backup
    restore <name>          Restore Redis backup
    list-backups            List available backups
    logs [lines]            Show Redis logs (default: 50 lines)
    flush                   Flush all Redis data (requires --force)

OPTIONS:
    --verbose, -v           Show detailed output
    --dry-run               Preview actions without executing
    --force                 Force operation (skip confirmations)

EXAMPLES:
    resource-redis status
    resource-redis inject "SET mykey 'hello world'"
    resource-redis inject "GET mykey"
    resource-redis credentials --format pretty
    resource-redis monitor 10
    resource-redis backup daily-backup
    resource-redis flush --force

For more information: https://docs.vrooli.com/resources/redis
EOF
}

# Main command router
resource_cli::main() {
    # Parse common options first
    local remaining_args
    remaining_args=$(resource_cli::parse_options "$@")
    set -- $remaining_args
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        # Standard resource commands
        inject|validate|status|start|stop|install|uninstall|credentials)
            resource_cli::$command "$@"
            ;;
            
        # Redis-specific commands
        monitor)
            redis_monitor "$@"
            ;;
        backup)
            redis_backup "$@"
            ;;
        restore)
            redis_restore "$@"
            ;;
        list-backups)
            redis_list_backups "$@"
            ;;
        logs)
            redis_logs "$@"
            ;;
        flush)
            redis_flush "$@"
            ;;
            
        help|--help|-h)
            resource_cli::show_help
            ;;
        *)
            log::error "Unknown command: $command"
            echo ""
            resource_cli::show_help
            exit 1
            ;;
    esac
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    resource_cli::main "$@"
fi