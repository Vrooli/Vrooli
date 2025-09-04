#!/usr/bin/env bash
################################################################################
# Simplified Stop Manager for Vrooli
# Clean, modern approach to stopping scenarios and resources
################################################################################

set -euo pipefail

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true

# Configuration
STOP_TIMEOUT="${STOP_TIMEOUT:-30}"
FORCE_STOP="${FORCE_STOP:-false}"
DRY_RUN="${DRY_RUN:-false}"

# Paths
SCENARIOS_DIR="${APP_ROOT}/scenarios"
RESOURCES_DIR="${APP_ROOT}/resources"

################################################################################
# Core Functions
################################################################################

# Get appropriate signal based on force mode
stop::get_signal() {
    [[ "$FORCE_STOP" == "true" ]] && echo "KILL" || echo "TERM"
}

# Execute command or show in dry-run
stop::execute() {
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY-RUN] Would execute: $*"
        return 0
    else
        eval "$@"
    fi
}

# Stop a single scenario by name
stop::scenario() {
    local scenario_name="$1"
    local scenario_dir="$SCENARIOS_DIR/$scenario_name"
    
    [[ ! -d "$scenario_dir" ]] && return 1
    
    # Try lifecycle stop if available
    if [[ -f "$scenario_dir/.vrooli/service.json" ]]; then
        if command -v jq >/dev/null 2>&1 && \
           jq -e '.lifecycle.stop.steps' "$scenario_dir/.vrooli/service.json" >/dev/null 2>&1; then
            # Source lifecycle if needed
            if ! command -v lifecycle::execute_phase >/dev/null 2>&1; then
                source "${APP_ROOT}/scripts/lib/utils/lifecycle.sh" 2>/dev/null || true
            fi
            
            if (cd "$scenario_dir" && lifecycle::execute_phase "stop" 2>/dev/null); then
                log::success "Stopped $scenario_name via lifecycle"
                return 0
            fi
        fi
    fi
    
    # Simple fallback: find and stop processes by working directory
    local stopped=false
    
    # Stop API processes
    if stop::execute "pkill -$(stop::get_signal) -f '${scenario_name}-api' 2>/dev/null"; then
        stopped=true
    fi
    
    # Stop UI processes (node server.js in scenario's ui directory)
    local ui_pids=$(pgrep -f 'node server.js' 2>/dev/null || true)
    for pid in $ui_pids; do
        local cwd=$(readlink "/proc/$pid/cwd" 2>/dev/null || true)
        if [[ "$cwd" == "$scenario_dir/ui" ]]; then
            stop::execute "kill -$(stop::get_signal) $pid 2>/dev/null" && stopped=true
        fi
    done
    
    if [[ "$stopped" == "true" ]]; then
        log::success "Stopped $scenario_name"
    fi
    return 0
}

# Stop a single resource by name
stop::resource() {
    local resource_name="$1"
    local resource_dir="$RESOURCES_DIR/$resource_name"
    
    # Try cli.sh manage stop
    if [[ -f "$resource_dir/cli.sh" ]]; then
        if stop::execute "(cd '$resource_dir' && ./cli.sh manage stop) 2>&1"; then
            log::success "Stopped $resource_name via cli.sh"
            return 0
        fi
    fi
    
    # Fallback to Docker stop
    local containers=$(docker ps --format '{{.Names}}' 2>/dev/null | grep -E "$resource_name" 2>/dev/null || true)
    if [[ -n "$containers" ]]; then
        for container in $containers; do
            stop::execute "docker stop $container 2>/dev/null" && \
                log::success "Stopped container: $container"
        done
        return 0
    fi
    
    return 1
}

################################################################################
# Main Commands
################################################################################

# Stop all scenarios
stop::all_scenarios() {
    log::info "Stopping all scenarios..."
    
    # Stop orchestrator
    stop::execute "pkill -$(stop::get_signal) -f 'app_orchestrator' 2>/dev/null" || true
    
    # Stop each scenario
    [[ -d "$SCENARIOS_DIR" ]] || return 0
    
    for scenario_dir in "$SCENARIOS_DIR"/*/; do
        [[ -d "$scenario_dir" ]] || continue
        stop::scenario "$(basename "$scenario_dir")" || true
    done
    
    # Clean up PID files
    [[ "$DRY_RUN" != "true" ]] && rm -f /tmp/vrooli-apps/*.pid 2>/dev/null || true
}

# Stop all resources  
stop::all_resources() {
    log::info "Stopping all resources..."
    
    [[ -d "$RESOURCES_DIR" ]] || return 0
    
    # Detect running resources by checking Docker
    if ! command -v docker >/dev/null 2>&1; then
        log::warning "Docker not available, skipping resource stop"
        return 0
    fi
    
    for resource_dir in "$RESOURCES_DIR"/*/; do
        [[ -d "$resource_dir" ]] || continue
        local resource_name=$(basename "$resource_dir")
        
        # Check if resource has running containers
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -qE "$resource_name" 2>/dev/null; then
            stop::resource "$resource_name" || true
        fi
    done
}

# Stop all Docker containers
stop::all_containers() {
    log::info "Stopping all Docker containers..."
    
    command -v docker >/dev/null 2>&1 || return 0
    
    local containers=$(docker ps -q 2>/dev/null || true)
    [[ -z "$containers" ]] && return 0
    
    if [[ "$FORCE_STOP" == "true" ]]; then
        stop::execute "docker kill $containers 2>/dev/null" || true
    else
        stop::execute "docker stop -t $STOP_TIMEOUT $containers 2>/dev/null" || true
    fi
}

# Stop everything
stop::all() {
    log::info "Stopping all Vrooli components..."
    stop::all_scenarios
    stop::all_resources
    stop::all_containers
}

################################################################################
# Entry Point
################################################################################

stop::main() {
    local target="${1:-all}"
    shift || true
    
    # Parse flags
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force) FORCE_STOP=true ;;
            --dry-run) DRY_RUN=true ;;
            --timeout) STOP_TIMEOUT="$2"; shift ;;
            *) ;;
        esac
        shift || true
    done
    
    case "$target" in
        all) stop::all ;;
        scenarios|scenario) stop::all_scenarios ;;
        resources|resource) stop::all_resources ;;
        containers|container|docker) stop::all_containers ;;
        *)
            # Try as specific scenario or resource
            if [[ -d "$SCENARIOS_DIR/$target" ]]; then
                stop::scenario "$target"
            elif [[ -d "$RESOURCES_DIR/$target" ]]; then
                stop::resource "$target"
            else
                log::error "Unknown target: $target"
                return 1
            fi
            ;;
    esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    stop::main "$@"
fi