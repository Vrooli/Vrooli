#!/bin/bash
# Airbyte Core Library Functions

set -euo pipefail

# Resource metadata
RESOURCE_NAME="airbyte"
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_DIR="${RESOURCE_DIR}/config"
DATA_DIR="${RESOURCE_DIR}/data"

# Load configuration
source "${CONFIG_DIR}/defaults.sh"

# Logging functions
log_info() {
    echo "[INFO] $*" >&2
}

log_error() {
    echo "[ERROR] $*" >&2
}

log_debug() {
    if [[ "${DEBUG:-false}" == "true" ]]; then
        echo "[DEBUG] $*" >&2
    fi
}

# Command: info
cmd_info() {
    local format="${1:---text}"
    
    if [[ "$format" == "--json" ]]; then
        cat "${CONFIG_DIR}/runtime.json"
    else
        echo "Airbyte Resource Information:"
        echo "  Startup Order: 550"
        echo "  Dependencies: postgres"
        echo "  Startup Time: 30-60s"
        echo "  Startup Timeout: 120s"
        echo "  Recovery Attempts: 3"
        echo "  Priority: high"
        echo ""
        echo "Ports:"
        echo "  Webapp: ${AIRBYTE_WEBAPP_PORT}"
        echo "  API Server: ${AIRBYTE_SERVER_PORT}"
        echo "  Temporal: ${AIRBYTE_TEMPORAL_PORT}"
    fi
}

# Command: manage
cmd_manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            manage_install "$@"
            ;;
        start)
            manage_start "$@"
            ;;
        stop)
            manage_stop "$@"
            ;;
        restart)
            manage_restart "$@"
            ;;
        uninstall)
            manage_uninstall "$@"
            ;;
        *)
            echo "Error: Unknown manage subcommand: $subcommand" >&2
            exit 1
            ;;
    esac
}

# Install Airbyte
manage_install() {
    log_info "Installing Airbyte..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is required but not installed"
        exit 1
    fi
    
    # Create directories
    mkdir -p "${DATA_DIR}"
    mkdir -p "${DATA_DIR}/workspace"
    mkdir -p "${DATA_DIR}/local-logs"
    
    # Create docker-compose.yml
    cat > "${RESOURCE_DIR}/docker-compose.yml" << EOF
version: '3.8'

services:
  airbyte-db:
    image: postgres:13-alpine
    container_name: airbyte-db
    environment:
      POSTGRES_DB: airbyte
      POSTGRES_USER: airbyte
      POSTGRES_PASSWORD: airbyte
    volumes:
      - airbyte_db_data:/var/lib/postgresql/data
    networks:
      - airbyte
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U airbyte"]
      interval: 10s
      timeout: 5s
      retries: 5

  airbyte-temporal:
    image: temporalio/auto-setup:1.13.0
    container_name: airbyte-temporal
    ports:
      - "${AIRBYTE_TEMPORAL_PORT}:7233"
    environment:
      - DB=postgresql
      - DB_PORT=5432
      - POSTGRES_USER=airbyte
      - POSTGRES_PWD=airbyte
      - POSTGRES_SEEDS=airbyte-db
      - DYNAMIC_CONFIG_FILE_PATH=config/dynamicconfig/development.yaml
    networks:
      - airbyte
    depends_on:
      airbyte-db:
        condition: service_healthy

  airbyte-server:
    image: airbyte/server:${AIRBYTE_VERSION}
    container_name: airbyte-server
    ports:
      - "${AIRBYTE_SERVER_PORT}:8001"
    environment:
      DATABASE_URL: postgresql://airbyte:airbyte@airbyte-db:5432/airbyte
      TEMPORAL_HOST: airbyte-temporal:7233
      WEBAPP_URL: http://localhost:${AIRBYTE_WEBAPP_PORT}
      WORKSPACE_ROOT: /workspace
      LOCAL_ROOT: /local
      TRACKING_STRATEGY: segment
      AIRBYTE_VERSION: ${AIRBYTE_VERSION}
    volumes:
      - ${DATA_DIR}/workspace:/workspace
      - ${DATA_DIR}/local-logs:/local
    networks:
      - airbyte
    depends_on:
      - airbyte-temporal
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8001/api/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 5

  airbyte-scheduler:
    image: airbyte/scheduler:${AIRBYTE_VERSION}
    container_name: airbyte-scheduler
    environment:
      DATABASE_URL: postgresql://airbyte:airbyte@airbyte-db:5432/airbyte
      TEMPORAL_HOST: airbyte-temporal:7233
      WORKSPACE_ROOT: /workspace
      LOCAL_ROOT: /local
      TRACKING_STRATEGY: segment
      AIRBYTE_VERSION: ${AIRBYTE_VERSION}
    volumes:
      - ${DATA_DIR}/workspace:/workspace
      - ${DATA_DIR}/local-logs:/local
    networks:
      - airbyte
    depends_on:
      - airbyte-server

  airbyte-worker:
    image: airbyte/worker:${AIRBYTE_VERSION}
    container_name: airbyte-worker
    environment:
      DATABASE_URL: postgresql://airbyte:airbyte@airbyte-db:5432/airbyte
      TEMPORAL_HOST: airbyte-temporal:7233
      WORKSPACE_ROOT: /workspace
      LOCAL_ROOT: /local
      LOCAL_DOCKER_MOUNT: /tmp/airbyte_local
      TRACKING_STRATEGY: segment
      AIRBYTE_VERSION: ${AIRBYTE_VERSION}
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ${DATA_DIR}/workspace:/workspace
      - ${DATA_DIR}/local-logs:/local
      - /tmp/airbyte_local:/tmp/airbyte_local
    networks:
      - airbyte
    depends_on:
      - airbyte-server

  airbyte-webapp:
    image: airbyte/webapp:${AIRBYTE_VERSION}
    container_name: airbyte-webapp
    ports:
      - "${AIRBYTE_WEBAPP_PORT}:80"
    environment:
      AIRBYTE_VERSION: ${AIRBYTE_VERSION}
      API_URL: http://airbyte-server:8001/api/v1/
      TRACKING_STRATEGY: segment
    networks:
      - airbyte
    depends_on:
      - airbyte-server

networks:
  airbyte:
    name: airbyte_network

volumes:
  airbyte_db_data:
EOF
    
    log_info "Pulling Docker images..."
    docker compose -f "${RESOURCE_DIR}/docker-compose.yml" pull
    
    log_info "Airbyte installed successfully"
}

