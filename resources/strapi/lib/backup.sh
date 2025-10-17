#!/usr/bin/env bash
# Strapi Backup and Restore Library
# Provides backup/restore capabilities for content and configuration

set -euo pipefail

# Prevent multiple sourcing
[[ -n "${STRAPI_BACKUP_LOADED:-}" ]] && return 0
readonly STRAPI_BACKUP_LOADED=1

# Source core library
BACKUP_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${BACKUP_SCRIPT_DIR}/lib/core.sh"

# Backup configuration
readonly BACKUP_DIR="${STRAPI_BACKUP_DIR:-${HOME}/.vrooli/backups/strapi}"
readonly MAX_BACKUPS="${STRAPI_MAX_BACKUPS:-10}"

#######################################
# Create backup directory
#######################################
backup::init() {
    mkdir -p "${BACKUP_DIR}"
}

#######################################
# Create full backup
#######################################
backup::create() {
    local backup_name="${1:-strapi-backup-$(date +%Y%m%d-%H%M%S)}"
    local backup_path="${BACKUP_DIR}/${backup_name}.tar.gz"
    
    backup::init
    
    core::info "Creating backup: ${backup_name}"
    
    # Check if Strapi is running
    if ! core::is_running; then
        core::error "Strapi must be running to create backup"
        return 1
    fi
    
    # Create temporary backup directory
    local temp_dir="/tmp/${backup_name}"
    mkdir -p "${temp_dir}"
    
    # Backup database
    if backup::export_database "${temp_dir}/database.sql"; then
        core::success "Database exported"
    else
        core::error "Failed to export database"
        rm -rf "${temp_dir}"
        return 1
    fi
    
    # Backup uploads (if local storage)
    if [[ -d "${STRAPI_DATA_DIR:-/app}/public/uploads" ]]; then
        cp -r "${STRAPI_DATA_DIR:-/app}/public/uploads" "${temp_dir}/uploads"
        core::success "Media files backed up"
    fi
    
    # Backup configuration
    if [[ -d "${STRAPI_CONFIG_DIR:-/app/config}" ]]; then
        cp -r "${STRAPI_CONFIG_DIR:-/app/config}" "${temp_dir}/config"
        core::success "Configuration backed up"
    fi
    
    # Backup content types
    if [[ -d "${STRAPI_DATA_DIR:-/app}/src/api" ]]; then
        cp -r "${STRAPI_DATA_DIR:-/app}/src/api" "${temp_dir}/api"
        core::success "Content types backed up"
    fi
    
    # Create compressed archive
    tar -czf "${backup_path}" -C "/tmp" "${backup_name}"
    rm -rf "${temp_dir}"
    
    # Rotate old backups
    backup::rotate
    
    core::success "Backup created: ${backup_path}"
    echo "${backup_path}"
}

#######################################
# Export database
#######################################
backup::export_database() {
    local output_file="${1}"
    
    # Get database credentials
    local db_host="${POSTGRES_HOST:-localhost}"
    local db_port="${POSTGRES_PORT:-5432}"
    local db_name="${STRAPI_DATABASE_NAME:-strapi}"
    local db_user="${POSTGRES_USER:-postgres}"
    local db_pass="${POSTGRES_PASSWORD:-postgres}"
    
    # Export using pg_dump
    if command -v pg_dump >/dev/null 2>&1; then
        PGPASSWORD="${db_pass}" pg_dump \
            -h "${db_host}" \
            -p "${db_port}" \
            -U "${db_user}" \
            -d "${db_name}" \
            --no-owner \
            --no-acl \
            > "${output_file}"
        return $?
    else
        # Try Docker if pg_dump not available
        if docker::is_available && docker ps | grep -q postgres; then
            docker exec postgres pg_dump \
                -U "${db_user}" \
                -d "${db_name}" \
                --no-owner \
                --no-acl \
                > "${output_file}"
            return $?
        else
            core::error "pg_dump not available and PostgreSQL container not running"
            return 1
        fi
    fi
}

#######################################
# Restore from backup
#######################################
backup::restore() {
    local backup_path="${1}"
    
    if [[ ! -f "${backup_path}" ]]; then
        core::error "Backup file not found: ${backup_path}"
        return 1
    fi
    
    core::info "Restoring from backup: ${backup_path}"
    
    # Stop Strapi if running
    if core::is_running; then
        core::info "Stopping Strapi for restore..."
        core::stop
    fi
    
    # Extract backup
    local temp_dir="/tmp/strapi-restore-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "${temp_dir}"
    tar -xzf "${backup_path}" -C "${temp_dir}"
    
    # Find extracted directory
    local backup_dir=$(find "${temp_dir}" -maxdepth 1 -type d -name "strapi-backup-*" | head -1)
    
    if [[ ! -d "${backup_dir}" ]]; then
        core::error "Invalid backup archive structure"
        rm -rf "${temp_dir}"
        return 1
    fi
    
    # Restore database
    if [[ -f "${backup_dir}/database.sql" ]]; then
        if backup::import_database "${backup_dir}/database.sql"; then
            core::success "Database restored"
        else
            core::error "Failed to restore database"
            rm -rf "${temp_dir}"
            return 1
        fi
    fi
    
    # Restore uploads
    if [[ -d "${backup_dir}/uploads" ]]; then
        rm -rf "${STRAPI_DATA_DIR:-/app}/public/uploads"
        cp -r "${backup_dir}/uploads" "${STRAPI_DATA_DIR:-/app}/public/"
        core::success "Media files restored"
    fi
    
    # Restore configuration
    if [[ -d "${backup_dir}/config" ]]; then
        rm -rf "${STRAPI_CONFIG_DIR:-/app/config}"
        cp -r "${backup_dir}/config" "${STRAPI_CONFIG_DIR:-/app}/"
        core::success "Configuration restored"
    fi
    
    # Restore content types
    if [[ -d "${backup_dir}/api" ]]; then
        rm -rf "${STRAPI_DATA_DIR:-/app}/src/api"
        mkdir -p "${STRAPI_DATA_DIR:-/app}/src"
        cp -r "${backup_dir}/api" "${STRAPI_DATA_DIR:-/app}/src/"
        core::success "Content types restored"
    fi
    
    # Clean up
    rm -rf "${temp_dir}"
    
    # Rebuild Strapi
    core::info "Rebuilding Strapi..."
    cd "${STRAPI_DATA_DIR:-/app}"
    npm run build
    
    # Start Strapi
    core::start
    
    core::success "Restore completed successfully"
}

#######################################
# Import database
#######################################
backup::import_database() {
    local input_file="${1}"
    
    # Get database credentials
    local db_host="${POSTGRES_HOST:-localhost}"
    local db_port="${POSTGRES_PORT:-5432}"
    local db_name="${STRAPI_DATABASE_NAME:-strapi}"
    local db_user="${POSTGRES_USER:-postgres}"
    local db_pass="${POSTGRES_PASSWORD:-postgres}"
    
    # Import using psql
    if command -v psql >/dev/null 2>&1; then
        PGPASSWORD="${db_pass}" psql \
            -h "${db_host}" \
            -p "${db_port}" \
            -U "${db_user}" \
            -d "${db_name}" \
            < "${input_file}"
        return $?
    else
        # Try Docker if psql not available
        if docker::is_available && docker ps | grep -q postgres; then
            docker exec -i postgres psql \
                -U "${db_user}" \
                -d "${db_name}" \
                < "${input_file}"
            return $?
        else
            core::error "psql not available and PostgreSQL container not running"
            return 1
        fi
    fi
}

#######################################
# List available backups
#######################################
backup::list() {
    backup::init
    
    echo "Available Strapi Backups"
    echo "========================"
    
    if [[ -d "${BACKUP_DIR}" ]]; then
        local count=0
        for backup in "${BACKUP_DIR}"/*.tar.gz; do
            if [[ -f "${backup}" ]]; then
                local name=$(basename "${backup}")
                local size=$(du -h "${backup}" | cut -f1)
                local date=$(stat -c %y "${backup}" 2>/dev/null || stat -f "%Sm" "${backup}" 2>/dev/null)
                printf "%-40s %8s  %s\n" "${name}" "${size}" "${date}"
                ((count++))
            fi
        done
        
        if [[ $count -eq 0 ]]; then
            echo "No backups found"
        else
            echo ""
            echo "Total: ${count} backup(s)"
        fi
    else
        echo "Backup directory does not exist"
    fi
}

#######################################
# Rotate old backups
#######################################
backup::rotate() {
    local backup_count=$(ls -1 "${BACKUP_DIR}"/*.tar.gz 2>/dev/null | wc -l)
    
    if [[ $backup_count -gt $MAX_BACKUPS ]]; then
        local to_delete=$((backup_count - MAX_BACKUPS))
        core::info "Rotating backups (keeping last ${MAX_BACKUPS})"
        
        ls -1t "${BACKUP_DIR}"/*.tar.gz | tail -n "$to_delete" | while read -r old_backup; do
            rm -f "${old_backup}"
            core::info "Deleted old backup: $(basename "${old_backup}")"
        done
    fi
}

#######################################
# Schedule automatic backups
#######################################
backup::schedule() {
    local frequency="${1:-daily}"  # daily, weekly, monthly
    
    core::info "Setting up automatic backups (${frequency})"
    
    # Create backup script
    local backup_script="${BACKUP_DIR}/auto-backup.sh"
    cat > "${backup_script}" << EOF
#!/usr/bin/env bash
# Automatic Strapi backup script
source "${BACKUP_SCRIPT_DIR}/lib/backup.sh"
backup::create "strapi-auto-\$(date +%Y%m%d-%H%M%S)"
EOF
    chmod +x "${backup_script}"
    
    # Add to crontab
    local cron_schedule
    case "${frequency}" in
        daily)
            cron_schedule="0 2 * * *"  # 2 AM daily
            ;;
        weekly)
            cron_schedule="0 2 * * 0"  # 2 AM Sunday
            ;;
        monthly)
            cron_schedule="0 2 1 * *"  # 2 AM first day of month
            ;;
        *)
            core::error "Invalid frequency: ${frequency}"
            return 1
            ;;
    esac
    
    # Add to crontab (if not already present)
    if ! crontab -l 2>/dev/null | grep -q "${backup_script}"; then
        (crontab -l 2>/dev/null; echo "${cron_schedule} ${backup_script}") | crontab -
        core::success "Automatic backups scheduled (${frequency})"
    else
        core::info "Automatic backups already scheduled"
    fi
}