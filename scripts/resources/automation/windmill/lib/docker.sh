#!/usr/bin/env bash
# Windmill Docker Management Functions
# All Docker Compose operations for Windmill multi-container setup

#######################################
# Start Windmill services using Docker Compose
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::compose_up() {
    log::info "Starting Windmill services..."
    
    if ! windmill::compose_cmd up -d; then
        log::error "Failed to start Windmill services"
        return 1
    fi
    
    log::success "Windmill services started successfully"
    return 0
}

#######################################
# Stop Windmill services using Docker Compose
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::compose_down() {
    log::info "Stopping Windmill services..."
    
    if ! windmill::compose_cmd down; then
        log::error "Failed to stop Windmill services"
        return 1
    fi
    
    log::success "Windmill services stopped successfully"
    return 0
}

#######################################
# Stop Windmill services and remove volumes
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::compose_down_volumes() {
    log::info "Stopping Windmill services and removing volumes..."
    
    if ! windmill::compose_cmd down -v; then
        log::error "Failed to stop Windmill services and remove volumes"
        return 1
    fi
    
    log::success "Windmill services stopped and volumes removed"
    return 0
}

#######################################
# Restart Windmill services
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::compose_restart() {
    log::info "Restarting Windmill services..."
    
    if ! windmill::compose_cmd restart; then
        log::error "Failed to restart Windmill services"
        return 1
    fi
    
    log::success "Windmill services restarted successfully"
    return 0
}

#######################################
# Scale worker services to specified count
# Arguments:
#   $1 - Number of worker replicas
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::scale_workers() {
    local worker_count="$1"
    
    if ! [[ "$worker_count" =~ ^[1-9][0-9]*$ ]]; then
        log::error "Invalid worker count: $worker_count (must be positive integer)"
        return 1
    fi
    
    log::info "Scaling workers to $worker_count replicas..."
    
    # Update the environment file with new worker count
    if [[ -f "$WINDMILL_ENV_FILE" ]]; then
        sed -i "s|^WINDMILL_WORKER_REPLICAS=.*|WINDMILL_WORKER_REPLICAS=$worker_count|" "$WINDMILL_ENV_FILE"
    fi
    
    # Scale the workers
    if ! windmill::compose_cmd up -d --scale "windmill-worker=$worker_count"; then
        log::error "Failed to scale workers to $worker_count"
        return 1
    fi
    
    log::success "Workers scaled to $worker_count replicas successfully"
    
    # Show worker status
    windmill::show_worker_info "$worker_count" "$worker_count"
    
    return 0
}

#######################################
# Restart only worker containers
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::restart_workers() {
    log::info "Restarting Windmill workers..."
    
    # Restart default workers
    if ! windmill::compose_cmd restart windmill-worker; then
        log::error "Failed to restart default workers"
        return 1
    fi
    
    # Restart native worker if enabled
    if windmill::compose_cmd ps --services | grep -q "windmill-worker-native"; then
        if ! windmill::compose_cmd restart windmill-worker-native; then
            log::error "Failed to restart native worker"
            return 1
        fi
    fi
    
    log::success "Workers restarted successfully"
    return 0
}

#######################################
# Restart specific service
# Arguments:
#   $1 - Service name (server|worker|db|lsp|multiplayer)
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::restart_service() {
    local service="$1"
    local container_name
    
    case "$service" in
        "server")
            container_name="windmill-app"
            ;;
        "worker"|"workers")
            windmill::restart_workers
            return $?
            ;;
        "db"|"database")
            container_name="windmill-db"
            ;;
        "lsp")
            container_name="windmill-lsp"
            ;;
        "multiplayer")
            container_name="windmill-multiplayer"
            ;;
        *)
            log::error "Unknown service: $service"
            log::info "Available services: server, worker, db, lsp, multiplayer"
            return 1
            ;;
    esac
    
    log::info "Restarting $service service..."
    
    if ! windmill::compose_cmd restart "$container_name"; then
        log::error "Failed to restart $service service"
        return 1
    fi
    
    log::success "$service service restarted successfully"
    return 0
}

#######################################
# Show logs for Windmill services
# Arguments:
#   $1 - Service name (server|worker|db|lsp|all) or container name
#   $2 - Follow logs flag (true/false)
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::show_logs() {
    local service="${1:-all}"
    local follow="${2:-false}"
    local compose_args=("logs")
    
    if [[ "$follow" == "true" ]]; then
        compose_args+=("--follow")
    fi
    
    # Add timestamps and tail last 100 lines
    compose_args+=("--timestamps" "--tail=100")
    
    case "$service" in
        "server")
            compose_args+=("windmill-app")
            ;;
        "worker"|"workers")
            compose_args+=("windmill-worker" "windmill-worker-native")
            ;;
        "db"|"database")
            compose_args+=("windmill-db")
            ;;
        "lsp")
            compose_args+=("windmill-lsp")
            ;;
        "multiplayer")
            compose_args+=("windmill-multiplayer")
            ;;
        "all")
            # Don't specify service to show all logs
            ;;
        *)
            # Assume it's a specific container name
            compose_args+=("$service")
            ;;
    esac
    
    log::info "Showing logs for: $service"
    
    if [[ "$follow" == "true" ]]; then
        log::info "Press Ctrl+C to stop following logs"
    fi
    
    windmill::compose_cmd "${compose_args[@]}"
}

