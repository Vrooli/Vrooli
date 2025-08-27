#!/usr/bin/env bash
################################################################################
# Unified Stop Manager for Vrooli
# Central hub for all stop operations - apps, resources, containers, processes
# 
# Usage:
#   stop::main all                    # Stop everything
#   stop::main apps                   # Stop generated apps only
#   stop::main resources              # Stop resources only
#   stop::main containers             # Stop Docker containers only
#   stop::main processes              # Stop system processes only
#   stop::main <name>                 # Stop specific app/resource
#
# Options (set via environment variables):
#   STOP_TIMEOUT=30                   # Seconds to wait for graceful stop
#   FORCE_STOP=false                  # Use SIGKILL instead of SIGTERM
#   DRY_RUN=false                     # Show what would be stopped
#   VERBOSE=false                     # Detailed output
#   PARALLEL=true                     # Stop resources in parallel
#
################################################################################

set -euo pipefail

# Ensure we're sourced with proper environment
if [[ -z "${APP_ROOT:-}" ]]; then
    APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
fi

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
source "${var_LOG_FILE:-${APP_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true

# Configuration with defaults (avoid readonly for CLI compatibility)
STOP_TIMEOUT="${STOP_TIMEOUT:-30}"
FORCE_STOP="${FORCE_STOP:-false}"
DRY_RUN="${DRY_RUN:-false}"
VERBOSE="${VERBOSE:-false}"
PARALLEL="${PARALLEL:-true}"
QUIET="${QUIET:-false}"
GLOBAL_TIMEOUT="${GLOBAL_TIMEOUT:-300}"  # 5 minutes max for entire stop operation

# Paths
GENERATED_APPS_DIR="${GENERATED_APPS_DIR:-$HOME/generated-apps}"
RESOURCES_DIR="${var_RESOURCES_DIR:-${APP_ROOT}/resources}"
PID_DIR="/tmp/vrooli-apps"
ORCHESTRATOR_PID="/tmp/vrooli-orchestrator.pid"
ORCHESTRATOR_LOCK="/tmp/vrooli-orchestrator.lock"

# State tracking
declare -g TOTAL_STOPPED=0
declare -g STOP_ERRORS=0
declare -ga STOPPED_ITEMS=()
declare -ga FAILED_ITEMS=()

################################################################################
# Utility Functions
################################################################################

# Log with optional verbose mode
stop::log() {
    local level="$1"
    shift
    
    if [[ "$QUIET" == "true" ]]; then
        return 0
    fi
    
    case "$level" in
        verbose)
            [[ "$VERBOSE" == "true" ]] && log::info "$@"
            ;;
        info)
            log::info "$@"
            ;;
        success)
            log::success "$@"
            ;;
        warning)
            log::warning "$@"
            ;;
        error)
            log::error "$@"
            ((STOP_ERRORS++)) || true
            ;;
        dry-run)
            [[ "$DRY_RUN" == "true" ]] && log::info "[DRY-RUN] $@"
            ;;
    esac
}

# Check if we're in dry-run mode
stop::is_dry_run() {
    [[ "$DRY_RUN" == "true" ]]
}

# Execute command or show in dry-run
stop::execute() {
    local cmd="$*"
    
    if stop::is_dry_run; then
        stop::log dry-run "Would execute: $cmd"
        return 0
    else
        stop::log verbose "Executing: $cmd"
        eval "$cmd"
        return $?
    fi
}

# Wait for process to stop with timeout
stop::wait_for_process() {
    local pid="$1"
    local timeout="${2:-$STOP_TIMEOUT}"
    local elapsed=0
    
    while [[ $elapsed -lt $timeout ]]; do
        if ! kill -0 "$pid" 2>/dev/null; then
            return 0
        fi
        sleep 1
        ((elapsed++)) || true
    done
    
    return 1  # Timeout reached
}

# Get signal based on force mode
stop::get_signal() {
    if [[ "$FORCE_STOP" == "true" ]]; then
        echo "KILL"
    else
        echo "TERM"
    fi
}

# Escape special characters for regex patterns
stop::escape_regex() {
    local input="$1"
    # Escape special regex characters: . * ^ $ + ? { } [ ] ( ) | \
    printf '%s\n' "$input" | sed 's/[\[\.*^$()+?{|}]/\\&/g'
}

