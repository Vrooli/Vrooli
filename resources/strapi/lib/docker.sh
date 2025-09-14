#!/usr/bin/env bash
# Strapi Docker Management Library
# Provides Docker-based deployment options

set -euo pipefail

# Prevent multiple sourcing
[[ -n "${STRAPI_DOCKER_LOADED:-}" ]] && return 0
readonly STRAPI_DOCKER_LOADED=1

# Source core library
DOCKER_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${DOCKER_SCRIPT_DIR}/lib/core.sh"

# Docker configuration
readonly DOCKER_COMPOSE_FILE="${DOCKER_SCRIPT_DIR}/docker/docker-compose.yml"
readonly DOCKER_ENV_FILE="${DOCKER_SCRIPT_DIR}/docker/.env"

#######################################
# Check if Docker is available
#######################################
docker::is_available() {
    command -v docker >/dev/null 2>&1 && \
    docker info >/dev/null 2>&1
}

#######################################
# Check if Docker Compose is available
#######################################
docker::compose_available() {
    if command -v docker-compose >/dev/null 2>&1; then
        return 0
    elif docker compose version >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

#######################################
# Get Docker Compose command
#######################################
docker::get_compose_cmd() {
    if command -v docker-compose >/dev/null 2>&1; then
        echo "docker-compose"
    elif docker compose version >/dev/null 2>&1; then
        echo "docker compose"
    else
        return 1
    fi
}

#######################################
# Create Docker environment file
#######################################
docker::create_env() {
    mkdir -p "$(dirname "${DOCKER_ENV_FILE}")"
    
    cat > "${DOCKER_ENV_FILE}" << EOF
# Database Configuration
POSTGRES_HOST=${POSTGRES_HOST:-postgres}
POSTGRES_PORT=${POSTGRES_PORT:-5432}
POSTGRES_USER=${POSTGRES_USER:-postgres}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-postgres}
STRAPI_DATABASE_NAME=${STRAPI_DATABASE_NAME:-strapi}

# Strapi Configuration
STRAPI_HOST=${STRAPI_HOST:-0.0.0.0}
STRAPI_PORT=${STRAPI_PORT:-1337}

# Security Keys (auto-generated if not set)
JWT_SECRET=${JWT_SECRET:-$(openssl rand -base64 32)}
ADMIN_JWT_SECRET=${ADMIN_JWT_SECRET:-$(openssl rand -base64 32)}
APP_KEYS=${APP_KEYS:-$(openssl rand -base64 32),$(openssl rand -base64 32)}
API_TOKEN_SALT=${API_TOKEN_SALT:-$(openssl rand -base64 32)}
TRANSFER_TOKEN_SALT=${TRANSFER_TOKEN_SALT:-$(openssl rand -base64 32)}

# Admin Credentials
STRAPI_ADMIN_EMAIL=${STRAPI_ADMIN_EMAIL:-admin@vrooli.local}
STRAPI_ADMIN_PASSWORD=${STRAPI_ADMIN_PASSWORD:-$(openssl rand -base64 12)}
EOF
    
    core::success "Docker environment file created"
}

#######################################
# Start Strapi with Docker Compose
#######################################
docker::start() {
    if ! docker::is_available; then
        core::error "Docker is not available"
        return 1
    fi
    
    if ! docker::compose_available; then
        core::error "Docker Compose is not available"
        return 1
    fi
    
    # Create environment file if it doesn't exist
    if [[ ! -f "${DOCKER_ENV_FILE}" ]]; then
        docker::create_env
    fi
    
    # Ensure network exists
    docker network create vrooli-network 2>/dev/null || true
    
    local compose_cmd=$(docker::get_compose_cmd)
    
    core::info "Starting Strapi with Docker Compose..."
    cd "$(dirname "${DOCKER_COMPOSE_FILE}")"
    ${compose_cmd} up -d
    
    # Wait for health check
    local max_wait=60
    local elapsed=0
    
    core::info "Waiting for Strapi to be healthy..."
    while [[ $elapsed -lt $max_wait ]]; do
        if docker exec vrooli-strapi curl -sf http://localhost:1337/health >/dev/null 2>&1; then
            core::success "Strapi is healthy and ready"
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
    done
    
    core::error "Strapi failed to become healthy within ${max_wait} seconds"
    return 1
}

#######################################
# Stop Strapi Docker containers
#######################################
docker::stop() {
    if ! docker::compose_available; then
        core::warning "Docker Compose is not available"
        return 2
    fi
    
    local compose_cmd=$(docker::get_compose_cmd)
    
    core::info "Stopping Strapi Docker containers..."
    cd "$(dirname "${DOCKER_COMPOSE_FILE}")"
    ${compose_cmd} down
    
    core::success "Strapi Docker containers stopped"
}

#######################################
# Show Docker container status
#######################################
docker::status() {
    if ! docker::is_available; then
        core::error "Docker is not available"
        return 1
    fi
    
    echo "Strapi Docker Status"
    echo "===================="
    
    # Check container status
    if docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -q "vrooli-strapi"; then
        docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep "strapi"
    else
        echo "No Strapi containers running"
    fi
}

#######################################
# View Docker container logs
#######################################
docker::logs() {
    local tail_lines="${1:-50}"
    
    if ! docker::is_available; then
        core::error "Docker is not available"
        return 1
    fi
    
    if docker ps -q -f name=vrooli-strapi >/dev/null 2>&1; then
        docker logs --tail "$tail_lines" vrooli-strapi
    else
        core::error "Strapi container is not running"
        return 1
    fi
}

#######################################
# Clean Docker resources
#######################################
docker::clean() {
    local keep_volumes="${1:-false}"
    
    if ! docker::compose_available; then
        core::warning "Docker Compose is not available"
        return 2
    fi
    
    local compose_cmd=$(docker::get_compose_cmd)
    
    core::info "Cleaning Docker resources..."
    cd "$(dirname "${DOCKER_COMPOSE_FILE}")"
    
    if [[ "$keep_volumes" == "--keep-volumes" ]]; then
        ${compose_cmd} down
    else
        ${compose_cmd} down -v
    fi
    
    core::success "Docker resources cleaned"
}