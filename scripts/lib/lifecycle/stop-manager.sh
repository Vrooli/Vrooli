#!/usr/bin/env bash
################################################################################
# Simplified Stop Manager for Vrooli
# Clean, modern approach to stopping scenarios and resources
# Integrated with the unified Go API for proper scenario management
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
SCENARIO_STATE_DIR="${HOME}/.vrooli/state/scenarios"

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

# Clean up port lock files for a scenario
stop::cleanup_port_locks() {
    local scenario_name="$1"
    
    # Only clean up if state directory exists
    [[ -d "$SCENARIO_STATE_DIR" ]] || return 0
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::debug "[DRY-RUN] Would clean up port locks for $scenario_name"
        return 0
    fi
    
    # Remove all lock files for this scenario
    for lock_file in "$SCENARIO_STATE_DIR"/.port_*.lock; do
        [[ -f "$lock_file" ]] || continue
        
        # Check if this lock belongs to the scenario
        local lock_content
        lock_content=$(cat "$lock_file" 2>/dev/null | cut -d: -f1)
        
        if [[ "$lock_content" == "$scenario_name" ]]; then
            rm -f "$lock_file" 2>/dev/null
        fi
    done
    
    # Also remove state file
    rm -f "$SCENARIO_STATE_DIR/${scenario_name}.json" 2>/dev/null
}

# Stop a single scenario by name
stop::scenario() {
    local scenario_name="$1"
    local scenario_dir="$SCENARIOS_DIR/$scenario_name"
    local signal="$(stop::get_signal)"
    
    # Check if scenario exists
    if [[ ! -d "$scenario_dir" ]]; then
        log::error "Scenario not found: $scenario_name"
        return 1
    fi
    
    # Check dry-run mode
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY-RUN] Would stop scenario: $scenario_name"
        return 0
    fi
    
    # Source lifecycle if needed
    if ! command -v lifecycle::execute_phase >/dev/null 2>&1; then
        source "${APP_ROOT}/scripts/lib/utils/lifecycle.sh" 2>/dev/null || {
            log::error "Failed to source lifecycle.sh"
            return 1
        }
    fi
    
    # First, try to stop using PID-tracked processes
    if command -v lifecycle::stop_scenario_processes >/dev/null 2>&1; then
        log::info "Stopping PID-tracked processes for $scenario_name..."
        lifecycle::stop_scenario_processes "$scenario_name" "$signal"
    fi
    
    # Then execute the stop lifecycle phase (for cleanup commands)
    (cd "$scenario_dir" && lifecycle::execute_phase "stop") || {
        log::warning "Stop lifecycle phase failed for: $scenario_name (processes may still be stopped)"
    }
    
    # Clean up port lock files after stop attempt
    stop::cleanup_port_locks "$scenario_name"
    
    log::success "Stopped $scenario_name"
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
    
    # Clean up ALL scenario lock files when stopping all
    if [[ -d "$SCENARIO_STATE_DIR" ]] && [[ "$DRY_RUN" != "true" ]]; then
        rm -f "$SCENARIO_STATE_DIR"/.port_*.lock 2>/dev/null
        rm -f "$SCENARIO_STATE_DIR"/*.json 2>/dev/null
        log::debug "Cleaned up all scenario port locks and state files"
    fi
    
    # Find scenarios with running PID-tracked processes
    local scenarios=()
    local processes_dir="$HOME/.vrooli/processes/scenarios"
    
    # Source lifecycle functions if needed
    if ! command -v lifecycle::discover_running_scenarios >/dev/null 2>&1; then
        source "${APP_ROOT}/scripts/lib/utils/lifecycle.sh" 2>/dev/null || {
            log::error "Failed to source lifecycle.sh"
            return 1
        }
    fi
    
    # Use new PID-based detection
    if [[ -d "$processes_dir" ]]; then
        for scenario_dir in "$processes_dir"/*; do
            [[ -d "$scenario_dir" ]] || continue
            
            local scenario_name=$(basename "$scenario_dir")
            
            # Verify scenario directory exists and has running processes
            if [[ -d "$SCENARIOS_DIR/$scenario_name" ]] && command -v lifecycle::is_scenario_running >/dev/null 2>&1; then
                if lifecycle::is_scenario_running "$scenario_name"; then
                    scenarios+=("$scenario_name")
                fi
            fi
        done
    fi
    
    if [[ ${#scenarios[@]} -eq 0 ]]; then
        log::info "No running scenarios found"
        return 0
    fi
    
    log::info "Found ${#scenarios[@]} running scenarios to stop: ${scenarios[*]}"
    
    # Stop scenarios in batches (5 at a time to avoid overwhelming the system)
    local batch_size=5
    local stopped_count=0
    local failed_count=0
    
    for ((i=0; i<${#scenarios[@]}; i+=batch_size)); do
        local batch=("${scenarios[@]:i:batch_size}")
        log::debug "Stopping batch: ${batch[*]}"
        
        # Stop scenarios in this batch in parallel
        local pids=()
        for scenario_name in "${batch[@]}"; do
            (
                if stop::scenario "$scenario_name"; then
                    exit 0
                else
                    exit 1
                fi
            ) &
            pids+=($!)
        done
        
        # Wait for batch to complete
        for pid in "${pids[@]}"; do
            if wait "$pid"; then
                ((stopped_count++))
            else
                ((failed_count++))
            fi
        done
    done
    
    if [[ $failed_count -eq 0 ]]; then
        log::success "Successfully stopped all $stopped_count scenarios"
    else
        log::warning "Stopped $stopped_count scenarios, $failed_count failed"
    fi
    
    return 0
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