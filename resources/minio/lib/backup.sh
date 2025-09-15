#!/usr/bin/env bash
################################################################################
# MinIO Backup and Restore Library
# 
# Provides backup and restore functionality for MinIO data preservation
################################################################################

set -euo pipefail

# Ensure dependencies are loaded
if [[ -z "${MINIO_CLI_DIR:-}" ]]; then
    MINIO_CLI_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}/.." && builtin pwd)"
fi

################################################################################
# Backup Functions
################################################################################

minio::backup::create() {
    local backup_name="${1:-}"
    local backup_dir="${2:-${HOME}/.minio/backups}"
    
    if [[ -z "$backup_name" ]]; then
        backup_name="minio-backup-$(date +%Y%m%d-%H%M%S)"
    fi
    
    log::info "Creating MinIO backup: $backup_name"
    
    # Check if MinIO is installed
    if ! minio::is_installed; then
        log::error "MinIO is not installed"
        return 1
    fi
    
    # Create backup directory if it doesn't exist
    mkdir -p "$backup_dir"
    
    local backup_path="$backup_dir/$backup_name"
    local container_name="${MINIO_CONTAINER_NAME:-minio}"
    local data_dir="${HOME}/.minio/data"
    
    log::info "Backing up MinIO data to: $backup_path"
    
    # Create backup directory
    mkdir -p "$backup_path"
    
    # Backup the actual data directory if it exists
    if [[ -d "$data_dir" ]]; then
        log::info "Backing up data directory..."
        mkdir -p "$backup_path/data"
        cp -r "$data_dir"/* "$backup_path/data/" 2>/dev/null || true
        log::success "✓ Data directory backed up"
    fi
    
    # Load actual credentials (avoid readonly variable issue)
    local creds_file="${HOME}/.minio/config/credentials"
    local root_user="minioadmin"
    local root_password="minioadmin"
    
    if [[ -f "$creds_file" ]]; then
        root_user=$(grep "^MINIO_ROOT_USER=" "$creds_file" | cut -d= -f2- || echo "minioadmin")
        root_password=$(grep "^MINIO_ROOT_PASSWORD=" "$creds_file" | cut -d= -f2- || echo "minioadmin")
    fi
    
    # Note: Bucket information is already contained in the data directory backup
    log::info "Bucket data included in data directory backup"
    
    # Save credentials for restore
    local creds_backup="$backup_path/credentials.env"
    {
        echo "MINIO_ROOT_USER=$root_user"
        echo "MINIO_ROOT_PASSWORD=$root_password"
        echo "BACKUP_DATE=$(date -Iseconds)"
        echo "MINIO_VERSION=$(docker inspect "$container_name" --format '{{.Config.Image}}' 2>/dev/null || echo 'unknown')"
    } > "$creds_backup"
    chmod 600 "$creds_backup"
    
    # Create backup metadata
    local metadata_file="$backup_path/metadata.json"
    {
        echo "{"
        echo "  \"backup_name\": \"$backup_name\","
        echo "  \"backup_date\": \"$(date -Iseconds)\","
        echo "  \"data_dir\": \"$data_dir\","
        echo "  \"backup_type\": \"full\""
        echo "}"
    } > "$metadata_file"
    
    log::success "Backup completed: $backup_path"
    echo "Backup location: $backup_path"
    
    # Show backup size
    local backup_size
    backup_size=$(du -sh "$backup_path" 2>/dev/null | cut -f1)
    echo "Backup size: $backup_size"
    
    return 0
}

minio::backup::list() {
    local backup_dir="${1:-${HOME}/.minio/backups}"
    
    log::info "Available MinIO backups:"
    
    if [[ ! -d "$backup_dir" ]]; then
        log::warning "No backup directory found at: $backup_dir"
        return 0
    fi
    
    local count=0
    for backup in "$backup_dir"/*; do
        if [[ -d "$backup" ]] && [[ -f "$backup/metadata.json" ]]; then
            local backup_name=$(basename "$backup")
            local backup_date=$(jq -r '.backup_date // "unknown"' "$backup/metadata.json" 2>/dev/null)
            local backup_type=$(jq -r '.backup_type // "unknown"' "$backup/metadata.json" 2>/dev/null)
            local backup_size=$(du -sh "$backup" 2>/dev/null | cut -f1)
            
            echo ""
            echo "  📦 $backup_name"
            echo "     Date: $backup_date"
            echo "     Type: $backup_type"
            echo "     Size: $backup_size"
            ((count++))
        fi
    done
    
    if [[ $count -eq 0 ]]; then
        log::warning "No backups found in: $backup_dir"
    else
        echo ""
        echo "Total backups: $count"
    fi
    
    return 0
}

minio::backup::restore() {
    local backup_name="${1:-}"
    local backup_dir="${2:-${HOME}/.minio/backups}"
    
    if [[ -z "$backup_name" ]]; then
        log::error "Backup name required"
        echo "Usage: resource-minio backup restore <backup-name>"
        minio::backup::list "$backup_dir"
        return 1
    fi
    
    local backup_path="$backup_dir/$backup_name"
    
    if [[ ! -d "$backup_path" ]]; then
        log::error "Backup not found: $backup_path"
        minio::backup::list "$backup_dir"
        return 1
    fi
    
    if [[ ! -f "$backup_path/metadata.json" ]]; then
        log::error "Invalid backup: missing metadata.json"
        return 1
    fi
    
    log::info "Restoring MinIO backup: $backup_name"
    
    local data_dir="${HOME}/.minio/data"
    local container_name="${MINIO_CONTAINER_NAME:-minio}"
    
    # Stop MinIO if running (to safely restore data)
    if minio::is_running; then
        log::info "Stopping MinIO for data restore..."
        docker stop "$container_name" &>/dev/null || true
    fi
    
    # Restore data directory if backup exists
    if [[ -d "$backup_path/data" ]]; then
        log::info "Restoring data directory..."
        
        # Backup current data (just in case)
        if [[ -d "$data_dir" ]]; then
            mv "$data_dir" "${data_dir}.backup-$(date +%Y%m%d-%H%M%S)" 2>/dev/null || true
        fi
        
        # Create data directory
        mkdir -p "$data_dir"
        
        # Restore backup data
        cp -r "$backup_path/data"/* "$data_dir/" 2>/dev/null || true
        log::success "✓ Data directory restored"
    fi
    
    # Restore credentials if available
    if [[ -f "$backup_path/credentials.env" ]]; then
        log::info "Restoring credentials..."
        local creds_dir="${HOME}/.minio/config"
        mkdir -p "$creds_dir"
        cp "$backup_path/credentials.env" "$creds_dir/credentials" 2>/dev/null || true
        chmod 600 "$creds_dir/credentials"
        log::success "✓ Credentials restored"
    fi
    
    # Restart MinIO
    log::info "Starting MinIO with restored data..."
    docker start "$container_name" &>/dev/null || true
    
    # Wait for MinIO to be healthy
    local max_attempts=30
    local attempt=0
    while [[ $attempt -lt $max_attempts ]]; do
        if minio::is_healthy; then
            log::success "✓ MinIO started successfully with restored data"
            break
        fi
        sleep 1
        ((attempt++))
    done
    
    if [[ $attempt -eq $max_attempts ]]; then
        log::warning "MinIO took longer than expected to start"
    fi
    
    # Verify MinIO is working
    log::info "Verifying MinIO status..."
    vrooli resource minio status
    
    log::success "Restore completed successfully"
    return 0
}

minio::backup::delete() {
    local backup_name="${1:-}"
    local backup_dir="${2:-${HOME}/.minio/backups}"
    
    if [[ -z "$backup_name" ]]; then
        log::error "Backup name required"
        echo "Usage: resource-minio backup delete <backup-name>"
        minio::backup::list "$backup_dir"
        return 1
    fi
    
    local backup_path="$backup_dir/$backup_name"
    
    if [[ ! -d "$backup_path" ]]; then
        log::error "Backup not found: $backup_path"
        return 1
    fi
    
    log::info "Deleting backup: $backup_name"
    
    # Show backup info before deletion
    if [[ -f "$backup_path/metadata.json" ]]; then
        local backup_date=$(jq -r '.backup_date // "unknown"' "$backup_path/metadata.json" 2>/dev/null)
        local bucket_count=$(jq -r '.buckets | length' "$backup_path/metadata.json" 2>/dev/null || echo "0")
        echo "  Date: $backup_date"
        echo "  Buckets: $bucket_count"
    fi
    
    # Confirm deletion
    echo ""
    read -p "Are you sure you want to delete this backup? (y/N) " -n 1 -r
    echo ""
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log::info "Deletion cancelled"
        return 0
    fi
    
    # Delete backup
    rm -rf "$backup_path"
    
    log::success "Backup deleted: $backup_name"
    return 0
}

################################################################################
# Export Functions
################################################################################

# Make functions available
export -f minio::backup::create
export -f minio::backup::list
export -f minio::backup::restore
export -f minio::backup::delete