# Start Airbyte
manage_start() {
    local wait_flag="${1:---no-wait}"
    
    log_info "Starting Airbyte services..."
    
    cd "${RESOURCE_DIR}"
    docker compose up -d
    
    if [[ "$wait_flag" == "--wait" ]]; then
        log_info "Waiting for services to be healthy..."
        local max_attempts=30
        local attempt=0
        
        while [ $attempt -lt $max_attempts ]; do
            if check_health; then
                log_info "Airbyte is ready!"
                return 0
            fi
            
            attempt=$((attempt + 1))
            log_debug "Health check attempt $attempt/$max_attempts"
            sleep 5
        done
        
        log_error "Airbyte failed to start within timeout"
        return 1
    fi
}

# Stop Airbyte
manage_stop() {
    log_info "Stopping Airbyte services..."
    
    cd "${RESOURCE_DIR}"
    docker compose down
    
    log_info "Airbyte stopped"
}

# Restart Airbyte
manage_restart() {
    manage_stop
    sleep 2
    manage_start "$@"
}

# Uninstall Airbyte
manage_uninstall() {
    local keep_data="${1:---remove-data}"
    
    log_info "Uninstalling Airbyte..."
    
    # Stop services
    manage_stop
    
    # Remove containers and images
    cd "${RESOURCE_DIR}"
    docker compose down --rmi all --volumes
    
    # Remove data if requested
    if [[ "$keep_data" != "--keep-data" ]]; then
        log_info "Removing data directory..."
        rm -rf "${DATA_DIR}"
    fi
    
    # Remove docker-compose file
    rm -f "${RESOURCE_DIR}/docker-compose.yml"
    
    log_info "Airbyte uninstalled"
}

# Check health
check_health() {
    timeout 5 curl -sf "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/health" > /dev/null 2>&1
}

# Command: test
cmd_test() {
    local test_type="${1:-all}"
    shift || true
    
    source "${RESOURCE_DIR}/lib/test.sh"
    
    case "$test_type" in
        smoke)
            test_smoke "$@"
            ;;
        integration)
            test_integration "$@"
            ;;
        unit)
            test_unit "$@"
            ;;
        all)
            test_all "$@"
            ;;
        *)
            echo "Error: Unknown test type: $test_type" >&2
            exit 1
            ;;
    esac
}

# Command: content
cmd_content() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        list)
            content_list "$@"
            ;;
        add)
            content_add "$@"
            ;;
        get)
            content_get "$@"
            ;;
        remove)
            content_remove "$@"
            ;;
        execute)
            content_execute "$@"
            ;;
        *)
            echo "Error: Unknown content subcommand: $subcommand" >&2
            exit 1
            ;;
    esac
}

