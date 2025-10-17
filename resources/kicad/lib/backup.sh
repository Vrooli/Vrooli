#!/bin/bash
# KiCad Cloud Backup Functions (Minio Integration)

# Get script directory and APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
KICAD_BACKUP_LIB_DIR="${APP_ROOT}/resources/kicad/lib"

# Source common functions if not already sourced
if ! declare -f kicad::init_dirs &>/dev/null; then
    source "${KICAD_BACKUP_LIB_DIR}/common.sh"
fi

# Source logging functions
source "${APP_ROOT}/scripts/lib/utils/logging.sh"

# Minio configuration
MINIO_HOST="${MINIO_HOST:-localhost}"
MINIO_PORT="${MINIO_PORT:-9000}"
MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-minioadmin}"
MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-minioadmin}"
MINIO_BUCKET="${MINIO_BUCKET:-kicad-backups}"

# Check if Minio is available
kicad::backup::check_minio() {
    # Check if Minio CLI is installed
    if ! command -v mc &>/dev/null; then
        log::warning "Minio CLI (mc) not installed. Installing..."
        
        # Determine platform
        local platform=""
        if [[ "$OSTYPE" == "darwin"* ]]; then
            platform="darwin-amd64"
        else
            platform="linux-amd64"
        fi
        
        # Download and install mc
        local mc_url="https://dl.min.io/client/mc/release/${platform}/mc"
        local mc_path="${KICAD_DATA_DIR}/mc"
        
        if curl -fsSL "$mc_url" -o "$mc_path" 2>/dev/null && chmod +x "$mc_path"; then
            export PATH="${KICAD_DATA_DIR}:$PATH"
            log::success "Minio CLI installed"
        else
            log::error "Failed to install Minio CLI"
            return 1
        fi
    fi
    
    # Configure Minio alias
    mc alias set kicad-minio "http://${MINIO_HOST}:${MINIO_PORT}" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY" &>/dev/null
    
    # Check connection
    if mc admin info kicad-minio &>/dev/null; then
        # Create bucket if it doesn't exist
        mc mb "kicad-minio/${MINIO_BUCKET}" --ignore-existing &>/dev/null
        return 0
    else
        log::warning "Minio server not accessible at ${MINIO_HOST}:${MINIO_PORT}"
        return 1
    fi
}

# Backup project to Minio
kicad::backup::cloud() {
    local project="${1:-}"
    local timestamp="${2:-$(date +%Y%m%d-%H%M%S)}"
    
    if [[ -z "$project" ]]; then
        echo "Usage: resource-kicad backup cloud <project-name> [timestamp]"
        return 1
    fi
    
    kicad::init_dirs
    local project_path="${KICAD_PROJECTS_DIR}/${project}"
    
    if [[ ! -d "$project_path" ]]; then
        log::error "Project not found: $project"
        return 1
    fi
    
    # Check Minio availability
    if ! kicad::backup::check_minio; then
        log::warning "Cloud backup unavailable - Minio not accessible"
        echo "To enable cloud backup:"
        echo "  1. Start Minio: vrooli resource minio develop"
        echo "  2. Retry backup command"
        return 1
    fi
    
    echo "Creating cloud backup for project: $project"
    
    # Create tar archive
    local backup_file="/tmp/kicad-${project}-${timestamp}.tar.gz"
    tar -czf "$backup_file" -C "${KICAD_PROJECTS_DIR}" "$project" 2>/dev/null
    
    if [[ ! -f "$backup_file" ]]; then
        log::error "Failed to create backup archive"
        return 1
    fi
    
    # Upload to Minio
    local remote_path="kicad-minio/${MINIO_BUCKET}/${project}/${project}-${timestamp}.tar.gz"
    
    if mc cp "$backup_file" "$remote_path" &>/dev/null; then
        log::success "Backup uploaded to Minio: ${MINIO_BUCKET}/${project}/${project}-${timestamp}.tar.gz"
        rm -f "$backup_file"
        
        # Keep only last 5 backups
        mc ls "kicad-minio/${MINIO_BUCKET}/${project}/" --json | \
            jq -r '.key' | \
            sort -r | \
            tail -n +6 | \
            while read -r old_backup; do
                mc rm "kicad-minio/${MINIO_BUCKET}/${project}/${old_backup}" &>/dev/null
            done
        
        return 0
    else
        log::error "Failed to upload backup to Minio"
        rm -f "$backup_file"
        return 1
    fi
}

