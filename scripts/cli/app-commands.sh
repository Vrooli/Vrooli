#!/usr/bin/env bash
################################################################################
# Vrooli CLI - App Management Commands
# 
# Thin wrapper around the Vrooli App HTTP API
#
# Usage:
#   vrooli app <subcommand> [options]
#
################################################################################

set -euo pipefail

# API configuration
API_PORT="${VROOLI_APP_API_PORT:-8094}"
API_BASE="http://localhost:${API_PORT}"

# Source utilities for display
# shellcheck disable=SC1091
source "$(dirname "${BASH_SOURCE[0]}")/../../scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"

# Check if API is running
check_api() {
    if ! curl -s "${API_BASE}/health" >/dev/null 2>&1; then
        log::error "App API is not running"
        echo "Start it with: cd scripts/cli/api && go run main.go"
        return 1
    fi
}

# Show help
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
    clean                   Remove old backups

OPTIONS:
    --help, -h              Show this help message

EXAMPLES:
    vrooli app list                           # List all apps
    vrooli app status research-assistant      # Check app status
    vrooli app protect research-assistant     # Protect from regeneration
    vrooli app backup my-app                  # Create manual backup
    vrooli app restore my-app --from latest   # Restore from latest backup

For more information: https://docs.vrooli.com/cli/app-management
EOF
}

# List all apps
app_list() {
    check_api || return 1
    
    log::header "Generated Apps"
    
    local response
    response=$(curl -s "${API_BASE}/apps")
    
    if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log::error "Failed to list apps"
        return 1
    fi
    
    local apps
    apps=$(echo "$response" | jq -r '.data')
    
    echo ""
    echo "Location: ${GENERATED_APPS_DIR:-$HOME/generated-apps}"
    echo ""
    printf "%-30s %-15s %-20s %s\n" "APP NAME" "STATUS" "LAST MODIFIED" "TYPE"
    printf "%-30s %-15s %-20s %s\n" "--------" "------" "-------------" "----"
    
    echo "$apps" | jq -c '.[]' | while IFS= read -r app; do
        local name status modified
        name=$(echo "$app" | jq -r '.name')
        modified=$(echo "$app" | jq -r '.modified' | cut -d'T' -f1)
        
        # Determine status
        status="✅ Clean"
        if [[ $(echo "$app" | jq -r '.has_git') == "false" ]]; then
            status="⚠️  No Git"
        elif [[ $(echo "$app" | jq -r '.customized') == "true" ]]; then
            status="🔧 Modified"
        fi
        
        [[ $(echo "$app" | jq -r '.protected') == "true" ]] && status="$status 🔒"
        
        printf "%-30s %-15s %-20s\n" "$name" "$status" "$modified"
    done
    
    local count
    count=$(echo "$apps" | jq '. | length')
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Total: $count apps"
}

