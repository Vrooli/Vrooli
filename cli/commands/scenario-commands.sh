#!/usr/bin/env bash
# Vrooli CLI - Scenario Management Commands (Direct Execution)
set -euo pipefail

# Get CLI directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${APP_ROOT}/scripts/lib/scenario/runner.sh"

# Show help for scenario commands
show_scenario_help() {
    cat << EOF
🚀 Vrooli Scenario Commands

USAGE:
    vrooli scenario <subcommand> [options]

SUBCOMMANDS:
    start <name> [options]  Start a scenario
    start <name1> <name2>...Start multiple scenarios
    start-all               Start all available scenarios
    restart <name> [options] Restart a scenario (stop then start)
    stop <name>             Stop a running scenario
    stop-all                Stop all running scenarios
    test <name>             Test a scenario
    list [--json]           List available scenarios
    logs <name> [options]   View logs for a scenario
    status [name] [--json]  Show scenario status
    open <name> [options]   Open scenario in browser
    port <name> <port>      Get port number for scenario

OPTIONS FOR START:
    --clean-stale           Clean stale port locks before starting

OPTIONS FOR LOGS:
    --follow, -f            Follow log output (live view)
    --step <name>           View specific background step log
    --runtime               View all background process logs
    --lifecycle             View lifecycle log (default behavior)

EXAMPLES:
    vrooli scenario start make-it-vegan         # Start a specific scenario
    vrooli scenario start make-it-vegan --clean-stale # Start with stale lock cleanup
    vrooli scenario start picker-wheel invoice-generator # Start multiple scenarios
    vrooli scenario start-all                   # Start all scenarios
    vrooli scenario restart ecosystem-manager    # Restart a scenario
    vrooli scenario stop swarm-manager           # Stop a specific scenario
    vrooli scenario stop-all                     # Stop all scenarios
    vrooli scenario test system-monitor          # Test a scenario
    vrooli scenario list                         # List available scenarios
    vrooli scenario list --json                  # List scenarios in JSON format
    vrooli scenario logs system-monitor          # Shows lifecycle execution
    vrooli scenario logs system-monitor --follow # Follow lifecycle log
    vrooli scenario status                       # Show all scenarios
    vrooli scenario status swarm-manager --json  # Show specific scenario in JSON
    vrooli scenario open app-monitor             # Open scenario in browser
    vrooli scenario open app-monitor --print-url # Print URL instead of opening
    vrooli scenario port app-monitor UI_PORT     # Get UI port number
EOF
}

