#!/usr/bin/env bash
# Configuration Migration Utility
# Safely migrates configurations from user home to project root

# Source shared secrets management library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
# shellcheck disable=SC1091
source "$PROJECT_ROOT/scripts/helpers/utils/secrets.sh"

#######################################
# Configuration Migration Script
# This script safely migrates configurations from:
#   ${HOME}/.vrooli/ → ${PROJECT_ROOT}/.vrooli/
#######################################

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

#######################################
# Print colored log messages
#######################################
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

#######################################
# Show migration status overview
#######################################
show_migration_status() {
    local user_config_dir="${HOME}/.vrooli"
    local project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    
    echo
    log_info "Configuration Migration Status Check"
    echo "======================================"
    echo
    
    log_info "Source (user home): $user_config_dir"
    log_info "Target (project):   $project_config_dir"
    echo
    
    # Check if user config directory exists
    if [[ ! -d "$user_config_dir" ]]; then
        log_success "No user home configuration found - migration not needed"
        return 0
    fi
    
    # Check if project config directory exists
    if [[ ! -d "$project_config_dir" ]]; then
        log_info "Project configuration directory will be created"
    fi
    
    # Show what would be migrated
    log_info "Files/directories that would be migrated:"
    find "$user_config_dir" -type f -o -type d | while read -r item; do
        local relative_path="${item#$user_config_dir}"
        if [[ -n "$relative_path" ]]; then
            echo "  • $relative_path"
        fi
    done
    
    echo
    
    # Check for conflicts
    local conflicts=0
    if [[ -d "$project_config_dir" ]]; then
        find "$user_config_dir" -type f | while read -r source_file; do
            local relative_path="${source_file#$user_config_dir}"
            local target_file="${project_config_dir}${relative_path}"
            
            if [[ -f "$target_file" ]]; then
                if ! cmp -s "$source_file" "$target_file"; then
                    log_warn "Conflict: $relative_path (files differ)"
                    conflicts=$((conflicts + 1))
                fi
            fi
        done
    fi
    
    if [[ $conflicts -gt 0 ]]; then
        log_warn "Found $conflicts file conflicts - manual review recommended"
    else
        log_success "No conflicts detected"
    fi
}

#######################################
# Backup existing project configuration
#######################################
backup_project_config() {
    local project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    local backup_dir="${project_config_dir}.backup.$(date +%Y%m%d-%H%M%S)"
    
    if [[ -d "$project_config_dir" ]]; then
        log_info "Creating backup of existing project configuration..."
        if cp -r "$project_config_dir" "$backup_dir"; then
            log_success "Backup created: $backup_dir"
            echo "$backup_dir"
            return 0
        else
            log_error "Failed to create backup"
            return 1
        fi
    fi
    
    return 0
}

#######################################
# Perform the actual migration
#######################################
perform_migration() {
    local user_config_dir="${HOME}/.vrooli"
    local project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    local dry_run="${1:-false}"
    
    if [[ ! -d "$user_config_dir" ]]; then
        log_info "No user configuration to migrate"
        return 0
    fi
    
    if [[ "$dry_run" == "true" ]]; then
        log_info "DRY RUN - No actual changes will be made"
    fi
    
    # Create project config directory if it doesn't exist
    if [[ "$dry_run" == "false" ]]; then
        mkdir -p "$project_config_dir"
    else
        log_info "Would create: $project_config_dir"
    fi
    
    # Copy files and directories
    find "$user_config_dir" -mindepth 1 | while read -r source_item; do
        local relative_path="${source_item#$user_config_dir}"
        local target_item="${project_config_dir}${relative_path}"
        
        if [[ -d "$source_item" ]]; then
            if [[ "$dry_run" == "false" ]]; then
                mkdir -p "$target_item"
                log_info "Created directory: ${relative_path}"
            else
                log_info "Would create directory: ${relative_path}"
            fi
        elif [[ -f "$source_item" ]]; then
            if [[ -f "$target_item" ]]; then
                if cmp -s "$source_item" "$target_item"; then
                    log_info "Identical file exists, skipping: ${relative_path}"
                    continue
                else
                    log_warn "File conflict: ${relative_path}"
                    if [[ "$dry_run" == "false" ]]; then
                        local backup_file="${target_item}.backup.$(date +%Y%m%d-%H%M%S)"
                        cp "$target_item" "$backup_file"
                        log_info "Backed up existing file to: ${backup_file#$project_config_dir}"
                    else
                        log_info "Would backup existing file"
                    fi
                fi
            fi
            
            if [[ "$dry_run" == "false" ]]; then
                cp "$source_item" "$target_item"
                log_success "Migrated file: ${relative_path}"
            else
                log_info "Would migrate file: ${relative_path}"
            fi
        fi
    done
}

