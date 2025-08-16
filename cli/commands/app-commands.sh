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
API_PORT="${VROOLI_API_PORT:-8090}"
API_BASE="http://localhost:${API_PORT}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Source utilities for display
# shellcheck disable=SC1091
VROOLI_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# Unset source guards to ensure utilities are properly loaded when exec'd from CLI
unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED 2>/dev/null || true
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"

# Check if API is running
check_api() {
    if ! curl -s --connect-timeout 2 "${API_BASE}/health" >/dev/null 2>&1; then
        return 1  # Return failure silently, let callers handle it
    fi
    return 0
}

# Show help
show_app_help() {
    cat << EOF
🚀 Vrooli App Management Commands

USAGE:
    vrooli app <subcommand> [options]

RUNTIME COMMANDS:
    start <name>            Start a specific app (runs setup if needed, then develop)
    stop <name>             Stop a specific app
    restart <name>          Restart a specific app
    logs <name>             Show logs for a specific app

MANAGEMENT COMMANDS:
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
    --follow                Follow logs in real-time (for logs command)

EXAMPLES:
    vrooli app start research-assistant       # Start an app (setup + develop)
    vrooli app stop research-assistant        # Stop an app
    vrooli app restart research-assistant     # Restart an app
    vrooli app logs research-assistant        # Show app logs
    vrooli app logs research-assistant --follow # Follow logs real-time
    vrooli app list                           # List all apps
    vrooli app status research-assistant      # Check app status
    vrooli app protect research-assistant     # Protect from regeneration

For more information: https://docs.vrooli.com/cli/app-management
EOF
}