# Main handler
main() {
    if [[ $# -eq 0 ]] || [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
        show_scenario_help
        return 0
    fi
    
    local subcommand="$1"; shift
    case "$subcommand" in
        start|run)  # Support both 'start' and 'run' (run is silent alias)
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
            
            # Start each scenario
            local overall_result=0
            if [[ ${#scenario_names[@]} -gt 1 ]]; then
                log::info "Starting ${#scenario_names[@]} scenarios: ${scenario_names[*]}"
            fi
            
            for scenario_name in "${scenario_names[@]}"; do
                if [[ ${#scenario_names[@]} -gt 1 ]]; then
                    log::info "Starting scenario: $scenario_name"
                fi
                scenario::run "$scenario_name" develop "$@" || overall_result=$?
            done
            
            return $overall_result
            ;;
        start-all)
            # Start all scenarios via API endpoint
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
            ;;
        restart)
            # Restart scenario - simply stop then start
            local scenario_name="${1:-}"
            [[ -z "$scenario_name" ]] && { 
                log::error "Scenario name required"
                log::info "Usage: vrooli scenario restart <name>"
                return 1
            }
            shift
            
            log::info "Restarting scenario: $scenario_name"
            
            # Stop if running (ignore errors - scenario might not be running)
            "$0" stop "$scenario_name" 2>/dev/null || true
            
            # Brief pause
            sleep 1
            
            # Start with any additional options passed through
            "$0" start "$scenario_name" "$@"
            ;;
        stop)
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
            ;;
        stop-all)
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
            ;;
        test)
            local scenario_name="${1:-}"
            [[ -z "$scenario_name" ]] && { 
                log::error "Scenario name required"
                log::info "Usage: vrooli scenario test <name>"
                return 1
            }
            shift
            scenario::run "$scenario_name" test "$@"
            ;;
        list)
            scenario::list "$@"
            ;;
        logs)
            local scenario_name="${1:-}"
            [[ -z "$scenario_name" ]] && { 
                log::error "Scenario name required"
                log::info "Usage: vrooli scenario logs <name> [options]"
                log::info "Options:"
                log::info "  --follow, -f        Follow log output in real-time"
                log::info "  --step <name>       View specific background step log"
                log::info "  --runtime           View all background process logs"
                log::info "  --lifecycle         View lifecycle log (default behavior)"
                log::info ""
                log::info "Available scenarios with logs:"
                ls -1 "${HOME}/.vrooli/logs/scenarios/" 2>/dev/null || echo "  (none found)"
                return 1
            }
            shift
            
            # Parse flags
            local follow=false
            local step_name=""
            local show_lifecycle=false
            local show_runtime=false
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --follow|-f)
                        follow=true
                        shift
                        ;;
                    --step)
                        step_name="${2:-}"
                        if [[ -z "$step_name" ]]; then
                            log::error "--step requires a step name"
                            return 1
                        fi
                        shift 2
                        ;;
                    --lifecycle)
                        show_lifecycle=true
                        shift
                        ;;
                    --runtime)
                        show_runtime=true
                        shift
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            # Show runtime logs if requested
            if [[ "$show_runtime" == "true" ]]; then
                local logs_dir="${HOME}/.vrooli/logs/scenarios/${scenario_name}"
                if [[ ! -d "$logs_dir" ]]; then
                    log::warn "No runtime logs found for scenario: $scenario_name"
                    return 1
                fi
                
                # Check for log files
                local log_files=("$logs_dir"/*.log)
                if [[ ! -e "${log_files[0]}" ]]; then
                    log::warn "No runtime log files found in $logs_dir"
                    return 1
                fi
                
                # Display runtime logs
                if [[ "$follow" == "true" ]]; then
                    log::info "Following runtime logs for scenario: $scenario_name"
                    log::info "Press Ctrl+C to stop viewing"
                    echo ""
                    tail -f "$logs_dir"/*.log
                else
                    log::info "Showing recent runtime logs for scenario: $scenario_name"
                    echo ""
                    # Show last 50 lines from each log file
                    for log_file in "$logs_dir"/*.log; do
                        if [[ -f "$log_file" ]]; then
                            echo "==> $(basename "$log_file") <=="
                            tail -50 "$log_file"
                            echo ""
                        fi
                    done
                    log::info "Tip: Use --step <name> to view a specific background process log"
                fi
                return 0
            fi
            
            # Show specific step log if requested
            if [[ -n "$step_name" ]]; then
                local logs_dir="${HOME}/.vrooli/logs/scenarios/${scenario_name}"
                
                # Find log files matching the step name
                local step_log=""
                shopt -s nullglob
                for log_file in "$logs_dir"/vrooli.*."${scenario_name}"."${step_name}".log; do
                    if [[ -f "$log_file" ]]; then
                        step_log="$log_file"
                        break
                    fi
                done
                shopt -u nullglob
                
                if [[ -z "$step_log" ]]; then
                    log::error "No log found for step '$step_name'"
                    log::info "This could mean:"
                    log::info "  • The step hasn't been reached yet (check earlier steps)"
                    log::info "  • The step isn't a background process (check --lifecycle)"
                    log::info "  • The step name is incorrect"
                    echo ""
                    log::info "Available background step logs:"
                    shopt -s nullglob
                    for log_file in "$logs_dir"/vrooli.*.log; do
                        if [[ -f "$log_file" ]]; then
                            local basename=$(basename "$log_file")
                            # Extract step name from log filename (format: vrooli.phase.scenario.step.log)
                            local extracted_step=$(echo "$basename" | sed -E "s/vrooli\.[^.]+\.${scenario_name}\.(.+)\.log/\1/")
                            echo "  • $extracted_step"
                        fi
                    done
                    shopt -u nullglob
                    return 1
                fi
                
                if [[ "$follow" == "true" ]]; then
                    log::info "Following log for step '$step_name' in scenario: $scenario_name"
                    log::info "Press Ctrl+C to stop viewing"
                    echo ""
                    tail -f "$step_log"
                else
                    log::info "Showing recent log for step '$step_name' in scenario: $scenario_name"
                    echo ""
                    echo "==> $(basename "$step_log") <=="
                    tail -100 "$step_log"
                    echo ""
                fi
                return 0
            fi
            
            # Default behavior: show lifecycle log with discovery information
            local lifecycle_log="${HOME}/.vrooli/logs/${scenario_name}.log"
            local logs_dir="${HOME}/.vrooli/logs/scenarios/${scenario_name}"
            
            # Check if lifecycle log exists
            if [[ ! -f "$lifecycle_log" ]]; then
                log::warn "No lifecycle log found for scenario: $scenario_name"
                log::info "This scenario may not have been run yet"
                
                # Check if there are any background logs
                if [[ -d "$logs_dir" ]]; then
                    local log_files=("$logs_dir"/*.log)
                    if [[ -e "${log_files[0]}" ]]; then
                        log::info "Background process logs are available. Use --runtime to view them"
                    fi
                fi
                return 1
            fi
            
            # Display lifecycle log
            if [[ "$follow" == "true" ]]; then
                log::info "Following lifecycle log for scenario: $scenario_name"
                log::info "Press Ctrl+C to stop viewing"
                echo ""
                tail -f "$lifecycle_log"
            else
                log::info "Showing recent lifecycle execution for scenario: $scenario_name"
                echo ""
                echo "==> Lifecycle execution log <=="
                echo "────────────────────────────────────────────────────────"
                # Show more lines from lifecycle log to capture full execution flow
                tail -100 "$lifecycle_log"
                echo ""
                
                # Show discovery information
                echo "────────────────────────────────────────────────────────"
                echo "📋 BACKGROUND STEP LOGS AVAILABLE:"
                echo ""
                
                # Find service.json to determine expected background steps
                local service_json=""
                if [[ -f "${APP_ROOT}/scenarios/${scenario_name}/.vrooli/service.json" ]]; then
                    service_json="${APP_ROOT}/scenarios/${scenario_name}/.vrooli/service.json"
                elif [[ -f "${APP_ROOT}/scenarios/${scenario_name}/service.json" ]]; then
                    service_json="${APP_ROOT}/scenarios/${scenario_name}/service.json"
                fi
                
                # List available logs with step extraction
                local found_steps=()
                for log_file in "$logs_dir"/vrooli.*.log; do
                    if [[ -f "$log_file" ]]; then
                        local basename=$(basename "$log_file")
                        # Extract phase and step from log filename (format: vrooli.phase.scenario.step.log)
                        if [[ "$basename" =~ vrooli\.([^.]+)\.${scenario_name}\.(.+)\.log ]]; then
                            local phase="${BASH_REMATCH[1]}"
                            local step="${BASH_REMATCH[2]}"
                            found_steps+=("${step}:${phase}")
                            echo "  ✅ ${step} (${phase})"
                            echo "     View: vrooli scenario logs ${scenario_name} --step ${step}"
                            echo ""
                        fi
                    fi
                done
                
                # Parse service.json to find expected background steps if possible
                if [[ -n "$service_json" ]] && [[ -f "$service_json" ]] && command -v jq >/dev/null 2>&1; then
                    # Look for background steps in all lifecycle phases
                    local expected_steps=$(jq -r '
                        .lifecycle | 
                        to_entries[] | 
                        select(.value | type == "object") |
                        select(.value.steps) |
                        .key as $phase |
                        .value.steps[]? | 
                        select(.background == true) | 
                        "\(.name):\($phase)"
                    ' "$service_json" 2>/dev/null || true)
                    
                    # Check for missing expected steps
                    if [[ -n "$expected_steps" ]]; then
                        while IFS= read -r expected; do
                            local found=false
                            for fs in "${found_steps[@]}"; do
                                if [[ "$fs" == "$expected" ]]; then
                                    found=true
                                    break
                                fi
                            done
                            
                            if [[ "$found" == "false" ]]; then
                                IFS=':' read -r step phase <<< "$expected"
                                echo "  ⚠️  ${step} (${phase}) - [NOT FOUND]"
                                echo "     Expected but missing - step may not have been reached"
                                echo "     Would view with: vrooli scenario logs ${scenario_name} --step ${step}"
                                echo ""
                            fi
                        done <<< "$expected_steps"
                    fi
                else
                    # If we can't parse service.json, just note common missing steps
                    if [[ ! -f "${logs_dir}/vrooli.develop.${scenario_name}.start-ui.log" ]] && \
                       [[ -f "${logs_dir}/vrooli.develop.${scenario_name}.start-api.log" ]]; then
                        echo "  ⚠️  start-ui (develop) - [NOT FOUND]"
                        echo "     Expected but missing - step may not have been reached"
                        echo "     Would view with: vrooli scenario logs ${scenario_name} --step start-ui"
                        echo ""
                    fi
                fi
                
                echo "💡 Tips:"
                echo "  • Use --runtime to view all background process logs"
                echo "  • Use --step <name> to view a specific background process log"
                echo "  • Use --follow or -f to watch logs in real-time"
                echo "  • Missing logs usually mean the step wasn't reached"
                echo "  • The lifecycle log above shows the complete execution sequence"
            fi
            ;;
        status)
            local scenario_name="${1:-}"
            local json_output=false
            
            # Parse arguments for --json flag
            if [[ "$scenario_name" == "--json" ]]; then
                json_output=true
                scenario_name=""
            elif [[ "${2:-}" == "--json" ]]; then
                json_output=true
            fi
            
            # Get API port from environment (main Go API, not orchestrator)
            local api_port="${VROOLI_API_PORT:-8092}"
            local api_url="http://localhost:${api_port}"
            
            # Check if API is reachable
            # Use simplest curl command for maximum compatibility
            if ! timeout 10 curl -s "${api_url}/health" >/dev/null 2>&1; then
                log::error "Vrooli API is not accessible at ${api_url}"
                log::info "The API may not be running. Start it with: vrooli develop"
                return 1
            fi
            
            if [[ -z "$scenario_name" ]]; then
                # Show all scenarios status
                if [[ "$json_output" != "true" ]]; then
                    log::info "Fetching status for all scenarios from API..."
                    echo ""
                fi
                
                local response
                response=$(curl -s "${api_url}/scenarios" 2>/dev/null)
                
                if [[ -z "$response" ]]; then
                    if [[ "$json_output" == "true" ]]; then
                        echo '{"error": "Failed to get response from API"}'
                    else
                        log::error "Failed to get response from API"
                    fi
                    return 1
                fi
                
                # If JSON output requested, enhance with summary statistics
                if [[ "$json_output" == "true" ]]; then
                    # Parse the data to calculate summary statistics
                    local data_check=$(echo "$response" | jq -r '.data' 2>/dev/null)
                    local total=0
                    local running=0
                    
                    if [[ "$data_check" != "null" ]] && [[ "$data_check" != "" ]]; then
                        total=$(echo "$response" | jq -r '.data | length' 2>/dev/null)
                        running=$(echo "$response" | jq -r '.data | map(select(.status == "running")) | length' 2>/dev/null)
                    fi
                    
                    # Create enhanced JSON response with summary
                    local enhanced_response=$(echo "$response" | jq --argjson total "$total" --argjson running "$running" --argjson stopped "$((total - running))" '
                    {
                        "success": .success,
                        "summary": {
                            "total_scenarios": $total,
                            "running": $running,
                            "stopped": $stopped
                        },
                        "scenarios": (.data // []),
                        "raw_response": .
                    }')
                    echo "$enhanced_response"
                    return 0
                fi
                
                # Parse JSON response from native Go API format
                local total running
                if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
                    # Handle case where data is null (no running scenarios)
                    local data_check=$(echo "$response" | jq -r '.data' 2>/dev/null)
                    if [[ "$data_check" == "null" ]] || [[ "$data_check" == "" ]]; then
                        total=0
                        running=0
                    else
                        total=$(echo "$response" | jq -r '.data | length' 2>/dev/null)
                        running=$(echo "$response" | jq -r '.data | map(select(.status == "running")) | length' 2>/dev/null)
                    fi
                else
                    total=0
                    running=0
                fi
                
                echo "📊 SCENARIO STATUS SUMMARY"
                echo "═══════════════════════════════════════"
                echo "Total Scenarios: $total"
                echo "Running: $running"
                echo "Stopped: $((total - running))"
                
                # Check for system warnings
                local system_warnings=$(echo "$response" | jq -r '.system_warnings // null' 2>/dev/null)
                if [[ "$system_warnings" != "null" ]] && [[ -n "$system_warnings" ]]; then
                    echo ""
                    echo "$response" | jq -r '.system_warnings[]? | .emoji + " " + .message + " (system-wide)"' 2>/dev/null
                fi
                echo ""
                
                if [[ "$total" -gt 0 ]] && [[ "$data_check" != "null" ]]; then
                    echo "SCENARIO                          STATUS      RUNTIME         PORT(S)"
                    echo "───────────────────────────────────────────────────────────────────────────"
                    
                    # Parse individual scenarios from native Go API format
                    echo "$response" | jq -r '.data[] | "\(.name)|\(.status)|" + (.runtime // "N/A") + "|" + (.ports | if . == {} or . == null then "" else (. | to_entries | map("\(.key):http://localhost:\(.value)") | join(", ")) end) + "|" + (.health_status // "unknown")' 2>/dev/null | while IFS='|' read -r name status runtime port_list health; do
                        # Color code status based on actual health
                        local status_display
                        if [ "$status" = "running" ]; then
                            case "$health" in
                                healthy)
                                    status_display="🟢 healthy"
                                    ;;
                                degraded)
                                    status_display="🟡 degraded"
                                    ;;
                                unhealthy)
                                    status_display="🔴 unhealthy"
                                    ;;
                                running|unknown)
                                    status_display="🟡 running"
                                    ;;
                                *)
                                    status_display="🟡 running"
                                    ;;
                            esac
                        else
                            case "$status" in
                                stopped)
                                    # Check if there's error context for better status display
                                    local error_context
                                    error_context=$(echo "$response" | jq -r --arg name "$name" '.data[] | select(.name == $name) | .last_error // ""' 2>/dev/null || echo "")
                                    if [[ -n "$error_context" ]] && [[ "$error_context" != "null" ]] && [[ "$error_context" != "" ]]; then
                                        if [[ "$error_context" =~ PORT.*CONFLICT|port.*conflict|Lock|LOCK ]]; then
                                            status_display="⚫ port conflict"
                                        elif [[ "$error_context" =~ RANGE.*EXHAUSTED|range.*exhausted ]]; then
                                            status_display="⚫ no ports"
                                        else
                                            status_display="⚫ error"
                                        fi
                                    else
                                        status_display="⚫ stopped"
                                    fi
                                    ;;
                                error)
                                    status_display="🔴 error"
                                    ;;
                                starting)
                                    status_display="🟡 starting"
                                    ;;
                                failed)
                                    status_display="🔴 failed"
                                    ;;
                                *)
                                    status_display="❓ $status"
                                    ;;
                            esac
                        fi
                        
                        printf "%-33s %-11s %-15s %s\n" "$name" "$status_display" "$runtime" "$port_list"
                    done
                fi
            else
                # Show specific scenario status
                if [[ "$json_output" != "true" ]]; then
                    log::info "Fetching status for scenario: $scenario_name"
                    echo ""
                fi
                
                local response
                response=$(curl -s "${api_url}/scenarios/${scenario_name}/status" 2>/dev/null)
                
                if echo "$response" | grep -q "not found"; then
                    if [[ "$json_output" == "true" ]]; then
                        echo "{\"error\": \"Scenario '$scenario_name' not found in API\"}"
                    else
                        log::error "Scenario '$scenario_name' not found in API"
                        log::info "Use 'vrooli scenario status' to see all available scenarios"
                    fi
                    return 1
                fi
                
                # If JSON output requested, enhance with metadata
                if [[ "$json_output" == "true" ]]; then
                    # Create enhanced JSON response for individual scenario
                    local enhanced_response=$(echo "$response" | jq --arg scenario_name "$scenario_name" '
                    {
                        "success": .success,
                        "scenario_name": $scenario_name,
                        "scenario_data": .data,
                        "raw_response": .,
                        "metadata": {
                            "query_type": "individual_scenario",
                            "timestamp": (now | strftime("%Y-%m-%d %H:%M:%S UTC"))
                        }
                    }')
                    echo "$enhanced_response"
                    return 0
                fi
                
                # Parse and display detailed status from native Go API format
                local status pid started_at stopped_at restart_count port_info runtime_formatted
                status=$(echo "$response" | jq -r '.data.status // "unknown"' 2>/dev/null)
                # Get PIDs from processes array (just show count for now)
                local process_count
                process_count=$(echo "$response" | jq -r '.data.processes | length // 0' 2>/dev/null)
                pid="$process_count processes"
                started_at=$(echo "$response" | jq -r '.data.started_at // "never"' 2>/dev/null)
                stopped_at="N/A"  # New orchestrator doesn't track stopped_at yet
                restart_count=0   # New orchestrator doesn't track restart_count yet
                runtime_formatted=$(echo "$response" | jq -r '.data.runtime // "N/A"' 2>/dev/null)
                
                echo "📋 SCENARIO: $scenario_name"
                echo "═══════════════════════════════════════"
                
                # Status with color and detailed failure types
                local health_status
                health_status=$(echo "$response" | jq -r '.data.health_status // "unknown"' 2>/dev/null)
                local last_error
                last_error=$(echo "$response" | jq -r '.data.last_error // ""' 2>/dev/null)
                
                case "$status" in
                    running)
                        case "$health_status" in
                            healthy)
                                echo "Status:        🟢 RUNNING (healthy)"
                                ;;
                            degraded)
                                echo "Status:        🟡 RUNNING (degraded)"
                                ;;
                            unhealthy)
                                echo "Status:        🔴 RUNNING (unhealthy)"
                                ;;
                            *)
                                echo "Status:        🟢 RUNNING"
                                ;;
                        esac
                        ;;
                    stopped)
                        if [[ -n "$last_error" ]] && [[ "$last_error" != "null" ]]; then
                            if [[ "$last_error" =~ PORT.*CONFLICT|port.*conflict|Lock|LOCK ]]; then
                                echo "Status:        ⚫ STOPPED (port conflict)"
                                echo "Issue:         Port allocation failed"
                                echo "Solution:      rm ~/.vrooli/state/scenarios/.port_*.lock"
                            elif [[ "$last_error" =~ RANGE.*EXHAUSTED|range.*exhausted ]]; then
                                echo "Status:        ⚫ STOPPED (no ports available)"
                                echo "Issue:         Port range exhausted"
                                echo "Solution:      vrooli resource restart"
                            else
                                echo "Status:        ⚫ STOPPED (error)"
                                echo "Last Error:    $last_error"
                            fi
                        else
                            echo "Status:        ⚫ STOPPED"
                        fi
                        ;;
                    error)
                        echo "Status:        🔴 ERROR"
                        if [[ -n "$last_error" ]] && [[ "$last_error" != "null" ]]; then
                            echo "Error Detail:  $last_error"
                        fi
                        ;;
                    starting)
                        echo "Status:        🟡 STARTING"
                        ;;
                    failed)
                        echo "Status:        🔴 FAILED"
                        if [[ -n "$last_error" ]] && [[ "$last_error" != "null" ]]; then
                            echo "Failure:       $last_error"
                        fi
                        ;;
                    *)
                        echo "Status:        ❓ $status"
                        ;;
                esac
                
                [[ "$pid" != "null" ]] && [[ "$pid" != "N/A" ]] && echo "Process ID:    $pid"
                [[ "$started_at" != "null" ]] && [[ "$started_at" != "never" ]] && echo "Started:       $started_at"
                [[ "$runtime_formatted" != "null" ]] && [[ "$runtime_formatted" != "N/A" ]] && echo "Runtime:       $runtime_formatted"
                [[ "$stopped_at" != "null" ]] && [[ "$stopped_at" != "N/A" ]] && echo "Stopped:       $stopped_at"
                [[ "$restart_count" != "0" ]] && echo "Restarts:      $restart_count"
                
                # Show allocated ports from native Go API format
                local ports
                ports=$(echo "$response" | jq -r '.data.allocated_ports // {}' 2>/dev/null)
                if [[ "$ports" != "{}" ]] && [[ "$ports" != "null" ]]; then
                    echo ""
                    echo "Allocated Ports:"
                    echo "$ports" | jq -r 'to_entries[] | "  \(.key): \(.value)"' 2>/dev/null
                fi
                
                # Show port status if available
                local port_status
                port_status=$(echo "$response" | jq -r '.port_status // {}' 2>/dev/null)
                if [[ "$port_status" != "{}" ]] && [[ "$port_status" != "null" ]]; then
                    echo ""
                    echo "Port Status:"
                    echo "$port_status" | jq -r 'to_entries[] | "  \(.key): http://localhost:\(.value.port) - \(if .value.listening then "✓ listening" else "✗ not listening" end)"' 2>/dev/null
                fi
                
                echo ""
                if [[ "$status" == "stopped" ]]; then
                    log::info "Start with: vrooli scenario start $scenario_name"
                elif [[ "$status" == "running" ]]; then
                    log::info "View logs with: vrooli scenario logs $scenario_name"
                fi
            fi
            ;;
        port)
            # Ultra-fast command to get a specific port for a scenario
            # Reads directly from process JSON files, no HTTP calls
            local scenario_name="${1:-}"
            local port_name="${2:-}"
            
            if [[ -z "$scenario_name" ]] || [[ -z "$port_name" ]]; then
                log::error "Scenario name and port name required"
                echo "Usage: vrooli scenario port <scenario-name> <port-name>"
                echo ""
                echo "Examples:"
                echo "  vrooli scenario port ecosystem-manager API_PORT"
                echo "  vrooli scenario port ecosystem-manager UI_PORT"
                return 1
            fi
            
            # Map port names to process step names
            # This is a convention: API_PORT -> start-api, UI_PORT -> start-ui
            local step_name=""
            case "$port_name" in
                API_PORT)
                    step_name="start-api"
                    ;;
                UI_PORT)
                    step_name="start-ui"
                    ;;
                *)
                    # For other port names, try lowercase without _PORT suffix
                    # e.g., WORKER_PORT -> start-worker
                    local base_name="${port_name%_PORT}"
                    step_name="start-${base_name,,}"
                    ;;
            esac
            
            # Direct file read - extremely fast
            local process_file="$HOME/.vrooli/processes/scenarios/${scenario_name}/${step_name}.json"
            
            if [[ ! -f "$process_file" ]]; then
                # Process not running or doesn't exist
                return 1
            fi
            
            # Extract port directly from JSON file (single jq call, very fast)
            local port_value
            port_value=$(jq -r '.port // empty' "$process_file" 2>/dev/null)
            
            if [[ -n "$port_value" ]] && [[ "$port_value" != "null" ]]; then
                # Just output the port number, nothing else (for easy scripting)
                echo "$port_value"
                return 0
            else
                # Port not found in process file
                return 1
            fi
            ;;
        open)
            # Open scenario in browser if it has a UI
            local scenario_name="${1:-}"
            local port_name="UI_PORT"
            local print_url=false
            
            # Parse options
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --port)
                        port_name="${2:-}"
                        if [[ -z "$port_name" ]]; then
                            log::error "--port requires a port name"
                            return 1
                        fi
                        shift 2
                        ;;
                    --print-url)
                        print_url=true
                        shift
                        ;;
                    *)
                        if [[ -z "$scenario_name" ]]; then
                            scenario_name="$1"
                        fi
                        shift
                        ;;
                esac
            done
            
            if [[ -z "$scenario_name" ]]; then
                log::error "Scenario name required"
                echo "Usage: vrooli scenario open <scenario-name> [options]"
                echo ""
                echo "Options:"
                echo "  --port <name>    Open specific port (default: UI_PORT)"
                echo "  --print-url      Print URL instead of opening browser"
                echo ""
                echo "Examples:"
                echo "  vrooli scenario open app-monitor"
                echo "  vrooli scenario open app-monitor --port API_PORT"
                echo "  vrooli scenario open app-monitor --print-url"
                return 1
            fi
            
            # Use the existing port command logic to get the port
            local port_value
            local port_result=0
            
            # Inline port logic instead of recursive call
            # Map port names to process step names
            local step_name=""
            case "$port_name" in
                API_PORT)
                    step_name="start-api"
                    ;;
                UI_PORT)
                    step_name="start-ui"
                    ;;
                *)
                    # For other port names, try lowercase without _PORT suffix
                    local base_name="${port_name%_PORT}"
                    step_name="start-${base_name,,}"
                    ;;
            esac
            
            # Direct file read - extremely fast
            local process_file="$HOME/.vrooli/processes/scenarios/${scenario_name}/${step_name}.json"
            
            if [[ ! -f "$process_file" ]]; then
                # Process not running or doesn't exist
                port_result=1
            else
                # Extract port directly from JSON file
                port_value=$(jq -r '.port // empty' "$process_file" 2>/dev/null)
                
                if [[ -z "$port_value" ]] || [[ "$port_value" == "null" ]]; then
                    # Port not found in process file
                    port_result=1
                fi
            fi
            
            if [[ $port_result -ne 0 ]] || [[ -z "$port_value" ]]; then
                case "$port_name" in
                    UI_PORT)
                        log::error "No UI port found for scenario '$scenario_name'"
                        log::info "This scenario may not have a web interface, or it's not running"
                        ;;
                    API_PORT)
                        log::error "No API port found for scenario '$scenario_name'"
                        ;;
                    *)
                        log::error "No port '$port_name' found for scenario '$scenario_name'"
                        ;;
                esac
                log::info "Start the scenario first: vrooli scenario start $scenario_name"
                return 1
            fi
            
            local url="http://localhost:$port_value"
            
            if [[ "$print_url" == "true" ]]; then
                echo "$url"
                return 0
            fi
            
            log::info "Opening $scenario_name at $url"
            
            # Detect platform and open browser
            case "$(uname -s)" in
                Linux*)
                    if command -v xdg-open >/dev/null 2>&1; then
                        xdg-open "$url" >/dev/null 2>&1 &
                    elif command -v firefox >/dev/null 2>&1; then
                        firefox "$url" >/dev/null 2>&1 &
                    elif command -v google-chrome >/dev/null 2>&1; then
                        google-chrome "$url" >/dev/null 2>&1 &
                    elif command -v chromium >/dev/null 2>&1; then
                        chromium "$url" >/dev/null 2>&1 &
                    else
                        log::warn "No browser found. Please open manually: $url"
                        return 1
                    fi
                    ;;
                Darwin*)
                    open "$url"
                    ;;
                CYGWIN*|MINGW32*|MSYS*|MINGW*)
                    start "$url"
                    ;;
                *)
                    log::warn "Unknown platform. Please open manually: $url"
                    echo "$url"
                    return 1
                    ;;
            esac
            ;;
        # Removed: convert, convert-all, validate, enable, disable
        *)
            log::error "Unknown scenario command: $subcommand"
            echo ""
            show_scenario_help
            return 1
            ;;
    esac
}

main "$@"