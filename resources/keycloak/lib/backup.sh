#!/usr/bin/env bash
################################################################################
# Keycloak Backup and Restore Functions
################################################################################

set -euo pipefail

# Determine script location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/common.sh"

# Get Keycloak port from registry
KEYCLOAK_PORT=$(source "${APP_ROOT}/scripts/resources/port_registry.sh" && port_registry::get keycloak)

# Backup directory configuration
BACKUP_DIR="${RESOURCE_DIR}/backups"
BACKUP_RETENTION_DAYS=${KEYCLOAK_BACKUP_RETENTION_DAYS:-7}
BACKUP_MAX_COUNT=${KEYCLOAK_BACKUP_MAX_COUNT:-30}
BACKUP_MIN_COUNT=${KEYCLOAK_BACKUP_MIN_COUNT:-5}

################################################################################
# Backup Functions
################################################################################

backup::init() {
    # Create backup directory if it doesn't exist
    if [[ ! -d "$BACKUP_DIR" ]]; then
        mkdir -p "$BACKUP_DIR"
        log::info "Created backup directory: $BACKUP_DIR"
    fi
}

backup::create() {
    local realm="${1:-}"
    local backup_name="${2:-}"
    
    if [[ -z "$realm" ]]; then
        log::error "Realm name is required"
        return 1
    fi
    
    backup::init
    
    # Generate backup filename
    local timestamp=$(date +%Y%m%d_%H%M%S)
    if [[ -z "$backup_name" ]]; then
        backup_name="keycloak_${realm}_${timestamp}"
    fi
    local backup_file="${BACKUP_DIR}/${backup_name}.json"
    
    log::info "Creating backup for realm: $realm"
    
    # Use the existing realm export function if available
    if [[ "${realm}" == "master" ]]; then
        # For master realm, export all realms
        local temp_export="/tmp/keycloak_export_${timestamp}.json"
        
        # Use docker exec to run export inside container
        if docker exec "${KEYCLOAK_CONTAINER_NAME}" \
            /opt/keycloak/bin/kc.sh export \
            --file "/tmp/export.json" \
            --realm "${realm}" 2>/dev/null; then
            
            # Copy exported file from container
            docker cp "${KEYCLOAK_CONTAINER_NAME}:/tmp/export.json" "$backup_file"
            docker exec "${KEYCLOAK_CONTAINER_NAME}" rm -f /tmp/export.json
            
            # Compress the backup
            gzip "$backup_file"
            backup_file="${backup_file}.gz"
            
            log::success "Backup created: $backup_file"
            echo "$backup_file"
            return 0
        fi
    fi
    
    # Fallback to API export for specific realms
    if command -v keycloak::realm::export_tenant &>/dev/null; then
        local temp_export="/tmp/keycloak_export_${timestamp}.json"
        if keycloak::realm::export_tenant "$realm" "$temp_export"; then
            mv "$temp_export" "$backup_file"
            gzip "$backup_file"
            backup_file="${backup_file}.gz"
            
            log::success "Backup created: $backup_file"
            echo "$backup_file"
            return 0
        fi
    fi
    
    log::error "Failed to export realm: $realm"
    return 1
}

backup::list() {
    backup::init
    
    log::info "Available backups:"
    
    if [[ -d "$BACKUP_DIR" ]]; then
        local count=0
        while IFS= read -r backup; do
            if [[ -n "$backup" ]]; then
                local size=$(du -h "$backup" | cut -f1)
                local modified=$(stat -c %y "$backup" 2>/dev/null | cut -d' ' -f1,2 | cut -d. -f1)
                local name=$(basename "$backup")
                
                log::info "  📦 $name"
                log::info "     Size: $size"
                log::info "     Modified: $modified"
                ((count++))
            fi
        done < <(find "$BACKUP_DIR" -name "*.json.gz" -type f 2>/dev/null | sort -r)
        
        if [[ $count -eq 0 ]]; then
            log::warning "No backups found"
        else
            log::success "Found $count backup(s)"
        fi
    else
        log::warning "Backup directory does not exist"
    fi
}

