#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
KEYCLOAK_LIB_DIR="${APP_ROOT}/resources/keycloak/lib"

# Dependencies are expected to be sourced by caller

# Initialize PostgreSQL database for Keycloak
keycloak::init_postgres_db() {
    # Check if PostgreSQL is running
    if ! docker ps --format "table {{.Names}}" | grep -q "vrooli-postgres-main"; then
        log::warning "PostgreSQL container not found. Using H2 database instead."
        KEYCLOAK_DB="dev-file"
        return 0
    fi
    
    # Create database and user if they don't exist
    log::info "Creating Keycloak database and user in PostgreSQL..."
    
    # Execute SQL commands in the PostgreSQL container
    # First try to create the user (will fail silently if exists)
    docker exec vrooli-postgres-main psql -U vrooli -d vrooli -c "CREATE USER keycloak WITH PASSWORD 'keycloak';" 2>/dev/null || true
    
    # Then create the database (will fail silently if exists)
    docker exec vrooli-postgres-main psql -U vrooli -d vrooli -c "CREATE DATABASE keycloak OWNER keycloak;" 2>/dev/null || true
    
    # Grant privileges
    docker exec vrooli-postgres-main psql -U vrooli -d vrooli -c "GRANT ALL PRIVILEGES ON DATABASE keycloak TO keycloak;" 2>/dev/null || true
    
    # Test the connection
    if docker exec vrooli-postgres-main psql -U keycloak -d keycloak -c "SELECT 1;" &>/dev/null; then
        log::success "PostgreSQL database initialized for Keycloak"
        return 0
    else
        log::warning "Failed to create PostgreSQL database. Using H2 database instead."
        KEYCLOAK_DB="dev-file"
        return 0
    fi
}