# Show app status
app_status() {
    local app_name="${1:-}"
    [[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
    
    check_api || return 1
    
    local response
    response=$(curl -s "${API_BASE}/apps/${app_name}")
    
    if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log::error "$(echo "$response" | jq -r '.error // "Failed to get app status"')"
        return 1
    fi
    
    local app backups
    app=$(echo "$response" | jq -r '.data.app')
    backups=$(echo "$response" | jq -r '.data.backups')
    
    log::header "App Status: $app_name"
    echo ""
    echo "📍 Location: $(echo "$app" | jq -r '.path')"
    echo "📅 Modified: $(echo "$app" | jq -r '.modified' | cut -d'T' -f1)"
    
    # Git status
    if [[ $(echo "$app" | jq -r '.has_git') == "true" ]]; then
        echo ""
        echo "Git Repository:"
        echo "━━━━━━━━━━━━━━━"
        if [[ $(echo "$app" | jq -r '.customized') == "true" ]]; then
            echo "🔧 Has customizations"
        else
            echo "✅ Clean (no customizations)"
        fi
    else
        echo ""
        echo "⚠️  No Git repository"
    fi
    
    # Protection
    echo ""
    echo "Protection Status:"
    echo "━━━━━━━━━━━━━━━━━"
    if [[ $(echo "$app" | jq -r '.protected') == "true" ]]; then
        echo "🔒 Protected"
    else
        echo "🔓 Not Protected"
    fi
    
    # Backups
    echo ""
    echo "Backups:"
    echo "━━━━━━━━━━━━━━━"
    if [[ $backups -gt 0 ]]; then
        echo "💾 Available backups: $backups"
        echo "   Use 'vrooli app restore $app_name --from latest' to restore"
    else
        echo "📭 No backups found"
    fi
}

# Protect app
app_protect() {
    local app_name="${1:-}"
    [[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
    
    check_api || return 1
    
    if curl -s -X POST "${API_BASE}/apps/${app_name}/protect" | jq -e '.success' >/dev/null 2>&1; then
        log::success "✅ App protected: $app_name"
        echo "This app will not be automatically regenerated during 'vrooli setup'"
    else
        log::error "Failed to protect app"
        return 1
    fi
}

# Unprotect app
app_unprotect() {
    local app_name="${1:-}"
    [[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
    
    check_api || return 1
    
    if curl -s -X DELETE "${API_BASE}/apps/${app_name}/protect" | jq -e '.success' >/dev/null 2>&1; then
        log::success "✅ Protection removed: $app_name"
    else
        log::error "Failed to unprotect app"
        return 1
    fi
}

# Show diff (git-based, doesn't need API)
app_diff() {
    local app_name="${1:-}"
    [[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
    
    local app_path="${GENERATED_APPS_DIR:-$HOME/generated-apps}/$app_name"
    
    if [[ ! -d "$app_path/.git" ]]; then
        log::error "No git repository found for: $app_name"
        return 1
    fi
    
    log::header "Changes in: $app_name"
    
    cd "$app_path" || return 1
    
    # Get initial commit
    local initial_commit
    initial_commit=$(git rev-list --max-parents=0 HEAD 2>/dev/null | head -1)
    
    echo ""
    echo "Comparing against initial generation"
    echo ""
    
    # Show diff stats
    git diff --stat "$initial_commit" HEAD
    
    cd - >/dev/null 2>&1 || true
}

# Regenerate app
app_regenerate() {
    local app_name="${1:-}"
    shift
    [[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
    
    local force=false no_backup=false
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force) force=true ;;
            --no-backup) no_backup=true ;;
            *) log::error "Unknown option: $1"; return 1 ;;
        esac
        shift
    done
    
    check_api || return 1
    
    # Check if protected
    local response
    response=$(curl -s "${API_BASE}/apps/${app_name}")
    
    if echo "$response" | jq -e '.data.app.protected' >/dev/null 2>&1; then
        if [[ $(echo "$response" | jq -r '.data.app.protected') == "true" ]] && [[ "$force" != "true" ]]; then
            log::error "App is protected: $app_name"
            echo "To regenerate anyway, use: vrooli app regenerate $app_name --force"
            return 1
        fi
    fi
    
    # Backup if customized and not skipped
    if [[ $(echo "$response" | jq -r '.data.app.customized // false') == "true" ]] && [[ "$no_backup" != "true" ]]; then
        log::info "Creating backup before regeneration..."
        curl -s -X POST "${API_BASE}/apps/${app_name}/backup" >/dev/null
    fi
    
    # Regenerate using scenario-to-app
    log::info "Regenerating app: $app_name"
    if "${var_SCRIPTS_SCENARIOS_DIR}/tools/scenario-to-app.sh" "$app_name" --force; then
        log::success "✅ App regenerated successfully: $app_name"
        
        # Remove protection if forced
        if [[ "$force" == "true" ]]; then
            curl -s -X DELETE "${API_BASE}/apps/${app_name}/protect" >/dev/null 2>&1
        fi
    else
        log::error "Failed to regenerate app: $app_name"
        return 1
    fi
}

# Create backup
app_backup() {
    local app_name="${1:-}"
    [[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
    
    check_api || return 1
    
    log::info "Creating backup: $app_name"
    
    local response
    response=$(curl -s -X POST "${API_BASE}/apps/${app_name}/backup")
    
    if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log::success "✅ Backup created: $(echo "$response" | jq -r '.data.backup')"
    else
        log::error "$(echo "$response" | jq -r '.error // "Failed to create backup"')"
        return 1
    fi
}

# Restore app
app_restore() {
    local app_name="${1:-}"
    shift
    [[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
    
    local backup_name=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --from) backup_name="${2:-}"; shift 2 ;;
            *) log::error "Unknown option: $1"; return 1 ;;
        esac
    done
    
    [[ -z "$backup_name" ]] && { log::error "Backup name required (use --from)"; return 1; }
    
    check_api || return 1
    
    # Confirm restoration
    local app_path="${GENERATED_APPS_DIR:-$HOME/generated-apps}/$app_name"
    if [[ -d "$app_path" ]]; then
        log::warning "This will replace the current app at: $app_path"
        flow::confirm "Continue with restoration?" || { log::info "Cancelled"; return 0; }
    fi
    
    log::info "Restoring from: $backup_name"
    
    local response
    response=$(curl -s -X POST "${API_BASE}/apps/${app_name}/restore" \
        -H "Content-Type: application/json" \
        -d "{\"backup\": \"$backup_name\"}")
    
    if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log::success "✅ $(echo "$response" | jq -r '.data')"
    else
        log::error "$(echo "$response" | jq -r '.error // "Failed to restore"')"
        return 1
    fi
}

# Clean backups (simplified - just removes old backups locally)
app_clean() {
    local days="${1:-30}"
    
    log::header "Cleaning Old Backups"
    echo "Removing backups older than $days days..."
    
    local backup_dir="${GENERATED_APPS_DIR:-$HOME/generated-apps}/.backups"
    if [[ ! -d "$backup_dir" ]]; then
        log::info "No backups directory found"
        return 0
    fi
    
    local count=0
    while IFS= read -r -d '' backup; do
        local name
        name=$(basename "$backup")
        echo "  Removing: $name"
        rm -rf "$backup"
        ((count++))
    done < <(find "$backup_dir" -maxdepth 1 -type d -mtime "+$days" -print0)
    
    if [[ $count -gt 0 ]]; then
        log::success "✅ Removed $count old backup(s)"
    else
        log::info "No old backups to remove"
    fi
}

# Main handler
main() {
    if [[ $# -eq 0 ]] || [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
        show_app_help
        return 0
    fi
    
    local subcommand="$1"
    shift
    
    case "$subcommand" in
        list) app_list "$@" ;;
        status) app_status "$@" ;;
        protect) app_protect "$@" ;;
        unprotect) app_unprotect "$@" ;;
        diff) app_diff "$@" ;;
        regenerate) app_regenerate "$@" ;;
        backup) app_backup "$@" ;;
        restore) app_restore "$@" ;;
        clean) app_clean "$@" ;;
        *)
            log::error "Unknown app command: $subcommand"
            echo ""
            show_app_help
            return 1
            ;;
    esac
}

main "$@"