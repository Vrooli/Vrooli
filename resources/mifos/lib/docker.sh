#!/usr/bin/env bash
################################################################################
# Mifos Docker Library
# 
# Docker-related functions for managing Mifos containers
################################################################################

set -euo pipefail

# ==============================================================================
# START
# ==============================================================================
mifos::docker::start() {
    log::header "Starting Mifos X Platform"
    
    # Ensure dependencies are running
    mifos::docker::ensure_dependencies || return 1
    
    # Ensure data directories exist
    mkdir -p "${MIFOS_DATA_DIR}" "${MIFOS_LOG_DIR}"
    
    # Generate docker-compose file if needed
    mifos::docker::generate_compose || return 1
    
    # Start containers
    log::info "Starting Mifos containers..."
    if docker-compose -f "${MIFOS_DOCKER_COMPOSE}" up -d; then
        log::success "Mifos containers started"
    else
        log::error "Failed to start Mifos containers"
        return 1
    fi
    
    # Wait for health
    if [[ "${1:-}" == "--wait" ]]; then
        mifos::docker::wait_for_health || return 1
    fi
    
    log::success "Mifos X is running"
    return 0
}

# ==============================================================================
# STOP
# ==============================================================================
mifos::docker::stop() {
    log::header "Stopping Mifos X Platform"
    
    if [[ ! -f "${MIFOS_DOCKER_COMPOSE}" ]]; then
        log::warning "Docker compose file not found"
        return 0
    fi
    
    log::info "Stopping Mifos containers..."
    if docker-compose -f "${MIFOS_DOCKER_COMPOSE}" down; then
        log::success "Mifos containers stopped"
        return 0
    else
        log::error "Failed to stop Mifos containers"
        return 1
    fi
}

# ==============================================================================
# RESTART
# ==============================================================================
mifos::docker::restart() {
    log::header "Restarting Mifos X Platform"
    
    mifos::docker::stop || return 1
    sleep 2
    mifos::docker::start "$@" || return 1
    
    return 0
}

# ==============================================================================
# LOGS
# ==============================================================================
mifos::docker::logs() {
    local service="${1:-}"
    local follow="${2:-}"
    
    if [[ ! -f "${MIFOS_DOCKER_COMPOSE}" ]]; then
        log::error "Docker compose file not found"
        return 1
    fi
    
    local opts=""
    [[ "${follow}" == "--follow" ]] && opts="-f"
    
    if [[ -n "${service}" ]]; then
        docker-compose -f "${MIFOS_DOCKER_COMPOSE}" logs ${opts} "${service}"
    else
        docker-compose -f "${MIFOS_DOCKER_COMPOSE}" logs ${opts}
    fi
}

# ==============================================================================
# ENSURE DEPENDENCIES
# ==============================================================================
mifos::docker::ensure_dependencies() {
    log::info "Ensuring dependencies are running..."
    
    # Check PostgreSQL
    if ! vrooli resource postgres status &>/dev/null; then
        log::info "Starting PostgreSQL dependency..."
        if ! vrooli resource postgres manage start; then
            log::error "Failed to start PostgreSQL"
            return 1
        fi
        # Wait for PostgreSQL to be ready
        sleep 5
    fi
    
    log::success "Dependencies are ready"
    return 0
}

# ==============================================================================
# WAIT FOR HEALTH
# ==============================================================================
mifos::docker::wait_for_health() {
    log::info "Waiting for Mifos to be healthy..."
    
    local retries="${MIFOS_HEALTH_CHECK_RETRIES:-24}"
    local interval="${MIFOS_HEALTH_CHECK_INTERVAL:-5}"
    
    for ((i=1; i<=retries; i++)); do
        if mifos::health_check 5; then
            log::success "Mifos is healthy"
            return 0
        fi
        log::info "Health check attempt ${i}/${retries}..."
        sleep "${interval}"
    done
    
    log::error "Mifos failed to become healthy after ${retries} attempts"
    return 1
}

# ==============================================================================
# GENERATE DOCKER COMPOSE
# ==============================================================================
mifos::docker::generate_compose() {
    log::info "Generating docker-compose configuration..."
    
    mkdir -p "$(dirname "${MIFOS_DOCKER_COMPOSE}")"
    
    # Build our mock server image if needed
    if [[ ! -f "${MIFOS_CLI_DIR}/Dockerfile" ]]; then
        log::error "Dockerfile not found for mock server"
        return 1
    fi
    
    # Build the image
    log::info "Building Mifos mock server image..."
    docker build -t mifos-mock:latest "${MIFOS_CLI_DIR}" || {
        log::error "Failed to build mock server image"
        return 1
    }
    
    cat > "${MIFOS_DOCKER_COMPOSE}" << EOF
version: '3.8'

services:
  ${MIFOS_CONTAINER_PREFIX}-backend:
    image: mifos-mock:latest
    container_name: ${MIFOS_CONTAINER_PREFIX}-backend
    ports:
      - "${MIFOS_PORT}:8080"
    environment:
      - MIFOS_PORT=8080
    volumes:
      - ${MIFOS_DATA_DIR}:/data
      - ${MIFOS_LOG_DIR}:/logs
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/fineract-provider/api/v1/offices"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 20s
    restart: unless-stopped

  ${MIFOS_CONTAINER_PREFIX}-webapp:
    image: ${MIFOS_WEBAPP_IMAGE}:${MIFOS_WEBAPP_VERSION}
    container_name: ${MIFOS_CONTAINER_PREFIX}-webapp
    ports:
      - "${MIFOS_WEBAPP_PORT}:80"
    environment:
      - API_URL=http://localhost:${MIFOS_PORT}
      - FINERACT_API_URL=http://localhost:${MIFOS_PORT}
      - FINERACT_PLATFORM_TENANT_IDENTIFIER=${FINERACT_TENANT}
    depends_on:
      - ${MIFOS_CONTAINER_PREFIX}-backend
    restart: unless-stopped
EOF
    
    log::success "Docker compose configuration generated"
    return 0
}