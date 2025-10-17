#!/bin/bash

# Wiki.js Common Functions

set -euo pipefail

# Configuration
WIKIJS_CONTAINER="wikijs"
WIKIJS_IMAGE="requarks/wiki:2"
WIKIJS_DB_NAME="wikijs"
WIKIJS_DB_USER="wikijs"
WIKIJS_DATA_DIR="${APP_ROOT}/.vrooli/data/resources/wikijs"
WIKIJS_CONFIG_FILE="$WIKIJS_DATA_DIR/config.yml"

# Get Wiki.js port from registry
get_wikijs_port() {
    local port_registry="${APP_ROOT:-/root/Vrooli}/scripts/resources/port_registry.sh"
    if [[ -f "$port_registry" ]]; then
        local port=$("$port_registry" get wikijs 2>/dev/null || echo "")
        if [[ -n "$port" ]]; then
            echo "$port"
        else
            echo "3010"  # Use the registered port
        fi
    else
        echo "3010"
    fi
}

# Get database connection info
get_db_config() {
    # Use host.docker.internal to connect from container to host
    # This works on Mac/Windows, for Linux we need the host IP
    local db_host="${WIKIJS_DB_HOST:-host.docker.internal}"
    
    # On Linux, host.docker.internal might not work, so we use the gateway IP
    if [[ "$OSTYPE" == "linux-gnu"* ]] && ! getent hosts host.docker.internal &>/dev/null; then
        # Get the default gateway IP (host machine from container perspective)
        db_host=$(docker run --rm alpine ip route | awk '/default/ { print $3 }' 2>/dev/null || echo "172.17.0.1")
    fi
    
    local db_port="${WIKIJS_DB_PORT:-5433}"  # Use correct PostgreSQL port
    local db_pass="${WIKIJS_DB_PASS:-wikijs_pass}"
    
    # Try to get port from PostgreSQL resource if available
    if command -v resource-postgres &>/dev/null; then
        local pg_status=$(resource-postgres status 2>/dev/null || true)
        if [[ -n "$pg_status" ]]; then
            # Extract port from the connection string
            local conn_string=$(echo "$pg_status" | grep -oP 'postgresql://[^:]+:\K\d+' | head -1 || echo "")
            if [[ -n "$conn_string" ]]; then
                db_port="$conn_string"
            fi
        fi
    fi
    
    echo "host=$db_host port=$db_port name=$WIKIJS_DB_NAME user=$WIKIJS_DB_USER pass=$db_pass"
}

# Check if Wiki.js is installed
is_installed() {
    # Check if data directory exists (config file is created on first run)
    [[ -d "$WIKIJS_DATA_DIR" ]]
}

# Check if Wiki.js container is running
is_running() {
    docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${WIKIJS_CONTAINER}$"
}

# Get container status
get_container_status() {
    if is_running; then
        echo "running"
    elif docker ps -a --format "{{.Names}}" 2>/dev/null | grep -q "^${WIKIJS_CONTAINER}$"; then
        echo "stopped"
    else
        echo "not_found"
    fi
}

# Check Wiki.js health
check_health() {
    local port=$(get_wikijs_port)
    
    # Check if Wiki.js is responding with proper timeout (per v2.0 contract)
    if timeout 5 curl -sf "http://localhost:${port}/" &>/dev/null; then
        return 0
    fi
    
    return 1
}

# Get Wiki.js version
get_version() {
    if is_running; then
        docker exec "$WIKIJS_CONTAINER" node -e "console.log(require('/wiki/package.json').version)" 2>/dev/null || echo "unknown"
    else
        echo "not_running"
    fi
}

# Create database if needed
ensure_database() {
    if ! command -v resource-postgres &>/dev/null; then
        echo "Warning: PostgreSQL resource not available, skipping database setup"
        return 0
    fi
    
    # Check if database exists
    local db_exists=$(resource-postgres query "SELECT 1 FROM pg_database WHERE datname='$WIKIJS_DB_NAME'" 2>/dev/null || echo "")
    
    if [[ -z "$db_exists" ]]; then
        echo "Creating Wiki.js database..."
        resource-postgres query "CREATE DATABASE $WIKIJS_DB_NAME" 2>/dev/null || true
        resource-postgres query "CREATE USER $WIKIJS_DB_USER WITH PASSWORD '$WIKIJS_DB_PASS'" 2>/dev/null || true
        resource-postgres query "GRANT ALL PRIVILEGES ON DATABASE $WIKIJS_DB_NAME TO $WIKIJS_DB_USER" 2>/dev/null || true
    fi
}

# Wait for Wiki.js to be ready
wait_for_ready() {
    local max_wait=60
    local elapsed=0
    
    echo "Waiting for Wiki.js to be ready..."
    while [[ $elapsed -lt $max_wait ]]; do
        if check_health; then
            echo "Wiki.js is ready!"
            return 0
        fi
        sleep 2
        ((elapsed+=2))
    done
    
    echo "Wiki.js failed to become ready within ${max_wait} seconds"
    return 1
}

# Get API endpoint
get_api_endpoint() {
    local port=$(get_wikijs_port)
    echo "http://localhost:${port}/graphql"
}

# Make GraphQL query
graphql_query() {
    local query="$1"
    local port=$(get_wikijs_port)
    
    curl -sf "http://localhost:${port}/graphql" \
        -H "Content-Type: application/json" \
        -d "{\"query\":\"$query\"}" 2>/dev/null
}

# Get admin token (for API access)
get_admin_token() {
    # This would normally be retrieved from the database or initial setup
    # For now, return a placeholder
    echo "${WIKIJS_ADMIN_TOKEN:-admin-token-placeholder}"
}