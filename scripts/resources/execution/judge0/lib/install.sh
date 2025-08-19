#!/usr/bin/env bash
# Judge0 Installation Module
# Handles Judge0 installation and initial setup

#######################################
# Main installation function
# Arguments:
#   $1 - Force install (yes/no)
#######################################
judge0::install::main() {
    local force="${1:-no}"
    
    log::header "$JUDGE0_MSG_INSTALL_START"
    
    # Check if already installed
    if judge0::is_installed && [[ "$force" != "yes" ]]; then
        log::warning "Judge0 is already installed. Use --force yes to reinstall."
        return 0
    fi
    
    # Run installation steps
    if ! judge0::install::check_requirements; then
        return 1
    fi
    
    if ! judge0::install::create_infrastructure; then
        return 1
    fi
    
    if ! judge0::install::setup_api_key; then
        return 1
    fi
    
    if ! judge0::install::create_compose_file; then
        return 1
    fi
    
    if ! judge0::install::start_services; then
        return 1
    fi
    
    if ! judge0::install::verify_installation; then
        return 1
    fi
    
    judge0::install::show_success_message
    
    # Auto-install CLI if available
    # shellcheck disable=SC1091
    "${var_SCRIPTS_RESOURCES_LIB_DIR}/install-resource-cli.sh" "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)" 2>/dev/null || true
    
    return 0
}

#######################################
# Check system requirements
#######################################
judge0::install::check_requirements() {
    log::info "$JUDGE0_MSG_INSTALL_CHECKING"
    
    # Check Docker
    if ! command -v docker >/dev/null 2>&1; then
        log::error "$JUDGE0_MSG_ERR_DOCKER"
        return 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        log::error "$JUDGE0_MSG_ERR_DOCKER"
        return 1
    fi
    
    # Check port availability
    if ports::is_port_in_use "$JUDGE0_PORT"; then
        log::error "$JUDGE0_MSG_ERR_PORT"
        return 1
    fi
    
    # Check system resources
    local total_memory=$(system::get_total_memory_mb)
    if [[ $total_memory -lt 2048 ]]; then
        log::warning "$JUDGE0_MSG_WARN_RESOURCES"
    fi
    
    # Check disk space
    local free_space=$(system::get_free_space_gb "$HOME")
    if [[ $free_space -lt 5 ]]; then
        log::warning "$JUDGE0_MSG_WARN_DISK"
    fi
    
    return 0
}

#######################################
# Create infrastructure (network, volumes, directories)
#######################################
judge0::install::create_infrastructure() {
    log::info "$JUDGE0_MSG_INSTALL_DOCKER"
    
    # Create network
    log::info "$JUDGE0_MSG_INSTALL_NETWORK"
    if ! docker::create_network "$JUDGE0_NETWORK_NAME"; then
        log::error "Failed to create Docker network"
        return 1
    fi
    
    # Create volume
    log::info "$JUDGE0_MSG_INSTALL_VOLUME"
    if ! docker volume create "$JUDGE0_VOLUME_NAME" >/dev/null 2>&1; then
        log::error "Failed to create Docker volume"
        return 1
    fi
    
    # Create directories
    if ! judge0::create_directories; then
        return 1
    fi
    
    return 0
}

#######################################
# Setup API key for authentication
#######################################
judge0::install::setup_api_key() {
    log::info "$JUDGE0_MSG_INSTALL_API_KEY"
    
    # Check if API key already exists
    local existing_key=$(judge0::get_api_key)
    
    if [[ -n "$existing_key" ]]; then
        export JUDGE0_API_KEY="$existing_key"
        log::info "Using existing API key"
    else
        # Generate new API key
        JUDGE0_API_KEY=$(judge0::generate_api_key)
        export JUDGE0_API_KEY
        judge0::save_api_key "$JUDGE0_API_KEY"
        log::success "Generated new API key"
    fi
    
    return 0
}

