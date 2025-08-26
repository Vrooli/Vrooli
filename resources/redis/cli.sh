#!/usr/bin/env bash
################################################################################
# Redis Resource CLI - v2.0 Universal Contract Compliant
# 
# High-performance key-value store and caching solution
#
# Usage:
#   resource-redis <command> [options]
#   resource-redis <group> <subcommand> [options]
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

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${REDIS_CLI_DIR}/config/defaults.sh"

# Source Redis libraries
for lib in common docker install status backup; do
    lib_file="${REDIS_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "redis" "Redis cache and data structure server management" "v2"

# Override default handlers to point directly to Redis implementations
CLI_COMMAND_HANDLERS["manage::install"]="redis::install::main"
CLI_COMMAND_HANDLERS["manage::uninstall"]="redis::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="redis::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="redis::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="redis::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="redis::common::is_healthy"

# Override content handlers for Redis-specific key-value operations
CLI_COMMAND_HANDLERS["content::add"]="redis::content::set"
CLI_COMMAND_HANDLERS["content::list"]="redis::content::keys" 
CLI_COMMAND_HANDLERS["content::get"]="redis::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="redis::content::delete"
CLI_COMMAND_HANDLERS["content::execute"]="redis::content::execute"

# Add Redis-specific content subcommands for backup/restore operations
cli::register_subcommand "content" "backup" "Create Redis data backup" "redis::backup::create"
cli::register_subcommand "content" "restore" "Restore Redis data from backup" "redis::backup::restore"
cli::register_subcommand "content" "list-backups" "List available backups" "redis::backup::list"
cli::register_subcommand "content" "flush" "Flush all Redis data" "redis::content::flush"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "redis::status::show"
cli::register_command "logs" "Show Redis logs" "redis::docker::logs"
cli::register_command "ping" "Test Redis connectivity" "redis::common::ping"

# Content operation functions - implement missing handlers for key-value operations
redis::content::set() {
    local key="${1:-}"
    local value="${2:-}"
    
    if [[ -z "$key" || -z "$value" ]]; then
        log::error "Usage: content add <key> <value>"
        return 1
    fi
    
    redis::docker::exec_cli "SET '$key' '$value'"
}

redis::content::keys() {
    local pattern="${1:-*}"
    redis::docker::exec_cli "KEYS '$pattern'"
}

redis::content::get() {
    local key="${1:-}"
    
    if [[ -z "$key" ]]; then
        log::error "Usage: content get <key>"
        return 1
    fi
    
    redis::docker::exec_cli "GET '$key'"
}

redis::content::delete() {
    local key="${1:-}"
    
    if [[ -z "$key" ]]; then
        log::error "Usage: content remove <key>"
        return 1
    fi
    
    redis::docker::exec_cli "DEL '$key'"
}

redis::content::execute() {
    local command="${1:-}"
    
    if [[ -z "$command" ]]; then
        log::error "Usage: content execute '<redis-command>'"
        return 1
    fi
    
    redis::docker::exec_cli "$command"
}

redis::content::flush() {
    local force="${1:-}"
    
    if [[ "$force" != "--force" ]]; then
        log::error "This will delete ALL Redis data. Use --force to confirm."
        return 1
    fi
    
    log::warning "Flushing all Redis data..."
    redis::docker::exec_cli "FLUSHALL"
}

# Docker logs function if not exists in lib/docker.sh
redis::docker::logs() {
    local lines="${1:-50}"
    local container_name="${REDIS_CONTAINER_NAME:-redis}"
    
    if ! docker ps --format '{{.Names}}' | grep -q "^${container_name}$"; then
        log::error "Redis container '${container_name}' is not running"
        return 1
    fi
    
    docker logs --tail="$lines" "$container_name"
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi