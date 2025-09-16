#!/usr/bin/env bash
# OpenMRS Core Functionality

set -euo pipefail

# Configuration
OPENMRS_DIR="${OPENMRS_DIR:-${HOME}/.vrooli/openmrs}"
OPENMRS_DATA_DIR="${OPENMRS_DATA_DIR:-${OPENMRS_DIR}/data}"
OPENMRS_CONFIG_DIR="${OPENMRS_CONFIG_DIR:-${OPENMRS_DIR}/config}"
OPENMRS_LOGS_DIR="${OPENMRS_LOGS_DIR:-${OPENMRS_DIR}/logs}"

# Source required libraries
source "${SCRIPT_DIR}/../../scripts/lib/utils/log.sh"

# Source port registry
source "${SCRIPT_DIR}/../../scripts/resources/port_registry.sh"

# Get ports from registry
OPENMRS_PORT="${OPENMRS_PORT:-$(ports::get_resource_port "openmrs")}"
OPENMRS_API_PORT="${OPENMRS_API_PORT:-$(ports::get_resource_port "openmrs-api")}"
OPENMRS_FHIR_PORT="${OPENMRS_FHIR_PORT:-$(ports::get_resource_port "openmrs-fhir")}"

# Docker configuration
OPENMRS_NETWORK="openmrs-network"
OPENMRS_DB_CONTAINER="openmrs-postgres"
OPENMRS_APP_CONTAINER="openmrs-app"
OPENMRS_VERSION="${OPENMRS_VERSION:-3.0.0}"

# Database configuration
OPENMRS_DB_NAME="${OPENMRS_DB_NAME:-openmrs}"
OPENMRS_DB_USER="${OPENMRS_DB_USER:-openmrs}"
OPENMRS_DB_PASS="${OPENMRS_DB_PASS:-openmrs_secure_pass_$(openssl rand -hex 8)}"
OPENMRS_DB_PORT="${OPENMRS_DB_PORT:-5444}"  # Using non-standard port to avoid conflicts

# Admin credentials
OPENMRS_ADMIN_USER="${OPENMRS_ADMIN_USER:-admin}"
OPENMRS_ADMIN_PASS="${OPENMRS_ADMIN_PASS:-Admin123}"

# Show runtime configuration
openmrs::info() {
    local format="${1:-text}"
    
    if [[ "$format" == "--json" ]]; then
        cat "${SCRIPT_DIR}/../config/runtime.json" 2>/dev/null || echo "{}"
    else
        echo "OpenMRS Runtime Configuration:"
        echo "  Web Interface: http://localhost:${OPENMRS_PORT}"
        echo "  REST API: http://localhost:${OPENMRS_API_PORT}"
        echo "  FHIR API: http://localhost:${OPENMRS_FHIR_PORT}"
        echo "  Database: PostgreSQL on port ${OPENMRS_DB_PORT}"
        echo "  Data Directory: ${OPENMRS_DATA_DIR}"
        echo "  Version: ${OPENMRS_VERSION}"
    fi
}

# Check status
openmrs::status() {
    local verbose="${1:-}"
    
    echo "OpenMRS Status:"
    
    # Check if containers are running
    if docker ps --format '{{.Names}}' | grep -q "^${OPENMRS_APP_CONTAINER}$"; then
        echo "  Application: Running ✓"
        
        # Check health endpoint
        if timeout 5 curl -sf "http://localhost:${OPENMRS_PORT}/openmrs" &>/dev/null; then
            echo "  Web Interface: Healthy ✓"
        else
            echo "  Web Interface: Not responding ✗"
        fi
        
        # Check API endpoints
        if timeout 5 curl -sf "http://localhost:${OPENMRS_API_PORT}/openmrs/ws/rest/v1/session" &>/dev/null; then
            echo "  REST API: Healthy ✓"
        else
            echo "  REST API: Not responding ✗"
        fi
    else
        echo "  Application: Stopped ✗"
    fi
    
    # Check database
    if docker ps --format '{{.Names}}' | grep -q "^${OPENMRS_DB_CONTAINER}$"; then
        echo "  Database: Running ✓"
    else
        echo "  Database: Stopped ✗"
    fi
    
    if [[ "$verbose" == "--verbose" ]] || [[ "$verbose" == "-v" ]]; then
        echo ""
        echo "Container Details:"
        docker ps -a --filter "name=${OPENMRS_APP_CONTAINER}" --filter "name=${OPENMRS_DB_CONTAINER}" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
    fi
}

