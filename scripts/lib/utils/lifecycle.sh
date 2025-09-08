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
    
    echo "$steps" | jq -c '.[]' | while IFS= read -r step; do
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
        
        # Execute command
        if [[ "$is_background" == "true" ]]; then
            (
                cd "$(pwd)" &&
                VROOLI_LIFECYCLE_MANAGED=true \
                setsid bash -lc "exec -a '$process_name' bash -lc $(quote "$cmd")"
            ) &
            log::info "Started background process: $process_name"
        else
            # Foreground execution
            (cd "$(pwd)" && bash -c "$cmd") || {
                log::error "Step failed: $process_name"
                return 1
            }
        fi
    done
    
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
    local app_name="${SCENARIO_NAME:-${PWD##*/}}"
    
    # Check if setup is needed (only if setup.sh is available)
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
    lifecycle::execute_phase "develop"
}

################################################################################
# CLI Mode - Direct Execution
################################################################################

#######################################
# Main entry point for CLI execution
# Usage: lifecycle.sh <scenario_name> <phase> [options]
#######################################
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