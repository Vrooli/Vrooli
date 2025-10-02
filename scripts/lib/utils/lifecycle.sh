#!/usr/bin/env bash
################################################################################
# Lifecycle Phase Executor
# Clean, focused script for executing lifecycle phases from service.json
################################################################################

set -euo pipefail

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true
source "${APP_ROOT}/scripts/lib/utils/setup.sh" 2>/dev/null || true
source "${APP_ROOT}/scripts/lib/network/ports.sh" 2>/dev/null || true

################################################################################
# Core Functions
################################################################################

quote() { printf '%q' "$1"; }

################################################################################
# PID-Based Process Tracking System
################################################################################

#######################################
# Start a background process with PID tracking
# Arguments:
#   $1 - Phase name (develop, setup, etc.)
#   $2 - Step name (start-api, start-ui, etc.)
#   $3 - Command to execute
# Returns:
#   0 on success, 1 on failure
#######################################
lifecycle::start_tracked_process() {
    local phase="$1"
    local step_name="$2" 
    local cmd="$3"
    local app_name="${SCENARIO_NAME:-${PWD##*/}}"
    
    # Create identifiers
    local process_id="vrooli.${phase}.${app_name}.${step_name}"
    local process_dir="$HOME/.vrooli/processes/scenarios/$app_name"
    local log_file="$HOME/.vrooli/logs/scenarios/${app_name}/${process_id}.log"
    
    # Ensure directories exist
    mkdir -p "$process_dir" "$HOME/.vrooli/logs/scenarios/${app_name}"
    
    # Extract port from environment if available
    local port=""
    if [[ "$step_name" == "start-api" && -n "${API_PORT:-}" ]]; then
        port="$API_PORT"
    elif [[ "$step_name" == "start-ui" && -n "${UI_PORT:-}" ]]; then
        port="$UI_PORT"
    fi
    
    # Start process in new process group with identifiable environment
    (
        cd "$(pwd)"
        
        # Set environment variables for child processes
        export VROOLI_PROCESS_ID="$process_id"
        export VROOLI_PHASE="$phase"
        export VROOLI_SCENARIO="$app_name"
        export VROOLI_STEP="$step_name"
        export VROOLI_LIFECYCLE_MANAGED="true"
        
        # Execute the actual command in new session/process group
        exec setsid bash -c "$cmd"
    ) >> "$log_file" 2>&1 &
    
    local pid=$!
    local pgid=$pid  # With setsid, the PID becomes the PGID
    
    # Create comprehensive process metadata WITH PORT AND PGID
    cat > "$process_dir/${step_name}.json" << EOF_JSON
{
    "pid": $pid,
    "pgid": $pgid,
    "process_id": "$process_id", 
    "phase": "$phase",
    "scenario": "$app_name",
    "step": "$step_name",
    "command": $(printf '%s' "$cmd" | jq -Rs .),
    "working_dir": "$(pwd)",
    "log_file": "$log_file",
    "port": ${port:-null},
    "started_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "status": "running"
}
EOF_JSON
    
    # Also create simple PID file for quick checks
    echo "$pid" > "$process_dir/${step_name}.pid"

    # Give the process a moment to fail fast (e.g., port conflicts)
    sleep 0.2
    if ! kill -0 "$pid" 2>/dev/null; then
        local wait_status=0
        if ! wait "$pid" 2>/dev/null; then
            wait_status=$?
        fi

        log::error "Process $process_id exited immediately (status: $wait_status)"

        if [[ -n "$port" ]]; then
            local conflict_info
            conflict_info=$(lsof -nP -iTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)
            if [[ -n "$conflict_info" ]]; then
                log::warning "🚨 Port $port is already in use. The following listeners were detected:" \
                    && while IFS= read -r line; do
                        [[ -z "$line" ]] && continue
                        log::warning "    $line"
                    done <<< "$conflict_info"
            else
                log::warning "Port $port appears free; check recent logs for other startup issues."
            fi
        fi

        if [[ -s "$log_file" ]]; then
            log::warning "Last log lines for $process_id:" \
                && tail -n 20 "$log_file" | while IFS= read -r line; do
                    log::warning "    $line"
                done
        fi

        rm -f "$process_dir/${step_name}.json" "$process_dir/${step_name}.pid"
        return 1
    fi

    log::info "Started background process: $process_id (PID: $pid)"
    return 0
}
lifecycle::discover_running_scenarios() {
    local scenarios_dir="$HOME/.vrooli/processes/scenarios"
    local -A running_scenarios
    local total_running=0
    
    [[ -d "$scenarios_dir" ]] || return 0
    
    for scenario_dir in "$scenarios_dir"/*; do
        [[ -d "$scenario_dir" ]] || continue
        
        local scenario_name=$(basename "$scenario_dir")
        local running_steps=0
        
        for step_file in "$scenario_dir"/*.json; do
            [[ -f "$step_file" ]] || continue
            
            local pid=$(jq -r '.pid' "$step_file" 2>/dev/null)
            local step_name=$(jq -r '.step' "$step_file" 2>/dev/null)
            
            # Check if process is actually running
            if [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 "$pid" 2>/dev/null; then
                running_steps=$((running_steps + 1))
            else
                # Cleanup dead process metadata
                rm -f "$step_file" "$scenario_dir/${step_name}.pid" 2>/dev/null
            fi
        done
        
        if [[ $running_steps -gt 0 ]]; then
            running_scenarios["$scenario_name"]=$running_steps
            total_running=$((total_running + 1))
        fi
    done
    
    # Output results in a structured format
    echo "total_running:$total_running"
    for scenario in "${!running_scenarios[@]}"; do
        echo "scenario:$scenario:${running_scenarios[$scenario]}"
    done
}

#######################################
# Check if a specific scenario is running
# Arguments:
#   $1 - Scenario name
# Returns:
#   0 if running, 1 if not running
#######################################
lifecycle::is_scenario_running() {
    local scenario_name="$1"
    local scenario_dir="$HOME/.vrooli/processes/scenarios/$scenario_name"
    
    [[ -d "$scenario_dir" ]] || return 1
    
    for step_file in "$scenario_dir"/*.json; do
        [[ -f "$step_file" ]] || continue
        
        local pid=$(jq -r '.pid' "$step_file" 2>/dev/null)
        
        # Check if process is actually running
        if [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 "$pid" 2>/dev/null; then
            return 0  # Found at least one running process
        else
            # Cleanup dead process metadata
            local step_name=$(jq -r '.step' "$step_file" 2>/dev/null)
            rm -f "$step_file" "$scenario_dir/${step_name}.pid" 2>/dev/null
        fi
    done
    
    return 1  # No running processes found
}

#######################################
# Stop all processes for a scenario
# Arguments:
#   $1 - Scenario name
#   $2 - Signal (optional, default: TERM)
# Returns:
#   0 on success, 1 on failure
#######################################
lifecycle::stop_scenario_processes() {
    local scenario_name="$1"
    local signal="${2:-TERM}"
    local scenario_dir="$HOME/.vrooli/processes/scenarios/$scenario_name"
    
    [[ -d "$scenario_dir" ]] || {
        log::debug "No process directory found for scenario: $scenario_name"
        return 0
    }
    
    local stopped_count=0
    local -a process_groups=()
    local -a step_names=()
    local -a step_files=()
    
    # Phase 1: Collect all process groups and send SIGTERM
    for step_file in "$scenario_dir"/*.json; do
        [[ -f "$step_file" ]] || continue
        
        local pid=$(jq -r '.pid' "$step_file" 2>/dev/null)
        local pgid=$(jq -r '.pgid // .pid' "$step_file" 2>/dev/null)  # Fallback to PID if no PGID
        local step_name=$(jq -r '.step' "$step_file" 2>/dev/null)
        
        if [[ "$pgid" =~ ^[0-9]+$ ]]; then
            if kill -0 -"$pgid" 2>/dev/null; then
                log::info "Stopping $scenario_name:$step_name process group (PGID: $pgid)"
                if kill -"$signal" -"$pgid" 2>/dev/null; then
                    process_groups+=("$pgid")
                    step_names+=("$step_name")
                    step_files+=("$step_file")
                    ((stopped_count++))
                fi
            else
                log::debug "Process group $pgid already stopped"
            fi
        fi
    done
    
    # Phase 2: Wait for graceful shutdown
    if [[ ${#process_groups[@]} -gt 0 ]]; then
        log::debug "Waiting 2 seconds for graceful shutdown..."
        sleep 2
        
        # Phase 3: Verify and force kill survivors
        local force_killed=0
        for i in "${!process_groups[@]}"; do
            local pgid="${process_groups[$i]}"
            local step_name="${step_names[$i]}"
            
            if kill -0 -"$pgid" 2>/dev/null; then
                log::info "Force killing survivors in $scenario_name:$step_name (PGID: $pgid)"
                kill -KILL -"$pgid" 2>/dev/null && ((force_killed++))
            fi
        done
        
        if [[ $force_killed -gt 0 ]]; then
            log::debug "Force killed $force_killed process groups"
            sleep 1  # Brief pause after force kill
        fi
    fi
    
    # Phase 4: Cleanup metadata files
    for step_file in "${step_files[@]}"; do
        if [[ -f "$step_file" ]]; then
            local step_name=$(jq -r '.step' "$step_file" 2>/dev/null)
            rm -f "$step_file" "$scenario_dir/${step_name}.pid" 2>/dev/null
        fi
    done
    
    # Phase 5: Clean up any allocated port locks/state for this scenario
    local scenario_state_dir="$HOME/.vrooli/state/scenarios"
    if [[ -d "$scenario_state_dir" ]]; then
        for lock_file in "$scenario_state_dir"/.port_*.lock; do
            [[ -f "$lock_file" ]] || continue
            if [[ "$(cut -d: -f1 "$lock_file" 2>/dev/null)" == "$scenario_name" ]]; then
                rm -f "$lock_file" 2>/dev/null || true
            fi
        done
        rm -f "$scenario_state_dir/${scenario_name}.json" 2>/dev/null || true
    fi

    log::info "Stopped $stopped_count process groups for scenario: $scenario_name"
    return 0
}

#######################################
# Execute a lifecycle phase from service.json
# Simple and focused - just runs the steps
# Arguments:
#   $1 - Phase name (setup, develop, test, stop, etc.)
# Returns:
#   0 on success, 1 on failure
#######################################
lifecycle::execute_phase() {
    local phase="${1:-}"
    [[ -z "$phase" ]] && return 1
    
    # Find service.json
    local service_json=""
    if [[ -f ".vrooli/service.json" ]]; then
        service_json="$(pwd)/.vrooli/service.json"
    elif [[ -f "service.json" ]]; then
        service_json="$(pwd)/service.json"
    else
        log::error "No service.json found"
        return 1
    fi
    
    # NEW: Get complete environment (if in scenario mode)
    if [[ -n "${SCENARIO_NAME:-}" ]]; then
        # Get complete environment
        local env_result
        env_result=$(ports::get_scenario_environment "$SCENARIO_NAME" "$service_json")
        
        if [[ $(echo "$env_result" | jq -r '.success') == "true" ]]; then
            # Export ALL environment variables
            local export_commands
            export_commands=$(echo "$env_result" | jq -r '.env_vars | to_entries[] | "export " + .key + "=" + (.value | @sh)')
            
            if [[ -n "$export_commands" ]]; then
                eval "$export_commands"
                log::info "Environment setup complete for scenario: $SCENARIO_NAME"
                
                # Optional: Log if scenario was already running
                local is_running
                is_running=$(echo "$env_result" | jq -r '.is_running')
                if [[ "$is_running" == "true" ]]; then
                    log::info "Scenario processes are currently running"
                fi
            fi
        else
            local error_msg
            error_msg=$(echo "$env_result" | jq -r '.message // .error // "Unknown error"')
            echo ""
            echo "🚨 SCENARIO STARTUP FAILED: Port Allocation Error"
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            echo "Scenario: $SCENARIO_NAME"
            echo "Error: $error_msg"
            echo ""
            echo "💡 Common Solutions:"
            echo "   • Clean stale port locks: rm ~/.vrooli/state/scenarios/.port_*.lock"
            echo "   • Restart resources: vrooli resource restart"
            echo "   • Check running scenarios: vrooli status --verbose"
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            echo ""
            log::error "Environment setup failed: $error_msg"
            return 1
        fi
    fi
    
    # Get phase configuration
    local phase_config
    phase_config=$(jq -c ".lifecycle.$phase // {}" "$service_json" 2>/dev/null)
    
    if [[ "$phase_config" == "{}" ]]; then
        log::warning "No configuration for phase: $phase"
        return 0
    fi
    
    # Get phase description
    local description
    description=$(echo "$phase_config" | jq -r '.description // ""')
    [[ -n "$description" ]] && log::info "$description"
    
    # Get steps
    local steps
    steps=$(echo "$phase_config" | jq -c '.steps // []')
    
    if [[ "$steps" == "[]" ]]; then
        log::debug "No steps defined for phase: $phase"
        return 0
    fi
    
    # Execute each step
    local step_count=0
    local total_steps
    total_steps=$(echo "$steps" | jq 'length')
    
    while IFS= read -r step; do
        step_count=$((step_count + 1))
        
        local name=$(echo "$step" | jq -r '.name // "unnamed"')
        local cmd=$(echo "$step" | jq -r '.run // ""')
        local desc=$(echo "$step" | jq -r '.description // ""')
        local is_background=$(echo "$step" | jq -r '.background // false')
        
        [[ -z "$cmd" ]] && continue
        
        log::info "[$step_count/$total_steps] $name"
        [[ -n "$desc" ]] && echo "  → $desc"
        
        if [[ "${DRY_RUN:-false}" == "true" ]]; then
            echo "[DRY-RUN] Would execute: $cmd"
            continue
        fi

        # Background execution with identifiable process name
        local app_name="${SCENARIO_NAME:-${PWD##*/}}"
        local process_name="vrooli.$phase.$app_name.$name"
        
        # Execute command using new PID tracking system
        if [[ "$is_background" == "true" ]]; then
            if ! lifecycle::start_tracked_process "$phase" "$name" "$cmd"; then
                log::error "Background step failed: $phase.$app_name.$name"
                return 1
            fi
        else
            # Foreground execution
            if (cd "$(pwd)" && LIFECYCLE_PHASE="$phase" VROOLI_LIFECYCLE_MANAGED="true" bash -c "$cmd"); then
                # Command succeeded
                true
            else
                # Command failed
                if [[ "$phase" == "stop" ]]; then
                    # For stop phase, log as warning since PID tracking may have already handled it
                    log::warning "Stop step completed with non-zero exit (likely process already stopped): $name"
                else
                    # For other phases, treat as error
                    log::error "Step failed: $phase.$app_name.$name"
                    return 1
                fi
            fi
        fi
    done < <(echo "$steps" | jq -c '.[]')
    
    log::success "Phase '$phase' completed"
    return 0
}

#######################################
# Execute develop phase with auto-setup
# Checks if setup is needed and runs it first
# Arguments:
#   $@ - Additional arguments
# Returns:
#   0 on success, 1 on failure
#######################################
lifecycle::develop_with_auto_setup() {
    local scenario_name="${SCENARIO_NAME:-}"
    
    # Check if scenario is already running and healthy
    if [[ -n "$scenario_name" ]]; then
        if lifecycle::is_scenario_running "$scenario_name"; then
            if lifecycle::is_scenario_healthy "$scenario_name"; then
                log::success "✓ Scenario '$scenario_name' is already running and healthy"
                
                # Show ports for user convenience
                local scenario_dir="$HOME/.vrooli/processes/scenarios/$scenario_name"
                if [[ -f "$scenario_dir/start-api.json" ]]; then
                    local api_port=$(jq -r '.port // ""' "$scenario_dir/start-api.json" 2>/dev/null)
                    [[ -n "$api_port" && "$api_port" != "null" ]] && echo "  API: http://localhost:$api_port"
                fi
                if [[ -f "$scenario_dir/start-ui.json" ]]; then
                    local ui_port=$(jq -r '.port // ""' "$scenario_dir/start-ui.json" 2>/dev/null)
                    [[ -n "$ui_port" && "$ui_port" != "null" ]] && echo "  UI: http://localhost:$ui_port"
                fi
                
                return 0  # Already running and healthy, nothing to do
            else
                log::warning "⚠ Scenario '$scenario_name' is running but unhealthy, restarting..."
                lifecycle::stop_scenario_processes "$scenario_name"
                sleep 2  # Give processes time to clean up
            fi
        else
            log::info "Starting scenario '$scenario_name'..."
        fi
    fi
    
    # Check if setup is needed
    if command -v setup::is_needed >/dev/null 2>&1; then
        if setup::is_needed "$(pwd)"; then
            log::info "Running setup before develop..."
            lifecycle::execute_phase "setup" || {
                log::error "Setup failed"
                return 1
            }
            
            # Mark setup complete
            if command -v setup::mark_complete >/dev/null 2>&1; then
                setup::mark_complete
            fi
        fi
    fi
    
    # Run develop phase
    log::info "Starting develop phase..."
    lifecycle::execute_phase "develop" || return 1
}
lifecycle::main() {
    # Only run main if executed directly (not sourced)
    if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
        return 0
    fi
    
    local scenario_name="${1:-}"
    local phase="${2:-}"
    
    if [[ -z "$scenario_name" || -z "$phase" ]]; then
        echo "Usage: $0 <scenario_name> <phase>"
        echo "  scenario_name: Name of scenario to run"
        echo "  phase: Lifecycle phase (setup, develop, test, stop)"
        exit 1
    fi
    
    # Setup scenario context
    local scenario_dir="${APP_ROOT}/scenarios/$scenario_name"
    
    if [[ ! -d "$scenario_dir" ]]; then
        log::error "Scenario not found: $scenario_name"
        exit 1
    fi
    
    # Set environment for scenario execution
    export SCENARIO_NAME="$scenario_name"
    export SCENARIO_MODE=true
    
    # Change to scenario directory
    cd "$scenario_dir" || exit 1
    
    # Execute the requested phase
    case "$phase" in
        develop)
            lifecycle::develop_with_auto_setup
            ;;
        *)
            lifecycle::execute_phase "$phase"
            ;;
    esac
}

