#!/bin/bash
# Eclipse Ditto Core Library Functions

set -euo pipefail

# Resource metadata
RESOURCE_NAME="eclipse-ditto"
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_DIR="${RESOURCE_DIR}/config"
DATA_DIR="${RESOURCE_DIR}/data"
DOCKER_COMPOSE_FILE="${RESOURCE_DIR}/docker-compose.yml"

# Load configuration
source "${CONFIG_DIR}/defaults.sh"

# Load additional libraries
source "${RESOURCE_DIR}/lib/twins.sh"

# Logging functions
log_info() {
    echo "[INFO] $*" >&2
}

log_error() {
    echo "[ERROR] $*" >&2
}

log_debug() {
    if [[ "${DITTO_DEBUG}" == "true" ]]; then
        echo "[DEBUG] $*" >&2
    fi
}

# Command: help
cmd_help() {
    cat << EOF
Eclipse Ditto Digital Twin Platform - Resource Management CLI

USAGE:
    resource-eclipse-ditto <command> [options]

COMMANDS:
    help                Show this help message
    info [--json]       Display resource configuration and runtime information
    
    manage <subcommand> Lifecycle management commands:
        install         Install Eclipse Ditto with Docker
        start [--wait]  Start Ditto services
        stop            Stop Ditto services  
        restart         Restart Ditto services
        uninstall       Remove Ditto and clean up
    
    test <type>         Run validation tests:
        smoke           Quick health check (<30s)
        integration     Full functionality test (<120s)
        unit            Library function tests (<60s)
        all             Run all test suites
    
    content <subcommand> Content management:
        list            List digital twins
        add <file>      Import twin definition from JSON
        get <id>        Get twin by ID
        remove <id>     Delete twin by ID
        execute <cmd>   Execute twin command
    
    status [--json]     Show service status
    logs [--tail N]     View service logs
    credentials         Display API credentials
    
    twin <subcommand>   Digital twin operations:
        create <id>     Create new digital twin
        update <id>     Update twin properties
        query <filter>  Query twins
        watch <id>      Stream twin changes (WebSocket)
        command <id>    Send command to twin

EXAMPLES:
    # Start Eclipse Ditto
    resource-eclipse-ditto manage start --wait
    
    # Create a digital twin
    resource-eclipse-ditto twin create "device:sensor:001"
    
    # Watch twin changes in real-time
    resource-eclipse-ditto twin watch "device:sensor:001"
    
    # Run health check
    resource-eclipse-ditto test smoke

For more information, see: resources/eclipse-ditto/README.md
EOF
}

