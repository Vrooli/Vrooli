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
    run <name>              Run a scenario directly
    test <name>             Test a scenario
    list                    List available scenarios
    logs <name> [options]   View logs for a scenario
    status [name] [--json]  Show scenario status from orchestrator

OPTIONS FOR LOGS:
    --follow, -f            Follow log output (live view)
    --step <name>           View specific background step log
    --runtime               View all background process logs
    --lifecycle             View lifecycle log (default behavior)

EXAMPLES:
    vrooli scenario run make-it-vegan
    vrooli scenario test swarm-manager
    vrooli scenario list
    vrooli scenario logs system-monitor              # Shows lifecycle execution
    vrooli scenario logs system-monitor --follow      # Follow lifecycle log
    vrooli scenario logs swarm-manager --runtime      # Show all background logs
    vrooli scenario logs swarm-manager --step start-api  # Show specific service
    vrooli scenario status              # Show all scenarios
    vrooli scenario status --json       # Show all scenarios in JSON format
    vrooli scenario status system-monitor  # Show specific scenario
    vrooli scenario status system-monitor --json  # Show specific scenario in JSON
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
        run)
            local scenario_name="${1:-}"
            [[ -z "$scenario_name" ]] && { 
                log::error "Scenario name required"
                log::info "Usage: vrooli scenario run <name>"
                return 1
            }
            shift
            scenario::run "$scenario_name" develop "$@"
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
            scenario::list
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
            
            # Get orchestrator port from registry or environment
            local orchestrator_port="${ORCHESTRATOR_PORT:-}"
            if [[ -z "$orchestrator_port" ]]; then
                # Try to get from port registry
                if [[ -f "${APP_ROOT}/scripts/resources/port_registry.sh" ]]; then
                    source "${APP_ROOT}/scripts/resources/port_registry.sh"
                    orchestrator_port=$(ports::get_resource_port "vrooli-orchestrator")
                fi
                # Fallback to default if still empty
                orchestrator_port="${orchestrator_port:-9500}"
            fi
            local orchestrator_api="http://localhost:${orchestrator_port}"
            
            # Check if orchestrator is reachable
            # Use simplest curl command for maximum compatibility
            if ! timeout 10 curl -s "${orchestrator_api}/health" >/dev/null 2>&1; then
                log::error "Orchestrator API is not accessible at ${orchestrator_api}"
                log::info "The orchestrator may not be running. Start it with: vrooli develop"
                return 1
            fi
            
            if [[ -z "$scenario_name" ]]; then
                # Show all scenarios status
                if [[ "$json_output" != "true" ]]; then
                    log::info "Fetching status for all scenarios from orchestrator..."
                    echo ""
                fi
                
                local response
                response=$(curl -s "${orchestrator_api}/scenarios" 2>/dev/null)
                
                if [[ -z "$response" ]]; then
                    if [[ "$json_output" == "true" ]]; then
                        echo '{"error": "Failed to get response from orchestrator"}'
                    else
                        log::error "Failed to get response from orchestrator"
                    fi
                    return 1
                fi
                
                # If JSON output requested, just pass through the response
                if [[ "$json_output" == "true" ]]; then
                    echo "$response"
                    return 0
                fi
                
                # Parse JSON response from new orchestrator format
                local total running
                if echo "$response" | jq -e '.success' >/dev/null 2>&1; then
                    total=$(echo "$response" | jq -r '.data | length' 2>/dev/null)
                    running=$(echo "$response" | jq -r '.data | map(select(.status == "running")) | length' 2>/dev/null)
                else
                    total=0
                    running=0
                fi
                
                echo "📊 SCENARIO STATUS SUMMARY"
                echo "═══════════════════════════════════════"
                echo "Total Scenarios: $total"
                echo "Running: $running"
                echo "Stopped: $((total - running))"
                echo ""
                
                if [[ "$total" -gt 0 ]]; then
                    echo "SCENARIO                          STATUS      RUNTIME         PORT(S)"
                    echo "───────────────────────────────────────────────────────────────────────────"
                    
                    # Parse individual scenarios from new orchestrator format
                    echo "$response" | jq -r '.data[] | "\(.name)|\(.status)|" + (.runtime // "N/A") + "|" + (.allocated_ports | if . == {} or . == null then "" else (. | to_entries | map("\(.key):http://localhost:\(.value)") | join(", ")) end) + "|running"' 2>/dev/null | while IFS='|' read -r name status runtime port_list health; do
                        # Color code status based on actual health
                        local status_display
                        if [ "$status" = "running" ]; then
                            case "$health" in
                                healthy)
                                    status_display="🟢 running"
                                    ;;
                                degraded)
                                    status_display="🟡 degraded"
                                    ;;
                                unhealthy)
                                    status_display="🔴 unhealthy"
                                    ;;
                                *)
                                    status_display="🟢 running"
                                    ;;
                            esac
                        else
                            case "$status" in
                                stopped)
                                    status_display="⚫ stopped"
                                    ;;
                                error)
                                    status_display="🔴 error"
                                    ;;
                                starting)
                                    status_display="🟡 starting"
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
                response=$(curl -s "${orchestrator_api}/scenarios/${scenario_name}" 2>/dev/null)
                
                if echo "$response" | grep -q "not found"; then
                    if [[ "$json_output" == "true" ]]; then
                        echo "{\"error\": \"Scenario '$scenario_name' not found in orchestrator\"}"
                    else
                        log::error "Scenario '$scenario_name' not found in orchestrator"
                        log::info "Use 'vrooli scenario status' to see all available scenarios"
                    fi
                    return 1
                fi
                
                # If JSON output requested, just pass through the response
                if [[ "$json_output" == "true" ]]; then
                    echo "$response"
                    return 0
                fi
                
                # Parse and display detailed status from new orchestrator format
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
                
                # Status with color
                case "$status" in
                    running)
                        echo "Status:        🟢 RUNNING"
                        ;;
                    stopped)
                        echo "Status:        ⚫ STOPPED"
                        ;;
                    error)
                        echo "Status:        🔴 ERROR"
                        ;;
                    starting)
                        echo "Status:        🟡 STARTING"
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
                
                # Show allocated ports from new orchestrator format
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
                    log::info "Start with: vrooli scenario run $scenario_name"
                elif [[ "$status" == "running" ]]; then
                    log::info "View logs with: vrooli scenario logs $scenario_name"
                fi
            fi
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