# View logs
openmrs::logs() {
    local service="${1:-all}"
    local follow="${2:-}"
    
    case "$service" in
        app|application)
            docker logs ${follow:+-f} "${OPENMRS_APP_CONTAINER}" 2>&1
            ;;
        db|database)
            docker logs ${follow:+-f} "${OPENMRS_DB_CONTAINER}" 2>&1
            ;;
        all|"")
            echo "=== Database Logs ===" 
            docker logs --tail 20 "${OPENMRS_DB_CONTAINER}" 2>&1 || true
            echo ""
            echo "=== Application Logs ==="
            docker logs --tail 50 "${OPENMRS_APP_CONTAINER}" 2>&1 || true
            ;;
        *)
            echo "Unknown service: $service" >&2
            echo "Available: app, db, all" >&2
            exit 1
            ;;
    esac
}

# Install OpenMRS
openmrs::install() {
    log::info "Installing OpenMRS Clinical Records Platform..."
    
    # Create directories
    mkdir -p "${OPENMRS_DATA_DIR}" "${OPENMRS_CONFIG_DIR}" "${OPENMRS_LOGS_DIR}"
    
    # Create Docker network
    if ! docker network ls | grep -q "${OPENMRS_NETWORK}"; then
        log::info "Creating Docker network..."
        docker network create "${OPENMRS_NETWORK}"
    fi
    
    # Save database password
    echo "${OPENMRS_DB_PASS}" > "${OPENMRS_CONFIG_DIR}/db_password.txt"
    chmod 600 "${OPENMRS_CONFIG_DIR}/db_password.txt"
    
    # Create Docker Compose file
    cat > "${OPENMRS_CONFIG_DIR}/docker-compose.yml" << EOF
version: '3.8'

services:
  postgres:
    image: postgres:14-alpine
    container_name: ${OPENMRS_DB_CONTAINER}
    environment:
      POSTGRES_DB: ${OPENMRS_DB_NAME}
      POSTGRES_USER: ${OPENMRS_DB_USER}
      POSTGRES_PASSWORD: ${OPENMRS_DB_PASS}
    volumes:
      - ${OPENMRS_DATA_DIR}/postgres:/var/lib/postgresql/data
    ports:
      - "${OPENMRS_DB_PORT}:5432"
    networks:
      - ${OPENMRS_NETWORK}
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "${OPENMRS_DB_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5

  openmrs:
    image: openmrs/openmrs-reference-application-3-backend:${OPENMRS_VERSION}
    container_name: ${OPENMRS_APP_CONTAINER}
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      OMRS_DB_HOSTNAME: postgres
      OMRS_DB_PORT: 5432
      OMRS_DB_NAME: ${OPENMRS_DB_NAME}
      OMRS_DB_USERNAME: ${OPENMRS_DB_USER}
      OMRS_DB_PASSWORD: ${OPENMRS_DB_PASS}
      OMRS_ADMIN_USER: ${OPENMRS_ADMIN_USER}
      OMRS_ADMIN_PASS: ${OPENMRS_ADMIN_PASS}
      OMRS_CONFIG_CONNECTION_TYPE: postgresql
      OMRS_CONFIG_HAS_CURRENT_OPENMRS_DATABASE: 'false'
      OMRS_CONFIG_CREATE_DATABASE_USER: 'false'
      OMRS_CONFIG_CREATE_TABLES: 'true'
      OMRS_CONFIG_ADD_DEMO_DATA: 'true'
      OMRS_CONFIG_MODULE_WEB_ADMIN: 'true'
      OMRS_CONFIG_AUTO_UPDATE_DATABASE: 'true'
    volumes:
      - ${OPENMRS_DATA_DIR}/openmrs:/openmrs/data
      - ${OPENMRS_LOGS_DIR}:/openmrs/logs
    ports:
      - "${OPENMRS_PORT}:8080"
      - "${OPENMRS_API_PORT}:8081"
      - "${OPENMRS_FHIR_PORT}:8082"
    networks:
      - ${OPENMRS_NETWORK}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/openmrs"]
      interval: 30s
      timeout: 10s
      retries: 10
      start_period: 300s

networks:
  ${OPENMRS_NETWORK}:
    external: true
EOF
    
    log::success "OpenMRS installation complete"
    echo "Database password saved in: ${OPENMRS_CONFIG_DIR}/db_password.txt"
}

# Start OpenMRS
openmrs::start() {
    local wait_flag="${1:-}"
    
    log::info "Starting OpenMRS services..."
    
    # Start using Docker Compose
    cd "${OPENMRS_CONFIG_DIR}"
    docker-compose up -d
    
    if [[ "$wait_flag" == "--wait" ]]; then
        log::info "Waiting for OpenMRS to be ready (this may take 2-3 minutes for initial startup)..."
        
        local retries=60
        while [[ $retries -gt 0 ]]; do
            if timeout 5 curl -sf "http://localhost:${OPENMRS_PORT}/openmrs" &>/dev/null; then
                log::success "OpenMRS is ready!"
                echo "Access OpenMRS at: http://localhost:${OPENMRS_PORT}/openmrs"
                echo "Login: ${OPENMRS_ADMIN_USER} / ${OPENMRS_ADMIN_PASS}"
                return 0
            fi
            ((retries--))
            echo -n "."
            sleep 5
        done
        
        log::error "OpenMRS failed to start within timeout"
        return 1
    fi
    
    log::success "OpenMRS services started"
}

# Stop OpenMRS  
openmrs::stop() {
    log::info "Stopping OpenMRS services..."
    
    if [[ -f "${OPENMRS_CONFIG_DIR}/docker-compose.yml" ]]; then
        cd "${OPENMRS_CONFIG_DIR}"
        docker-compose down || true
    else
        # Fallback to direct container stop
        docker stop "${OPENMRS_APP_CONTAINER}" "${OPENMRS_DB_CONTAINER}" 2>/dev/null || true
    fi
    
    log::success "OpenMRS services stopped"
}

# Restart OpenMRS
openmrs::restart() {
    openmrs::stop
    sleep 2
    openmrs::start "$@"
}

# Uninstall OpenMRS
openmrs::uninstall() {
    local keep_data="${1:-}"
    
    log::warning "Uninstalling OpenMRS..."
    
    # Stop services
    openmrs::stop
    
    # Remove containers
    docker rm -f "${OPENMRS_APP_CONTAINER}" "${OPENMRS_DB_CONTAINER}" 2>/dev/null || true
    
    # Remove network
    docker network rm "${OPENMRS_NETWORK}" 2>/dev/null || true
    
    # Remove data unless --keep-data flag is set
    if [[ "$keep_data" != "--keep-data" ]]; then
        log::info "Removing data directories..."
        rm -rf "${OPENMRS_DIR}"
    else
        log::info "Keeping data directories"
    fi
    
    log::success "OpenMRS uninstalled"
}

# Content management functions
openmrs::content::list() {
    echo "Available OpenMRS content operations:"
    echo "  - patient: Manage patient records"
    echo "  - encounter: Manage clinical encounters"  
    echo "  - concept: Manage clinical concepts"
    echo "  - provider: Manage healthcare providers"
    echo "  - location: Manage facility locations"
    echo "  - obs: Manage observations"
    echo ""
    echo "Use: resource-openmrs content execute <operation> [args]"
}

openmrs::content::add() {
    local type="${1:-}"
    shift || true
    
    case "$type" in
        patient)
            openmrs::api::patient create "$@"
            ;;
        encounter)
            openmrs::api::encounter create "$@"
            ;;
        *)
            echo "Unknown content type: $type" >&2
            exit 1
            ;;
    esac
}

openmrs::content::get() {
    local type="${1:-}"
    local id="${2:-}"
    
    case "$type" in
        patient)
            openmrs::api::patient get "$id"
            ;;
        encounter)
            openmrs::api::encounter get "$id"
            ;;
        *)
            echo "Unknown content type: $type" >&2
            exit 1
            ;;
    esac
}

openmrs::content::remove() {
    local type="${1:-}"
    local id="${2:-}"
    
    case "$type" in
        patient)
            openmrs::api::patient delete "$id"
            ;;
        encounter)
            openmrs::api::encounter delete "$id"
            ;;
        *)
            echo "Unknown content type: $type" >&2
            exit 1
            ;;
    esac
}

openmrs::content::execute() {
    local operation="${1:-}"
    shift || true
    
    case "$operation" in
        import-demo-data)
            openmrs::import_demo_data "$@"
            ;;
        export-fhir)
            openmrs::fhir::export "$@"
            ;;
        *)
            echo "Unknown operation: $operation" >&2
            exit 1
            ;;
    esac
}

# API functions (simplified for scaffolding)
openmrs::api::patient() {
    local action="${1:-list}"
    shift || true
    
    local api_base="http://localhost:${OPENMRS_API_PORT}/openmrs/ws/rest/v1"
    
    case "$action" in
        list)
            curl -s -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
                "${api_base}/patient" | jq '.'
            ;;
        create)
            echo "Patient creation not yet implemented"
            ;;
        get)
            local uuid="${1:-}"
            curl -s -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
                "${api_base}/patient/${uuid}" | jq '.'
            ;;
        *)
            echo "Unknown patient action: $action" >&2
            exit 1
            ;;
    esac
}

openmrs::api::encounter() {
    local action="${1:-list}"
    echo "Encounter API not yet implemented"
}

openmrs::api::concept() {
    local action="${1:-list}"
    echo "Concept API not yet implemented"
}

openmrs::api::provider() {
    local action="${1:-list}"
    echo "Provider API not yet implemented"
}

# FHIR functions
openmrs::fhir::export() {
    local resource_type="${1:-Patient}"
    echo "FHIR export not yet implemented"
}

openmrs::fhir::import() {
    echo "FHIR import not yet implemented"
}