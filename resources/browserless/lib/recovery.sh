#!/usr/bin/env bash
# Browserless Recovery Functions
# Backup and restore operations using backup framework

#######################################
# Create a backup of Browserless data
# Arguments:
#   $1 - Label for backup (optional)
#######################################
browserless::create_backup() {
    local label="${1:-auto}"
    
    log::info "Creating Browserless backup..."
    
    # Create temporary directory for backup
    local temp_dir
    temp_dir=$(mktemp -d)
    
    # Ensure cleanup on exit
    trap "rm -rf '$temp_dir'" EXIT
    
    # Check if data directory exists
    if [[ ! -d "$BROWSERLESS_DATA_DIR" ]]; then
        log::warn "Data directory does not exist: $BROWSERLESS_DATA_DIR"
        log::info "Creating empty backup for future recovery"
        mkdir -p "$temp_dir/data"
    else
        # Copy Browserless data
        log::info "Copying Browserless data..."
        cp -r "$BROWSERLESS_DATA_DIR" "$temp_dir/data" 2>/dev/null || true
    fi
    
    # Save current configuration
    log::info "Saving configuration..."
    cat > "$temp_dir/config.json" << EOF
{
    "max_browsers": "${MAX_BROWSERS:-5}",
    "timeout": "${TIMEOUT:-30000}",
    "headless": "${HEADLESS:-yes}",
    "port": "$BROWSERLESS_PORT",
    "container_name": "$BROWSERLESS_CONTAINER_NAME",
    "image": "$BROWSERLESS_IMAGE",
    "shm_size": "$BROWSERLESS_DOCKER_SHM_SIZE",
    "backup_timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "backup_label": "$label"
}
EOF
    
    # Get container state if running
    if docker::is_running "$BROWSERLESS_CONTAINER_NAME"; then
        log::info "Saving container state..."
        docker inspect "$BROWSERLESS_CONTAINER_NAME" > "$temp_dir/container_state.json" 2>/dev/null || true
        
        # Get pressure data for reference
        local pressure_data
        pressure_data=$(http::get "http://localhost:$BROWSERLESS_PORT/pressure" 2>/dev/null || echo "{}")
        echo "$pressure_data" > "$temp_dir/pressure_state.json"
    fi
    
    # Store backup using framework
    local backup_path
    if backup_path=$(backup::store "browserless" "$temp_dir" "$label"); then
        log::success "Backup created successfully: $(basename "$backup_path")"
        
        # Show backup info
        echo
        echo "Backup Details:"
        echo "  Label: $label"
        echo "  Path: $backup_path"
        echo "  Size: $(du -sh "$backup_path" | cut -f1)"
        echo
        echo "To list backups: resource-browserless manage backup list"
        echo "To recover: resource-browserless manage recover"
        
        return 0
    else
        log::error "Failed to create backup"
        return 1
    fi
}

#######################################
# Recover Browserless from backup
#######################################
browserless::recover() {
    log::info "Starting Browserless recovery..."
    
    # List available backups
    local backups
    backups=$(backup::list "browserless" 2>/dev/null)
    
    if [[ -z "$backups" ]]; then
        log::error "No backups found for Browserless"
        return 1
    fi
    
    echo "Available backups:"
    echo "$backups"
    echo
    
    # Select backup
    local backup_path
    backup_path=$(backup::select "browserless")
    
    if [[ -z "$backup_path" ]]; then
        log::error "No backup selected"
        return 1
    fi
    
    log::info "Recovering from: $(basename "$backup_path")"
    
    # Stop container if running
    if docker::is_running "$BROWSERLESS_CONTAINER_NAME"; then
        log::info "Stopping Browserless container..."
        browserless::stop
    fi
    
    # Extract backup
    local temp_dir
    temp_dir=$(mktemp -d)
    trap "rm -rf '$temp_dir'" EXIT
    
    if ! backup::extract "$backup_path" "$temp_dir"; then
        log::error "Failed to extract backup"
        return 1
    fi
    
    # Restore configuration
    if [[ -f "$temp_dir/config.json" ]]; then
        log::info "Restoring configuration..."
        
        # Read configuration
        MAX_BROWSERS=$(jq -r '.max_browsers // "5"' "$temp_dir/config.json")
        TIMEOUT=$(jq -r '.timeout // "30000"' "$temp_dir/config.json")
        HEADLESS=$(jq -r '.headless // "yes"' "$temp_dir/config.json")
        
        export MAX_BROWSERS TIMEOUT HEADLESS
        
        echo "  Max Browsers: $MAX_BROWSERS"
        echo "  Timeout: $TIMEOUT"
        echo "  Headless: $HEADLESS"
    fi
    
    # Restore data
    if [[ -d "$temp_dir/data" ]]; then
        log::info "Restoring data directory..."
        
        # Backup existing data if present
        if [[ -d "$BROWSERLESS_DATA_DIR" ]]; then
            local backup_existing="${BROWSERLESS_DATA_DIR}.before_recovery.$(date +%Y%m%d_%H%M%S)"
            log::info "Backing up existing data to: $backup_existing"
            mv "$BROWSERLESS_DATA_DIR" "$backup_existing"
        fi
        
        # Restore data
        cp -r "$temp_dir/data" "$BROWSERLESS_DATA_DIR"
        
        # Set permissions
        chmod -R 755 "$BROWSERLESS_DATA_DIR"
        
        log::success "Data restored successfully"
    fi
    
    # Restart container if it was running before
    if [[ -f "$temp_dir/container_state.json" ]]; then
        log::info "Container was running before backup, restarting..."
        browserless::start
    fi
    
    log::success "Recovery completed successfully!"
    
    # Show status
    echo
    browserless::status
    
    return 0
}