# Execute main if running directly
lifecycle::main "$@"
#######################################
# Check if a scenario is healthy (not just running)
# Arguments:
#   $1 - Scenario name
# Returns:
#   0 if healthy, 1 if not healthy
#######################################
lifecycle::is_scenario_healthy() {
    local scenario_name="$1"
    local service_json="${APP_ROOT}/scenarios/${scenario_name}/.vrooli/service.json"
    
    # First check if it's running at all
    if ! lifecycle::is_scenario_running "$scenario_name"; then
        return 1  # Not running, so not healthy
    fi
    
    # Get health check configuration from service.json
    if [[ ! -f "$service_json" ]]; then
        # No service.json, assume healthy if running
        return 0
    fi
    
    # Extract health endpoints
    local api_endpoint=$(jq -r '.lifecycle.health.endpoints.api // "/health"' "$service_json" 2>/dev/null)
    local ui_endpoint=$(jq -r '.lifecycle.health.endpoints.ui // "/health"' "$service_json" 2>/dev/null)
    
    # Get ports from environment or process files
    local scenario_dir="$HOME/.vrooli/processes/scenarios/$scenario_name"
    local api_port=""
    local ui_port=""
    
    # Try to get ports from process metadata
    if [[ -f "$scenario_dir/start-api.json" ]]; then
        # Extract port from environment in the JSON
        api_port=$(jq -r '.port // ""' "$scenario_dir/start-api.json" 2>/dev/null)
    fi
    
    if [[ -f "$scenario_dir/start-ui.json" ]]; then
        ui_port=$(jq -r '.port // ""' "$scenario_dir/start-ui.json" 2>/dev/null)
    fi
    
    # Fallback: get ports from service.json
    if [[ -z "$api_port" ]]; then
        api_port=$(jq -r '.ports.api.port // ""' "$service_json" 2>/dev/null)
    fi
    if [[ -z "$ui_port" ]]; then
        ui_port=$(jq -r '.ports.ui.port // ""' "$service_json" 2>/dev/null)
    fi
    
    # Check API health if port is available
    if [[ -n "$api_port" && "$api_port" != "null" ]]; then
        local api_url="http://localhost:${api_port}${api_endpoint}"
        if ! curl -sf --max-time 5 "$api_url" >/dev/null 2>&1; then
            log::debug "API health check failed for $scenario_name at $api_url"
            return 1  # API not healthy
        fi
    fi
    
    # Check UI health if port is available
    if [[ -n "$ui_port" && "$ui_port" != "null" ]]; then
        local ui_url="http://localhost:${ui_port}${ui_endpoint}"
        if ! curl -sf --max-time 5 "$ui_url" >/dev/null 2>&1; then
            log::debug "UI health check failed for $scenario_name at $ui_url"
            return 1  # UI not healthy
        fi
    fi
    
    return 0  # All checks passed
}