backup::restore() {
    local backup_file="${1:-}"
    local target_realm="${2:-}"
    
    if [[ -z "$backup_file" ]]; then
        log::error "Backup file is required"
        return 1
    fi
    
    # Handle relative or absolute paths
    if [[ ! "$backup_file" = /* ]]; then
        backup_file="${BACKUP_DIR}/${backup_file}"
    fi
    
    if [[ ! -f "$backup_file" ]]; then
        log::error "Backup file not found: $backup_file"
        return 1
    fi
    
    log::info "Restoring from backup: $backup_file"
    
    # Decompress if needed
    local temp_file="/tmp/keycloak_restore_$(date +%s).json"
    if [[ "$backup_file" == *.gz ]]; then
        gunzip -c "$backup_file" > "$temp_file"
    else
        cp "$backup_file" "$temp_file"
    fi
    
    # Extract realm name from backup if not provided
    if [[ -z "$target_realm" ]]; then
        target_realm=$(jq -r '.realm' "$temp_file" 2>/dev/null)
        if [[ -z "$target_realm" || "$target_realm" == "null" ]]; then
            log::error "Could not determine realm name from backup"
            rm -f "$temp_file"
            return 1
        fi
    fi
    
    # Update realm name in backup if different
    if [[ -n "$target_realm" ]]; then
        jq ".realm = \"$target_realm\"" "$temp_file" > "${temp_file}.tmp"
        mv "${temp_file}.tmp" "$temp_file"
    fi
    
    # Import the realm
    local access_token
    access_token=$(keycloak::get_admin_token) || {
        log::error "Failed to get admin token"
        rm -f "$temp_file"
        return 1
    }
    
    # Check if realm exists
    if curl -sf \
        -H "Authorization: Bearer $access_token" \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${target_realm}" &>/dev/null; then
        
        log::warning "Realm $target_realm already exists. Updating..."
        
        # Update existing realm
        if curl -sf \
            -X PUT \
            -H "Authorization: Bearer $access_token" \
            -H "Content-Type: application/json" \
            -d "@$temp_file" \
            "http://localhost:${KEYCLOAK_PORT}/admin/realms/${target_realm}"; then
            log::success "Realm updated from backup: $target_realm"
        else
            log::error "Failed to update realm from backup"
            rm -f "$temp_file"
            return 1
        fi
    else
        # Create new realm
        if curl -sf \
            -X POST \
            -H "Authorization: Bearer $access_token" \
            -H "Content-Type: application/json" \
            -d "@$temp_file" \
            "http://localhost:${KEYCLOAK_PORT}/admin/realms"; then
            log::success "Realm restored from backup: $target_realm"
        else
            log::error "Failed to restore realm from backup"
            rm -f "$temp_file"
            return 1
        fi
    fi
    
    rm -f "$temp_file"
    return 0
}

backup::cleanup() {
    local retention_days="${1:-$BACKUP_RETENTION_DAYS}"
    local max_count="${2:-$BACKUP_MAX_COUNT}"
    local min_count="${3:-$BACKUP_MIN_COUNT}"
    
    log::info "Applying backup rotation policy:"
    log::info "  - Retention: $retention_days days"
    log::info "  - Max backups: $max_count"
    log::info "  - Min backups: $min_count"
    
    if [[ ! -d "$BACKUP_DIR" ]]; then
        log::warning "Backup directory does not exist"
        return 0
    fi
    
    # Count total backups
    local total_backups=$(find "$BACKUP_DIR" -name "*.json.gz" -type f 2>/dev/null | wc -l)
    log::info "Current backup count: $total_backups"
    
    # Apply retention policy (remove old backups)
    local removed_by_age=0
    while IFS= read -r backup; do
        if [[ -n "$backup" ]]; then
            # Only remove if we'll have min_count left
            local remaining=$((total_backups - removed_by_age))
            if [[ $remaining -gt $min_count ]]; then
                log::warning "Removing old backup (age): $(basename "$backup")"
                rm -f "$backup"
                ((removed_by_age++))
            else
                log::info "Keeping backup to maintain minimum count: $(basename "$backup")"
            fi
        fi
    done < <(find "$BACKUP_DIR" -name "*.json.gz" -type f -mtime +$retention_days 2>/dev/null | sort)
    
    # Apply max count policy (remove oldest if over limit)
    local current_count=$((total_backups - removed_by_age))
    local removed_by_count=0
    
    if [[ $current_count -gt $max_count ]]; then
        local to_remove=$((current_count - max_count))
        log::info "Removing $to_remove backup(s) to stay under max count"
        
        while IFS= read -r backup; do
            if [[ $to_remove -gt 0 && -n "$backup" ]]; then
                log::warning "Removing old backup (count): $(basename "$backup")"
                rm -f "$backup"
                ((removed_by_count++))
                ((to_remove--))
            fi
        done < <(find "$BACKUP_DIR" -name "*.json.gz" -type f -printf '%T@ %p\n' 2>/dev/null | sort -n | cut -d' ' -f2)
    fi
    
    local total_removed=$((removed_by_age + removed_by_count))
    if [[ $total_removed -gt 0 ]]; then
        log::success "Removed $total_removed backup(s) (age: $removed_by_age, count: $removed_by_count)"
    else
        log::info "No backups removed"
    fi
    
    # Show final status
    local final_count=$(find "$BACKUP_DIR" -name "*.json.gz" -type f 2>/dev/null | wc -l)
    log::info "Final backup count: $final_count"
    
    return 0
}

backup::schedule() {
    local realm="${1:-master}"
    local cron_schedule="${2:-0 2 * * *}"  # Default: 2 AM daily
    
    log::info "Scheduling automatic backups for realm: $realm"
    
    # Create backup script
    local backup_script="${RESOURCE_DIR}/scripts/auto-backup.sh"
    cat > "$backup_script" <<EOF
#!/usr/bin/env bash
# Auto-generated backup script for Keycloak
source "${RESOURCE_DIR}/lib/backup.sh"
backup::create "$realm" "auto_backup_$realm"
backup::cleanup
EOF
    chmod +x "$backup_script"
    
    # Add to crontab
    local cron_entry="$cron_schedule $backup_script >> ${RESOURCE_DIR}/logs/backup.log 2>&1"
    
    # Check if already scheduled
    if crontab -l 2>/dev/null | grep -q "$backup_script"; then
        log::warning "Backup already scheduled for realm: $realm"
    else
        (crontab -l 2>/dev/null; echo "$cron_entry") | crontab -
        log::success "Backup scheduled: $cron_schedule"
    fi
}