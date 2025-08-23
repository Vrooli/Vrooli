#!/bin/bash
# Common functions and variables for Odoo resource

# Resource identification
export ODOO_RESOURCE_NAME="odoo"
export ODOO_RESOURCE_CATEGORY="execution"

# Paths  
export ODOO_BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export ODOO_DATA_DIR="${var_DATA_DIR:-/home/matthalloran8/Vrooli/data}/resources/odoo"
export ODOO_CONFIG_DIR="$ODOO_BASE_DIR/config"
export ODOO_LOG_FILE="${var_LOG_DIR:-/home/matthalloran8/Vrooli/logs}/odoo.log"

# Docker configuration
export ODOO_CONTAINER_NAME="vrooli-odoo"
export ODOO_PG_CONTAINER_NAME="vrooli-odoo-postgres"
export ODOO_NETWORK_NAME="vrooli-odoo-net"
export ODOO_IMAGE="odoo:17.0"
export ODOO_PG_IMAGE="postgres:15-alpine"

# Network configuration
export ODOO_PORT="${ODOO_PORT:-8069}"
export ODOO_LONGPOLLING_PORT="${ODOO_LONGPOLLING_PORT:-8072}"
export ODOO_PG_PORT="${ODOO_PG_PORT:-5469}"

# Database configuration
export ODOO_DB_NAME="odoo"
export ODOO_DB_USER="odoo"
export ODOO_DB_PASSWORD="${ODOO_DB_PASSWORD:-odoo_secure_password_$(date +%s)}"
export ODOO_ADMIN_EMAIL="${ODOO_ADMIN_EMAIL:-admin@example.com}"
export ODOO_ADMIN_PASSWORD="${ODOO_ADMIN_PASSWORD:-admin}"

# Create required directories
odoo_init_dirs() {
    mkdir -p "$ODOO_DATA_DIR"/{addons,config,filestore,sessions}
    mkdir -p "$ODOO_DATA_DIR/postgres"
    mkdir -p "$(dirname "$ODOO_LOG_FILE")"
}

# Check if Odoo is installed
odoo_is_installed() {
    docker image inspect "$ODOO_IMAGE" &>/dev/null && \
    docker image inspect "$ODOO_PG_IMAGE" &>/dev/null
}

# Check if Odoo is running
odoo_is_running() {
    docker ps --format "{{.Names}}" | grep -q "^${ODOO_CONTAINER_NAME}$"
}

# Get container logs
odoo_logs() {
    local lines="${1:-50}"
    
    if odoo_is_running; then
        docker logs --tail "$lines" "$ODOO_CONTAINER_NAME" 2>&1
    elif [[ -f "$ODOO_LOG_FILE" ]]; then
        tail -n "$lines" "$ODOO_LOG_FILE"
    else
        echo "No logs available"
        return 1
    fi
}

# Register with port registry
odoo_register_ports() {
    local port_registry="$ODOO_BASE_DIR/../../port_registry.sh"
    if [[ -f "$port_registry" ]]; then
        "$port_registry" register "odoo" "$ODOO_PORT" "Odoo Web Interface"
        "$port_registry" register "odoo-longpolling" "$ODOO_LONGPOLLING_PORT" "Odoo Longpolling"
        "$port_registry" register "odoo-postgres" "$ODOO_PG_PORT" "Odoo PostgreSQL"
    fi
}

# Export functions
export -f odoo_init_dirs
export -f odoo_is_installed
export -f odoo_is_running
export -f odoo_logs
export -f odoo_register_ports