# List content (sources, destinations, connections)
content_list() {
    local type="sources"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                type="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local endpoint
    case "$type" in
        sources)
            endpoint="source_definitions/list"
            ;;
        destinations)
            endpoint="destination_definitions/list"
            ;;
        connections)
            endpoint="connections/list"
            ;;
        *)
            log_error "Unknown type: $type"
            exit 1
            ;;
    esac
    
    curl -sf "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/${endpoint}" | jq '.'
}

# Add content
content_add() {
    local type=""
    local config_file=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                type="$2"
                shift 2
                ;;
            --config)
                config_file="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$type" || -z "$config_file" ]]; then
        log_error "Both --type and --config are required"
        exit 1
    fi
    
    if [[ ! -f "$config_file" ]]; then
        log_error "Config file not found: $config_file"
        exit 1
    fi
    
    local endpoint
    case "$type" in
        source)
            endpoint="sources/create"
            ;;
        destination)
            endpoint="destinations/create"
            ;;
        connection)
            endpoint="connections/create"
            ;;
        *)
            log_error "Unknown type: $type"
            exit 1
            ;;
    esac
    
    curl -X POST \
        -H "Content-Type: application/json" \
        -d "@${config_file}" \
        "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/${endpoint}" | jq '.'
}

# Get content details
content_get() {
    local id=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --id)
                id="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$id" ]]; then
        log_error "--id is required"
        exit 1
    fi
    
    # Try different endpoints to find the resource
    for endpoint in "sources/get" "destinations/get" "connections/get"; do
        result=$(curl -sf -X POST \
            -H "Content-Type: application/json" \
            -d "{\"sourceId\":\"$id\"}" \
            "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/${endpoint}" 2>/dev/null)
        
        if [[ $? -eq 0 ]]; then
            echo "$result" | jq '.'
            return 0
        fi
    done
    
    log_error "Resource not found: $id"
    exit 1
}

# Remove content
content_remove() {
    local id=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --id)
                id="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$id" ]]; then
        log_error "--id is required"
        exit 1
    fi
    
    # Try different endpoints to delete the resource
    for endpoint in "sources/delete" "destinations/delete" "connections/delete"; do
        result=$(curl -sf -X POST \
            -H "Content-Type: application/json" \
            -d "{\"sourceId\":\"$id\"}" \
            "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/${endpoint}" 2>/dev/null)
        
        if [[ $? -eq 0 ]]; then
            echo "Resource deleted: $id"
            return 0
        fi
    done
    
    log_error "Failed to delete resource: $id"
    exit 1
}

# Execute sync
content_execute() {
    local connection_id=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --connection-id)
                connection_id="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$connection_id" ]]; then
        log_error "--connection-id is required"
        exit 1
    fi
    
    curl -X POST \
        -H "Content-Type: application/json" \
        -d "{\"connectionId\":\"$connection_id\"}" \
        "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/connections/sync" | jq '.'
}

# Command: status
cmd_status() {
    echo "Airbyte Service Status:"
    echo ""
    
    # Check each service
    for service in webapp server worker scheduler temporal db; do
        container_name="airbyte-${service}"
        if [[ "$service" == "db" ]]; then
            container_name="airbyte-db"
        fi
        
        if docker ps --format "table {{.Names}}" | grep -q "^${container_name}$"; then
            echo "  ${service}: running"
        else
            echo "  ${service}: stopped"
        fi
    done
    
    echo ""
    echo "Health Check:"
    if check_health; then
        echo "  API: healthy"
    else
        echo "  API: unhealthy"
    fi
}

# Command: logs
cmd_logs() {
    local service="server"
    local follow=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --service)
                service="$2"
                shift 2
                ;;
            -f|--follow)
                follow="-f"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    container_name="airbyte-${service}"
    if [[ "$service" == "db" ]]; then
        container_name="airbyte-db"
    fi
    
    docker logs $follow "$container_name"
}

# Command: credentials
cmd_credentials() {
    echo "Airbyte API Credentials:"
    echo ""
    echo "  API URL: http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/"
    echo "  Webapp URL: http://localhost:${AIRBYTE_WEBAPP_PORT}"
    echo ""
    echo "Default credentials (first-time setup):"
    echo "  Username: airbyte"
    echo "  Password: password"
    echo ""
    echo "Note: Change default credentials after first login"
}