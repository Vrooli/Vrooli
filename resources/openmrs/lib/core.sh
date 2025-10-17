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
OPENMRS_DB_CONTAINER="openmrs-mysql"
OPENMRS_APP_CONTAINER="openmrs-app"
OPENMRS_VERSION="${OPENMRS_VERSION:-demo}"

# Database configuration
OPENMRS_DB_NAME="${OPENMRS_DB_NAME:-openmrs}"
OPENMRS_DB_USER="${OPENMRS_DB_USER:-openmrs}"
OPENMRS_DB_PASS="${OPENMRS_DB_PASS:-openmrs_secure_pass_$(openssl rand -hex 8)}"
OPENMRS_DB_ROOT_PASS="${OPENMRS_DB_ROOT_PASS:-root_secure_pass_$(openssl rand -hex 8)}"
OPENMRS_DB_PORT="${OPENMRS_DB_PORT:-3316}"  # Using non-standard port to avoid conflicts

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
        echo "  Database: MySQL on port ${OPENMRS_DB_PORT}"
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
    
    # Save database passwords
    echo "${OPENMRS_DB_PASS}" > "${OPENMRS_CONFIG_DIR}/db_password.txt"
    echo "${OPENMRS_DB_ROOT_PASS}" > "${OPENMRS_CONFIG_DIR}/db_root_password.txt"
    chmod 600 "${OPENMRS_CONFIG_DIR}/db_password.txt"
    chmod 600 "${OPENMRS_CONFIG_DIR}/db_root_password.txt"
    
    # Create Docker Compose file
    cat > "${OPENMRS_CONFIG_DIR}/docker-compose.yml" << EOF
services:
  mysql:
    image: mysql:5.7
    container_name: ${OPENMRS_DB_CONTAINER}
    environment:
      MYSQL_ROOT_PASSWORD: ${OPENMRS_DB_ROOT_PASS}
      MYSQL_DATABASE: ${OPENMRS_DB_NAME}
      MYSQL_USER: ${OPENMRS_DB_USER}
      MYSQL_PASSWORD: ${OPENMRS_DB_PASS}
    volumes:
      - ${OPENMRS_DATA_DIR}/mysql:/var/lib/mysql
    ports:
      - "${OPENMRS_DB_PORT}:3306"
    networks:
      - ${OPENMRS_NETWORK}
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-p${OPENMRS_DB_ROOT_PASS}"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 30s

  openmrs:
    image: openmrs/openmrs-reference-application-distro:${OPENMRS_VERSION}
    container_name: ${OPENMRS_APP_CONTAINER}
    depends_on:
      mysql:
        condition: service_healthy
    environment:
      DB_DATABASE: ${OPENMRS_DB_NAME}
      DB_HOST: mysql
      DB_USERNAME: ${OPENMRS_DB_USER}
      DB_PASSWORD: ${OPENMRS_DB_PASS}
      DB_CREATE_TABLES: 'true'
      DB_AUTO_UPDATE: 'true'
      MODULE_WEB_ADMIN: 'true'
      DEMO: 'true'
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
    echo "Root password saved in: ${OPENMRS_CONFIG_DIR}/db_root_password.txt"
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

# API functions with full CRUD operations
openmrs::api::patient() {
    local action="${1:-list}"
    shift || true
    
    local api_base="http://localhost:${OPENMRS_PORT}/openmrs/ws/rest/v1"
    
    case "$action" in
        list)
            echo "Fetching patient list..."
            curl -s -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
                -H "Content-Type: application/json" \
                "${api_base}/patient?v=default" | jq '.results[] | {uuid: .uuid, display: .display, gender: .person.gender, age: .person.age}'
            ;;
        create)
            local given_name="${1:-John}"
            local family_name="${2:-Doe}"
            local gender="${3:-M}"
            local birthdate="${4:-1990-01-01}"
            
            echo "Creating patient: $given_name $family_name..."
            
            local patient_data='{
                "identifiers": [{
                    "identifier": "'"$(uuidgen | tr '[:lower:]' '[:upper:]')"'",
                    "identifierType": "05a29f94-c0ed-11e2-94be-8c13b969e334",
                    "location": "6351fcf4-e311-4a19-90f9-35667d99a8af",
                    "preferred": true
                }],
                "person": {
                    "gender": "'"$gender"'",
                    "names": [{
                        "givenName": "'"$given_name"'",
                        "familyName": "'"$family_name"'"
                    }],
                    "birthdate": "'"$birthdate"'"
                }
            }'
            
            curl -s -X POST \
                -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
                -H "Content-Type: application/json" \
                -d "$patient_data" \
                "${api_base}/patient" | jq '{uuid: .uuid, display: .display}'
            ;;
        get)
            local uuid="${1:-}"
            if [[ -z "$uuid" ]]; then
                echo "Error: Patient UUID required" >&2
                return 1
            fi
            echo "Fetching patient $uuid..."
            curl -s -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
                -H "Content-Type: application/json" \
                "${api_base}/patient/${uuid}?v=full" | jq '.'
            ;;
        update)
            local uuid="${1:-}"
            local field="${2:-}"
            local value="${3:-}"
            if [[ -z "$uuid" ]] || [[ -z "$field" ]] || [[ -z "$value" ]]; then
                echo "Error: UUID, field, and value required" >&2
                return 1
            fi
            echo "Updating patient $uuid..."
            local update_data="{\"$field\": \"$value\"}"
            curl -s -X POST \
                -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
                -H "Content-Type: application/json" \
                -d "$update_data" \
                "${api_base}/patient/${uuid}" | jq '.'
            ;;
        delete)
            local uuid="${1:-}"
            if [[ -z "$uuid" ]]; then
                echo "Error: Patient UUID required" >&2
                return 1
            fi
            echo "Deleting patient $uuid..."
            curl -s -X DELETE \
                -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
                "${api_base}/patient/${uuid}?purge=true"
            echo "Patient deleted"
            ;;
        *)
            echo "Unknown patient action: $action" >&2
            echo "Available actions: list, create, get, update, delete" >&2
            return 1
            ;;
    esac
}

