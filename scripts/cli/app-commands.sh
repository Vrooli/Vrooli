#!/usr/bin/env bash
################################################################################
# Vrooli CLI - App Management Commands
# 
# Handles all app-related operations including status checking, protection,
# regeneration, backup, and restoration.
#
# Usage:
#   vrooli app <subcommand> [options]
#
################################################################################

set -euo pipefail

# Get CLI directory
CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
# shellcheck disable=SC1091
source "${CLI_DIR}/../../scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"

# Default apps directory
APPS_DIR="${GENERATED_APPS_DIR:-$HOME/generated-apps}"

# Show help for app commands
show_app_help() {
    cat << EOF
🚀 Vrooli App Management Commands

USAGE:
    vrooli app <subcommand> [options]

SUBCOMMANDS:
    list                    List all generated apps with their status
    status <name>           Show detailed status of a specific app
    protect <name>          Mark app as protected from auto-regeneration
    unprotect <name>        Remove protection from app
    diff <name>             Show changes from original generation
    regenerate <name>       Regenerate app from scenario
    backup <name>           Create manual backup of app
    restore <name>          Restore app from backup
    clean                   Remove all backups older than 30 days

OPTIONS:
    --help, -h              Show this help message

EXAMPLES:
    vrooli app list                           # List all apps
    vrooli app status research-assistant      # Check app status
    vrooli app protect research-assistant     # Protect from regeneration
    vrooli app diff research-assistant        # Show modifications
    vrooli app regenerate research-assistant  # Regenerate (with backup if modified)
    vrooli app backup my-app                  # Create manual backup
    vrooli app restore my-app --from latest   # Restore from latest backup

For more information: https://docs.vrooli.com/cli/app-management
EOF
}

