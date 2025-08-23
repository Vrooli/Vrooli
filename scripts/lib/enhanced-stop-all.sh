#!/usr/bin/env bash
################################################################################
# SAFE Enhanced Stop-All Script for Vrooli Apps
# Fixed critical safety issues:
# - PID validation before killing
# - Restricted pattern matching
# - User ownership checks
# - No killing of system processes
################################################################################

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
GENERATED_APPS_DIR="${GENERATED_APPS_DIR:-$HOME/generated-apps}"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
PROCESS_TRACKING_DIR="$HOME/.vrooli/process-tracking"
MIN_SAFE_PID=1000  # Don't kill PIDs below this (system processes)
FORCE_MODE=false

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Validate PID is safe to kill
is_safe_pid() {
    local pid="$1"
    
    # Must be a number
    if ! [[ "$pid" =~ ^[0-9]+$ ]]; then
        return 1
    fi
    
    # Don't kill low PIDs (system processes)
    if [[ $pid -lt $MIN_SAFE_PID ]]; then
        log_warning "Refusing to kill system process PID $pid"
        return 1
    fi
    
    # Don't kill our own process or parent
    if [[ $pid -eq $$ ]] || [[ $pid -eq $PPID ]]; then
        return 1
    fi
    
    # Don't kill init process
    if [[ $pid -eq 1 ]]; then
        log_error "Refusing to kill init process!"
        return 1
    fi
    
    # Check if we own the process (safer)
    if [[ -f "/proc/$pid/status" ]]; then
        local uid=$(grep "^Uid:" "/proc/$pid/status" 2>/dev/null | awk '{print $2}')
        if [[ "$uid" != "$(id -u)" ]] && [[ $(id -u) -ne 0 ]]; then
            log_warning "Process $pid not owned by current user"
            return 1
        fi
    fi
    
    return 0
}

# Safe kill with validation
safe_kill() {
    local pid="$1"
    local signal="${2:-TERM}"
    
    if ! is_safe_pid "$pid"; then
        return 1
    fi
    
    # Verify process exists before killing
    if ! kill -0 "$pid" 2>/dev/null; then
        return 1  # Process doesn't exist
    fi
    
    # Send signal
    if kill -"$signal" "$pid" 2>/dev/null; then
        return 0
    fi
    
    return 1
}

################################################################################
# Method 1: Use Python Process Tracker (Safest)
################################################################################
stop_via_tracker() {
    STOPPED_COUNT=0
    
    # Try safe tracker first
    local safe_tracker="$VROOLI_ROOT/scripts/lib/process-tracker-safe.py"
    local tracker="$VROOLI_ROOT/scripts/lib/process-tracker.py"
    
    if [[ -f "$safe_tracker" ]]; then
        log_info "Method 1: Using SAFE process tracker..."
        if python3 "$safe_tracker" stop all $([[ "$FORCE_MODE" == "true" ]] && echo "--force") 2>/dev/null; then
            STOPPED_COUNT=$(python3 "$safe_tracker" status 2>/dev/null | wc -l || echo 0)
        fi
    elif [[ -f "$tracker" ]]; then
        log_warning "Using original tracker (has security issues)..."
        if python3 "$tracker" stop all 2>/dev/null; then
            STOPPED_COUNT=$(python3 "$tracker" status 2>/dev/null | wc -l || echo 0)
        fi
    fi
    
    return 0
}

################################################################################
# Method 2: Stop via Process Manager (with validation)
################################################################################
stop_via_process_manager() {
    STOPPED_COUNT=0
    
    if [[ -f "$VROOLI_ROOT/scripts/lib/process-manager.sh" ]]; then
        log_info "Method 2: Using process manager..."
        
        # Source process manager
        source "$VROOLI_ROOT/scripts/lib/process-manager.sh" 2>/dev/null || true
        
        if type -t pm::list >/dev/null 2>&1; then
            # Stop only vrooli.develop.* processes
            while IFS= read -r process; do
                # Validate process name format
                if [[ "$process" =~ ^vrooli\.develop\.[a-zA-Z0-9_-]+$ ]]; then
                    if pm::stop "$process" 2>/dev/null; then
                        ((STOPPED_COUNT++))
                        log_info "  Stopped: $process"
                    fi
                fi
            done < <(pm::list 2>/dev/null | grep "^vrooli\.develop\." || true)
        fi
    fi
    
    return 0
}