# List all apps
app_list() {
    log::header "Generated Apps"
    
    local apps
    local use_api=false
    
    # Try to use API if available
    if check_api; then
        local response
        response=$(curl -s --connect-timeout 2 "${API_BASE}/apps" 2>/dev/null)
        
        if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
            apps=$(echo "$response" | jq -r '.data')
            use_api=true
        fi
    fi
    
    # Fallback to filesystem if API not available
    if [[ "$use_api" == "false" ]]; then
        # Build JSON array from filesystem
        apps="[]"
        local generated_dir="${GENERATED_APPS_DIR:-$HOME/generated-apps}"
        
        if [[ -d "$generated_dir" ]]; then
            for app_dir in "$generated_dir"/*; do
                [[ ! -d "$app_dir" ]] && continue
                
                local app_name
                app_name=$(basename "$app_dir")
                
                # Skip hidden directories and backups
                [[ "$app_name" == .* && "$app_name" != .tmp-* ]] && continue
                [[ "$app_name" == "backups" ]] && continue
                
                local has_git="false"
                [[ -d "$app_dir/.git" ]] && has_git="true"
                
                local customized="false"
                if [[ "$has_git" == "true" ]]; then
                    # Check if there are uncommitted changes
                    if cd "$app_dir" 2>/dev/null && git diff --quiet HEAD 2>/dev/null; then
                        customized="false"
                    else
                        customized="true"
                    fi
                    cd - >/dev/null 2>&1
                fi
                
                local modified
                modified=$(stat -c '%Y' "$app_dir" 2>/dev/null || stat -f '%m' "$app_dir" 2>/dev/null || echo "0")
                modified=$(date -d "@$modified" '+%Y-%m-%d' 2>/dev/null || date -r "$modified" '+%Y-%m-%d' 2>/dev/null || echo "unknown")
                
                local protected="false"
                [[ -f "$app_dir/.protected" ]] && protected="true"
                
                # Add to apps array
                local app_json
                app_json=$(jq -n \
                    --arg name "$app_name" \
                    --arg path "$app_dir" \
                    --arg has_git "$has_git" \
                    --arg customized "$customized" \
                    --arg protected "$protected" \
                    --arg modified "$modified" \
                    '{name: $name, path: $path, has_git: ($has_git == "true"), customized: ($customized == "true"), protected: ($protected == "true"), modified: $modified}')
                
                apps=$(echo "$apps" | jq --argjson app "$app_json" '. + [$app]')
            done
        fi
    fi
    
    # Source process manager for status checks
    local process_manager="${VROOLI_ROOT}/scripts/lib/process-manager.sh"
    if [[ -f "$process_manager" ]]; then
        # shellcheck disable=SC1090
        source "$process_manager" 2>/dev/null || true
    fi
    
    echo ""
    echo "Location: ${GENERATED_APPS_DIR:-$HOME/generated-apps}"
    echo ""
    printf "%-30s %-12s %-25s %-15s %-20s\n" "APP NAME" "RUNTIME" "URL" "GIT STATUS" "LAST MODIFIED"
    printf "%-30s %-12s %-25s %-15s %-20s\n" "--------" "-------" "---" "----------" "-------------"
    
    # Process each app (properly handle JSON array)
    local app_count=0
    while IFS= read -r app; do
        # Skip empty lines
        [[ -z "$app" ]] && continue
        
        # Process app in subshell to prevent errors from breaking the loop
        (
            # Extract app data (with error handling)
            local name git_status modified runtime_state url_display
            name=$(echo "$app" | jq -r '.name' 2>/dev/null)
            [[ -z "$name" || "$name" == "null" ]] && exit 0
            
            modified=$(echo "$app" | jq -r '.modified' 2>/dev/null || echo "unknown")
            [[ "$modified" == "null" ]] && modified="unknown"
            modified=$(echo "$modified" | cut -d'T' -f1 2>/dev/null || echo "$modified")
        
        # Determine git status
        git_status="✅ Clean"
        local has_git customized protected
        has_git=$(echo "$app" | jq -r '.has_git' 2>/dev/null || echo "false")
        customized=$(echo "$app" | jq -r '.customized' 2>/dev/null || echo "false")
        protected=$(echo "$app" | jq -r '.protected' 2>/dev/null || echo "false")
        
        if [[ "$has_git" == "false" ]]; then
            git_status="⚠️  No Git"
        elif [[ "$customized" == "true" ]]; then
            git_status="🔧 Modified"
        fi
        
        if [[ "$protected" == "true" ]]; then
            git_status="$git_status 🔒"
        fi
        
        # Get runtime state using process manager
        runtime_state="○ Stopped"
        url_display="-"
        
        # Check if any background processes are running for this app
        if type -t pm::list >/dev/null 2>&1; then
            local has_running_processes=false
            while IFS= read -r process; do
                if [[ "$process" == vrooli.develop.* ]] && pm::is_running "$process" 2>/dev/null; then
                    has_running_processes=true
                    break
                fi
            done < <(pm::list 2>/dev/null)
            
            if [[ "$has_running_processes" == "true" ]]; then
                runtime_state="● Running"
                # Try to get port from app's config
                local app_path="${GENERATED_APPS_DIR:-$HOME/generated-apps}/$name"
                local port=""
                if [[ -f "$app_path/package.json" ]]; then
                    port=$(jq -r '.config.port // .port // ""' "$app_path/package.json" 2>/dev/null)
                fi
                if [[ -z "$port" ]] && [[ -f "$app_path/.env" ]]; then
                    port=$(grep -E '^PORT=' "$app_path/.env" | cut -d= -f2 | tr -d '"' 2>/dev/null)
                fi
                if [[ -n "$port" ]]; then
                    url_display="http://localhost:$port"
                fi
            fi
        fi
        
            # Format output - using simpler color handling
            if [[ "$url_display" != "-" ]]; then
                # Print with colored URL
                printf "%-30s %-12s ${BLUE}%-25s${NC} %-15s %-20s\n" \
                    "$name" "$runtime_state" "$url_display" "$git_status" "$modified"
            else
                # Print without color
                printf "%-30s %-12s %-25s %-15s %-20s\n" \
                    "$name" "$runtime_state" "$url_display" "$git_status" "$modified"
            fi
        ) || true  # Continue even if subshell fails
        
        app_count=$((app_count + 1))
    done < <(echo "$apps" | jq -c '.[]' 2>/dev/null)
    
    local count
    count=$(echo "$apps" | jq '. | length')
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Count running apps
    local running_count=0
    if type -t pm::list >/dev/null 2>&1; then
        while IFS= read -r process; do
            if [[ "$process" == vrooli.*.develop ]] && pm::is_running "$process" 2>/dev/null; then
                running_count=$((running_count + 1))
            fi
        done < <(pm::list 2>/dev/null)
    fi
    
    echo "Total: $count apps ($running_count running)"
    echo ""
    echo "Use 'vrooli app start <name>' to start an app"
    echo "Use 'vrooli app logs <name>' to view logs"
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
            # Ignore shell redirections and operators that might be passed accidentally
            '2>&1'|'1>&2'|'&>'|'>'|'<'|'|'|'&&'|'||'|';') ;; # Silently ignore
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
    
    # Regenerate using scenario convert command
    log::info "Regenerating app: $app_name"
    
    # Use the CLI's own scenario convert command (dogfooding)
    if "${VROOLI_ROOT}/cli/vrooli" scenario convert "$app_name" --force; then
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
            # Ignore shell redirections and operators that might be passed accidentally
            '2>&1'|'1>&2'|'&>'|'>'|'<'|'|'|'&&'|'||'|';') ;; # Silently ignore
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

# Start an app using process manager
app_start() {
    local app_name="${1:-}"
    [[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
    
    # Check if process manager is available
    local process_manager="${VROOLI_ROOT}/scripts/lib/process-manager.sh"
    if [[ ! -f "$process_manager" ]]; then
        log::error "Process manager not found"
        echo "Please run 'vrooli setup' to install the process manager"
        return 1
    fi
    
    log::info "Starting app: $app_name"
    
    # Check if app exists
    local app_path="${GENERATED_APPS_DIR:-$HOME/generated-apps}/$app_name"
    if [[ ! -d "$app_path" ]]; then
        log::error "App not found: $app_name"
        echo "Use 'vrooli app list' to see available apps"
        return 1
    fi
    
    # Source process manager
    # shellcheck disable=SC1090
    source "$process_manager"
    
    # Check if app has been set up before
    local setup_marker="$app_path/.vrooli/.setup-complete"
    if [[ ! -f "$setup_marker" ]]; then
        log::info "First-time setup required for: $app_name"
        log::info "Running setup phase..."
        
        if (cd "$app_path" && ./scripts/manage.sh setup); then
            # Create setup completion marker
            mkdir -p "$app_path/.vrooli"
            echo "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" > "$setup_marker"
            log::success "✅ Setup completed for: $app_name"
        else
            log::error "Setup failed for: $app_name"
            return 1
        fi
    else
        log::info "App already set up (setup completed: $(cat "$setup_marker" 2>/dev/null || echo 'unknown'))"
    fi
    
    # Start the app by running manage.sh develop
    # manage.sh will handle background processes internally via process manager
    log::info "Starting development environment for: $app_name"
    if (cd "$app_path" && ./scripts/manage.sh develop); then
        # Try to get port from app's package.json or config
        local port=""
        if [[ -f "$app_path/package.json" ]]; then
            port=$(jq -r '.config.port // .port // ""' "$app_path/package.json" 2>/dev/null)
        fi
        
        # If no port in package.json, check for common port files
        if [[ -z "$port" ]] && [[ -f "$app_path/.env" ]]; then
            port=$(grep -E '^PORT=' "$app_path/.env" | cut -d= -f2 | tr -d '"' 2>/dev/null)
        fi
        
        # Default ports for known app types
        if [[ -z "$port" ]]; then
            case "$app_name" in
                *api*) port="5000" ;;
                *frontend*|*ui*) port="3000" ;;
                *admin*) port="3001" ;;
                *) port="" ;;
            esac
        fi
        
        # Display success with URL if available
        if [[ -n "$port" ]]; then
            echo ""
            echo -e "   ${GREEN}➜${NC} URL: ${BLUE}http://localhost:$port${NC}"
            echo ""
            echo "   Other commands:"
            echo "     vrooli app stop $app_name      # Stop the app"
            echo "     vrooli app logs $app_name      # View logs"
            echo "     vrooli app restart $app_name   # Restart"
        fi
        return 0
    else
        return 1
    fi
}

# Stop an app using process manager
app_stop() {
    local app_name="${1:-}"
    [[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
    
    # Check if process manager is available
    local process_manager="${VROOLI_ROOT}/scripts/lib/process-manager.sh"
    if [[ ! -f "$process_manager" ]]; then
        log::error "Process manager not found"
        return 1
    fi
    
    log::info "Stopping app: $app_name"
    
    # Source process manager
    # shellcheck disable=SC1090
    source "$process_manager"
    
    # Stop all background processes for this app
    # Background processes are named: vrooli.develop.*
    local stopped_any=false
    
    if command -v pm::list >/dev/null 2>&1; then
        while IFS= read -r process; do
            if [[ "$process" == vrooli.develop.* ]]; then
                if pm::stop "$process"; then
                    stopped_any=true
                fi
            fi
        done < <(pm::list 2>/dev/null)
    fi
    
    if [[ "$stopped_any" == "true" ]]; then
        log::success "✓ Stopped background processes for: $app_name"
        return 0
    else
        log::info "No running processes found for: $app_name"
        return 0
    fi
}

# Restart an app using orchestrator
app_restart() {
    local app_name="${1:-}"
    [[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
    
    log::info "Restarting app: $app_name"
    
    # Stop then start (suppress stop output to avoid clutter)
    if app_stop "$app_name" >/dev/null 2>&1; then
        sleep 2
        app_start "$app_name"
    else
        # If stop failed, try to start anyway (might not be running)
        log::info "App not running, starting it..."
        app_start "$app_name"
    fi
}

# Show logs for an app
app_logs() {
    local app_name="${1:-}"
    shift
    [[ -z "$app_name" ]] && { log::error "App name required"; return 1; }
    
    local follow=false
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --follow|-f) follow=true ;;
            *) ;;
        esac
        shift
    done
    
    # Check if process manager is available
    local process_manager="${VROOLI_ROOT}/scripts/lib/process-manager.sh"
    if [[ ! -f "$process_manager" ]]; then
        log::error "Process manager not found"
        return 1
    fi
    
    # Source process manager
    # shellcheck disable=SC1090
    source "$process_manager"
    
    # Get logs from all background processes for this app
    local found_processes=()
    
    if command -v pm::list >/dev/null 2>&1; then
        while IFS= read -r process; do
            if [[ "$process" == vrooli.develop.* ]]; then
                found_processes+=("$process")
            fi
        done < <(pm::list 2>/dev/null)
    fi
    
    if [[ ${#found_processes[@]} -eq 0 ]]; then
        log::info "No running processes found for: $app_name"
        return 0
    fi
    
    # Show logs from all background processes
    for process in "${found_processes[@]}"; do
        echo ""
        log::info "Logs for: $process"
        echo "────────────────────────────────────────────────────────"
        
        if [[ "$follow" == "true" ]]; then
            pm::logs "$process" 50 --follow
            break  # Only follow one process to avoid confusion
        else
            pm::logs "$process" 20
        fi
    done
    
    if [[ "$follow" != "true" ]]; then
        echo ""
        echo "Use --follow to see logs in real-time"
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
        # Runtime commands
        start) app_start "$@" ;;
        stop) app_stop "$@" ;;
        restart) app_restart "$@" ;;
        logs) app_logs "$@" ;;
        
        # Management commands
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