#!/usr/bin/env bash
set -euo pipefail

BACKUP_APP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${BACKUP_APP_DIR}/../../lib/utils/var.sh"

# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"
# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/env.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

#######################################
# Vrooli-specific backup logic
# 
# This handles Vrooli application-specific backup operations:
# - PostgreSQL database backups
# - Redis data backups  
# - Configuration files backup
# - User uploads backup (if using local storage)
# - JWT keys backup
#######################################

backup::vrooli_database() {
    log::info "Creating PostgreSQL database backup..."
    
    if [[ "${PG_DUMP_AVAILABLE:-false}" != "true" ]]; then
        log::error "pg_dump not available - cannot backup PostgreSQL database"
        log::error "Install postgresql-client package and retry"
        exit $ERROR_MISSING_BACKUP_TOOLS
    fi
    
    # Get database connection info from environment or defaults
    local db_host="${DATABASE_HOST:-localhost}"
    local db_port="${DATABASE_PORT:-5432}"
    local db_name="${DATABASE_NAME:-vrooli}"
    local db_user="${DATABASE_USER:-vrooli}"
    
    # Use environment variable for password
    export PGPASSWORD="${DATABASE_PASSWORD:-}"
    
    local backup_file="${BACKUP_SESSION_DIR}/postgres_${BACKUP_TIMESTAMP}.sql"
    
    log::info "Backing up database: $db_name@$db_host:$db_port"
    
    if pg_dump \
        --host="$db_host" \
        --port="$db_port" \
        --username="$db_user" \
        --dbname="$db_name" \
        --no-password \
        --verbose \
        --clean \
        --if-exists \
        --create \
        --format=plain \
        --file="$backup_file"; then
        
        log::success "✅ Database backup created: $(basename "$backup_file")"
        
        # Compress the backup
        if gzip "$backup_file"; then
            log::success "✅ Database backup compressed: $(basename "$backup_file").gz"
        fi
    else
        log::error "Failed to create database backup"
        exit $ERROR_DATABASE_BACKUP_FAILED
    fi
    
    # Clear password variable
    unset PGPASSWORD
}

backup::vrooli_redis() {
    log::info "Creating Redis data backup..."
    
    local redis_host="${REDIS_HOST:-localhost}"
    local redis_port="${REDIS_PORT:-6379}"
    local redis_db="${REDIS_DB:-0}"
    
    # Try to save Redis data
    if command -v redis-cli &> /dev/null; then
        log::info "Triggering Redis BGSAVE..."
        
        if redis-cli -h "$redis_host" -p "$redis_port" BGSAVE | grep -q "Background saving started"; then
            log::info "Waiting for Redis background save to complete..."
            
            # Wait for BGSAVE to complete (check every 2 seconds, max 60 seconds)
            local wait_count=0
            local max_wait=30
            
            while [[ $wait_count -lt $max_wait ]]; do
                if redis-cli -h "$redis_host" -p "$redis_port" LASTSAVE | tail -1 > "${BACKUP_SESSION_DIR}/.redis_last_save_new"; then
                    if [[ -f "${BACKUP_SESSION_DIR}/.redis_last_save_old" ]]; then
                        if ! diff "${BACKUP_SESSION_DIR}/.redis_last_save_old" "${BACKUP_SESSION_DIR}/.redis_last_save_new" > /dev/null; then
                            log::success "✅ Redis background save completed"
                            break
                        fi
                    else
                        cp "${BACKUP_SESSION_DIR}/.redis_last_save_new" "${BACKUP_SESSION_DIR}/.redis_last_save_old"
                    fi
                fi
                
                sleep 2
                ((wait_count++))
                
                if [[ $wait_count -eq $max_wait ]]; then
                    log::warning "Redis BGSAVE timeout - backup may be incomplete"
                fi
            done
            
            # Clean up temp files
            trash::safe_remove "${BACKUP_SESSION_DIR}/.redis_last_save_old" --temp
            trash::safe_remove "${BACKUP_SESSION_DIR}/.redis_last_save_new" --temp
        else
            log::warning "Failed to trigger Redis BGSAVE - Redis data backup skipped"
        fi
    else
        log::warning "redis-cli not available - Redis data backup skipped"
    fi
}

backup::vrooli_configuration() {
    log::info "Backing up Vrooli configuration files..."
    
    local config_backup_dir="${BACKUP_SESSION_DIR}/configuration"
    mkdir -p "$config_backup_dir"
    
    # Backup important configuration files (excluding sensitive data)
    local config_files=(
        ".vrooli/service.json"
        "package.json"
        "packages/server/prisma/schema.prisma"
        "docker-compose.yml"
        "docker-compose.prod.yml"
    )
    
    for config_file in "${config_files[@]}"; do
        local full_path="${var_ROOT_DIR}/${config_file}"
        if [[ -f "$full_path" ]]; then
            local dest_dir="${config_backup_dir}/$(dirname "$config_file")"
            mkdir -p "$dest_dir"
            cp "$full_path" "$dest_dir/"
            log::info "✅ Backed up: $config_file"
        else
            log::info "ℹ️  Skipped (not found): $config_file"
        fi
    done
    
    log::success "✅ Configuration backup complete"
}