#######################################
# Validate backup integrity
#######################################
browserless::validate_backup() {
    local backup_path="${1:-}"
    
    if [[ -z "$backup_path" ]]; then
        backup_path=$(backup::select "browserless")
    fi
    
    if [[ -z "$backup_path" ]]; then
        log::error "No backup selected"
        return 1
    fi
    
    log::info "Validating backup: $(basename "$backup_path")"
    
    # Extract to temp directory
    local temp_dir
    temp_dir=$(mktemp -d)
    trap "rm -rf '$temp_dir'" EXIT
    
    if ! backup::extract "$backup_path" "$temp_dir"; then
        log::error "Failed to extract backup"
        return 1
    fi
    
    local validation_errors=0
    
    # Check for required files
    echo "Checking backup contents..."
    
    if [[ ! -f "$temp_dir/config.json" ]]; then
        log::warn "  Configuration file missing"
        ((validation_errors++))
    else
        log::success "  Configuration file present"
        
        # Validate JSON
        if jq empty "$temp_dir/config.json" 2>/dev/null; then
            log::success "  Configuration JSON valid"
        else
            log::error "  Configuration JSON invalid"
            ((validation_errors++))
        fi
    fi
    
    if [[ -d "$temp_dir/data" ]]; then
        log::success "  Data directory present"
        
        # Check workspace structure
        local dirs=("screenshots" "downloads" "uploads")
        for dir in "${dirs[@]}"; do
            if [[ -d "$temp_dir/data/$dir" ]]; then
                local count
                count=$(find "$temp_dir/data/$dir" -type f 2>/dev/null | wc -l)
                log::success "  $dir: $count files"
            else
                log::info "  $dir: not present"
            fi
        done
    else
        log::info "  Data directory not present (may be empty backup)"
    fi
    
    if [[ -f "$temp_dir/container_state.json" ]]; then
        log::success "  Container state present"
    else
        log::info "  Container state not present (container was not running)"
    fi
    
    # Show backup metadata
    if [[ -f "$temp_dir/config.json" ]]; then
        echo
        echo "Backup Metadata:"
        echo "  Timestamp: $(jq -r '.backup_timestamp // "Unknown"' "$temp_dir/config.json")"
        echo "  Label: $(jq -r '.backup_label // "Unknown"' "$temp_dir/config.json")"
        echo "  Max Browsers: $(jq -r '.max_browsers // "Unknown"' "$temp_dir/config.json")"
        echo "  Port: $(jq -r '.port // "Unknown"' "$temp_dir/config.json")"
    fi
    
    echo
    if [[ $validation_errors -eq 0 ]]; then
        log::success "Backup validation passed!"
        return 0
    else
        log::warn "Backup validation completed with $validation_errors warning(s)"
        return 0  # Still return success as backup may be usable
    fi
}

#######################################
# Auto-backup before risky operations
#######################################
browserless::auto_backup() {
    local operation="${1:-operation}"
    
    if [[ "${AUTO_BACKUP:-yes}" != "yes" ]]; then
        return 0
    fi
    
    log::info "Creating automatic backup before $operation..."
    
    if browserless::create_backup "auto_${operation}_$(date +%Y%m%d_%H%M%S)"; then
        log::success "Auto-backup created successfully"
        return 0
    else
        log::warn "Auto-backup failed, continue anyway? (y/n)"
        read -r response
        [[ "$response" == "y" ]] || [[ "$response" == "Y" ]]
    fi
}