# Safe container name matching with fallback
stop::match_containers() {
    local resource_name="$1"
    local escaped_name
    escaped_name=$(stop::escape_regex "$resource_name")
    
    # Try regex pattern matching first
    local containers
    containers=$(docker ps --format '{{.Names}}' 2>/dev/null | \
        grep -E "(^${escaped_name}$|^vrooli-${escaped_name}|^${escaped_name}-|-${escaped_name}$)" 2>/dev/null || true)
    
    # If regex fails or returns nothing, try literal matching as fallback
    if [[ -z "$containers" ]]; then
        stop::log verbose "Regex matching failed for '$resource_name', trying literal search" >&2
        containers=$(docker ps --format '{{.Names}}' 2>/dev/null | \
            grep -F "$resource_name" 2>/dev/null || true)
    fi
    
    echo "$containers"
}

# Execute with timeout to prevent hangs
stop::with_timeout() {
    local timeout_seconds="${STOP_TIMEOUT:-30}"
    local cmd="$*"
    
    stop::log verbose "Executing with ${timeout_seconds}s timeout: $cmd"
    
    if command -v timeout >/dev/null 2>&1; then
        timeout "$timeout_seconds" bash -c "$cmd"
        local exit_code=$?
        if [[ $exit_code -eq 124 ]]; then
            stop::log error "Command timed out after ${timeout_seconds}s: $cmd"
            return 1
        fi
        return $exit_code
    else
        # Fallback for systems without timeout command
        stop::log verbose "timeout command not available, executing without timeout"
        eval "$cmd"
        return $?
    fi
}

################################################################################
# Main Entry Point
################################################################################

stop::main() {
    local target="${1:-all}"
    shift || true
    
    # Parse any additional flags
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force)
                FORCE_STOP=true
                ;;
            --dry-run|--check)
                [[ -z "${DRY_RUN:-}" ]] && DRY_RUN=true
                ;;
            --verbose|-v)
                VERBOSE=true
                ;;
            --quiet|-q)
                QUIET=true
                ;;
            --timeout)
                STOP_TIMEOUT="$2"
                shift
                ;;
            *)
                # Unknown flag or argument
                ;;
        esac
        shift || true
    done
    
    stop::log info "Stop Manager: Target='$target', Force=$FORCE_STOP, DryRun=$DRY_RUN"
    stop::log info "Global timeout: ${GLOBAL_TIMEOUT}s, Individual timeout: ${STOP_TIMEOUT}s"
    
    # Execute the target stop operation
    stop::execute_target "$target" || {
        local exit_code=$?
        stop::log error "Stop operation failed with exit code: $exit_code"
        return $exit_code
    }
    
    # Report results
    stop::report_results
    
    # Return error if any failures
    [[ $STOP_ERRORS -eq 0 ]] && return 0 || return 1
}

# Execute the target stop operation (separated for timeout handling)
stop::execute_target() {
    local target="$1"
    
    case "$target" in
        all)
            stop::all
            ;;
        apps|app)
            stop::apps
            ;;
        resources|resource)
            stop::resources
            ;;
        containers|container|docker)
            stop::containers
            ;;
        processes|process)
            stop::processes
            ;;
        *)
            # Assume it's a specific app or resource name
            stop::specific "$target"
            ;;
    esac
}

################################################################################
# Core Stop Functions
################################################################################

# Stop everything
stop::all() {
    stop::log info "Stopping all Vrooli components..."
    
    # Order matters: apps first, then resources, then containers, then processes
    stop::apps
    stop::resources
    stop::containers
    stop::processes
}