backup::vrooli_secrets() {
    log::info "Backing up JWT keys and certificates..."
    
    local secrets_backup_dir="${BACKUP_SESSION_DIR}/secrets"
    mkdir -p "$secrets_backup_dir"
    chmod 700 "$secrets_backup_dir"
    
    # Backup JWT keys
    local jwt_keys=(
        "packages/server/src/assets/keys/jwt_priv.pem"
        "packages/server/src/assets/keys/jwt_pub.pem"
    )
    
    for key_file in "${jwt_keys[@]}"; do
        local full_path="${var_ROOT_DIR}/${key_file}"
        if [[ -f "$full_path" ]]; then
            local dest_dir="${secrets_backup_dir}/$(dirname "${key_file}")"
            mkdir -p "$dest_dir"
            cp "$full_path" "$dest_dir/"
            chmod 600 "${dest_dir}/$(basename "$key_file")"
            log::info "✅ Backed up: $key_file"
        else
            log::info "ℹ️  Skipped (not found): $key_file"
        fi
    done
    
    log::success "✅ Secrets backup complete"
}

backup::vrooli_user_data() {
    log::info "Backing up user data (if using local storage)..."
    
    local uploads_dir="${var_ROOT_DIR}/data/uploads"
    
    if [[ -d "$uploads_dir" && -n "$(ls -A "$uploads_dir" 2>/dev/null)" ]]; then
        log::info "Found local uploads directory, creating backup..."
        
        local user_data_backup="${BACKUP_SESSION_DIR}/user_data_${BACKUP_TIMESTAMP}.tar.gz"
        
        if tar -czf "$user_data_backup" -C "${var_ROOT_DIR}" "data/uploads"; then
            log::success "✅ User data backup created: $(basename "$user_data_backup")"
        else
            log::error "Failed to create user data backup"
        fi
    else
        log::info "ℹ️  No local user data found (using remote storage or empty)"
    fi
}

backup::vrooli_create_manifest() {
    log::info "Creating backup manifest..."
    
    local manifest_file="${BACKUP_SESSION_DIR}/backup_manifest.json"
    local backup_size
    backup_size=$(du -sb "${BACKUP_SESSION_DIR}" | cut -f1)
    
    cat > "$manifest_file" << EOF
{
  "backup": {
    "timestamp": "${BACKUP_TIMESTAMP}",
    "date": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "version": "$(cd "${var_ROOT_DIR}" && git rev-parse HEAD 2>/dev/null || echo 'unknown')",
    "size_bytes": ${backup_size},
    "components": {
      "database": $(ls "${BACKUP_SESSION_DIR}"/postgres_*.sql.gz >/dev/null 2>&1 && echo "true" || echo "false"),
      "redis": true,
      "configuration": $(ls "${BACKUP_SESSION_DIR}/configuration" >/dev/null 2>&1 && echo "true" || echo "false"),
      "secrets": $(ls "${BACKUP_SESSION_DIR}/secrets" >/dev/null 2>&1 && echo "true" || echo "false"),
      "user_data": $(ls "${BACKUP_SESSION_DIR}"/user_data_*.tar.gz >/dev/null 2>&1 && echo "true" || echo "false")
    },
    "environment": {
      "hostname": "$(hostname)",
      "user": "$(whoami)",
      "working_directory": "${var_ROOT_DIR}"
    }
  }
}
EOF
    
    log::success "✅ Backup manifest created: $(basename "$manifest_file")"
}

backup::vrooli_schedule_production() {
    if env::in_production; then
        log::info "Setting up production backup schedule..."
        
        # Create backup script
        local backup_script="${var_ROOT_DIR}/scripts/app/utils/scheduled-backup.sh"
        cat > "$backup_script" << 'EOF'
#!/usr/bin/env bash
# Scheduled backup script for production Vrooli
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$PROJECT_ROOT"

# Run backup with error handling
if ! ./scripts/manage.sh backup; then
    echo "Backup failed at $(date)" >> "${PROJECT_ROOT}/logs/backup-errors.log"
    exit 1
fi

echo "Backup completed successfully at $(date)" >> "${PROJECT_ROOT}/logs/backup.log"
EOF
        chmod +x "$backup_script"
        
        # Add to crontab (daily at 2 AM)
        local cron_entry="0 2 * * * ${backup_script} >/dev/null 2>&1"
        
        if ! crontab -l 2>/dev/null | grep -q "$backup_script"; then
            (crontab -l 2>/dev/null; echo "$cron_entry") | crontab -
            log::success "✅ Production backup scheduled (daily at 2 AM)"
        else
            log::info "Production backup already scheduled"
        fi
        
        # Create logs directory
        mkdir -p "${var_ROOT_DIR}/logs"
    fi
}

backup::vrooli_main() {
    log::header "🔄 Vrooli Application Backup"
    
    # Verify we have required environment variables
    if [[ -z "${BACKUP_SESSION_DIR:-}" ]]; then
        log::error "BACKUP_SESSION_DIR not set - universal backup phase should run first"
        exit $ERROR_BACKUP_ENVIRONMENT
    fi
    
    # Create database backup
    backup::vrooli_database
    
    # Create Redis backup
    backup::vrooli_redis
    
    # Backup configuration
    backup::vrooli_configuration
    
    # Backup secrets
    backup::vrooli_secrets
    
    # Backup user data
    backup::vrooli_user_data
    
    # Create manifest
    backup::vrooli_create_manifest
    
    # Schedule production backups
    backup::vrooli_schedule_production
    
    local backup_size
    backup_size=$(du -sh "${BACKUP_SESSION_DIR}" | cut -f1)
    
    log::success "✅ Vrooli backup complete!"
    log::info "Backup location: ${BACKUP_SESSION_DIR}"
    log::info "Backup size: ${backup_size}"
    log::info ""
    log::info "To restore this backup, use:"
    log::info "  ./scripts/manage.sh restore --backup-dir=\"${BACKUP_SESSION_DIR}\""
}

# Execute if called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    backup::vrooli_main "$@"
fi