#######################################
# Create Docker Compose file
#######################################
judge0::install::create_compose_file() {
    log::info "$JUDGE0_MSG_INSTALL_CONFIG"
    
    local compose_file="${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    
    cat > "$compose_file" << EOF
services:
  judge0-server:
    image: ${JUDGE0_IMAGE}:${JUDGE0_VERSION}
    container_name: ${JUDGE0_CONTAINER_NAME}
    restart: unless-stopped
    ports:
      - "${JUDGE0_PORT}:2358"
    environment:
      - JUDGE0_AUTHENTICATION_TOKEN=${JUDGE0_API_KEY}
      - ENABLE_SUBMISSION_DELETE=${JUDGE0_ENABLE_SUBMISSION_DELETE}
      - ENABLE_BATCHED_SUBMISSIONS=${JUDGE0_ENABLE_BATCHED_SUBMISSIONS}
      - ENABLE_WAIT_RESULT=${JUDGE0_ENABLE_WAIT_RESULT}
      - ENABLE_CALLBACKS=${JUDGE0_ENABLE_CALLBACKS}
      - CPU_TIME_LIMIT=${JUDGE0_CPU_TIME_LIMIT}
      - WALL_TIME_LIMIT=${JUDGE0_WALL_TIME_LIMIT}
      - MEMORY_LIMIT=${JUDGE0_MEMORY_LIMIT}
      - MAX_PROCESSES_AND_OR_THREADS=${JUDGE0_MAX_PROCESSES}
      - MAX_FILE_SIZE=${JUDGE0_MAX_FILE_SIZE}
      - STACK_LIMIT=${JUDGE0_STACK_LIMIT}
      - NUMBER_OF_WORKERS=${JUDGE0_WORKERS_COUNT}
      - REDIS_HOST=judge0-redis
      - REDIS_PORT=6379
      - POSTGRES_HOST=judge0-db
      - POSTGRES_PORT=5432
      - POSTGRES_DB=judge0
      - POSTGRES_USER=judge0
      - POSTGRES_PASSWORD=judge0
    volumes:
      - ${JUDGE0_VOLUME_NAME}:/var/judge0
      - ${JUDGE0_SUBMISSIONS_DIR}:/judge0/submissions
    networks:
      - ${JUDGE0_NETWORK_NAME}
    depends_on:
      - judge0-db
      - judge0-redis
    deploy:
      resources:
        limits:
          memory: ${JUDGE0_MEMORY_LIMIT_DOCKER}
          cpus: '${JUDGE0_CPU_LIMIT}'
    logging:
      driver: "json-file"
      options:
        max-size: "${JUDGE0_LOG_MAX_SIZE}"
        max-file: "${JUDGE0_LOG_MAX_FILES}"

  judge0-workers:
    image: ${JUDGE0_IMAGE}:${JUDGE0_VERSION}
    command: ["./scripts/workers"]
    restart: unless-stopped
    environment:
      - JUDGE0_AUTHENTICATION_TOKEN=${JUDGE0_API_KEY}
      - ENABLE_NETWORK=${JUDGE0_ENABLE_NETWORK}
      - CPU_TIME_LIMIT=${JUDGE0_CPU_TIME_LIMIT}
      - WALL_TIME_LIMIT=${JUDGE0_WALL_TIME_LIMIT}
      - MEMORY_LIMIT=${JUDGE0_MEMORY_LIMIT}
      - MAX_PROCESSES_AND_OR_THREADS=${JUDGE0_MAX_PROCESSES}
      - MAX_FILE_SIZE=${JUDGE0_MAX_FILE_SIZE}
      - STACK_LIMIT=${JUDGE0_STACK_LIMIT}
      - REDIS_HOST=judge0-redis
      - REDIS_PORT=6379
      - POSTGRES_HOST=judge0-db
      - POSTGRES_PORT=5432
      - POSTGRES_DB=judge0
      - POSTGRES_USER=judge0
      - POSTGRES_PASSWORD=judge0
    volumes:
      - ${JUDGE0_VOLUME_NAME}:/var/judge0
      - ${JUDGE0_SUBMISSIONS_DIR}:/judge0/submissions
    networks:
      - ${JUDGE0_NETWORK_NAME}
    depends_on:
      - judge0-db
      - judge0-redis
    deploy:
      replicas: ${JUDGE0_WORKERS_COUNT}
      resources:
        limits:
          memory: ${JUDGE0_WORKER_MEMORY_LIMIT}
          cpus: '${JUDGE0_WORKER_CPU_LIMIT}'
    logging:
      driver: "json-file"
      options:
        max-size: "${JUDGE0_LOG_MAX_SIZE}"
        max-file: "${JUDGE0_LOG_MAX_FILES}"

  judge0-db:
    image: postgres:15-alpine
    container_name: ${JUDGE0_CONTAINER_NAME}-db
    restart: unless-stopped
    environment:
      - POSTGRES_DB=judge0
      - POSTGRES_USER=judge0
      - POSTGRES_PASSWORD=judge0
    volumes:
      - judge0-postgres-data:/var/lib/postgresql/data
    networks:
      - ${JUDGE0_NETWORK_NAME}
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  judge0-redis:
    image: redis:7-alpine
    container_name: ${JUDGE0_CONTAINER_NAME}-redis
    restart: unless-stopped
    command: redis-server --appendonly yes
    volumes:
      - judge0-redis-data:/data
    networks:
      - ${JUDGE0_NETWORK_NAME}
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

volumes:
  ${JUDGE0_VOLUME_NAME}:
    external: true
  judge0-postgres-data:
  judge0-redis-data:

networks:
  ${JUDGE0_NETWORK_NAME}:
    external: true
EOF

    # Set appropriate permissions
    chmod 600 "$compose_file"
    
    return 0
}