#######################################
# Get service status from Docker Compose
# Outputs: Service status in table format
#######################################
windmill::get_compose_status() {
    if ! windmill::is_installed; then
        echo "Windmill is not installed"
        return 1
    fi
    
    log::info "Windmill service status:"
    echo
    
    # Show compose status
    windmill::compose_cmd ps
    
    echo
    
    # Show resource usage if services are running
    if windmill::is_running; then
        log::info "Resource usage:"
        echo
        
        # Get container IDs for resource monitoring
        local container_ids
        container_ids=$(windmill::compose_cmd ps -q 2>/dev/null)
        
        if [[ -n "$container_ids" ]]; then
            # Show resource stats for running containers
            docker stats --no-stream --format "table {{.Container}}\t{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}" $container_ids 2>/dev/null || true
        fi
    fi
}

#######################################
# Pull latest Windmill images
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::pull_images() {
    log::info "Pulling latest Windmill images..."
    
    if ! windmill::compose_cmd pull; then
        log::error "Failed to pull Windmill images"
        return 1
    fi
    
    log::success "Windmill images updated successfully"
    return 0
}

#######################################
# Remove Windmill containers and images
# Arguments:
#   $1 - Remove images flag (true/false, default: false)
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::cleanup() {
    local remove_images="${1:-false}"
    
    log::info "Cleaning up Windmill containers..."
    
    # Stop and remove containers and networks
    if ! windmill::compose_down; then
        log::warn "Failed to gracefully stop services, forcing removal..."
    fi
    
    # Remove containers forcefully if still running
    local containers
    containers=$(docker ps -aq --filter "name=${WINDMILL_PROJECT_NAME}" 2>/dev/null || true)
    if [[ -n "$containers" ]]; then
        log::info "Removing remaining containers..."
        docker rm -f $containers 2>/dev/null || true
    fi
    
    # Remove custom networks
    local networks
    networks=$(docker network ls --filter "name=${WINDMILL_PROJECT_NAME}" --format "{{.Name}}" 2>/dev/null || true)
    if [[ -n "$networks" ]]; then
        log::info "Removing networks..."
        for network in $networks; do
            docker network rm "$network" 2>/dev/null || true
        done
    fi
    
    # Remove images if requested
    if [[ "$remove_images" == "true" ]]; then
        log::info "Removing Windmill images..."
        
        local images
        images=$(docker images --filter "reference=ghcr.io/windmill-labs/windmill" --format "{{.ID}}" 2>/dev/null || true)
        if [[ -n "$images" ]]; then
            docker rmi $images 2>/dev/null || true
        fi
    fi
    
    log::success "Windmill cleanup completed"
    return 0
}

#######################################
# Create and start services with health checks
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::deploy() {
    log::info "Deploying Windmill services..."
    
    # Create and start services
    if ! windmill::compose_up; then
        return 1
    fi
    
    # Wait for services to become healthy
    if ! windmill::wait_for_health; then
        log::error "Windmill deployment failed - services not healthy"
        log::info "Check logs with: $0 --action logs"
        return 1
    fi
    
    log::success "Windmill deployed successfully!"
    return 0
}

