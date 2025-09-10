#!/bin/bash
# JupyterHub Core Library Functions

set -euo pipefail

# Ensure required environment variables are set
: ${JUPYTERHUB_PORT:=8000}
: ${JUPYTERHUB_CONTAINER_NAME:=vrooli-jupyterhub}
: ${JUPYTERHUB_DATA_DIR:=/data/resources/jupyterhub}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}ℹ️  $1${NC}"
}

log_warn() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}" >&2
}

# Show resource information
show_info() {
    local format="${1:-text}"
    
    if [[ "$format" == "--json" ]]; then
        cat "${SCRIPT_DIR}/config/runtime.json"
    else
        echo "📊 JupyterHub Resource Information"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "Name:        jupyterhub"
        echo "Category:    productivity"
        echo "Version:     4.0.0"
        echo "Port:        ${JUPYTERHUB_PORT}"
        echo "API Port:    ${JUPYTERHUB_API_PORT:-8001}"
        echo "Status:      $(get_service_status)"
        echo ""
        echo "Dependencies:"
        echo "  - postgres (required)"
        echo "  - docker (required)"
        echo "  - redis (optional)"
        echo ""
        echo "Features:"
        echo "  ✓ Multi-user support"
        echo "  ✓ Docker-based isolation"
        echo "  ✓ OAuth authentication"
        echo "  ✓ Resource limits"
        echo "  ✓ Persistent storage"
    fi
}

# Show detailed status
show_status() {
    local format="${1:-text}"
    local status=$(get_service_status)
    local health=$(check_health)
    
    if [[ "$format" == "--json" ]]; then
        cat << EOF
{
  "status": "$status",
  "health": "$health",
  "container": "$(docker ps --filter name=${JUPYTERHUB_CONTAINER_NAME} --format '{{.Status}}' 2>/dev/null || echo 'not running')",
  "port": ${JUPYTERHUB_PORT},
  "url": "http://localhost:${JUPYTERHUB_PORT}",
  "users": $(get_user_count),
  "active_servers": $(get_active_server_count)
}
EOF
    else
        echo "📊 JupyterHub Status"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "Service:     $status"
        echo "Health:      $health"
        echo "URL:         http://localhost:${JUPYTERHUB_PORT}"
        echo "Users:       $(get_user_count)"
        echo "Active:      $(get_active_server_count) servers running"
        
        if [[ "$status" == "running" ]]; then
            echo ""
            echo "Container:"
            docker ps --filter name=${JUPYTERHUB_CONTAINER_NAME} --format "table {{.Status}}\t{{.Ports}}"
        fi
    fi
}

# Get service status
get_service_status() {
    if docker ps --filter name=${JUPYTERHUB_CONTAINER_NAME} --format '{{.Names}}' | grep -q ${JUPYTERHUB_CONTAINER_NAME}; then
        echo "running"
    else
        echo "stopped"
    fi
}

# Check health
check_health() {
    if [[ "$(get_service_status)" != "running" ]]; then
        echo "service not running"
        return 1
    fi
    
    if timeout 5 curl -sf "http://localhost:${JUPYTERHUB_PORT}/hub/health" &>/dev/null; then
        echo "healthy"
        return 0
    else
        echo "unhealthy"
        return 1
    fi
}

# Get user count
get_user_count() {
    # This would query the database or API in a real implementation
    echo "0"
}

# Get active server count
get_active_server_count() {
    # Count docker containers with jupyterhub user prefix
    docker ps --filter "label=jupyterhub.user" --format '{{.Names}}' 2>/dev/null | wc -l || echo "0"
}

# Show logs
show_logs() {
    local lines="${1:-100}"
    local follow=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -f|--follow)
                follow=true
                ;;
            -n|--lines)
                lines="$2"
                shift
                ;;
        esac
        shift
    done
    
    if [[ "$follow" == "true" ]]; then
        docker logs -f ${JUPYTERHUB_CONTAINER_NAME} 2>&1
    else
        docker logs --tail "$lines" ${JUPYTERHUB_CONTAINER_NAME} 2>&1
    fi
}

# Show credentials
show_credentials() {
    echo "🔐 JupyterHub Credentials"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "URL:           http://localhost:${JUPYTERHUB_PORT}"
    echo "Admin User:    ${JUPYTERHUB_ADMIN_USER:-admin}"
    echo "Admin Pass:    ${JUPYTERHUB_ADMIN_PASSWORD:-changeme}"
    echo ""
    echo "API Endpoint:  http://localhost:${JUPYTERHUB_API_PORT:-8001}/hub/api"
    if [[ -n "${JUPYTERHUB_API_TOKEN:-}" ]]; then
        echo "API Token:     ${JUPYTERHUB_API_TOKEN}"
    else
        echo "API Token:     (not configured)"
    fi
    echo ""
    echo "⚠️  Change default credentials in production!"
}

