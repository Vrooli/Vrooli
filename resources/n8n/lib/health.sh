#!/usr/bin/env bash
# n8n Health Check Functions - Minimal wrapper using health framework

# Source guard to prevent multiple sourcing
[[ -n "${_N8N_HEALTH_SOURCED:-}" ]] && return 0
export _N8N_HEALTH_SOURCED=1

# Source core and frameworks
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
N8N_LIB_DIR="${APP_ROOT}/resources/n8n/lib"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/health-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/core.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/memory-monitor.sh" 2>/dev/null || true

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
# Enhanced memory health check with prediction
# Returns: 0 if healthy, 1 if warning, 2 if critical
#######################################
n8n::check_memory_health() {
    if ! docker::is_running "$N8N_CONTAINER_NAME"; then
        echo "Container not running"
        return 0
    fi
    
    # Use memory monitor if available
    if declare -f n8n::memory::check_and_alert >/dev/null 2>&1; then
        n8n::memory::check_and_alert
        return $?
    fi
    
    # Fallback basic check
    local mem_usage
    mem_usage=$(docker stats "$N8N_CONTAINER_NAME" --no-stream --format "{{.MemPerc}}" 2>/dev/null | sed 's/%//')
    
    if [[ -z "$mem_usage" ]]; then
        return 0
    fi
    
    if (( $(echo "$mem_usage >= 80" | bc -l) )); then
        log::warn "n8n memory usage at ${mem_usage}%"
        return 2
    elif (( $(echo "$mem_usage >= 70" | bc -l) )); then
        log::warn "n8n memory usage at ${mem_usage}%"
        return 1
    fi
    
    return 0
}

#######################################
# Start memory monitoring if not already running
# Returns: 0 if started or already running
#######################################
n8n::start_memory_monitoring() {
    if ! declare -f n8n::memory::monitor_loop >/dev/null 2>&1; then
        log::warn "Memory monitoring not available"
        return 1
    fi
    
    # Check if already running
    if [[ -f /tmp/n8n-memory-monitor.pid ]]; then
        local pid
        pid=$(cat /tmp/n8n-memory-monitor.pid)
        if kill -0 "$pid" 2>/dev/null; then
            log::info "Memory monitoring already running (PID: $pid)"
            return 0
        fi
    fi
    
    # Start monitoring
    n8n::memory::monitor_loop true
    return $?
}