#######################################
# Backup Windmill data (database and volumes)
# Arguments:
#   $1 - Backup directory path
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::backup_data() {
    local backup_dir="$1"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_path="$backup_dir/windmill_backup_$timestamp"
    
    log::info "Creating backup at: $backup_path"
    
    # Create backup directory
    mkdir -p "$backup_path" || {
        log::error "Failed to create backup directory: $backup_path"
        return 1
    }
    
    # Backup database if using internal PostgreSQL
    if windmill::compose_cmd ps --services | grep -q "windmill-db"; then
        log::info "Backing up PostgreSQL database..."
        
        if ! windmill::compose_cmd exec -T windmill-db pg_dump -U postgres -d windmill > "$backup_path/database.sql"; then
            log::error "Failed to backup database"
            return 1
        fi
        
        log::success "Database backup completed"
    fi
    
    # Backup volumes
    log::info "Backing up Docker volumes..."
    
    local volumes
    volumes=$(windmill::compose_cmd config --volumes 2>/dev/null || true)
    
    for volume in $volumes; do
        local volume_backup_dir="$backup_path/volumes/$volume"
        mkdir -p "$volume_backup_dir"
        
        log::info "Backing up volume: $volume"
        
        # Use a temporary container to copy volume data
        docker run --rm \
            -v "${WINDMILL_PROJECT_NAME}_$volume:/volume:ro" \
            -v "$volume_backup_dir:/backup" \
            alpine:latest \
            tar -czf "/backup/data.tar.gz" -C /volume . 2>/dev/null || {
            log::warn "Failed to backup volume: $volume"
        }
    done
    
    # Backup configuration files
    log::info "Backing up configuration files..."
    cp "$WINDMILL_ENV_FILE" "$backup_path/environment" 2>/dev/null || true
    cp "$WINDMILL_COMPOSE_FILE" "$backup_path/docker-compose.yml" 2>/dev/null || true
    
    # Create backup manifest
    cat > "$backup_path/backup_info.txt" << EOF
Windmill Backup Information
==========================
Backup Date: $(date)
Windmill Version: $(docker inspect ${WINDMILL_PROJECT_NAME}-server 2>/dev/null | jq -r '.[0].Config.Image' 2>/dev/null || "unknown")
Project Name: $WINDMILL_PROJECT_NAME
Backup Path: $backup_path

Files Included:
- database.sql (PostgreSQL dump)
- volumes/ (Docker volume backups)
- environment (Environment configuration)
- docker-compose.yml (Compose configuration)
EOF
    
    # Compress backup
    log::info "Compressing backup..."
    
    local compressed_backup="$backup_dir/windmill_backup_$timestamp.tar.gz"
    if tar -czf "$compressed_backup" -C "$backup_dir" "windmill_backup_$timestamp"; then
        # Remove uncompressed backup directory
        rm -rf "$backup_path"
        log::success "Backup completed: $compressed_backup"
        echo "$compressed_backup"
    else
        log::error "Failed to compress backup"
        return 1
    fi
    
    return 0
}

#######################################
# Restore Windmill data from backup
# Arguments:
#   $1 - Backup file path (.tar.gz)
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::restore_data() {
    local backup_file="$1"
    
    if [[ ! -f "$backup_file" ]]; then
        log::error "Backup file not found: $backup_file"
        return 1
    fi
    
    log::info "Restoring from backup: $backup_file"
    
    # Create temporary restore directory
    local restore_dir
    restore_dir=$(mktemp -d) || {
        log::error "Failed to create temporary directory"
        return 1
    }
    
    # Extract backup
    log::info "Extracting backup..."
    if ! tar -xzf "$backup_file" -C "$restore_dir"; then
        log::error "Failed to extract backup file"
        rm -rf "$restore_dir"
        return 1
    fi
    
    # Find the backup directory (should be only one)
    local backup_dir
    backup_dir=$(find "$restore_dir" -maxdepth 1 -type d -name "windmill_backup_*" | head -n1)
    
    if [[ ! -d "$backup_dir" ]]; then
        log::error "Invalid backup structure"
        rm -rf "$restore_dir"
        return 1
    fi
    
    # Stop services before restore
    log::info "Stopping Windmill services..."
    windmill::compose_down_volumes || true
    
    # Restore configuration files
    if [[ -f "$backup_dir/environment" ]]; then
        log::info "Restoring environment configuration..."
        cp "$backup_dir/environment" "$WINDMILL_ENV_FILE"
    fi
    
    # Start services to create volumes
    log::info "Starting services..."
    if ! windmill::compose_up; then
        log::error "Failed to start services for restore"
        rm -rf "$restore_dir"
        return 1
    fi
    
    # Wait for database to be ready
    sleep 10
    
    # Restore database
    if [[ -f "$backup_dir/database.sql" ]]; then
        log::info "Restoring database..."
        
        if ! windmill::compose_cmd exec -T windmill-db psql -U postgres -d windmill < "$backup_dir/database.sql"; then
            log::error "Failed to restore database"
            rm -rf "$restore_dir"
            return 1
        fi
        
        log::success "Database restored successfully"
    fi
    
    # Restore volumes
    if [[ -d "$backup_dir/volumes" ]]; then
        log::info "Restoring volumes..."
        
        for volume_backup in "$backup_dir/volumes"/*; do
            if [[ -d "$volume_backup" && -f "$volume_backup/data.tar.gz" ]]; then
                local volume_name=$(basename "$volume_backup")
                
                log::info "Restoring volume: $volume_name"
                
                # Use temporary container to restore volume data
                docker run --rm \
                    -v "${WINDMILL_PROJECT_NAME}_$volume_name:/volume" \
                    -v "$volume_backup:/backup" \
                    alpine:latest \
                    sh -c "cd /volume && tar -xzf /backup/data.tar.gz" || {
                    log::warn "Failed to restore volume: $volume_name"
                }
            fi
        done
    fi
    
    # Restart services
    log::info "Restarting services..."
    windmill::compose_restart
    
    # Wait for health
    if windmill::wait_for_health; then
        log::success "Windmill restored successfully from backup!"
    else
        log::warn "Restore completed but services may not be fully healthy"
    fi
    
    # Cleanup
    rm -rf "$restore_dir"
    
    return 0
}