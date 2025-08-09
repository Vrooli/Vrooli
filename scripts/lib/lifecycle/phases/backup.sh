#!/usr/bin/env bash
################################################################################
# Universal Backup Phase Handler
# 
# Handles generic backup tasks:
# - Backup directory creation and validation
# - Common backup utilities setup
# - Backup rotation and cleanup
# - Generic database backup orchestration
#
# App-specific logic should be in app/lifecycle/backup.sh
################################################################################

set -euo pipefail

# Get script directory
LIB_LIFECYCLE_PHASES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${LIB_LIFECYCLE_PHASES_DIR}/../../utils/var.sh"
# shellcheck disable=SC1091
source "${LIB_LIFECYCLE_PHASES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"

################################################################################
# Backup Functions
################################################################################

#######################################
# Create backup session directory
# Globals:
#   BACKUP_DIR
#   BACKUP_SESSION_DIR
#   BACKUP_TIMESTAMP
# Returns:
#   0 on success
#######################################
backup::create_backup_directory() {
    local backup_base_dir="${BACKUP_DIR:-${var_ROOT_DIR}/backups}"
    local timestamp=$(date +"%Y%m%d_%H%M%S")
    
    export BACKUP_SESSION_DIR="${backup_base_dir}/${timestamp}"
    export BACKUP_TIMESTAMP="$timestamp"
    
    log::info "Creating backup directory: ${BACKUP_SESSION_DIR}"
    
    if ! mkdir -p "${BACKUP_SESSION_DIR}"; then
        log::error "Failed to create backup directory: ${BACKUP_SESSION_DIR}"
        exit $ERROR_BACKUP_DIRECTORY_CREATION
    fi
    
    # Ensure proper permissions
    chmod 750 "${BACKUP_SESSION_DIR}"
    
    log::success "✅ Backup directory created: ${BACKUP_SESSION_DIR}"
}

#######################################
# Check and setup backup tools
# Globals:
#   RCLONE_AVAILABLE
#   PG_DUMP_AVAILABLE
# Returns:
#   0 on success, exits on missing required tools
#######################################
backup::setup_backup_tools() {
    log::info "Checking backup tool availability..."
    
    local missing_tools=()
    
    # Check for required backup tools
    if ! command -v tar &> /dev/null; then
        missing_tools+=("tar")
    fi
    
    if ! command -v gzip &> /dev/null; then
        missing_tools+=("gzip")
    fi
    
    # Check for optional tools
    if command -v rclone &> /dev/null; then
        log::info "✅ rclone available for remote backup sync"
        export RCLONE_AVAILABLE="true"
    else
        log::info "ℹ️  rclone not available (optional for remote backups)"
        export RCLONE_AVAILABLE="false"
    fi
    
    if command -v pg_dump &> /dev/null; then
        log::info "✅ pg_dump available for PostgreSQL backups"
        export PG_DUMP_AVAILABLE="true"
    else
        log::info "ℹ️  pg_dump not available (needed for PostgreSQL backups)"
        export PG_DUMP_AVAILABLE="false"
    fi
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log::error "Missing required backup tools: ${missing_tools[*]}"
        log::error "Please install missing tools and retry backup"
        exit $ERROR_MISSING_BACKUP_TOOLS
    fi
    
    log::success "✅ Backup tools check complete"
}

#######################################
# Rotate old backup directories based on retention
# Globals:
#   BACKUP_DIR
#   BACKUP_RETENTION_DAYS
# Returns:
#   0 on success
#######################################
backup::rotate_old_backups() {
    local backup_base_dir="${BACKUP_DIR:-${var_ROOT_DIR}/backups}"
    local retention_days="${BACKUP_RETENTION_DAYS:-30}"
    
    log::info "Rotating old backups (retention: ${retention_days} days)..."
    
    if [[ ! -d "$backup_base_dir" ]]; then
        log::info "No backup directory found, skipping rotation"
        return 0
    fi
    
    # Find and remove backups older than retention period
    local old_backups
    old_backups=$(find "$backup_base_dir" -maxdepth 1 -type d -name "[0-9]*_[0-9]*" -mtime +"$retention_days" || true)
    
    if [[ -n "$old_backups" ]]; then
        log::info "Removing old backups:"
        echo "$old_backups" | while IFS= read -r backup_dir; do
            log::info "  Removing: $(basename "$backup_dir")"
            rm -rf "$backup_dir"
        done
        log::success "✅ Old backup cleanup complete"
    else
        log::info "No old backups to remove"
    fi
}

#######################################
# Verify backup environment prerequisites
# Globals:
#   BACKUP_DIR
#   BACKUP_MIN_SPACE_GB
# Returns:
#   0 on success, exits on insufficient resources
#######################################
backup::verify_backup_environment() {
    log::info "Verifying backup environment..."
    
    # Check disk space
    local backup_base_dir="${BACKUP_DIR:-${var_ROOT_DIR}/backups}"
    local required_space="${BACKUP_MIN_SPACE_GB:-5}"
    
    # Get available space, handling cases where directory doesn't exist yet
    local parent_dir="$backup_base_dir"
    if [[ ! -d "$parent_dir" ]]; then
        parent_dir="$(dirname "$backup_base_dir")"
    fi
    
    local available_space
    if available_space=$(df -BG "$parent_dir" 2>/dev/null | awk 'NR==2 {print $4}' | sed 's/G//'); then
        if [[ "$available_space" -lt "$required_space" ]]; then
            log::error "Insufficient disk space for backup"
            log::error "Available: ${available_space}GB, Required: ${required_space}GB"
            exit $ERROR_INSUFFICIENT_DISK_SPACE
        fi
        log::info "✅ Disk space check: ${available_space}GB available"
    else
        log::warning "Unable to check disk space - proceeding with backup"
        log::info "ℹ️  To enable disk space checks, ensure the backup directory is accessible"
    fi
    
    # Verify backup directory permissions
    if [[ ! -w "$(dirname "$backup_base_dir")" ]]; then
        log::error "No write permission for backup directory: $backup_base_dir"
        exit $ERROR_BACKUP_PERMISSION_DENIED
    fi
    
    log::success "✅ Backup environment verification complete"
}

################################################################################
# Main Backup Logic
################################################################################

#######################################
# Run universal backup tasks
# Handles generic backup preparation
# Globals:
#   BACKUP_SESSION_DIR
# Returns:
#   0 on success, 1 on failure
#######################################
backup::universal::main() {
    # Initialize phase
    phase::init "Backup"
    
    log::header "🔄 Universal Backup Phase"
    
    # Verify environment
    backup::verify_backup_environment
    
    # Setup backup tools
    backup::setup_backup_tools
    
    # Create backup session directory
    backup::create_backup_directory
    
    # Rotate old backups
    backup::rotate_old_backups
    
    # Complete phase
    phase::complete
    
    log::success "✅ Universal backup preparation complete"
    log::info "Backup session directory: ${BACKUP_SESSION_DIR}"
    log::info "Next: App-specific backup logic will execute"
    
    return 0
}

#######################################
# Entry point for direct execution
#######################################
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Check if being called by lifecycle engine
    if [[ "${LIFECYCLE_PHASE:-}" == "backup" ]]; then
        backup::universal::main "$@"
    else
        log::error "This script should be called through the lifecycle engine"
        log::info "Use: ./scripts/manage.sh backup [options]"
        exit 1
    fi
fi