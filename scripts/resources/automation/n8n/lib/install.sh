#!/usr/bin/env bash
# n8n Installation Functions
# Install, uninstall, and configuration update functions

#######################################
# Update Vrooli configuration
#######################################
n8n::update_config() {
    # Create JSON with proper escaping
    local additional_config
    additional_config=$(cat <<EOF
{
    "features": {
        "workflows": true,
        "webhooks": true,
        "api": true,
        "templates": true
    },
    "ui": {
        "endpoint": "/",
        "port": "$N8N_PORT"
    },
    "webhook": {
        "url": "$WEBHOOK_URL"
    },
    "api": {
        "version": "v1",
        "restEndpoint": "/rest",
        "webhookEndpoint": "/webhook",
        "webhookTestEndpoint": "/webhook-test"
    },
    "container": {
        "name": "$N8N_CONTAINER_NAME",
        "image": "$N8N_IMAGE"
    }
}
EOF
)
    
    resources::update_config "automation" "n8n" "$N8N_BASE_URL" "$additional_config"
}

#######################################
# Complete n8n installation
#######################################
n8n::install() {
    log::header "🤖 Installing n8n Workflow Automation (Docker)"
    
    # Start rollback context
    resources::start_rollback_context "install_n8n_docker"
    
    # Check if already installed
    if n8n::container_exists && n8n::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "n8n is already installed and running"
        log::info "Use --force yes to reinstall"
        return 0
    fi
    
    # Check Docker
    if ! n8n::check_docker; then
        return 1
    fi
    
    # Validate port assignment
    if ! resources::validate_port "n8n" "$N8N_PORT"; then
        log::error "Port validation failed for n8n"
        log::info "You can set a custom port with: export N8N_CUSTOM_PORT=<port>"
        return 1
    fi
    
    # Build custom image if requested
    if [[ "$BUILD_IMAGE" == "yes" ]]; then
        if ! n8n::build_custom_image; then
            resources::handle_error \
                "Failed to build custom n8n image" \
                "system" \
                "Check Docker logs and Dockerfile"
            return 1
        fi
    fi
    
    # Generate password if needed
    if [[ "$BASIC_AUTH" == "yes" && -z "$AUTH_PASSWORD" ]]; then
        AUTH_PASSWORD=$(n8n::generate_password)
        log::info "Generated password for user '$AUTH_USERNAME'"
    fi
    
    # Detect webhook URL if not provided
    if [[ -z "$WEBHOOK_URL" ]]; then
        # Try to detect public IP or hostname
        if system::is_command "curl"; then
            local public_ip
            public_ip=$(curl -s -4 https://api.ipify.org 2>/dev/null || echo "")
            if [[ -n "$public_ip" ]]; then
                WEBHOOK_URL="http://$public_ip:$N8N_PORT"
                log::info "Auto-detected webhook URL: $WEBHOOK_URL"
            fi
        fi
        
        # Fallback to localhost
        if [[ -z "$WEBHOOK_URL" ]]; then
            WEBHOOK_URL="$N8N_BASE_URL"
            log::warn "Using localhost for webhook URL - external webhooks may not work"
        fi
    fi
    
    # Create directories
    if ! n8n::create_directories; then
        resources::handle_error \
            "Failed to create n8n directories" \
            "system" \
            "Check directory permissions"
        return 1
    fi
    
    # Create Docker network
    n8n::create_network
    
    # Start PostgreSQL if needed
    if [[ "$DATABASE_TYPE" == "postgres" ]]; then
        if ! n8n::start_postgres; then
            resources::handle_error \
                "Failed to start PostgreSQL" \
                "system" \
                "Check Docker logs"
            return 1
        fi
    fi
    
    # Start n8n container
    if ! n8n::start_container "$WEBHOOK_URL" "$AUTH_PASSWORD"; then
        resources::handle_error \
            "Failed to start n8n container" \
            "system" \
            "Check Docker logs: docker logs $N8N_CONTAINER_NAME"
        return 1
    fi
    
    # Wait for service to be ready
    log::info "Waiting for n8n to start..."
    
    # Wait for container to be running and port to be available
    local wait_time=0
    local max_wait=60
    while [ $wait_time -lt $max_wait ]; do
        if n8n::is_running && ss -tlnp 2>/dev/null | grep -q ":$N8N_PORT"; then
            log::info "Container is running and port is bound"
            break
        fi
        sleep 2
        wait_time=$((wait_time + 2))
        echo -n "."
    done
    echo
    
    if [ $wait_time -ge $max_wait ]; then
        resources::handle_error \
            "n8n failed to start within timeout" \
            "system" \
            "Check container logs for errors"
        return 1
    fi
    
    # Give n8n time to initialize and run migrations
    log::info "Waiting for n8n to complete initialization..."
    sleep 10
    
    if n8n::is_healthy; then
        log::success "✅ n8n is running and healthy on port $N8N_PORT"
        
        # Display access information
        echo
        log::header "🌐 n8n Access Information"
        log::info "URL: $N8N_BASE_URL"
        log::info "Webhook URL: $WEBHOOK_URL"
        
        if [[ "$BASIC_AUTH" == "yes" ]]; then
            log::info "Username: $AUTH_USERNAME"
            if [[ -n "$AUTH_PASSWORD" ]]; then
                log::warn "Password: $AUTH_PASSWORD (save this password!)"
            fi
        else
            log::warn "⚠️  No authentication enabled - n8n is publicly accessible"
        fi
        
        if [[ "$DATABASE_TYPE" == "postgres" ]]; then
            log::info "Database: PostgreSQL (persistent)"
        else
            log::info "Database: SQLite (file-based)"
        fi
        
        # Update Vrooli configuration
        if ! n8n::update_config; then
            log::warn "Failed to update Vrooli configuration"
            log::info "n8n is installed but may need manual configuration in Vrooli"
        fi
        
        # Clear rollback context on success
        ROLLBACK_ACTIONS=()
        OPERATION_ID=""
        
        echo
        log::header "🎯 Next Steps"
        log::info "1. Access n8n at: $N8N_BASE_URL"
        log::info "2. Create your first workflow"
        log::info "3. Configure webhook integrations using: $WEBHOOK_URL"
        log::info "4. Check the docs: https://docs.n8n.io"
        
        return 0
    else
        log::warn "n8n started but health check failed"
        log::info "Check logs: docker logs $N8N_CONTAINER_NAME"
        return 0
    fi
}

#######################################
# Uninstall n8n
#######################################
n8n::uninstall() {
    log::header "🗑️  Uninstalling n8n"
    
    if ! flow::is_yes "$YES"; then
        log::warn "This will remove n8n and all workflow data"
        read -p "Are you sure you want to continue? (y/N): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Uninstall cancelled"
            return 0
        fi
    fi
    
    # Stop and remove n8n container
    if n8n::container_exists; then
        log::info "Removing n8n container..."
        docker stop "$N8N_CONTAINER_NAME" 2>/dev/null || true
        docker rm "$N8N_CONTAINER_NAME" 2>/dev/null || true
        log::success "n8n container removed"
    fi
    
    # Stop and remove PostgreSQL container
    if docker ps -a --format '{{.Names}}' | grep -q "^${N8N_DB_CONTAINER_NAME}$"; then
        log::info "Removing PostgreSQL container..."
        docker stop "$N8N_DB_CONTAINER_NAME" 2>/dev/null || true
        docker rm "$N8N_DB_CONTAINER_NAME" 2>/dev/null || true
        log::success "PostgreSQL container removed"
    fi
    
    # Remove Docker network
    if docker network ls | grep -q "$N8N_NETWORK_NAME"; then
        log::info "Removing Docker network..."
        docker network rm "$N8N_NETWORK_NAME" 2>/dev/null || true
    fi
    
    # Backup data before removal
    if [[ -d "$N8N_DATA_DIR" ]]; then
        local backup_dir="$HOME/n8n-backup-$(date +%Y%m%d-%H%M%S)"
        log::info "Backing up n8n data to: $backup_dir"
        cp -r "$N8N_DATA_DIR" "$backup_dir" 2>/dev/null || true
    fi
    
    # Remove data directory
    read -p "Remove n8n data directory? (y/N): " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$N8N_DATA_DIR" 2>/dev/null || true
        log::info "Data directory removed"
    fi
    
    # Remove from Vrooli config
    resources::remove_config "automation" "n8n"
    
    log::success "✅ n8n uninstalled successfully"
}

#######################################
# Upgrade n8n to latest version
#######################################
n8n::upgrade() {
    log::header "⬆️  Upgrading n8n"
    
    if ! n8n::container_exists; then
        log::error "n8n is not installed"
        return 1
    fi
    
    # Pull latest image
    log::info "Pulling latest n8n image..."
    if ! docker pull "$N8N_IMAGE"; then
        log::error "Failed to pull latest image"
        return 1
    fi
    
    # Get current container configuration
    local current_config
    current_config=$(docker inspect "$N8N_CONTAINER_NAME" 2>/dev/null)
    
    # Stop current container
    n8n::stop
    
    # Backup data
    local backup_dir="${N8N_DATA_DIR}-backup-$(date +%Y%m%d-%H%M%S)"
    log::info "Backing up data to: $backup_dir"
    cp -r "$N8N_DATA_DIR" "$backup_dir" 2>/dev/null || true
    
    # Remove old container
    docker rm "$N8N_CONTAINER_NAME" 2>/dev/null || true
    
    # Start with new image (reuse existing configuration)
    n8n::start
    
    if n8n::is_healthy; then
        log::success "✅ n8n upgraded successfully"
        n8n::version
    else
        log::error "Upgrade may have failed - check logs"
        return 1
    fi
}

#######################################
# Reinstall n8n (preserving data)
#######################################
n8n::reinstall() {
    log::header "🔄 Reinstalling n8n"
    
    # Backup current data
    if [[ -d "$N8N_DATA_DIR" ]]; then
        local backup_dir="${N8N_DATA_DIR}-backup-$(date +%Y%m%d-%H%M%S)"
        log::info "Backing up data to: $backup_dir"
        cp -r "$N8N_DATA_DIR" "$backup_dir" || {
            log::error "Failed to backup data"
            return 1
        }
    fi
    
    # Force uninstall (preserving data)
    YES=yes n8n::uninstall
    
    # Restore data
    if [[ -d "$backup_dir" ]]; then
        log::info "Restoring data..."
        mkdir -p "$N8N_DATA_DIR"
        cp -r "$backup_dir"/* "$N8N_DATA_DIR/" 2>/dev/null || true
    fi
    
    # Install fresh
    FORCE=yes n8n::install
}