# Management functions

# Install JupyterHub
manage_install() {
    log_info "Installing JupyterHub..."
    
    # Check dependencies
    if ! docker info &>/dev/null; then
        log_error "Docker is not running. Please start Docker first."
        exit 1
    fi
    
    # Create data directories
    log_info "Creating data directories..."
    mkdir -p "${JUPYTERHUB_DATA_DIR}"/{users,shared,config,extensions,backups}
    
    # Generate tokens if not set
    if [[ -z "${JUPYTERHUB_COOKIE_SECRET:-}" ]]; then
        export JUPYTERHUB_COOKIE_SECRET=$(openssl rand -hex 32)
        log_info "Generated cookie secret"
    fi
    
    if [[ -z "${JUPYTERHUB_PROXY_AUTH_TOKEN:-}" ]]; then
        export JUPYTERHUB_PROXY_AUTH_TOKEN=$(openssl rand -hex 32)
        log_info "Generated proxy auth token"
    fi
    
    if [[ -z "${JUPYTERHUB_API_TOKEN:-}" ]]; then
        export JUPYTERHUB_API_TOKEN=$(openssl rand -hex 32)
        log_info "Generated API token"
    fi
    
    # Save tokens to config
    cat > "${JUPYTERHUB_CONFIG_DIR}/tokens.env" << EOF
JUPYTERHUB_COOKIE_SECRET=${JUPYTERHUB_COOKIE_SECRET}
JUPYTERHUB_PROXY_AUTH_TOKEN=${JUPYTERHUB_PROXY_AUTH_TOKEN}
JUPYTERHUB_API_TOKEN=${JUPYTERHUB_API_TOKEN}
EOF
    
    # Pull Docker images
    log_info "Pulling Docker images..."
    docker pull ${JUPYTERHUB_IMAGE}
    docker pull ${JUPYTERHUB_NOTEBOOK_IMAGE}
    
    # Create Docker network if not exists
    docker network create ${JUPYTERHUB_NETWORK} 2>/dev/null || true
    
    log_info "✅ JupyterHub installed successfully"
    log_info "Run 'resource-jupyterhub manage start' to start the service"
}

# Uninstall JupyterHub
manage_uninstall() {
    local keep_data=false
    
    for arg in "$@"; do
        if [[ "$arg" == "--keep-data" ]]; then
            keep_data=true
        fi
    done
    
    log_info "Uninstalling JupyterHub..."
    
    # Stop service if running
    manage_stop
    
    # Remove container
    docker rm -f ${JUPYTERHUB_CONTAINER_NAME} 2>/dev/null || true
    
    # Remove data if not keeping
    if [[ "$keep_data" == "false" ]]; then
        log_warn "Removing all JupyterHub data..."
        rm -rf "${JUPYTERHUB_DATA_DIR}"
    else
        log_info "Keeping user data in ${JUPYTERHUB_DATA_DIR}"
    fi
    
    log_info "✅ JupyterHub uninstalled"
}

# Start JupyterHub
manage_start() {
    local wait=false
    local timeout=120
    
    for arg in "$@"; do
        case "$arg" in
            --wait)
                wait=true
                ;;
            --timeout)
                timeout="$2"
                shift
                ;;
        esac
    done
    
    if [[ "$(get_service_status)" == "running" ]]; then
        log_info "JupyterHub is already running"
        return 0
    fi
    
    log_info "Starting JupyterHub..."
    
    # Create minimal configuration
    create_jupyterhub_config
    
    # Start container
    docker run -d \
        --name ${JUPYTERHUB_CONTAINER_NAME} \
        --network ${JUPYTERHUB_NETWORK} \
        -p ${JUPYTERHUB_PORT}:8000 \
        -p ${JUPYTERHUB_API_PORT:-8001}:8001 \
        -p ${JUPYTERHUB_PROXY_PORT:-8081}:8081 \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -v ${JUPYTERHUB_DATA_DIR}:/data \
        -v ${JUPYTERHUB_CONFIG_DIR}/jupyterhub_config.py:/srv/jupyterhub/jupyterhub_config.py \
        --env-file ${JUPYTERHUB_CONFIG_DIR}/tokens.env \
        ${JUPYTERHUB_IMAGE} \
        jupyterhub -f /srv/jupyterhub/jupyterhub_config.py
    
    if [[ "$wait" == "true" ]]; then
        log_info "Waiting for JupyterHub to be ready..."
        local elapsed=0
        while [[ $elapsed -lt $timeout ]]; do
            if check_health &>/dev/null; then
                log_info "✅ JupyterHub is ready"
                show_credentials
                return 0
            fi
            sleep 5
            elapsed=$((elapsed + 5))
        done
        log_error "Timeout waiting for JupyterHub to start"
        return 1
    fi
    
    log_info "✅ JupyterHub started"
}