# List cloud backups
kicad::backup::list() {
    local project="${1:-}"
    
    # Check Minio availability
    if ! kicad::backup::check_minio; then
        log::warning "Cannot list backups - Minio not accessible"
        return 1
    fi
    
    if [[ -z "$project" ]]; then
        echo "All KiCad project backups:"
        mc ls "kicad-minio/${MINIO_BUCKET}/" --recursive 2>/dev/null | grep -E "\.tar\.gz$" || echo "  No backups found"
    else
        echo "Backups for project: $project"
        mc ls "kicad-minio/${MINIO_BUCKET}/${project}/" 2>/dev/null | grep -E "\.tar\.gz$" || echo "  No backups found"
    fi
}

# Restore project from cloud backup
kicad::backup::restore() {
    local project="${1:-}"
    local timestamp="${2:-latest}"
    
    if [[ -z "$project" ]]; then
        echo "Usage: resource-kicad backup restore <project-name> [timestamp|latest]"
        return 1
    fi
    
    # Check Minio availability
    if ! kicad::backup::check_minio; then
        log::error "Cannot restore - Minio not accessible"
        return 1
    fi
    
    kicad::init_dirs
    
    # Find backup file
    local backup_file=""
    if [[ "$timestamp" == "latest" ]]; then
        backup_file=$(mc ls "kicad-minio/${MINIO_BUCKET}/${project}/" --json 2>/dev/null | \
            jq -r '.key' | \
            sort -r | \
            head -n1)
    else
        backup_file="${project}-${timestamp}.tar.gz"
    fi
    
    if [[ -z "$backup_file" ]]; then
        log::error "No backup found for project: $project"
        return 1
    fi
    
    echo "Restoring project from backup: $backup_file"
    
    # Download backup
    local temp_file="/tmp/${backup_file}"
    if ! mc cp "kicad-minio/${MINIO_BUCKET}/${project}/${backup_file}" "$temp_file" &>/dev/null; then
        log::error "Failed to download backup"
        return 1
    fi
    
    # Backup existing project if it exists
    local project_path="${KICAD_PROJECTS_DIR}/${project}"
    if [[ -d "$project_path" ]]; then
        mv "$project_path" "${project_path}.bak.$(date +%Y%m%d-%H%M%S)"
        echo "Existing project backed up"
    fi
    
    # Extract backup
    if tar -xzf "$temp_file" -C "${KICAD_PROJECTS_DIR}" 2>/dev/null; then
        log::success "Project restored from backup"
        rm -f "$temp_file"
        return 0
    else
        log::error "Failed to extract backup"
        rm -f "$temp_file"
        return 1
    fi
}

# Automated backup scheduling
kicad::backup::schedule() {
    local project="${1:-}"
    local interval="${2:-daily}"  # daily, hourly, weekly
    
    if [[ -z "$project" ]]; then
        echo "Usage: resource-kicad backup schedule <project-name> [daily|hourly|weekly]"
        return 1
    fi
    
    # Create backup script
    local backup_script="${KICAD_DATA_DIR}/scheduled-backup-${project}.sh"
    cat > "$backup_script" <<EOF
#!/bin/bash
source "${APP_ROOT}/resources/kicad/lib/backup.sh"
kicad::backup::cloud "$project"
EOF
    chmod +x "$backup_script"
    
    # Add to crontab (mock implementation for now)
    echo "Backup scheduled for project: $project ($interval)"
    echo "Note: Cron scheduling would be configured here in production"
    echo "Backup script created: $backup_script"
    
    return 0
}

# Export functions for CLI framework
export -f kicad::backup::check_minio
export -f kicad::backup::cloud
export -f kicad::backup::list
export -f kicad::backup::restore
export -f kicad::backup::schedule