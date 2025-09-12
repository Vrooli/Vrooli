#!/usr/bin/env bash
################################################################################
# Huginn Core Library - v2.0 Universal Contract Compliant
# 
# Core functionality for Huginn resource management
################################################################################

set -euo pipefail

# Export configuration for use by other scripts
huginn::export_config() {
    export HUGINN_PORT="${HUGINN_PORT:-4111}"
    export CONTAINER_NAME="${CONTAINER_NAME:-huginn}"
    export HUGINN_BASE_URL="${HUGINN_BASE_URL:-http://localhost:${HUGINN_PORT}}"
    export DEFAULT_ADMIN_USERNAME="${DEFAULT_ADMIN_USERNAME:-admin}"
    export DEFAULT_ADMIN_PASSWORD="${DEFAULT_ADMIN_PASSWORD:-changeme}"
}

# Initialize Huginn environment
huginn::init() {
    huginn::export_config
    
    # Ensure required directories exist
    local data_dir="${APP_ROOT}/data/huginn"
    [[ ! -d "${data_dir}" ]] && mkdir -p "${data_dir}"
    
    # Check Docker availability
    if ! command -v docker &>/dev/null; then
        log::error "Docker is required but not installed"
        return 1
    fi
}

# Get Huginn container status
huginn::get_status() {
    local container_name="${1:-${CONTAINER_NAME}}"
    
    if docker ps --filter "name=${container_name}" --format "{{.Status}}" 2>/dev/null | grep -q "Up"; then
        echo "running"
    elif docker ps -a --filter "name=${container_name}" --format "{{.Names}}" 2>/dev/null | grep -q "${container_name}"; then
        echo "stopped"
    else
        echo "not_installed"
    fi
}

# Wait for Huginn to be ready
huginn::wait_ready() {
    local max_wait="${1:-60}"
    local elapsed=0
    
    log::info "Waiting for Huginn to be ready..."
    
    while [[ ${elapsed} -lt ${max_wait} ]]; do
        if timeout 5 curl -sf "${HUGINN_BASE_URL}/health" &>/dev/null || \
           timeout 5 curl -sf "${HUGINN_BASE_URL}" &>/dev/null; then
            log::success "Huginn is ready"
            return 0
        fi
        
        sleep 2
        ((elapsed+=2))
        echo -n "."
    done
    
    echo ""
    log::error "Huginn failed to become ready within ${max_wait} seconds"
    return 1
}

# Validate Huginn environment
huginn::validate_environment() {
    local errors=0
    
    # Check Docker
    if ! command -v docker &>/dev/null; then
        log::error "Docker is not installed"
        ((errors++))
    fi
    
    # Check port availability (if not running)
    if [[ "$(huginn::get_status)" != "running" ]]; then
        if nc -z localhost "${HUGINN_PORT}" 2>/dev/null; then
            log::error "Port ${HUGINN_PORT} is already in use"
            ((errors++))
        fi
    fi
    
    # Check dependencies
    if [[ "${HUGINN_REQUIRES_POSTGRES:-true}" == "true" ]]; then
        if ! docker ps --filter "name=postgres" --format "{{.Names}}" 2>/dev/null | grep -q postgres; then
            log::warn "PostgreSQL dependency is not running"
        fi
    fi
    
    return ${errors}
}

# API request helper
huginn::api_request() {
    local method="${1}"
    local endpoint="${2}"
    local data="${3:-}"
    
    local url="${HUGINN_BASE_URL}${endpoint}"
    local auth="${DEFAULT_ADMIN_USERNAME}:${DEFAULT_ADMIN_PASSWORD}"
    
    local curl_opts=(-s -f -X "${method}")
    curl_opts+=(-u "${auth}")
    
    if [[ -n "${data}" ]]; then
        curl_opts+=(-H "Content-Type: application/json")
        curl_opts+=(-d "${data}")
    fi
    
    if ! timeout 10 curl "${curl_opts[@]}" "${url}"; then
        log::error "API request failed: ${method} ${endpoint}"
        return 1
    fi
}

# Get Huginn version
huginn::get_version() {
    local container_name="${1:-${CONTAINER_NAME}}"
    
    if [[ "$(huginn::get_status "${container_name}")" == "running" ]]; then
        docker exec "${container_name}" cat /app/VERSION 2>/dev/null || echo "unknown"
    else
        echo "not_running"
    fi
}

# Get resource usage
huginn::get_resource_usage() {
    local container_name="${1:-${CONTAINER_NAME}}"
    
    if [[ "$(huginn::get_status "${container_name}")" == "running" ]]; then
        docker stats "${container_name}" --no-stream --format "CPU: {{.CPUPerc}}, Memory: {{.MemUsage}}"
    else
        echo "Container not running"
    fi
}

# Backup Huginn data
huginn::backup() {
    local backup_dir="${APP_ROOT}/backups/huginn"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="${backup_dir}/huginn_backup_${timestamp}.tar.gz"
    
    log::info "Creating Huginn backup..."
    
    # Create backup directory
    mkdir -p "${backup_dir}"
    
    # Backup database if running
    if [[ "$(huginn::get_status)" == "running" ]]; then
        docker exec "${CONTAINER_NAME}" pg_dump -U huginn huginn_production > "${backup_dir}/database_${timestamp}.sql" 2>/dev/null || true
    fi
    
    # Backup data directory
    if [[ -d "${APP_ROOT}/data/huginn" ]]; then
        tar -czf "${backup_file}" -C "${APP_ROOT}/data" huginn/
        log::success "Backup created: ${backup_file}"
    else
        log::warn "No data directory to backup"
    fi
}

# Restore Huginn data
huginn::restore() {
    local backup_file="${1}"
    
    if [[ ! -f "${backup_file}" ]]; then
        log::error "Backup file not found: ${backup_file}"
        return 1
    fi
    
    log::info "Restoring Huginn from backup..."
    
    # Stop container if running
    if [[ "$(huginn::get_status)" == "running" ]]; then
        huginn::stop
    fi
    
    # Extract backup
    tar -xzf "${backup_file}" -C "${APP_ROOT}/data"
    
    log::success "Backup restored from: ${backup_file}"
}

# Export functions for use by other scripts
export -f huginn::export_config
export -f huginn::init
export -f huginn::get_status
export -f huginn::wait_ready
export -f huginn::validate_environment
export -f huginn::api_request
export -f huginn::get_version
export -f huginn::get_resource_usage
export -f huginn::backup
export -f huginn::restore