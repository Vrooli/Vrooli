#!/usr/bin/env bash
# n8n Health Check Functions - Minimal wrapper using health framework

# Source core and frameworks
N8N_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../lib/health-framework.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../lib/docker-utils.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/core.sh"

#######################################
# Tiered health check (wrapper for framework)
# Returns: Health tier status string
#######################################
n8n::tiered_health_check() {
    local config
    config=$(n8n::get_health_config)
    health::tiered_check "$config"
}

#######################################
# Check if n8n is healthy (backward compatibility)
# Returns: 0 if healthy, 1 otherwise
#######################################
n8n::is_healthy() {
    n8n::check_basic_health
}

#######################################
# Check filesystem corruption
# Returns: 0 if healthy, 1 if corrupted
#######################################
n8n::detect_filesystem_corruption() {
    health::check_filesystem "$N8N_DATA_DIR"
}

#######################################
# Check database health
# Returns: 0 if healthy, 1 if unhealthy
#######################################
n8n::check_database_health() {
    if [[ "$DATABASE_TYPE" == "sqlite" ]]; then
        local db_path="${N8N_DATA_DIR}/database.sqlite"
        if [[ -f "$db_path" ]]; then
            health::check_sqlite_integrity "$db_path"
        else
            return 0  # No database yet is fine
        fi
    else
        # PostgreSQL health check
        if docker::is_running "$N8N_DB_CONTAINER_NAME"; then
            docker exec "$N8N_DB_CONTAINER_NAME" pg_isready -U n8n >/dev/null 2>&1
        else
            return 1
        fi
    fi
}

#######################################
# Comprehensive health check
# Returns: 0 if all healthy, 1 if any issues
#######################################
n8n::comprehensive_health_check() {
    local tier
    tier=$(n8n::tiered_health_check)
    
    case "$tier" in
        "HEALTHY")
            return 0
            ;;
        "DEGRADED"|"UNHEALTHY")
            return 1
            ;;
        *)
            return 1
            ;;
    esac
}

#######################################
# PostgreSQL health check
# Returns: 0 if healthy, 1 otherwise
#######################################
n8n::postgres_is_healthy() {
    [[ "$DATABASE_TYPE" == "postgres" ]] && docker::is_running "$N8N_DB_CONTAINER_NAME"
}

#######################################
# Check if PostgreSQL exists
# Returns: 0 if exists, 1 otherwise
#######################################
n8n::postgres_exists() {
    [[ "$DATABASE_TYPE" == "postgres" ]] && docker::container_exists "$N8N_DB_CONTAINER_NAME"
}

#######################################
# Check if PostgreSQL is running
# Returns: 0 if running, 1 otherwise
#######################################
n8n::postgres_running() {
    [[ "$DATABASE_TYPE" == "postgres" ]] && docker::is_running "$N8N_DB_CONTAINER_NAME"
}

#######################################
# Check if n8n is installed
# Returns: 0 if installed, 1 otherwise
#######################################
n8n::is_installed() {
    docker::container_exists "$N8N_CONTAINER_NAME"
}

#######################################
# Check if port is available
# Returns: 0 if available, 1 if in use
#######################################
n8n::is_port_available() {
    docker::is_port_available "$1"
}

#######################################
# Validate API key setup
# Returns: 0 if valid, 1 otherwise
#######################################
n8n::validate_api_key_setup() {
    n8n::check_api_functionality
}