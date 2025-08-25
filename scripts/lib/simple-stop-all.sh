#!/usr/bin/env bash
################################################################################
# Simple Stop-All Script for Vrooli Apps
# Direct and effective approach to stopping all generated apps
################################################################################

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1" >&2; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1" >&2; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1" >&2; }

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"

# Configuration
GENERATED_APPS_DIR="${GENERATED_APPS_DIR:-$HOME/generated-apps}"

main() {
    log_info "Stopping all Vrooli apps..."
    
    local stopped_count=0
    
    # Method 1: Kill Python orchestrator
    log_info "Stopping orchestrator..."
    if pkill -f "app_orchestrator" 2>/dev/null; then
        ((stopped_count++)) || true
        log_success "  Stopped orchestrator"
    fi
    
    # Method 2: Kill all manage.sh processes from generated apps
    log_info "Stopping app management scripts..."
    local pids=$(pgrep -f "generated-apps.*manage\.sh" 2>/dev/null || true)
    if [[ -n "$pids" ]]; then
        for pid in $pids; do
            if kill -TERM "$pid" 2>/dev/null; then
                ((stopped_count++)) || true
                log_info "  Stopped PID $pid"
            fi
        done
        
        # Give them time to exit gracefully
        sleep 2
        
        # Force kill any remaining
        for pid in $pids; do
            kill -9 "$pid" 2>/dev/null || true
        done
    fi
    
    # Method 3: Kill API servers and UI servers from generated apps
    log_info "Stopping app API and UI servers..."
    
    # Find and kill Go API servers
    for app_dir in "$GENERATED_APPS_DIR"/*/; do
        if [[ -d "$app_dir" ]]; then
            local app_name=$(basename "$app_dir")
            
            # Kill Go API servers
            if pkill -f "${app_name}-api" 2>/dev/null; then
                ((stopped_count++)) || true
                log_info "  Stopped $app_name API"
            fi
            
            # Kill Node UI servers
            if pkill -f "$app_dir.*server\.js" 2>/dev/null; then
                ((stopped_count++)) || true
                log_info "  Stopped $app_name UI"
            fi
        fi
    done
    
    # Method 4: Clean up PID files
    log_info "Cleaning up PID files..."
    rm -f /tmp/vrooli-apps/*.pid 2>/dev/null || true
    rm -f /tmp/vrooli-orchestrator.pid 2>/dev/null || true
    rm -f /tmp/vrooli-orchestrator.lock 2>/dev/null || true
    
    # Method 5: Kill any process with working directory in generated-apps
    log_info "Checking for remaining processes..."
    for pid_dir in /proc/*/; do
        if [[ -d "$pid_dir" ]]; then
            local pid=$(basename "$pid_dir")
            if [[ "$pid" =~ ^[0-9]+$ ]]; then
                local cwd=$(readlink "${pid_dir}cwd" 2>/dev/null || true)
                if [[ "$cwd" == "$GENERATED_APPS_DIR"/* ]]; then
                    if kill -TERM "$pid" 2>/dev/null; then
                        ((stopped_count++)) || true
                        local app_name="${cwd#$GENERATED_APPS_DIR/}"
                        app_name="${app_name%%/*}"
                        log_info "  Stopped $app_name process (PID: $pid)"
                    fi
                fi
            fi
        fi
    done
    
    # Summary
    echo ""
    if [[ $stopped_count -gt 0 ]]; then
        log_success "Successfully stopped $stopped_count processes!"
    else
        log_info "No running apps found"
    fi
    
    # Final verification
    local remaining=$(pgrep -f "generated-apps" | wc -l)
    if [[ $remaining -gt 0 ]]; then
        log_warning "Found $remaining processes that may still be running"
        log_info "Run with '--force' to kill them: $0 --force"
        
        if [[ "${1:-}" == "--force" ]]; then
            log_warning "Force killing remaining processes..."
            pkill -9 -f "generated-apps"
            log_success "Force killed all remaining processes"
        fi
    else
        log_success "All apps stopped successfully!"
    fi
}

main "$@"