#######################################
# Validate migrated configuration
#######################################
validate_migration() {
    local project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    local project_config_file
    project_config_file="$(secrets::get_project_config_file)"
    
    log_info "Validating migrated configuration..."
    
    # Check if project config directory exists
    if [[ ! -d "$project_config_dir" ]]; then
        log_error "Project configuration directory not found"
        return 1
    fi
    
    # Check if main service.json exists and is valid JSON
    if [[ -f "$project_config_file" ]]; then
        if jq . "$project_config_file" >/dev/null 2>&1; then
            log_success "service.json is valid JSON"
        else
            log_error "service.json is invalid JSON"
            return 1
        fi
    else
        log_warn "service.json not found (this may be normal)"
    fi
    
    # Check if secrets.json exists and is valid JSON
    local secrets_file="${project_config_dir}/secrets.json"
    if [[ -f "$secrets_file" ]]; then
        if jq . "$secrets_file" >/dev/null 2>&1; then
            log_success "secrets.json is valid JSON"
        else
            log_error "secrets.json is invalid JSON"
            return 1
        fi
        
        # Check file permissions
        local perms
        perms=$(stat -c %a "$secrets_file")
        if [[ "$perms" == "600" ]]; then
            log_success "secrets.json has correct permissions (600)"
        else
            log_warn "secrets.json permissions: $perms (should be 600)"
            chmod 600 "$secrets_file"
            log_success "Fixed secrets.json permissions"
        fi
    else
        log_info "secrets.json not found (this may be normal)"
    fi
    
    log_success "Migration validation completed"
    return 0
}

#######################################
# Clean up user home configuration (after successful migration)
#######################################
cleanup_user_config() {
    local user_config_dir="${HOME}/.vrooli"
    local force="${1:-false}"
    
    if [[ ! -d "$user_config_dir" ]]; then
        log_info "No user configuration to clean up"
        return 0
    fi
    
    if [[ "$force" != "true" ]]; then
        echo
        log_warn "This will DELETE the old configuration directory: $user_config_dir"
        echo -n "Are you sure you want to proceed? (yes/no): "
        read -r confirm
        if [[ "$confirm" != "yes" ]]; then
            log_info "Cleanup cancelled"
            return 0
        fi
    fi
    
    # Create a final backup before deletion
    local final_backup="${user_config_dir}.final-backup.$(date +%Y%m%d-%H%M%S)"
    if cp -r "$user_config_dir" "$final_backup"; then
        log_success "Final backup created: $final_backup"
    else
        log_error "Failed to create final backup - aborting cleanup"
        return 1
    fi
    
    # Remove the user config directory
    if rm -rf "$user_config_dir"; then
        log_success "User configuration directory removed"
        log_info "Final backup preserved at: $final_backup"
    else
        log_error "Failed to remove user configuration directory"
        return 1
    fi
}

#######################################
# Main migration function
#######################################
main() {
    local action="${1:-status}"
    local dry_run="${2:-false}"
    
    case "$action" in
        "status")
            show_migration_status
            ;;
        "migrate")
            log_info "Starting configuration migration..."
            echo
            
            # Show status first
            show_migration_status
            echo
            
            # Confirm migration
            if [[ "$dry_run" != "true" ]]; then
                echo -n "Proceed with migration? (yes/no): "
                read -r confirm
                if [[ "$confirm" != "yes" ]]; then
                    log_info "Migration cancelled"
                    exit 0
                fi
            fi
            
            # Backup existing project config
            local backup_path
            backup_path=$(backup_project_config)
            
            # Perform migration
            if perform_migration "$dry_run"; then
                if [[ "$dry_run" == "false" ]]; then
                    log_success "Migration completed"
                    
                    # Validate migration
                    if validate_migration; then
                        log_success "Migration validation passed"
                    else
                        log_error "Migration validation failed"
                        exit 1
                    fi
                else
                    log_info "Dry run completed"
                fi
            else
                log_error "Migration failed"
                exit 1
            fi
            ;;
        "cleanup")
            cleanup_user_config "$dry_run"
            ;;
        "dry-run")
            main "migrate" "true"
            ;;
        *)
            echo "Usage: $0 [status|migrate|cleanup|dry-run]"
            echo
            echo "Commands:"
            echo "  status   - Show current migration status"
            echo "  migrate  - Perform the migration"
            echo "  cleanup  - Remove old user configuration (after migration)"
            echo "  dry-run  - Show what would be migrated without making changes"
            echo
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"