# Stop generated apps
stop::apps() {
    stop::log info "Stopping generated apps..."
    local count=0
    
    # 1. Stop Python orchestrator
    stop::log verbose "Checking for orchestrator process..."
    if stop::execute "pkill -$(stop::get_signal) -f 'app_orchestrator' 2>/dev/null"; then
        ((count++)) || true
        ((TOTAL_STOPPED++)) || true || true
        STOPPED_ITEMS+=("app_orchestrator")
        stop::log success "  Stopped app orchestrator"
    fi
    
    # 2. Stop manage.sh processes
    stop::log verbose "Checking for manage.sh processes..."
    local pids
    pids=$(pgrep -f "generated-apps.*manage\.sh" 2>/dev/null || true)
    if [[ -n "$pids" ]]; then
        for pid in $pids; do
            local signal
            signal=$(stop::get_signal)
            if stop::execute "kill -$signal $pid 2>/dev/null"; then
                ((count++)) || true || true
                ((TOTAL_STOPPED++)) || true || true || true
                stop::log info "  Stopped app management script (PID: $pid)"
                
                # Wait for graceful shutdown unless forced
                if [[ "$FORCE_STOP" != "true" ]]; then
                    stop::wait_for_process "$pid" 5 || {
                        stop::log warning "  Process $pid didn't stop gracefully, forcing..."
                        stop::execute "kill -KILL $pid 2>/dev/null" || true
                    }
                fi
            fi
        done
    fi
    
    # 3. Stop API and UI servers from generated apps
    stop::log verbose "Checking for app servers..."
    if [[ -d "$GENERATED_APPS_DIR" ]]; then
        for app_dir in "$GENERATED_APPS_DIR"/*/; do
            if [[ -d "$app_dir" ]]; then
                local app_name
                app_name=$(basename "$app_dir")
                
                # Stop Go API servers
                if stop::execute "pkill -$(stop::get_signal) -f '${app_name}-api' 2>/dev/null"; then
                    ((count++)) || true || true
                    ((TOTAL_STOPPED++)) || true || true || true
                    STOPPED_ITEMS+=("${app_name}-api")
                    stop::log info "  Stopped $app_name API"
                fi
                
                # Stop Node UI servers
                if stop::execute "pkill -$(stop::get_signal) -f '$app_dir.*server\.js' 2>/dev/null"; then
                    ((count++)) || true || true
                    ((TOTAL_STOPPED++)) || true || true || true
                    STOPPED_ITEMS+=("${app_name}-ui")
                    stop::log info "  Stopped $app_name UI"
                fi
            fi
        done
    fi
    
    # 4. Clean up PID files
    if ! stop::is_dry_run; then
        stop::log verbose "Cleaning up PID files..."
        rm -f "$PID_DIR"/*.pid 2>/dev/null || true
        rm -f "$ORCHESTRATOR_PID" 2>/dev/null || true
        rm -f "$ORCHESTRATOR_LOCK" 2>/dev/null || true
    fi
    
    if [[ $count -gt 0 ]]; then
        stop::log success "Stopped $count app processes"
    else
        stop::log info "No app processes were running"
    fi
}

# Stop resources
stop::resources() {
    stop::log info "Stopping resources..."
    local count=0
    local -a running_resources=()
    
    # Detect running resources
    stop::log verbose "Detecting running resources..."
    
    # Check Docker containers for resource patterns
    if command -v docker >/dev/null 2>&1; then
        while IFS= read -r resource_dir; do
            [[ -d "$resource_dir" ]] || continue
            
            local resource_name
            resource_name=$(basename "$resource_dir")
            local has_running=false
            
            # Check various naming patterns with escaped regex
            local escaped_name
            escaped_name=$(stop::escape_regex "$resource_name")
            local patterns=(
                "^${escaped_name}$"
                "^vrooli-${escaped_name}"
                "^${escaped_name}-"
                "-${escaped_name}$"
            )
            
            for pattern in "${patterns[@]}"; do
                if docker ps --format '{{.Names}}' 2>/dev/null | grep -qE "$pattern" 2>/dev/null; then
                    has_running=true
                    stop::log verbose "Found running container matching pattern '$pattern' for resource '$resource_name'"
                    break
                fi
            done
            
            # Fallback to literal search if regex fails
            if [[ "$has_running" == "false" ]]; then
                if docker ps --format '{{.Names}}' 2>/dev/null | grep -qF "$resource_name" 2>/dev/null; then
                    has_running=true
                    stop::log verbose "Found running container via literal search for resource '$resource_name'"
                fi
            fi
            
            if [[ "$has_running" == "true" ]]; then
                running_resources+=("$resource_dir:$resource_name")
                stop::log verbose "  Found running resource: $resource_name"
            fi
        done < <(find "$RESOURCES_DIR" -mindepth 1 -maxdepth 1 -type d 2>/dev/null)
    fi
    
    # Stop resources using best available method
    if [[ ${#running_resources[@]} -gt 0 ]]; then
        stop::log info "Stopping ${#running_resources[@]} resources..."
        
        for resource_info in "${running_resources[@]}"; do
            local resource_dir="${resource_info%%:*}"
            local resource_name="${resource_info#*:}"
            local stopped=false
            
            # Try resource CLI first
            if command -v "resource-${resource_name}" >/dev/null 2>&1; then
                if stop::execute "resource-${resource_name} stop 2>&1"; then
                    stopped=true
                    ((count++)) || true || true
                    ((TOTAL_STOPPED++)) || true || true || true
                    STOPPED_ITEMS+=("resource:$resource_name")
                    stop::log success "  Stopped $resource_name via CLI"
                fi
            fi
            
            # Try manage.sh if CLI failed
            if [[ "$stopped" == "false" ]] && [[ -f "$resource_dir/manage.sh" ]]; then
                if stop::execute "'$resource_dir/manage.sh' stop 2>&1"; then
                    stopped=true
                    ((count++)) || true || true
                    ((TOTAL_STOPPED++)) || true || true || true
                    STOPPED_ITEMS+=("resource:$resource_name")
                    stop::log success "  Stopped $resource_name via manage.sh"
                fi
            fi
            
            # Fallback to Docker stop
            if [[ "$stopped" == "false" ]]; then
                stop::log verbose "Attempting Docker container stop for resource: $resource_name"
                local containers
                containers=$(stop::match_containers "$resource_name")
                
                if [[ -n "$containers" ]]; then
                    stop::log verbose "Found containers for $resource_name: $containers"
                    for container in $containers; do
                        if stop::execute "docker stop $container 2>/dev/null"; then
                            stopped=true
                            ((count++)) || true || true
                            ((TOTAL_STOPPED++)) || true || true || true
                            STOPPED_ITEMS+=("container:$container")
                            stop::log success "  Stopped $container via Docker"
                        else
                            stop::log warning "  Failed to stop container: $container"
                        fi
                    done
                else
                    stop::log verbose "No matching containers found for resource: $resource_name"
                fi
            fi
            
            if [[ "$stopped" == "false" ]]; then
                FAILED_ITEMS+=("resource:$resource_name")
                stop::log warning "  Could not stop resource: $resource_name"
            fi
        done
    fi
    
    if [[ $count -gt 0 ]]; then
        stop::log success "Stopped $count resources"
    else
        stop::log info "No resources were running"
    fi
}

# Stop Docker containers
stop::containers() {
    stop::log info "Stopping Docker containers..."
    local count=0
    
    if ! command -v docker >/dev/null 2>&1; then
        stop::log warning "Docker not installed, skipping container stop"
        return 0
    fi
    
    # Get all running containers
    local containers=$(docker ps -q 2>/dev/null || true)
    
    if [[ -z "$containers" ]]; then
        stop::log info "No Docker containers running"
        return 0
    fi
    
    # Count containers
    local container_count=$(echo "$containers" | wc -l)
    stop::log info "Found $container_count running containers"
    
    # Stop containers
    if [[ "$FORCE_STOP" == "true" ]]; then
        stop::log warning "Force stopping all containers..."
        if stop::execute "docker kill $containers 2>/dev/null"; then
            ((TOTAL_STOPPED += container_count)) || true
            count=$container_count
            stop::log success "Force stopped $container_count containers"
        fi
    else
        stop::log info "Gracefully stopping containers (timeout: ${STOP_TIMEOUT}s)..."
        if stop::execute "docker stop -t $STOP_TIMEOUT $containers 2>/dev/null"; then
            ((TOTAL_STOPPED += container_count)) || true
            count=$container_count
            stop::log success "Stopped $container_count containers"
        fi
    fi
    
    if [[ $count -eq 0 ]]; then
        stop::log warning "Failed to stop some containers"
    fi
}

# Stop system processes
stop::processes() {
    stop::log info "Stopping system processes..."
    local count=0
    
    # Bash processes (Vrooli-specific scripts that can cause infinite loops)
    stop::log verbose "Checking for bash scenario processes..."
    local bash_patterns=(
        "scenario-to-app.sh"
        "app_orchestrator.py"
        "simple-multi-app-starter"
    )
    
    for pattern in "${bash_patterns[@]}"; do
        if stop::execute "pkill -$(stop::get_signal) -f '$pattern' 2>/dev/null"; then
            ((count++)) || true
            ((TOTAL_STOPPED++)) || true || true
            STOPPED_ITEMS+=("bash-process:$pattern")
            stop::log success "  Stopped bash processes matching '$pattern'"
        fi
    done
    
    # Python processes (excluding system services)
    local python_patterns=(
        "homeassistant"
        "uvicorn"
        "gunicorn"
        "supervisord"
        "sage-notebook"
        "jupyter"
        "prepline"
    )
    
    for pattern in "${python_patterns[@]}"; do
        if stop::execute "pkill -$(stop::get_signal) -f '$pattern' 2>/dev/null"; then
            ((count++)) || true
            ((TOTAL_STOPPED++)) || true || true
            STOPPED_ITEMS+=("process:$pattern")
            stop::log success "  Stopped $pattern processes"
        fi
    done
    
    # Node processes (excluding development tools)
    stop::log verbose "Checking for Node.js processes..."
    local node_pids=$(pgrep -f "node" 2>/dev/null || true)
    
    for pid in $node_pids; do
        # Skip if it's a development tool
        local cmd=$(ps -p "$pid" -o cmd --no-headers 2>/dev/null || true)
        if [[ -n "$cmd" ]]; then
            # Skip Cursor, VSCode, and system Node processes
            if echo "$cmd" | grep -qE "Cursor|cursor|vscode|VS Code|electron|chrome|typescript|dumb-init"; then
                continue
            fi
            
            if stop::execute "kill -$(stop::get_signal) $pid 2>/dev/null"; then
                ((count++)) || true || true
                ((TOTAL_STOPPED++)) || true || true || true
                stop::log info "  Stopped Node process (PID: $pid)"
            fi
        fi
    done
    
    # Clean up any processes in generated-apps directory
    stop::log verbose "Checking for processes in generated-apps..."
    for pid_dir in /proc/*/; do
        if [[ -d "$pid_dir" ]]; then
            local pid=$(basename "$pid_dir")
            if [[ "$pid" =~ ^[0-9]+$ ]]; then
                local cwd=$(readlink "${pid_dir}cwd" 2>/dev/null || true)
                if [[ "$cwd" == "$GENERATED_APPS_DIR"/* ]]; then
                    if stop::execute "kill -$(stop::get_signal) $pid 2>/dev/null"; then
                        ((count++)) || true || true
                        ((TOTAL_STOPPED++)) || true || true || true
                        stop::log info "  Stopped process in generated-apps (PID: $pid)"
                    fi
                fi
            fi
        fi
    done
    
    if [[ $count -gt 0 ]]; then
        stop::log success "Stopped $count system processes"
    else
        stop::log info "No system processes were running"
    fi
}

# Stop specific app or resource
stop::specific() {
    local name="$1"
    stop::log info "Stopping specific target: $name"
    
    # Check if it's explicitly typed
    if [[ "$name" == app:* ]]; then
        name="${name#app:}"
        stop::specific_app "$name"
    elif [[ "$name" == resource:* ]]; then
        name="${name#resource:}"
        stop::specific_resource "$name"
    else
        # Try to auto-detect
        local found=false
        
        # Check if it's an app
        if [[ -d "$GENERATED_APPS_DIR/$name" ]]; then
            stop::specific_app "$name"
            found=true
        fi
        
        # Check if it's a resource
        if [[ -d "$RESOURCES_DIR/$name" ]]; then
            stop::specific_resource "$name"
            found=true
        fi
        
        if [[ "$found" == "false" ]]; then
            stop::log error "Unknown target: $name (not found as app or resource)"
        fi
    fi
}

# Stop specific app
stop::specific_app() {
    local app_name="$1"
    local app_dir="$GENERATED_APPS_DIR/$app_name"
    
    if [[ ! -d "$app_dir" ]]; then
        stop::log error "App not found: $app_name"
        return 1
    fi
    
    stop::log info "Stopping app: $app_name"
    
    # Try manage.sh first
    if [[ -f "$app_dir/scripts/manage.sh" ]]; then
        if stop::execute "'$app_dir/scripts/manage.sh' stop 2>&1"; then
            ((TOTAL_STOPPED++)) || true || true
            STOPPED_ITEMS+=("app:$app_name")
            stop::log success "Stopped $app_name via manage.sh"
            return 0
        fi
    fi
    
    # Fallback to process killing
    local stopped=false
    
    # Kill API server
    if stop::execute "pkill -$(stop::get_signal) -f '${app_name}-api' 2>/dev/null"; then
        stopped=true
        ((TOTAL_STOPPED++)) || true
        stop::log success "Stopped $app_name API"
    fi
    
    # Kill UI server
    if stop::execute "pkill -$(stop::get_signal) -f '$app_dir.*server' 2>/dev/null"; then
        stopped=true
        ((TOTAL_STOPPED++)) || true
        stop::log success "Stopped $app_name UI"
    fi
    
    if [[ "$stopped" == "true" ]]; then
        STOPPED_ITEMS+=("app:$app_name")
    else
        FAILED_ITEMS+=("app:$app_name")
        stop::log warning "App may not be running: $app_name"
    fi
}

# Stop specific resource
stop::specific_resource() {
    local resource_name="$1"
    local resource_dir="$RESOURCES_DIR/$resource_name"
    
    stop::log info "Stopping resource: $resource_name"
    
    # Try resource CLI
    if command -v "resource-${resource_name}" >/dev/null 2>&1; then
        if stop::execute "resource-${resource_name} stop 2>&1"; then
            ((TOTAL_STOPPED++)) || true || true
            STOPPED_ITEMS+=("resource:$resource_name")
            stop::log success "Stopped $resource_name via CLI"
            return 0
        fi
    fi
    
    # Try manage.sh
    if [[ -f "$resource_dir/manage.sh" ]]; then
        if stop::execute "'$resource_dir/manage.sh' stop 2>&1"; then
            ((TOTAL_STOPPED++)) || true || true
            STOPPED_ITEMS+=("resource:$resource_name")
            stop::log success "Stopped $resource_name via manage.sh"
            return 0
        fi
    fi
    
    # Try Docker
    stop::log verbose "Attempting Docker container stop for specific resource: $resource_name"
    local containers
    containers=$(stop::match_containers "$resource_name")
    
    if [[ -n "$containers" ]]; then
        stop::log verbose "Found containers for $resource_name: $containers"
        local stopped=false
        for container in $containers; do
            if stop::execute "docker stop $container 2>/dev/null"; then
                stopped=true
                ((TOTAL_STOPPED++)) || true || true
                stop::log success "Stopped container: $container"
            else
                stop::log warning "Failed to stop container: $container"
            fi
        done
        
        if [[ "$stopped" == "true" ]]; then
            STOPPED_ITEMS+=("resource:$resource_name")
        else
            FAILED_ITEMS+=("resource:$resource_name")
            stop::log warning "All container stop attempts failed for: $resource_name"
        fi
    else
        FAILED_ITEMS+=("resource:$resource_name")
        stop::log warning "Resource may not be running: $resource_name"
    fi
}

################################################################################
# Reporting
################################################################################

stop::report_results() {
    # Don't report in quiet mode
    [[ "$QUIET" == "true" ]] && return 0
    
    echo ""
    stop::log info "════════════════════════════════════════"
    stop::log info "Stop Manager Results"
    stop::log info "════════════════════════════════════════"
    
    if [[ $TOTAL_STOPPED -gt 0 ]]; then
        stop::log success "Successfully stopped $TOTAL_STOPPED items"
        
        if [[ "$VERBOSE" == "true" ]] && [[ ${#STOPPED_ITEMS[@]} -gt 0 ]]; then
            stop::log info "Stopped items:"
            for item in "${STOPPED_ITEMS[@]}"; do
                echo "  • $item"
            done
        fi
    else
        stop::log info "Nothing needed to be stopped"
    fi
    
    if [[ ${#FAILED_ITEMS[@]} -gt 0 ]]; then
        stop::log warning "Failed to stop ${#FAILED_ITEMS[@]} items:"
        for item in "${FAILED_ITEMS[@]}"; do
            echo "  ✗ $item"
        done
    fi
    
    if [[ $STOP_ERRORS -gt 0 ]]; then
        stop::log error "Encountered $STOP_ERRORS errors during stop process"
    fi
    
    # Final status check
    if command -v docker >/dev/null 2>&1; then
        local remaining
        remaining=$(docker ps -q 2>/dev/null | wc -l || echo 0)
        remaining=${remaining//[[:space:]]/}  # Remove whitespace
        if [[ $remaining -gt 0 ]]; then
            stop::log warning "$remaining Docker containers still running"
        fi
    fi
    
    local remaining_apps
    remaining_apps=$(pgrep -f "generated-apps" 2>/dev/null | wc -l || echo 0)
    remaining_apps=${remaining_apps//[[:space:]]/}  # Remove whitespace
    if [[ $remaining_apps -gt 0 ]]; then
        stop::log warning "$remaining_apps app processes may still be running"
    fi
}

################################################################################
# Script Entry Point (when run directly)
################################################################################

# If sourced, don't run main
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Script is being run directly
    stop::main "$@"
    exit $?
fi

# When sourced, just make functions available
true