#######################################
# Start Judge0 services
#######################################
judge0::install::start_services() {
    log::info "$JUDGE0_MSG_INSTALL_STARTING"
    
    local compose_file="${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    
    # Start services
    if ! docker compose -f "$compose_file" up -d >/dev/null 2>&1; then
        log::error "Failed to start Judge0 services"
        docker compose -f "$compose_file" logs
        return 1
    fi
    
    # Wait for services to be healthy
    log::info "$JUDGE0_MSG_INSTALL_HEALTH"
    if ! judge0::wait_for_health; then
        log::error "Judge0 failed to become healthy"
        docker compose -f "$compose_file" logs
        return 1
    fi
    
    return 0
}

#######################################
# Verify installation
#######################################
judge0::install::verify_installation() {
    # Test API
    if ! judge0::api::test; then
        log::error "Judge0 API test failed"
        return 1
    fi
    
    # Test simple code execution
    local test_code='console.log("Hello, Judge0!");'
    local result=$(judge0::api::submit "$test_code" "javascript" "" "Hello, Judge0!" 2>/dev/null)
    
    if [[ -z "$result" ]]; then
        log::error "Judge0 test submission failed"
        return 1
    fi
    
    return 0
}

#######################################
# Show success message with next steps
#######################################
judge0::install::show_success_message() {
    log::success "$JUDGE0_MSG_INSTALL_SUCCESS"
    echo
    log::info "📋 Next steps:"
    echo "  • Check status: $0 --action status"
    echo "  • View logs: $0 --action logs"
    echo "  • List languages: $0 --action languages"
    echo "  • Submit code: $0 --action submit --code 'print(\"Hello\")' --language python"
    echo
    log::info "$JUDGE0_MSG_INFO_API"
    log::info "🔑 API Key saved to: ${JUDGE0_CONFIG_DIR}/api_key"
    echo
    log::warning "$JUDGE0_MSG_WARN_SECURITY"
}