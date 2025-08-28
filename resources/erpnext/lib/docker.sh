#!/bin/bash
# ERPNext Docker Functions

# Start with Docker Compose
erpnext::docker::start() {
    local compose_file="$ERPNEXT_RESOURCE_DIR/docker/docker-compose.yml"
    
    # Create docker-compose file if it doesn't exist
    if [ ! -f "$compose_file" ]; then
        erpnext::docker::create_compose || return 1
    fi
    
    # Start services
    docker-compose -f "$compose_file" up -d || {
        log::error "Failed to start ERPNext containers"
        return 1
    }
    
    return 0
}

# Stop Docker containers
erpnext::docker::stop() {
    local compose_file="$ERPNEXT_RESOURCE_DIR/docker/docker-compose.yml"
    
    if [ -f "$compose_file" ]; then
        docker-compose -f "$compose_file" down || {
            log::error "Failed to stop ERPNext containers"
            return 1
        }
    fi
    
    return 0
}

# Pull Docker images
erpnext::docker::pull() {
    log::info "Pulling ERPNext Docker images..."
    
    docker pull frappe/erpnext:${ERPNEXT_VERSION} || {
        log::error "Failed to pull ERPNext image"
        return 1
    }
    
    docker pull mariadb:10.6 || {
        log::error "Failed to pull MariaDB image"
        return 1
    }
    
    docker pull redis:7-alpine || {
        log::error "Failed to pull Redis image"
        return 1
    }
    
    return 0
}

# Remove Docker resources
erpnext::docker::remove() {
    local compose_file="$ERPNEXT_RESOURCE_DIR/docker/docker-compose.yml"
    
    if [ -f "$compose_file" ]; then
        docker-compose -f "$compose_file" down -v || {
            log::warn "Failed to remove ERPNext volumes"
        }
    fi
    
    return 0
}

# Show Docker logs
erpnext::docker::logs() {
    local compose_file="$ERPNEXT_RESOURCE_DIR/docker/docker-compose.yml"
    local service="${1:-erpnext-app}"
    local follow_flag=""
    
    # Check for follow flag
    if [[ "${*}" == *"--follow"* ]] || [[ "${*}" == *"-f"* ]]; then
        follow_flag="-f"
    fi
    
    if [ ! -f "$compose_file" ]; then
        log::error "Docker compose file not found"
        return 1
    fi
    
    docker-compose -f "$compose_file" logs $follow_flag "$service"
}

# Restart Docker containers
erpnext::docker::restart() {
    erpnext::docker::stop && erpnext::docker::start
}

# Create docker-compose.yml
erpnext::docker::create_compose() {
    local compose_file="$ERPNEXT_RESOURCE_DIR/docker/docker-compose.yml"
    
    mkdir -p "${compose_file%/*}"
    
    cat > "$compose_file" << EOF
services:
  erpnext-db:
    image: mariadb:10.6
    container_name: erpnext-db
    environment:
      MYSQL_ROOT_PASSWORD: ${ERPNEXT_ADMIN_PASSWORD}
      MYSQL_DATABASE: ${ERPNEXT_DB_NAME}
      MYSQL_USER: erpnext
      MYSQL_PASSWORD: ${ERPNEXT_ADMIN_PASSWORD}
    volumes:
      - erpnext-db-data:/var/lib/mysql
    networks:
      - erpnext-network
    restart: unless-stopped

  erpnext-redis:
    image: redis:7-alpine
    container_name: erpnext-redis
    networks:
      - erpnext-network
    restart: unless-stopped

  erpnext-app:
    image: ${ERPNEXT_IMAGE}
    container_name: erpnext-app
    ports:
      - "${ERPNEXT_PORT}:8000"
    environment:
      - SITE_NAME=${ERPNEXT_SITE_NAME}
      - DB_HOST=erpnext-db
      - DB_PORT=3306
      - REDIS_CACHE=redis://erpnext-redis:6379/0
      - REDIS_QUEUE=redis://erpnext-redis:6379/1
      - REDIS_SOCKETIO=redis://erpnext-redis:6379/2
      - ADMIN_PASSWORD=${ERPNEXT_ADMIN_PASSWORD}
      - INSTALL_APPS=erpnext
      - DB_ROOT_PASSWORD=${ERPNEXT_ADMIN_PASSWORD}
    volumes:
      - erpnext-app-data:/home/frappe/frappe-bench/sites
      - ${HOME}/.erpnext/apps:/home/frappe/frappe-bench/apps/custom
    depends_on:
      - erpnext-db
      - erpnext-redis
    networks:
      - erpnext-network
    restart: unless-stopped

volumes:
  erpnext-db-data:
  erpnext-app-data:

networks:
  erpnext-network:
    driver: bridge
EOF
    
    return 0
}