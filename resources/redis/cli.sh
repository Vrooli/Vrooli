#!/usr/bin/env bash
################################################################################
# Redis Resource CLI
# 
# Ultra-thin CLI wrapper that delegates directly to library functions
#
# Usage:
#   resource-redis <command> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    REDIS_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${REDIS_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
REDIS_CLI_DIR="${APP_ROOT}/resources/redis"

# Source standard variables
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source Redis configuration
# shellcheck disable=SC1091
source "${REDIS_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
redis::export_config 2>/dev/null || true

# Source Redis libraries - these contain the actual functionality
# shellcheck disable=SC1091
source "${REDIS_CLI_DIR}/lib/common.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${REDIS_CLI_DIR}/lib/docker.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${REDIS_CLI_DIR}/lib/status.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${REDIS_CLI_DIR}/lib/backup.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${REDIS_CLI_DIR}/lib/install.sh" 2>/dev/null || true

# Initialize CLI framework
cli::init "redis" "Redis cache and data structure server management"

# Override help to provide Redis-specific examples
cli::register_command "help" "Show this help message with examples" "redis::show_help"

# Register core commands - direct library function calls
cli::register_command "install" "Install Redis" "redis::install::main" "modifies-system"
cli::register_command "uninstall" "Uninstall Redis" "redis::cli_uninstall" "modifies-system"
cli::register_command "upgrade" "Upgrade Redis" "redis::install::upgrade" "modifies-system"
cli::register_command "start" "Start Redis" "redis::docker::start" "modifies-system"
cli::register_command "stop" "Stop Redis" "redis::docker::stop" "modifies-system"
cli::register_command "restart" "Restart Redis" "redis::cli_restart" "modifies-system"

# Register status and monitoring commands
cli::register_command "status" "Show service status" "redis::status::show"
cli::register_command "validate" "Validate installation" "redis::common::is_healthy"
cli::register_command "monitor" "Monitor Redis in real-time" "redis::cli_monitor"
cli::register_command "logs" "Show Redis logs" "redis::cli_logs"

# Register Redis commands
cli::register_command "inject" "Execute Redis command" "redis::cli_inject" "modifies-system"
cli::register_command "ping" "Test Redis connection" "redis::common::ping"
cli::register_command "flush" "Flush all Redis data (requires --force)" "redis::cli_flush" "modifies-system"

# Register backup commands
cli::register_command "backup" "Create Redis backup" "redis::cli_backup" "modifies-system"
cli::register_command "restore" "Restore Redis backup" "redis::cli_restore" "modifies-system"
cli::register_command "list-backups" "List available backups" "redis::backup::list"

# Register utility commands
cli::register_command "info" "Show Redis info" "redis::common::get_info"
cli::register_command "credentials" "Get n8n credentials" "redis::show_credentials"
cli::register_command "test" "Run integration tests" "redis::cli_test"

################################################################################
# CLI wrapper functions - minimal wrappers for commands that need argument handling
################################################################################

# Uninstall with force confirmation
redis::cli_uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Redis and all its data. Use --force to confirm."
        return 1
    fi
    
    redis::install::uninstall true
}

# Restart Redis
redis::cli_restart() {
    echo "Restarting Redis..."
    
    if redis::docker::stop; then
        sleep 2
        if redis::docker::start; then
            echo "✅ Redis restarted successfully"
            return 0
        fi
    fi
    
    return 1
}

# Monitor with interval
redis::cli_monitor() {
    local interval="${1:-5}"
    redis::status::monitor "$interval"
}

# Show logs with line count
redis::cli_logs() {
    local lines="${1:-50}"
    local follow="${2:-false}"
    local container_name="${REDIS_CONTAINER_NAME:-redis}"
    
    # Use shared utility with follow support
    docker_resource::show_logs_with_follow "$container_name" "$lines" "$follow"
}