################################################################################
# Method 3: Find by Environment Variables (with strict validation)
################################################################################
stop_via_environment() {
    STOPPED_COUNT=0
    log_info "Method 3: Searching by environment variables..."
    
    local pids=()
    
    # Only check processes we can read
    for environ_file in /proc/*/environ; do
        if [[ -r "$environ_file" ]]; then
            local pid=$(echo "$environ_file" | cut -d/ -f3)
            
            # Validate PID format
            if [[ "$pid" =~ ^[0-9]+$ ]]; then
                # Look for our specific markers
                if grep -q "VROOLI_APP_NAME=\|VROOLI_TRACKED=1\|VROOLI_SAFE=1" "$environ_file" 2>/dev/null; then
                    if is_safe_pid "$pid"; then
                        pids+=("$pid")
                    fi
                fi
            fi
        fi
    done 2>/dev/null || true
    
    # Stop found processes
    for pid in "${pids[@]}"; do
        local app_name="unknown"
        if [[ -r "/proc/$pid/environ" ]]; then
            app_name=$(tr '\0' '\n' < "/proc/$pid/environ" | grep "^VROOLI_APP_NAME=" | cut -d= -f2 || echo "unknown")
        fi
        
        log_info "  Stopping $app_name (PID: $pid)"
        
        if safe_kill "$pid" "$([[ "$FORCE_MODE" == "true" ]] && echo "KILL" || echo "TERM")"; then
            ((STOPPED_COUNT++))
            if [[ "$FORCE_MODE" != "true" ]]; then
                sleep 0.2
                # Force kill if still running
                safe_kill "$pid" "KILL" 2>/dev/null || true
            fi
        fi
    done
    
    return 0
}

################################################################################
# Method 4: Find by Working Directory (with strict path validation)
################################################################################
stop_via_working_directory() {
    STOPPED_COUNT=0
    log_info "Method 4: Searching by working directory..."
    
    if [[ ! -d "$GENERATED_APPS_DIR" ]]; then
        log_warning "  Generated apps directory not found"
        return 0
    fi
    
    # Resolve real path to prevent symlink attacks
    local real_gen_dir=$(realpath "$GENERATED_APPS_DIR" 2>/dev/null || echo "$GENERATED_APPS_DIR")
    
    local pids=()
    
    for cwd_link in /proc/*/cwd; do
        if [[ -L "$cwd_link" ]]; then
            local pid=$(echo "$cwd_link" | cut -d/ -f3)
            
            if [[ "$pid" =~ ^[0-9]+$ ]] && is_safe_pid "$pid"; then
                local cwd=$(readlink "$cwd_link" 2>/dev/null || true)
                
                # Strict path matching - must be direct child of generated-apps
                if [[ "$cwd" == "$real_gen_dir/"* ]]; then
                    # Extract app name and validate
                    local app_dir="${cwd#$real_gen_dir/}"
                    local app_name="${app_dir%%/*}"
                    
                    # Validate app name format
                    if [[ "$app_name" =~ ^[a-zA-Z0-9_-]+$ ]]; then
                        pids+=("$pid:$app_name")
                    fi
                fi
            fi
        fi
    done 2>/dev/null || true
    
    # Stop found processes
    for pid_info in "${pids[@]}"; do
        local pid="${pid_info%%:*}"
        local app_name="${pid_info##*:}"
        
        log_info "  Stopping $app_name (PID: $pid)"
        
        if safe_kill "$pid" "$([[ "$FORCE_MODE" == "true" ]] && echo "KILL" || echo "TERM")"; then
            ((STOPPED_COUNT++))
        fi
    done
    
    return 0
}

################################################################################
# Clean up orphaned resources (safely)
################################################################################
cleanup_orphaned_resources() {
    log_info "Cleaning up orphaned resources..."
    
    # Clean up stale lock files
    if [[ -d "$HOME/.vrooli/processes" ]]; then
        find "$HOME/.vrooli/processes" -type f \( -name "lock" -o -name "pid" \) 2>/dev/null | while read -r file; do
            local dir=$(dirname "$file")
            local pid_file="$dir/pid"
            
            if [[ -f "$pid_file" ]]; then
                local pid=$(cat "$pid_file" 2>/dev/null || echo "")
                if [[ -n "$pid" ]] && [[ "$pid" =~ ^[0-9]+$ ]]; then
                    if ! kill -0 "$pid" 2>/dev/null; then
                        log_info "  Cleaning stale lock for PID $pid"
                        rm -rf "$dir"
                    fi
                fi
            fi
        done
    fi
    
    # Clean up tracking database using safe tracker
    if [[ -f "$VROOLI_ROOT/scripts/lib/process-tracker-safe.py" ]]; then
        python3 "$VROOLI_ROOT/scripts/lib/process-tracker-safe.py" cleanup 2>/dev/null || true
    fi
}

################################################################################
# Main execution
################################################################################
main() {
    # Handle arguments
    for arg in "$@"; do
        case "$arg" in
            force|--force|-f)
                FORCE_MODE=true
                log_warning "Force mode enabled - using SIGKILL"
                ;;
            --help|-h)
                echo "Usage: $0 [--force]"
                echo "  --force  Use SIGKILL instead of SIGTERM"
                exit 0
                ;;
        esac
    done
    
    log_info "Starting SAFE enhanced stop-all for Vrooli apps..."
    echo ""
    
    # Global counter for stopped processes
    STOPPED_COUNT=0
    local total_stopped=0
    
    # Try all methods in order of safety/reliability
    stop_via_tracker
    total_stopped=$((total_stopped + STOPPED_COUNT))
    
    STOPPED_COUNT=0
    stop_via_process_manager
    total_stopped=$((total_stopped + STOPPED_COUNT))
    
    STOPPED_COUNT=0
    stop_via_environment
    total_stopped=$((total_stopped + STOPPED_COUNT))
    
    STOPPED_COUNT=0
    stop_via_working_directory
    total_stopped=$((total_stopped + STOPPED_COUNT))
    
    # Clean up orphaned resources
    cleanup_orphaned_resources
    
    # Summary
    echo ""
    log_success "Stop-all complete!"
    log_info "Total processes stopped: $total_stopped"
    
    # Final verification
    echo ""
    log_info "Verifying all apps are stopped..."
    
    local remaining=0
    for environ_file in /proc/*/environ; do
        if [[ -r "$environ_file" ]]; then
            if grep -q "VROOLI_APP_NAME=\|VROOLI_TRACKED=1" "$environ_file" 2>/dev/null; then
                local pid=$(echo "$environ_file" | cut -d/ -f3)
                if [[ "$pid" =~ ^[0-9]+$ ]] && is_safe_pid "$pid"; then
                    ((remaining++))
                fi
            fi
        fi
    done 2>/dev/null || true
    
    if [[ $remaining -eq 0 ]]; then
        log_success "All Vrooli apps successfully stopped!"
    else
        log_warning "Found $remaining processes that may still be running"
        log_info "Run with '--force' to force kill remaining processes"
    fi
    
    exit 0
}

main "$@"