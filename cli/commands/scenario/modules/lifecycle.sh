#!/usr/bin/env bash
# Scenario Lifecycle Management Module
# Handles start, stop, restart, start-all, stop-all operations

set -euo pipefail

# Start one or more scenarios
scenario::lifecycle::start() {
    # Handle multiple scenario names
    local -a scenario_names=()
    while [[ $# -gt 0 ]] && [[ ! "$1" =~ ^- ]]; do
        scenario_names+=("$1")
        shift
    done
    
    [[ ${#scenario_names[@]} -eq 0 ]] && { 
        log::error "Scenario name(s) required"
        log::info "Usage: vrooli scenario start <name> [name2] [name3]..."
        return 1
    }
    
    local open_after=false
    local -a passthrough_args=()

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --open)
                open_after=true
                shift
                ;;
            *)
                passthrough_args+=("$1")
                shift
                ;;
        esac
    done

    # Start each scenario
    local overall_result=0
    if [[ ${#scenario_names[@]} -gt 1 ]]; then
        log::info "Starting ${#scenario_names[@]} scenarios: ${scenario_names[*]}"
    fi
    
    for scenario_name in "${scenario_names[@]}"; do
        if [[ ${#scenario_names[@]} -gt 1 ]]; then
            log::info "Starting scenario: $scenario_name"
        fi

        if ! scenario::run "$scenario_name" develop "${passthrough_args[@]}"; then
            local start_exit=$?
            if [[ $overall_result -eq 0 ]]; then
                overall_result=$start_exit
            fi
            continue
        fi

        if [[ "$open_after" == "true" ]]; then
            if ! scenario::browser::open "$scenario_name"; then
                local open_exit=$?
                if [[ $overall_result -eq 0 ]]; then
                    overall_result=$open_exit
                fi
            fi
        fi
    done
    
    return $overall_result
}

# Start all scenarios via API endpoint
scenario::lifecycle::start_all() {
    local api_port="${VROOLI_API_PORT:-8092}"
    local api_url="http://localhost:${api_port}"
    
    # Check if API is reachable
    if ! timeout 10 curl -s "${api_url}/health" >/dev/null 2>&1; then
        log::error "Vrooli API is not accessible at ${api_url}"
        log::info "The API must be running first. Start it with: vrooli develop"
        return 1
    fi
    
    log::info "Starting all scenarios..."
    
    # Call the start-all endpoint
    local response
    response=$(curl -s -X POST "${api_url}/scenarios/start-all" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        log::error "Failed to get response from API"
        return 1
    fi
    
    # Parse response
    local success
    success=$(echo "$response" | jq -r '.success // false' 2>/dev/null)
    
    if [[ "$success" == "true" ]]; then
        local started_count failed_count
        started_count=$(echo "$response" | jq -r '.data.started | length // 0' 2>/dev/null)
        failed_count=$(echo "$response" | jq -r '.data.failed | length // 0' 2>/dev/null)
        
        log::success "Started $started_count scenarios"
        
        # Show started scenarios
        if [[ "$started_count" -gt 0 ]]; then
            echo ""
            echo "Started scenarios:"
            echo "$response" | jq -r '.data.started[]? | "  ✅ " + .name + ": " + .message' 2>/dev/null
        fi
        
        # Show failed scenarios
        if [[ "$failed_count" -gt 0 ]]; then
            echo ""
            echo "Failed to start:"
            echo "$response" | jq -r '.data.failed[]? | "  ❌ " + .name + ": " + .error' 2>/dev/null
        fi
    else
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error // "Unknown error"' 2>/dev/null)
        log::error "Failed to start scenarios: $error_msg"
        return 1
    fi
}

# Restart a scenario (stop then start)
scenario::lifecycle::restart() {
    local scenario_name="${1:-}"
    [[ -z "$scenario_name" ]] && { 
        log::error "Scenario name required"
        log::info "Usage: vrooli scenario restart <name>"
        return 1
    }
    shift
    
    log::info "Restarting scenario: $scenario_name"
    
    # Stop if running (ignore errors - scenario might not be running)
    scenario::lifecycle::stop "$scenario_name" 2>/dev/null || true
    
    # Brief pause
    sleep 1
    
    # Start with any additional options passed through
    scenario::lifecycle::start "$scenario_name" "$@"
}

# Stop a specific scenario
scenario::lifecycle::stop() {
    local scenario_name="${1:-}"
    [[ -z "$scenario_name" ]] && { 
        log::error "Scenario name required"
        log::info "Usage: vrooli scenario stop <name>"
        return 1
    }
    shift
    
    # Direct lifecycle stop (bypasses API for reliability)
    # This ensures we can stop scenarios even when API is down
    # Source lifecycle.sh while preventing automatic main() execution
    {
        # Temporarily override lifecycle::main to prevent automatic execution
        lifecycle::main() { return 0; }
        source "${APP_ROOT}/scripts/lib/utils/lifecycle.sh" >/dev/null 2>&1
        unset -f lifecycle::main  # Remove our override, restore original
    }
    
    log::info "Stopping scenario: $scenario_name"
    
    # Use lifecycle function directly
    if lifecycle::stop_scenario_processes "$scenario_name"; then
        log::success "Scenario '$scenario_name' stopped successfully"
    else
        log::error "Failed to stop scenario '$scenario_name'"
        return 1
    fi
}

# Stop all running scenarios
scenario::lifecycle::stop_all() {
    # Temporarily disable strict error handling for stop-all
    local orig_flags=$-
    set +euo pipefail
    
    log::info "Stopping all scenarios..."
    
    # Direct PID-based approach - no sourcing of complex lifecycle functions
    local scenarios_dir="$HOME/.vrooli/processes/scenarios"
    local stopped_count=0
    local failed_count=0
    local stopped_scenarios=()
    local failed_scenarios=()
    
    if [[ -d "$scenarios_dir" ]]; then
        # Process each scenario directory
        for scenario_dir in "$scenarios_dir"/*; do
            [[ -d "$scenario_dir" ]] || continue
            
            local scenario_name=$(basename "$scenario_dir")
            local scenario_stopped=false
            local scenario_processes=0
            
            # Stop all processes for this scenario
            for pid_file in "$scenario_dir"/*.pid; do
                [[ -f "$pid_file" ]] || continue
                
                local pid=$(cat "$pid_file" 2>/dev/null || echo "")
                local step_name=$(basename "$pid_file" .pid)
                
                if [[ -n "$pid" && "$pid" =~ ^[0-9]+$ ]]; then
                    ((scenario_processes++))
                    if kill -0 "$pid" 2>/dev/null; then
                        # Process is running, stop it
                        log::info "Stopping $scenario_name:$step_name (PID: $pid)"
                        if kill -TERM "$pid" 2>/dev/null; then
                            scenario_stopped=true
                            # Wait a moment for graceful shutdown
                            sleep 0.5
                            # Force kill if still running
                            kill -0 "$pid" 2>/dev/null && kill -KILL "$pid" 2>/dev/null
                        fi
                    fi
                fi
                
                # Clean up PID file and JSON metadata
                rm -f "$pid_file" "$scenario_dir/$step_name.json" 2>/dev/null
            done
            
            # Record results for this scenario
            if [[ $scenario_processes -gt 0 ]]; then
                if [[ "$scenario_stopped" == "true" ]]; then
                    stopped_scenarios+=("$scenario_name")
                    ((stopped_count++))
                else
                    failed_scenarios+=("$scenario_name") 
                    ((failed_count++))
                fi
            fi
        done
    fi
    
    # Report results
    if [[ $stopped_count -gt 0 ]]; then
        log::success "Stopped $stopped_count scenarios"
        echo ""
        echo "Stopped scenarios:"
        for scenario in "${stopped_scenarios[@]}"; do
            echo "  ✅ $scenario"
        done
    fi
    
    if [[ $failed_count -gt 0 ]]; then
        echo ""
        echo "Failed to stop:"
        for scenario in "${failed_scenarios[@]}"; do
            echo "  ❌ $scenario"
        done
    fi
    
    if [[ $stopped_count -eq 0 && $failed_count -eq 0 ]]; then
        log::info "No running scenarios found"
    fi
    
    # Restore original shell flags  
    set -$orig_flags
    
    # Return success if we stopped any scenarios
    [[ $stopped_count -gt 0 ]] || [[ $failed_count -eq 0 ]]
}

# Test a scenario
scenario::lifecycle::test() {
    local scenario_name="${1:-}"
    [[ -z "$scenario_name" ]] && { 
        log::error "Scenario name required"
        log::info "Usage: vrooli scenario test <name> [--allow-skip-missing-runtime] [--manage-runtime]"
        return 1
    }
    shift
    scenario::run "$scenario_name" test "$@"
}