# Inject Redis command
redis::cli_inject() {
    local command="${1:-}"
    
    if [[ -z "$command" ]]; then
        log::error "Redis command required for injection"
        echo "Usage: resource-redis inject '<redis-command>'"
        echo "Example: resource-redis inject 'SET mykey myvalue'"
        return 1
    fi
    
    # Use existing Redis CLI execution function
    if command -v redis::docker::exec_cli &>/dev/null; then
        redis::docker::exec_cli "$command"
    else
        # Fallback to direct docker exec
        if [[ -n "${REDIS_PASSWORD:-}" ]]; then
            # shellcheck disable=SC2086
            docker exec "${REDIS_CONTAINER_NAME:-redis}" redis-cli -a "$REDIS_PASSWORD" $command
        else
            # shellcheck disable=SC2086
            docker exec "${REDIS_CONTAINER_NAME:-redis}" redis-cli $command
        fi
    fi
}

# Flush with force confirmation
redis::cli_flush() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will delete ALL Redis data. Use --force to confirm."
        return 1
    fi
    
    log::warning "Flushing all Redis data..."
    redis::cli_inject "FLUSHALL"
}

# Run integration tests
redis::cli_test() {
    local test_script="${REDIS_CLI_DIR}/test/integration-test.sh"
    
    if [[ ! -f "$test_script" ]]; then
        log::error "Test script not found: $test_script"
        return 1
    fi
    
    # Export required variables for the test
    export REDIS_PORT="${REDIS_PORT:-6380}"
    export REDIS_CONTAINER_NAME="${REDIS_CONTAINER_NAME:-vrooli-redis-resource}"
    export REDIS_DATA_DIR="${REDIS_DATA_DIR:-${var_DATA_DIR}/redis}"
    export REDIS_MAX_MEMORY="${REDIS_MAX_MEMORY:-2gb}"
    export REDIS_DATABASES="${REDIS_DATABASES:-16}"
    
    # Run the test
    echo "Running Redis integration tests..."
    if bash "$test_script"; then
        echo "✅ All tests passed"
        return 0
    else
        echo "❌ Some tests failed"
        return 1
    fi
}

# Create backup with name
redis::cli_backup() {
    local backup_name="${1:-}"
    redis::backup::create "$backup_name"
}

# Restore backup with name validation
redis::cli_restore() {
    local backup_name="${1:-}"
    
    if [[ -z "$backup_name" ]]; then
        log::error "Backup name required for restore"
        return 1
    fi
    
    redis::backup::restore "$backup_name"
}

# Show credentials for n8n integration
redis::show_credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    credentials::parse_args "$@" || return $?
    
    # Get resource status
    local status
    status=$(credentials::get_resource_status "${REDIS_CONTAINER_NAME:-redis}")
    
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
                --argjson port "${REDIS_PORT:-6380}" \
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
    
    credentials::format_output "$response"
}

# Custom help function with examples
redis::show_help() {
    cli::_handle_help
    
    echo ""
    echo "⚡ Examples:"
    echo ""
    echo "  # Redis commands"
    echo "  resource-redis inject 'SET mykey myvalue'"
    echo "  resource-redis inject 'GET mykey'"
    echo "  resource-redis inject 'KEYS *'"
    echo "  resource-redis ping"
    echo ""
    echo "  # Data management"
    echo "  resource-redis backup my-backup"
    echo "  resource-redis restore my-backup"
    echo "  resource-redis list-backups"
    echo ""
    echo "  # Management"
    echo "  resource-redis status"
    echo "  resource-redis monitor"
    echo "  resource-redis logs 100"
    echo ""
    echo "  # Dangerous operations"
    echo "  resource-redis flush --force"
    echo "  resource-redis uninstall --force"
    echo ""
    echo "Default Port: ${REDIS_PORT:-6380}"
    echo "Connection: redis://localhost:${REDIS_PORT:-6380}"
    echo "Databases: 0-$((${REDIS_DATABASES:-16} - 1))"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi
