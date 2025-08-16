#!/usr/bin/env bash
# n8n Docker Management - Simplified using docker-resource-utils.sh

# Source guard to prevent multiple sourcing
[[ -n "${_N8N_DOCKER_SOURCED:-}" ]] && return 0
export _N8N_DOCKER_SOURCED=1

# Source core and frameworks
N8N_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/wait-utils.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/core.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/health.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/recovery.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/auto-credentials.sh"

#######################################
# Start n8n (delegates to core)
#######################################
n8n::start() {
    # Handle existing instance
    if docker::is_running "$N8N_CONTAINER_NAME" && [[ "${FORCE:-no}" != "yes" ]]; then
        log::info "n8n is already running on port $N8N_PORT"
        if n8n::check_basic_health; then
            log::success "Running instance is healthy"
            return 0
        else
            log::warn "Running instance has issues, restarting..."
            n8n::stop
        fi
    fi
    
    # Pre-flight checks
    if ! n8n::detect_filesystem_corruption; then
        log::warn "Filesystem corruption detected"
        if ! n8n::auto_recover; then
            log::error "Failed to recover from corruption"
            return 1
        fi
    fi
    
    # Ensure directories exist
    n8n::create_directories
    
    # Check container exists
    if ! docker::container_exists "$N8N_CONTAINER_NAME"; then
        log::error "n8n not installed. Run: ./manage.sh --action install"
        return 1
    fi
    
    # Start database if needed
    if [[ "$DATABASE_TYPE" == "postgres" ]] && n8n::postgres_exists && ! n8n::postgres_running; then
        log::info "Starting PostgreSQL..."
        docker::start_container "$N8N_DB_CONTAINER_NAME"
        sleep ${N8N_DB_START_WAIT:-3}
    fi
    
    # Start n8n
    log::info "Starting n8n..."
    docker::start_container "$N8N_CONTAINER_NAME"
    
    # Wait for ready
    # wait::for_http expects: (url, expected_code, timeout, headers)
    if wait::for_http "${N8N_BASE_URL}/healthz" 200 60; then
        log::success "✅ n8n is ready on port $N8N_PORT"
        log::info "Access n8n at: $N8N_BASE_URL"
        
        # Optionally update firewall rules for Docker-to-host connectivity
        # This ensures n8n can reach native services like Ollama
        # Only runs if N8N_UPDATE_FIREWALL is explicitly set to "true"
        if [[ "${N8N_UPDATE_FIREWALL:-false}" == "true" ]] && [[ -f "${var_ROOT_DIR}/scripts/lib/firewall/firewall.sh" ]]; then
            log::info "Updating firewall rules for Docker connectivity (requires sudo)..."
            if sudo "${var_ROOT_DIR}/scripts/lib/firewall/firewall.sh" setup 2>/dev/null; then
                log::debug "Firewall rules updated successfully"
            else
                log::warn "Failed to update firewall rules - n8n may not reach host services"
                log::info "To manually update: sudo ${var_ROOT_DIR}/scripts/lib/firewall/firewall.sh setup"
            fi
        elif [[ "${N8N_UPDATE_FIREWALL:-false}" == "true" ]]; then
            log::debug "Firewall script not found - skipping firewall update"
        fi
        
        return 0
    else
        log::error "n8n failed to start properly"
        return 1
    fi
}

#######################################
# Stop n8n
#######################################
n8n::stop() {
    log::info "Stopping n8n..."
    
    if docker::container_exists "$N8N_CONTAINER_NAME"; then
        docker::stop_container "$N8N_CONTAINER_NAME"
    fi
    
    if [[ "$DATABASE_TYPE" == "postgres" ]] && n8n::postgres_running; then
        log::info "Stopping PostgreSQL..."
        docker::stop_container "$N8N_DB_CONTAINER_NAME"
    fi
    
    log::info "n8n stopped"
}

#######################################
# Restart n8n
#######################################
n8n::restart() {
    log::info "Restarting n8n..."
    n8n::stop
    sleep ${N8N_RESTART_WAIT:-2}
    n8n::start
}

#######################################
# Show logs
#######################################
n8n::logs() {
    if ! docker::container_exists "$N8N_CONTAINER_NAME"; then
        log::error "n8n container does not exist"
        return 1
    fi
    
    log::info "Showing n8n logs (last ${LINES:-50} lines)..."
    docker::get_logs "$N8N_CONTAINER_NAME" "${LINES:-50}"
}