# Command: info
cmd_info() {
    local format="${1:---text}"
    
    if [[ "$format" == "--json" ]]; then
        cat "${CONFIG_DIR}/runtime.json"
    else
        echo "Eclipse Ditto Resource Information:"
        echo "  Startup Order: 600"
        echo "  Dependencies: postgres, redis"
        echo "  Optional: qdrant, n8n"
        echo "  Startup Time: 30-60s"
        echo "  Startup Timeout: ${DITTO_STARTUP_TIMEOUT}s"
        echo "  Recovery Attempts: 3"
        echo "  Priority: high"
        echo ""
        echo "Ports:"
        echo "  Gateway API: ${DITTO_GATEWAY_PORT}"
        echo "  MongoDB: ${DITTO_MONGODB_PORT}"
        echo ""
        echo "Features:"
        echo "  Digital Twins: ✓"
        echo "  REST API: ✓"
        echo "  WebSocket: ${DITTO_ENABLE_WEBSOCKET}"
        echo "  MQTT: ${DITTO_ENABLE_MQTT}"
        echo "  AMQP: ${DITTO_ENABLE_AMQP}"
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

# Install Eclipse Ditto
manage_install() {
    log_info "Installing Eclipse Ditto..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is required but not installed"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log_error "Docker Compose is required but not installed"
        exit 1
    fi
    
    # Create directories
    mkdir -p "${DATA_DIR}"
    mkdir -p "${DATA_DIR}/mongodb"
    mkdir -p "${DATA_DIR}/nginx"
    mkdir -p "${DATA_DIR}/logs"
    
    # Create docker-compose.yml
    create_docker_compose
    
    # Pull images
    log_info "Pulling Docker images..."
    cd "${RESOURCE_DIR}"
    if command -v docker-compose &> /dev/null; then
        docker-compose pull || true
    else
        docker compose pull || true
    fi
    
    log_info "Eclipse Ditto installed successfully"
}

# Create Docker Compose configuration
create_docker_compose() {
    cat > "${DOCKER_COMPOSE_FILE}" << 'EOF'
version: '3.8'

services:
  mongodb:
    image: mongo:7.0
    container_name: ditto-mongodb
    networks:
      - ditto-network
    volumes:
      - ./data/mongodb:/data/db
    environment:
      - MONGO_INITDB_DATABASE=ditto
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "mongosh", "--eval", "db.adminCommand('ping')"]
      interval: 10s
      timeout: 5s
      retries: 5

  policies:
    image: eclipse/ditto-policies:${DITTO_DOCKER_TAG:-3.5.6}
    container_name: ditto-policies
    networks:
      - ditto-network
    depends_on:
      mongodb:
        condition: service_healthy
    environment:
      - DITTO_MONGODB_URI=mongodb://mongodb:27017/ditto
      - JAVA_TOOL_OPTIONS=-Xms256m -Xmx512m
    restart: unless-stopped

  things:
    image: eclipse/ditto-things:${DITTO_DOCKER_TAG:-3.5.6}
    container_name: ditto-things
    networks:
      - ditto-network
    depends_on:
      mongodb:
        condition: service_healthy
      policies:
        condition: service_started
    environment:
      - DITTO_MONGODB_URI=mongodb://mongodb:27017/ditto
      - JAVA_TOOL_OPTIONS=-Xms256m -Xmx512m
    restart: unless-stopped

  things-search:
    image: eclipse/ditto-things-search:${DITTO_DOCKER_TAG:-3.5.6}
    container_name: ditto-things-search
    networks:
      - ditto-network
    depends_on:
      mongodb:
        condition: service_healthy
      things:
        condition: service_started
    environment:
      - DITTO_MONGODB_URI=mongodb://mongodb:27017/ditto
      - JAVA_TOOL_OPTIONS=-Xms256m -Xmx512m
    restart: unless-stopped

  connectivity:
    image: eclipse/ditto-connectivity:${DITTO_DOCKER_TAG:-3.5.6}
    container_name: ditto-connectivity
    networks:
      - ditto-network
    depends_on:
      mongodb:
        condition: service_healthy
      things:
        condition: service_started
    environment:
      - DITTO_MONGODB_URI=mongodb://mongodb:27017/ditto
      - JAVA_TOOL_OPTIONS=-Xms256m -Xmx512m
    restart: unless-stopped

  gateway:
    image: eclipse/ditto-gateway:${DITTO_DOCKER_TAG:-3.5.6}
    container_name: ditto-gateway
    networks:
      - ditto-network
    ports:
      - "${DITTO_GATEWAY_PORT:-8089}:8080"
    depends_on:
      - policies
      - things
      - things-search
      - connectivity
    environment:
      - DITTO_GATEWAY_AUTHENTICATION_DEVOPS_PASSWORD=${DITTO_PASSWORD:-ditto}
      - DITTO_GATEWAY_AUTHENTICATION_STATUS_PASSWORD=${DITTO_PASSWORD:-ditto}
      - ENABLE_DUMMY_AUTH=true
      - JAVA_TOOL_OPTIONS=-Xms256m -Xmx512m
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 5

  nginx:
    image: nginx:alpine
    container_name: ditto-nginx
    networks:
      - ditto-network
    ports:
      - "${DITTO_GATEWAY_PORT:-8089}:80"
    volumes:
      - ./config/nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      gateway:
        condition: service_healthy
    restart: unless-stopped

networks:
  ditto-network:
    name: ${DITTO_DOCKER_NETWORK:-ditto-network}
    driver: bridge
EOF
}