openmrs::api::encounter() {
    local action="${1:-list}"
    shift || true
    
    local api_base="http://localhost:${OPENMRS_PORT}/openmrs/ws/rest/v1"
    
    case "$action" in
        list)
            echo "Fetching encounter list..."
            curl -s -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
                -H "Content-Type: application/json" \
                "${api_base}/encounter?v=default" | jq '.'
            ;;
        create)
            local patient_uuid="${1:-}"
            local encounter_type="${2:-07000be2-26b6-4cce-8b40-866d8435b613}"
            local datetime="${3:-$(date -Iseconds)}"
            
            if [[ -z "$patient_uuid" ]]; then
                echo "Error: Patient UUID required" >&2
                return 1
            fi
            
            echo "Creating encounter for patient $patient_uuid..."
            
            local encounter_data='{
                "patient": "'"$patient_uuid"'",
                "encounterDatetime": "'"$datetime"'",
                "encounterType": "'"$encounter_type"'",
                "location": "6351fcf4-e311-4a19-90f9-35667d99a8af"
            }'
            
            curl -s -X POST \
                -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
                -H "Content-Type: application/json" \
                -d "$encounter_data" \
                "${api_base}/encounter" | jq '.'
            ;;
        *)
            echo "Encounter action '$action' not yet implemented" >&2
            return 1
            ;;
    esac
}

openmrs::api::concept() {
    local action="${1:-list}"
    shift || true
    
    local api_base="http://localhost:${OPENMRS_PORT}/openmrs/ws/rest/v1"
    
    case "$action" in
        list)
            echo "Fetching concept list..."
            curl -s -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
                -H "Content-Type: application/json" \
                "${api_base}/concept?v=default" | jq '.results[] | {uuid: .uuid, display: .display}'
            ;;
        search)
            local query="${1:-}"
            if [[ -z "$query" ]]; then
                echo "Error: Search query required" >&2
                return 1
            fi
            echo "Searching for concept: $query..."
            curl -s -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
                -H "Content-Type: application/json" \
                "${api_base}/concept?q=${query}&v=default" | jq '.'
            ;;
        *)
            echo "Concept action '$action' not yet implemented" >&2
            return 1
            ;;
    esac
}

openmrs::api::provider() {
    local action="${1:-list}"
    shift || true
    
    local api_base="http://localhost:${OPENMRS_PORT}/openmrs/ws/rest/v1"
    
    case "$action" in
        list)
            echo "Fetching provider list..."
            curl -s -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
                -H "Content-Type: application/json" \
                "${api_base}/provider?v=default" | jq '.'
            ;;
        *)
            echo "Provider action '$action' not yet implemented" >&2
            return 1
            ;;
    esac
}

# FHIR functions
# Demo data seeding function
openmrs::import_demo_data() {
    log::info "Importing demo data into OpenMRS..."
    
    # Wait for OpenMRS to be fully ready
    local retries=30
    while [[ $retries -gt 0 ]]; do
        if timeout 5 curl -sf "http://localhost:${OPENMRS_PORT}/openmrs" &>/dev/null; then
            break
        fi
        ((retries--))
        sleep 2
    done
    
    if [[ $retries -eq 0 ]]; then
        log::error "OpenMRS is not responding"
        return 1
    fi
    
    # Create demo patients
    echo "Creating demo patients..."
    
    local patients=(
        "John,Smith,M,1985-03-15"
        "Mary,Johnson,F,1990-07-22"
        "Robert,Williams,M,1978-11-08"
        "Patricia,Brown,F,1982-04-30"
        "Michael,Davis,M,1995-09-12"
    )
    
    for patient_data in "${patients[@]}"; do
        IFS=',' read -r first last gender dob <<< "$patient_data"
        echo "Creating patient: $first $last"
        openmrs::api::patient create "$first" "$last" "$gender" "$dob" || true
    done
    
    log::success "Demo data import complete"
}

openmrs::fhir::export() {
    local resource_type="${1:-Patient}"
    local api_base="http://localhost:${OPENMRS_PORT}/openmrs/ws/fhir2/R4"
    
    echo "Exporting FHIR $resource_type resources..."
    curl -s -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
        -H "Accept: application/fhir+json" \
        "${api_base}/${resource_type}" | jq '.'
}

openmrs::fhir::import() {
    local resource_file="${1:-}"
    if [[ -z "$resource_file" ]] || [[ ! -f "$resource_file" ]]; then
        echo "Error: Valid FHIR resource file required" >&2
        return 1
    fi
    
    local api_base="http://localhost:${OPENMRS_PORT}/openmrs/ws/fhir2/R4"
    
    echo "Importing FHIR resource from $resource_file..."
    curl -s -X POST \
        -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
        -H "Content-Type: application/fhir+json" \
        -d "@${resource_file}" \
        "${api_base}" | jq '.'
}