# List all generated apps with status
app_list() {
    log::header "Generated Apps"
    
    if [[ ! -d "$APPS_DIR" ]]; then
        log::warning "No apps directory found at: $APPS_DIR"
        return 0
    fi
    
    local app_count=0
    local customized_count=0
    
    echo ""
    echo "Location: $APPS_DIR"
    echo ""
    printf "%-30s %-15s %-20s %s\n" "APP NAME" "STATUS" "LAST MODIFIED" "COMMITS"
    printf "%-30s %-15s %-20s %s\n" "--------" "------" "-------------" "-------"
    
    for app_dir in "$APPS_DIR"/*; do
        [[ ! -d "$app_dir" ]] && continue
        [[ "$(basename "$app_dir")" == ".backups" ]] && continue
        
        local app_name
        app_name=$(basename "$app_dir")
        local status="✅ Clean"
        local last_modified="Unknown"
        local commit_count="N/A"
        
        # Check git status if repo exists
        if [[ -d "$app_dir/.git" ]]; then
            cd "$app_dir" 2>/dev/null || continue
            
            # Get last modified time
            local last_mod
            last_mod=$(git log -1 --format="%ar" 2>/dev/null || echo "")
            if [[ -n "$last_mod" ]]; then
                last_modified="$last_mod"
            fi
            
            # Get commit count
            commit_count=$(git rev-list --count HEAD 2>/dev/null || echo "0")
            
            # Check for modifications
            if ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null || [[ -n "$(git ls-files --others --exclude-standard 2>/dev/null)" ]]; then
                status="🔧 Modified"
                ((customized_count++))
            elif [[ "$commit_count" -gt 1 ]]; then
                status="📝 Customized"
                ((customized_count++))
            fi
            
            # Check if protected
            if [[ -f "$app_dir/.vrooli/.protected" ]]; then
                status="$status 🔒"
            fi
            
            cd - >/dev/null 2>&1 || true
        else
            status="⚠️  No Git"
        fi
        
        printf "%-30s %-15s %-20s %s\n" "$app_name" "$status" "$last_modified" "$commit_count"
        ((app_count++))
    done
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Total: $app_count apps ($customized_count customized)"
    
    # Check for backups
    if [[ -d "$APPS_DIR/.backups" ]]; then
        local backup_count
        backup_count=$(find "$APPS_DIR/.backups" -maxdepth 1 -type d | wc -l)
        ((backup_count--)) # Subtract the .backups directory itself
        if [[ $backup_count -gt 0 ]]; then
            echo "Backups: $backup_count available"
        fi
    fi
}

# Show detailed status of an app
app_status() {
    local app_name="${1:-}"
    
    if [[ -z "$app_name" ]]; then
        log::error "App name required"
        echo "Usage: vrooli app status <app-name>"
        return 1
    fi
    
    local app_path="$APPS_DIR/$app_name"
    
    if [[ ! -d "$app_path" ]]; then
        log::error "App not found: $app_name"
        return 1
    fi
    
    log::header "App Status: $app_name"
    echo ""
    echo "📍 Location: $app_path"
    
    # Check service.json for metadata
    if [[ -f "$app_path/.vrooli/service.json" ]]; then
        local display_name version
        display_name=$(jq -r '.service.displayName // .service.name // "Unknown"' "$app_path/.vrooli/service.json" 2>/dev/null)
        version=$(jq -r '.service.version // "Unknown"' "$app_path/.vrooli/service.json" 2>/dev/null)
        echo "📋 Display Name: $display_name"
        echo "🏷️  Version: $version"
    fi
    
    # Check git status
    if [[ -d "$app_path/.git" ]]; then
        cd "$app_path" || return 1
        
        echo ""
        echo "Git Repository:"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        # Get initial commit info
        local initial_commit
        initial_commit=$(git rev-list --max-parents=0 HEAD 2>/dev/null | head -1)
        if [[ -n "$initial_commit" ]]; then
            local gen_time scenario_hash
            gen_time=$(git show "$initial_commit" --format="%ai" --no-patch 2>/dev/null)
            scenario_hash=$(git show "$initial_commit" --no-patch | grep "Scenario hash:" | cut -d: -f2 | xargs)
            
            echo "🕐 Generated: $gen_time"
            [[ -n "$scenario_hash" ]] && echo "🔗 Scenario Hash: $scenario_hash"
        fi
        
        # Commit count
        local commit_count
        commit_count=$(git rev-list --count HEAD 2>/dev/null || echo "0")
        echo "📝 Total Commits: $commit_count"
        
        # Check for uncommitted changes
        local modified_files staged_files untracked_files
        modified_files=$(git diff --name-only 2>/dev/null | wc -l)
        staged_files=$(git diff --cached --name-only 2>/dev/null | wc -l)
        untracked_files=$(git ls-files --others --exclude-standard 2>/dev/null | wc -l)
        
        echo ""
        echo "Working Directory:"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        if [[ $modified_files -eq 0 ]] && [[ $staged_files -eq 0 ]] && [[ $untracked_files -eq 0 ]]; then
            echo "✅ Clean (no uncommitted changes)"
        else
            echo "🔧 Modifications Detected:"
            [[ $modified_files -gt 0 ]] && echo "   📄 Modified files: $modified_files"
            [[ $staged_files -gt 0 ]] && echo "   📦 Staged files: $staged_files"
            [[ $untracked_files -gt 0 ]] && echo "   ❓ Untracked files: $untracked_files"
            
            # Show first few modified files
            if [[ $modified_files -gt 0 ]]; then
                echo ""
                echo "   Recent changes:"
                git status --porcelain 2>/dev/null | head -5 | sed 's/^/   /'
                [[ $(git status --porcelain 2>/dev/null | wc -l) -gt 5 ]] && echo "   ..."
            fi
        fi
        
        cd - >/dev/null 2>&1 || true
    else
        echo ""
        echo "⚠️  No Git repository found"
        echo "   App was generated before version control was added"
        echo "   Regenerate to enable git tracking"
    fi
    
    # Check protection status
    echo ""
    echo "Protection Status:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    if [[ -f "$app_path/.vrooli/.protected" ]]; then
        local protected_date
        protected_date=$(stat -c %y "$app_path/.vrooli/.protected" 2>/dev/null | cut -d' ' -f1 || echo "Unknown")
        echo "🔒 Protected since: $protected_date"
        echo "   This app will not be auto-regenerated"
    else
        echo "🔓 Not Protected"
        echo "   Auto-regeneration: Enabled (if scenario changes)"
    fi
    
    # Check for backups
    echo ""
    echo "Backups:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    local backup_count=0
    if [[ -d "$APPS_DIR/.backups" ]]; then
        for backup in "$APPS_DIR/.backups/$app_name"-*; do
            [[ -d "$backup" ]] && ((backup_count++))
        done
    fi
    
    if [[ $backup_count -gt 0 ]]; then
        echo "💾 Available backups: $backup_count"
        echo "   Use 'vrooli app restore $app_name' to see options"
    else
        echo "📭 No backups found"
    fi
}

# Mark app as protected
app_protect() {
    local app_name="${1:-}"
    
    if [[ -z "$app_name" ]]; then
        log::error "App name required"
        echo "Usage: vrooli app protect <app-name>"
        return 1
    fi
    
    local app_path="$APPS_DIR/$app_name"
    
    if [[ ! -d "$app_path" ]]; then
        log::error "App not found: $app_name"
        return 1
    fi
    
    # Create .vrooli directory if it doesn't exist
    mkdir -p "$app_path/.vrooli"
    
    # Create protection marker file
    echo "Protected on $(date -u +"%Y-%m-%dT%H:%M:%SZ")" > "$app_path/.vrooli/.protected"
    
    log::success "✅ App protected: $app_name"
    echo "This app will not be automatically regenerated during 'vrooli setup' or auto-conversion"
    echo "To regenerate, use: vrooli app regenerate $app_name"
}

# Remove protection from app
app_unprotect() {
    local app_name="${1:-}"
    
    if [[ -z "$app_name" ]]; then
        log::error "App name required"
        echo "Usage: vrooli app unprotect <app-name>"
        return 1
    fi
    
    local app_path="$APPS_DIR/$app_name"
    
    if [[ ! -d "$app_path" ]]; then
        log::error "App not found: $app_name"
        return 1
    fi
    
    if [[ -f "$app_path/.vrooli/.protected" ]]; then
        rm -f "$app_path/.vrooli/.protected"
        log::success "✅ Protection removed: $app_name"
        echo "This app will now be regenerated if its scenario changes"
    else
        log::info "App was not protected: $app_name"
    fi
}

# Show differences from original generation
app_diff() {
    local app_name="${1:-}"
    
    if [[ -z "$app_name" ]]; then
        log::error "App name required"
        echo "Usage: vrooli app diff <app-name>"
        return 1
    fi
    
    local app_path="$APPS_DIR/$app_name"
    
    if [[ ! -d "$app_path" ]]; then
        log::error "App not found: $app_name"
        return 1
    fi
    
    if [[ ! -d "$app_path/.git" ]]; then
        log::error "No git repository found for: $app_name"
        echo "App was generated before version control was added"
        return 1
    fi
    
    cd "$app_path" || return 1
    
    log::header "Changes in: $app_name"
    echo ""
    
    # Get the initial commit
    local initial_commit
    initial_commit=$(git rev-list --max-parents=0 HEAD 2>/dev/null | head -1)
    
    if [[ -z "$initial_commit" ]]; then
        log::error "Cannot find initial generation commit"
        cd - >/dev/null 2>&1 || true
        return 1
    fi
    
    # Show summary
    echo "Comparing against initial generation:"
    git show "$initial_commit" --format="%h - %s (%ar)" --no-patch
    echo ""
    
    # Show file changes
    echo "Files changed since generation:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Get diff summary
    git diff --stat "$initial_commit" HEAD 2>/dev/null || echo "No committed changes"
    
    # Check for uncommitted changes
    if ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null; then
        echo ""
        echo "Uncommitted changes:"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        git status --short
    fi
    
    echo ""
    echo "To see detailed changes, run:"
    echo "  cd $app_path"
    echo "  git diff $initial_commit"
    
    cd - >/dev/null || true
}

# Regenerate app from scenario
app_regenerate() {
    local app_name="${1:-}"
    shift
    
    if [[ -z "$app_name" ]]; then
        log::error "App name required"
        echo "Usage: vrooli app regenerate <app-name> [--force] [--no-backup]"
        return 1
    fi
    
    # Parse options
    local force=false
    local no_backup=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force)
                force=true
                ;;
            --no-backup)
                no_backup=true
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
        shift
    done
    
    local app_path="$APPS_DIR/$app_name"
    
    # Check if app is protected
    if [[ -f "$app_path/.vrooli/.protected" ]] && [[ "$force" != "true" ]]; then
        log::error "App is protected: $app_name"
        echo "This app has been marked as protected from regeneration"
        echo "To regenerate anyway, use: vrooli app regenerate $app_name --force"
        return 1
    fi
    
    # Check for modifications and create backup if needed
    if [[ -d "$app_path/.git" ]] && [[ "$no_backup" != "true" ]]; then
        cd "$app_path" 2>/dev/null || true
        
        local is_modified=false
        if ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null || [[ -n "$(git ls-files --others --exclude-standard 2>/dev/null)" ]]; then
            is_modified=true
        elif [[ $(git rev-list --count HEAD 2>/dev/null || echo "0") -gt 1 ]]; then
            is_modified=true
        fi
        
        if [[ "$is_modified" == "true" ]]; then
            log::info "Creating backup before regeneration..."
            cd - >/dev/null 2>&1 || true
            app_backup "$app_name" || {
                log::error "Failed to create backup"
                return 1
            }
        fi
        
        cd - >/dev/null 2>&1 || true
    fi
    
    # Call scenario-to-app.sh to regenerate
    log::info "Regenerating app: $app_name"
    
    if "${var_SCRIPTS_SCENARIOS_DIR}/tools/scenario-to-app.sh" "$app_name" --force; then
        log::success "✅ App regenerated successfully: $app_name"
        
        # Remove protection if it was forced
        if [[ -f "$app_path/.vrooli/.protected" ]] && [[ "$force" == "true" ]]; then
            rm -f "$app_path/.vrooli/.protected"
            log::info "Protection removed after forced regeneration"
        fi
    else
        log::error "Failed to regenerate app: $app_name"
        return 1
    fi
}

# Create backup of app
app_backup() {
    local app_name="${1:-}"
    
    if [[ -z "$app_name" ]]; then
        log::error "App name required"
        echo "Usage: vrooli app backup <app-name>"
        return 1
    fi
    
    local app_path="$APPS_DIR/$app_name"
    
    if [[ ! -d "$app_path" ]]; then
        log::error "App not found: $app_name"
        return 1
    fi
    
    # Create backup directory
    local backup_dir="$APPS_DIR/.backups"
    local backup_name="${app_name}-$(date +%Y%m%d-%H%M%S)"
    local backup_path="$backup_dir/$backup_name"
    
    mkdir -p "$backup_dir"
    
    log::info "Creating backup: $backup_name"
    
    if cp -r "$app_path" "$backup_path" 2>/dev/null; then
        log::success "✅ Backup created: $backup_path"
        
        # Create metadata file
        cat > "$backup_path/.backup-metadata" << EOF
{
  "app_name": "$app_name",
  "backup_date": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "backup_name": "$backup_name"
}
EOF
        
        echo "Backup location: $backup_path"
    else
        log::error "Failed to create backup"
        return 1
    fi
}

# Restore app from backup
app_restore() {
    local app_name="${1:-}"
    shift
    
    if [[ -z "$app_name" ]]; then
        log::error "App name required"
        echo "Usage: vrooli app restore <app-name> [--from <backup-name>]"
        return 1
    fi
    
    local backup_name=""
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --from)
                backup_name="${2:-}"
                shift 2
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    local backup_dir="$APPS_DIR/.backups"
    
    # List available backups if no specific backup specified
    if [[ -z "$backup_name" ]]; then
        log::header "Available backups for: $app_name"
        echo ""
        
        local found_backup=false
        for backup in "$backup_dir/$app_name"-*; do
            if [[ -d "$backup" ]]; then
                found_backup=true
                local bname
                bname=$(basename "$backup")
                local bdate
                bdate=$(echo "$bname" | sed 's/.*-\([0-9]\{8\}-[0-9]\{6\}\)$/\1/')
                echo "  📦 $bname"
            fi
        done
        
        if [[ "$found_backup" == "false" ]]; then
            log::error "No backups found for: $app_name"
            return 1
        fi
        
        echo ""
        echo "To restore, run:"
        echo "  vrooli app restore $app_name --from <backup-name>"
        echo "  vrooli app restore $app_name --from latest"
        return 0
    fi
    
    # Find the backup to restore
    local backup_path=""
    
    if [[ "$backup_name" == "latest" ]]; then
        # Find the most recent backup
        backup_path=$(find "$backup_dir" -maxdepth 1 -type d -name "$app_name-*" | sort -r | head -1)
        if [[ -n "$backup_path" ]]; then
            backup_name=$(basename "$backup_path")
        fi
    else
        backup_path="$backup_dir/$backup_name"
    fi
    
    if [[ ! -d "$backup_path" ]]; then
        log::error "Backup not found: $backup_name"
        return 1
    fi
    
    local app_path="$APPS_DIR/$app_name"
    
    # Confirm restoration
    if [[ -d "$app_path" ]]; then
        log::warning "This will replace the current app at: $app_path"
        if ! flow::confirm "Continue with restoration?"; then
            log::info "Restoration cancelled"
            return 0
        fi
        
        # Create a safety backup of current state
        local safety_backup="${backup_dir}/${app_name}-before-restore-$(date +%Y%m%d-%H%M%S)"
        cp -r "$app_path" "$safety_backup" 2>/dev/null
        log::info "Safety backup created: $(basename "$safety_backup")"
        
        # Remove current app
        rm -rf "$app_path"
    fi
    
    # Restore from backup
    log::info "Restoring from: $backup_name"
    
    if cp -r "$backup_path" "$app_path" 2>/dev/null; then
        # Remove backup metadata file from restored app
        rm -f "$app_path/.backup-metadata"
        
        log::success "✅ App restored successfully: $app_name"
        echo "Restored from backup: $backup_name"
    else
        log::error "Failed to restore from backup"
        return 1
    fi
}

# Clean old backups
app_clean() {
    local days="${1:-30}"
    
    log::header "Cleaning Old Backups"
    echo "Removing backups older than $days days..."
    
    local backup_dir="$APPS_DIR/.backups"
    
    if [[ ! -d "$backup_dir" ]]; then
        log::info "No backups directory found"
        return 0
    fi
    
    local count=0
    while IFS= read -r -d '' backup; do
        log::info "Removing: $(basename "$backup")"
        rm -rf "$backup"
        ((count++))
    done < <(find "$backup_dir" -maxdepth 1 -type d -mtime "+$days" -print0)
    
    if [[ $count -gt 0 ]]; then
        log::success "✅ Removed $count old backup(s)"
    else
        log::info "No old backups to remove"
    fi
}

# Main command handler
main() {
    if [[ $# -eq 0 ]] || [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
        show_app_help
        return 0
    fi
    
    local subcommand="$1"
    shift
    
    case "$subcommand" in
        list)
            app_list "$@"
            ;;
        status)
            app_status "$@"
            ;;
        protect)
            app_protect "$@"
            ;;
        unprotect)
            app_unprotect "$@"
            ;;
        diff)
            app_diff "$@"
            ;;
        regenerate)
            app_regenerate "$@"
            ;;
        backup)
            app_backup "$@"
            ;;
        restore)
            app_restore "$@"
            ;;
        clean)
            app_clean "$@"
            ;;
        *)
            log::error "Unknown app command: $subcommand"
            echo ""
            show_app_help
            return 1
            ;;
    esac
}

# Execute main function
main "$@"