# Start Eclipse Ditto services
manage_start() {
    local wait_flag=""
    [[ "${1:-}" == "--wait" ]] && wait_flag="--wait"
    
    log_info "Starting Eclipse Ditto services..."
    
    if [[ ! -f "${DOCKER_COMPOSE_FILE}" ]]; then
        log_error "Docker Compose file not found. Run 'manage install' first."
        exit 1
    fi
    
    cd "${RESOURCE_DIR}"
    if command -v docker-compose &> /dev/null; then
        docker-compose up -d
    else
        docker compose up -d
    fi
    
    if [[ -n "$wait_flag" ]]; then
        wait_for_health
    fi
    
    log_info "Eclipse Ditto started successfully"
}

# Wait for services to be healthy
wait_for_health() {
    log_info "Waiting for Eclipse Ditto to be healthy..."
    
    local max_attempts=24  # 2 minutes with 5-second intervals
    local attempt=0
    
    while [[ $attempt -lt $max_attempts ]]; do
        if timeout 5 curl -sf "http://localhost:${DITTO_GATEWAY_PORT}/health" &>/dev/null; then
            log_info "Eclipse Ditto is healthy"
            return 0
        fi
        
        attempt=$((attempt + 1))
        log_debug "Health check attempt $attempt/$max_attempts"
        sleep 5
    done
    
    log_error "Eclipse Ditto failed to become healthy within timeout"
    return 1
}

# Stop Eclipse Ditto services
manage_stop() {
    log_info "Stopping Eclipse Ditto services..."
    
    if [[ ! -f "${DOCKER_COMPOSE_FILE}" ]]; then
        log_info "Docker Compose file not found, nothing to stop"
        return 0
    fi
    
    cd "${RESOURCE_DIR}"
    if command -v docker-compose &> /dev/null; then
        docker-compose down --timeout "${DITTO_SHUTDOWN_TIMEOUT}"
    else
        docker compose down --timeout "${DITTO_SHUTDOWN_TIMEOUT}"
    fi
    
    log_info "Eclipse Ditto stopped successfully"
}

# Restart Eclipse Ditto services
manage_restart() {
    manage_stop
    sleep 2
    manage_start "$@"
}

# Uninstall Eclipse Ditto
manage_uninstall() {
    log_info "Uninstalling Eclipse Ditto..."
    
    # Stop services if running
    manage_stop
    
    # Remove volumes
    if [[ -f "${DOCKER_COMPOSE_FILE}" ]]; then
        cd "${RESOURCE_DIR}"
        if command -v docker-compose &> /dev/null; then
            docker-compose down -v
        else
            docker compose down -v
        fi
    fi
    
    # Remove data directory
    if [[ -d "${DATA_DIR}" ]]; then
        log_info "Removing data directory..."
        rm -rf "${DATA_DIR}"
    fi
    
    # Remove docker-compose file
    rm -f "${DOCKER_COMPOSE_FILE}"
    
    log_info "Eclipse Ditto uninstalled successfully"
}