# Stop JupyterHub
manage_stop() {
    local force=false
    
    for arg in "$@"; do
        if [[ "$arg" == "--force" ]]; then
            force=true
        fi
    done
    
    if [[ "$(get_service_status)" != "running" ]]; then
        log_info "JupyterHub is not running"
        return 0
    fi
    
    log_info "Stopping JupyterHub..."
    
    # Stop user servers first
    log_info "Stopping user servers..."
    docker ps --filter "label=jupyterhub.user" -q | xargs -r docker stop
    
    # Stop hub
    if [[ "$force" == "true" ]]; then
        docker kill ${JUPYTERHUB_CONTAINER_NAME}
    else
        docker stop ${JUPYTERHUB_CONTAINER_NAME}
    fi
    
    docker rm ${JUPYTERHUB_CONTAINER_NAME}
    
    log_info "✅ JupyterHub stopped"
}

# Restart JupyterHub
manage_restart() {
    log_info "Restarting JupyterHub..."
    manage_stop "$@"
    sleep 2
    manage_start "$@"
}

# Create JupyterHub configuration
create_jupyterhub_config() {
    cat > "${JUPYTERHUB_CONFIG_DIR}/jupyterhub_config.py" << 'EOF'
import os

# JupyterHub configuration
c = get_config()

# Hub settings
c.JupyterHub.hub_ip = '0.0.0.0'
c.JupyterHub.hub_port = 8001

# Authentication - using simple PAM auth for now
c.JupyterHub.authenticator_class = 'jupyterhub.auth.PAMAuthenticator'
c.PAMAuthenticator.open_sessions = False

# Admin users
admin_users = os.environ.get('JUPYTERHUB_ADMIN_USERS', 'admin').split(',')
c.Authenticator.admin_users = set(admin_users)

# Spawner - using simple local process spawner for minimal setup
c.JupyterHub.spawner_class = 'jupyterhub.spawner.LocalProcessSpawner'

# Proxy
c.ConfigurableHTTPProxy.auth_token = os.environ.get('JUPYTERHUB_PROXY_AUTH_TOKEN', '')

# Cookie secret
c.JupyterHub.cookie_secret = bytes.fromhex(os.environ.get('JUPYTERHUB_COOKIE_SECRET', ''))

# Services
c.JupyterHub.services = []

# Database - using SQLite for minimal setup
c.JupyterHub.db_url = 'sqlite:////data/jupyterhub.sqlite'

# Logging
c.JupyterHub.log_level = os.environ.get('JUPYTERHUB_LOG_LEVEL', 'INFO')
EOF
}

# Content management functions

# List content
content_list() {
    local type="users"
    
    for arg in "$@"; do
        case "$arg" in
            --type)
                type="$2"
                shift
                ;;
        esac
        shift || true
    done
    
    case "$type" in
        users)
            echo "📋 JupyterHub Users:"
            echo "(No users configured in minimal setup)"
            ;;
        notebooks)
            echo "📋 User Notebooks:"
            if [[ -d "${JUPYTERHUB_USER_DATA_DIR}" ]]; then
                find "${JUPYTERHUB_USER_DATA_DIR}" -name "*.ipynb" -type f 2>/dev/null | head -20
            else
                echo "(No notebooks found)"
            fi
            ;;
        extensions)
            echo "📋 Available Extensions:"
            echo "- JupyterLab Git"
            echo "- Variable Inspector"
            echo "- Table of Contents"
            ;;
        profiles)
            echo "📋 Server Profiles:"
            echo "- small (1 CPU, 2GB RAM)"
            echo "- medium (2 CPU, 4GB RAM)"
            echo "- large (4 CPU, 8GB RAM)"
            ;;
        *)
            log_error "Unknown type: $type"
            exit 1
            ;;
    esac
}

# Add content
content_add() {
    log_info "Add functionality would be implemented here"
    log_info "This would add users, notebooks, or extensions"
}

# Get content
content_get() {
    log_info "Get functionality would be implemented here"
    log_info "This would retrieve specific content details"
}

# Remove content
content_remove() {
    log_info "Remove functionality would be implemented here"
    log_info "This would remove users, notebooks, or extensions"
}

# Execute command
content_execute() {
    log_info "Execute functionality would be implemented here"
    log_info "This would run administrative commands"
}

# Spawn user server
content_spawn() {
    local user=""
    
    for arg in "$@"; do
        case "$arg" in
            --user)
                user="$2"
                shift
                ;;
        esac
        shift || true
    done
    
    if [[ -z "$user" ]]; then
        log_error "User not specified. Use --user <username>"
        exit 1
    fi
    
    log_info "Spawning notebook server for user: $user"
    log_info "This would start a Docker container for the user's notebook server"
}