# Start Keycloak
keycloak::start() {
    log::info "Starting Keycloak"
    
    # Check if already running
    if keycloak::is_running; then
        log::warning "Keycloak is already running"
        return 0
    fi
    
    # Install if needed
    if ! keycloak::is_installed; then
        log::info "Docker not available, installing..."
        source "${KEYCLOAK_LIB_DIR}/install.sh"
        keycloak::install || return 1
    fi
    
    local port
    port=$(keycloak::get_port)
    
    # Remove existing container if it exists but is stopped
    if keycloak::container_exists && ! keycloak::is_running; then
        log::info "Removing existing stopped container"
        docker rm "${KEYCLOAK_CONTAINER_NAME}" >/dev/null 2>&1 || true
    fi
    
    # Initialize PostgreSQL database if using postgres
    if [[ "${KEYCLOAK_DB}" == "postgres" ]]; then
        log::info "Initializing PostgreSQL database for Keycloak..."
        keycloak::init_postgres_db
    fi
    
    # Build docker run command based on database type
    local docker_cmd="docker run -d"
    docker_cmd+=" --name \"${KEYCLOAK_CONTAINER_NAME}\""
    docker_cmd+=" --network \"${KEYCLOAK_NETWORK}\""
    docker_cmd+=" -p \"${port}:8080\""
    docker_cmd+=" -e \"KEYCLOAK_ADMIN=${KEYCLOAK_ADMIN_USER}\""
    docker_cmd+=" -e \"KEYCLOAK_ADMIN_PASSWORD=${KEYCLOAK_ADMIN_PASSWORD}\""
    docker_cmd+=" -e \"KC_DB=${KEYCLOAK_DB}\""
    
    # Add PostgreSQL-specific environment variables if using postgres
    if [[ "${KEYCLOAK_DB}" == "postgres" ]]; then
        docker_cmd+=" -e \"KC_DB_URL=${KEYCLOAK_DB_URL}\""
        docker_cmd+=" -e \"KC_DB_USERNAME=${KEYCLOAK_DB_USERNAME}\""
        docker_cmd+=" -e \"KC_DB_PASSWORD=${KEYCLOAK_DB_PASSWORD}\""
        docker_cmd+=" -e \"KC_DB_SCHEMA=${KEYCLOAK_DB_SCHEMA}\""
    fi
    
    docker_cmd+=" -e \"KC_HOSTNAME_STRICT=${KEYCLOAK_HOSTNAME_STRICT}\""
    docker_cmd+=" -e \"KC_HOSTNAME_STRICT_HTTPS=${KEYCLOAK_HOSTNAME_STRICT_HTTPS}\""
    docker_cmd+=" -e \"KC_HTTP_ENABLED=${KEYCLOAK_HTTP_ENABLED}\""
    docker_cmd+=" -e \"KC_HEALTH_ENABLED=${KEYCLOAK_HEALTH_ENABLED}\""
    docker_cmd+=" -e \"KC_METRICS_ENABLED=${KEYCLOAK_METRICS_ENABLED}\""
    docker_cmd+=" -e \"KC_FEATURES=${KEYCLOAK_FEATURES}\""
    docker_cmd+=" -e \"KC_LOG_LEVEL=${KEYCLOAK_LOG_LEVEL}\""
    docker_cmd+=" -e \"JAVA_OPTS=${KEYCLOAK_JVM_OPTS}\""
    docker_cmd+=" -v \"${KEYCLOAK_REALMS_DIR}:/opt/keycloak/data/import:ro\""
    docker_cmd+=" \"${KEYCLOAK_IMAGE}\""
    docker_cmd+=" start-dev"
    docker_cmd+=" --import-realm"
    
    # Start the container
    log::info "Starting Keycloak container on port ${port}..."
    if ! eval "${docker_cmd}" >/dev/null 2>&1; then
        log::error "Failed to start Keycloak container"
        log::info "Recovery hints:"
        log::info "  1. Check if port ${port} is already in use: lsof -i:${port}"
        log::info "  2. Check Docker daemon is running: docker ps"
        log::info "  3. Check network exists: docker network ls | grep ${KEYCLOAK_NETWORK}"
        log::info "  4. Check image availability: docker pull ${KEYCLOAK_IMAGE}"
        return 1
    fi
    
    # Store container ID as PID for consistency with other resources
    docker inspect --format='{{.State.Pid}}' "${KEYCLOAK_CONTAINER_NAME}" > "${KEYCLOAK_PID_FILE}" 2>/dev/null || true
    
    # Wait for service to be ready
    log::info "Waiting for Keycloak to be ready..."
    if keycloak::wait_for_ready 60; then
        log::success "Keycloak started successfully on port ${port}"
        log::info "Admin Console: http://localhost:${port}/admin"
        
        # Show credentials with security warning if using defaults
        if [[ "${KEYCLOAK_ADMIN_PASSWORD}" == "admin" ]]; then
            log::warning "Using DEFAULT credentials: ${KEYCLOAK_ADMIN_USER}/${KEYCLOAK_ADMIN_PASSWORD}"
            log::info "For production, set KEYCLOAK_ADMIN_PASSWORD environment variable"
        else
            log::info "Credentials: ${KEYCLOAK_ADMIN_USER}/[secure password set]"
        fi
        return 0
    else
        log::error "Failed to start Keycloak service"
        log::info "Recovery hints:"
        log::info "  1. Check container logs: docker logs ${KEYCLOAK_CONTAINER_NAME}"
        log::info "  2. Verify database connection: docker exec ${KEYCLOAK_CONTAINER_NAME} env | grep KC_DB"
        log::info "  3. Check health endpoint: curl -sf http://localhost:${port}/health"
        keycloak::stop
        return 1
    fi
}

# Stop Keycloak
keycloak::stop() {
    log::info "Stopping Keycloak"
    
    if keycloak::container_exists; then
        log::info "Stopping Keycloak container"
        docker stop "${KEYCLOAK_CONTAINER_NAME}" >/dev/null 2>&1 || true
        docker rm "${KEYCLOAK_CONTAINER_NAME}" >/dev/null 2>&1 || true
    fi
    
    # Clean up PID file
    if [[ -f "${KEYCLOAK_PID_FILE}" ]]; then
        rm -f "${KEYCLOAK_PID_FILE}"
    fi
    
    # Also kill any process on the port (failsafe)
    local port
    port=$(keycloak::get_port)
    local pids
    pids=$(lsof -ti:${port} 2>/dev/null || true)
    if [[ -n "${pids}" ]]; then
        echo "${pids}" | xargs kill -9 2>/dev/null || true
    fi
    
    log::success "Keycloak stopped"
    return 0
}

# Restart Keycloak
keycloak::restart() {
    log::info "Restarting Keycloak"
    keycloak::stop
    sleep 3
    keycloak::start
}