# Command: test
cmd_test() {
    source "${RESOURCE_DIR}/lib/test.sh"
    test_main "$@"
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

# List digital twins
content_list() {
    log_info "Listing digital twins..."
    
    curl -sf -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
        "http://localhost:${DITTO_GATEWAY_PORT}/api/2/search/things" | jq .
}

# Add digital twin from file
content_add() {
    local file="${1:-}"
    
    if [[ -z "$file" || ! -f "$file" ]]; then
        log_error "File not found: $file"
        exit 1
    fi
    
    log_info "Adding digital twin from $file..."
    
    curl -X PUT -H "Content-Type: application/json" \
        -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
        -d "@${file}" \
        "http://localhost:${DITTO_GATEWAY_PORT}/api/2/things"
}

# Get digital twin by ID
content_get() {
    local twin_id="${1:-}"
    
    if [[ -z "$twin_id" ]]; then
        log_error "Twin ID required"
        exit 1
    fi
    
    curl -sf -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
        "http://localhost:${DITTO_GATEWAY_PORT}/api/2/things/${twin_id}" | jq .
}

# Remove digital twin
content_remove() {
    local twin_id="${1:-}"
    
    if [[ -z "$twin_id" ]]; then
        log_error "Twin ID required"
        exit 1
    fi
    
    log_info "Removing digital twin: $twin_id"
    
    curl -X DELETE -u "${DITTO_USERNAME}:${DITTO_PASSWORD}" \
        "http://localhost:${DITTO_GATEWAY_PORT}/api/2/things/${twin_id}"
}

# Execute twin command
content_execute() {
    local command="${1:-}"
    shift || true
    
    case "$command" in
        create|update|query|watch|command)
            twin_${command} "$@"
            ;;
        *)
            log_error "Unknown command: $command"
            exit 1
            ;;
    esac
}

# Command: status
cmd_status() {
    local format="${1:---text}"
    
    if ! docker ps &>/dev/null; then
        echo "Status: Docker not running"
        exit 1
    fi
    
    local gateway_status="stopped"
    local mongodb_status="stopped"
    
    if docker ps --format "table {{.Names}}" | grep -q "ditto-gateway"; then
        gateway_status="running"
    fi
    
    if docker ps --format "table {{.Names}}" | grep -q "ditto-mongodb"; then
        mongodb_status="running"
    fi
    
    if [[ "$format" == "--json" ]]; then
        cat << EOF
{
  "status": "$([[ "$gateway_status" == "running" ]] && echo "running" || echo "stopped")",
  "services": {
    "gateway": "$gateway_status",
    "mongodb": "$mongodb_status"
  },
  "port": ${DITTO_GATEWAY_PORT},
  "health": "$([[ "$gateway_status" == "running" ]] && timeout 5 curl -sf "http://localhost:${DITTO_GATEWAY_PORT}/health" &>/dev/null && echo "healthy" || echo "unhealthy")"
}
EOF
    else
        echo "Eclipse Ditto Status:"
        echo "  Gateway: $gateway_status"
        echo "  MongoDB: $mongodb_status"
        echo "  Port: ${DITTO_GATEWAY_PORT}"
        echo "  Health: $([[ "$gateway_status" == "running" ]] && timeout 5 curl -sf "http://localhost:${DITTO_GATEWAY_PORT}/health" &>/dev/null && echo "healthy" || echo "unhealthy")"
    fi
}

# Command: logs
cmd_logs() {
    local tail_lines="${2:-50}"
    
    if [[ "${1:-}" == "--tail" ]]; then
        shift
        tail_lines="${1:-50}"
    fi
    
    cd "${RESOURCE_DIR}"
    if command -v docker-compose &> /dev/null; then
        docker-compose logs --tail "$tail_lines"
    else
        docker compose logs --tail "$tail_lines"
    fi
}

# Command: credentials
cmd_credentials() {
    echo "Eclipse Ditto Credentials:"
    echo "  API URL: http://localhost:${DITTO_GATEWAY_PORT}"
    echo "  Username: ${DITTO_USERNAME}"
    echo "  Password: ${DITTO_PASSWORD}"
    echo ""
    echo "Example curl command:"
    echo "  curl -u ${DITTO_USERNAME}:${DITTO_PASSWORD} http://localhost:${DITTO_GATEWAY_PORT}/api/2/things"
}

# Command: twin (delegated to twins.sh)
cmd_twin() {
    twin_command "$@"
}