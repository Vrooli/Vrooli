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
    logs <name> [--follow]  View logs for a scenario
    status [name] [--json]  Show scenario status from orchestrator

OPTIONS FOR LOGS:
    --follow, -f            Follow log output (live view)

EXAMPLES:
    vrooli scenario run make-it-vegan
    vrooli scenario test swarm-manager
    vrooli scenario list
    vrooli scenario logs system-monitor
    vrooli scenario logs system-monitor --follow
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
                log::info "Usage: vrooli scenario logs <name> [--follow|-f]"
                log::info "Available scenarios with logs:"
                ls -1 "${HOME}/.vrooli/logs/scenarios/" 2>/dev/null || echo "  (none found)"
                return 1
            }
            shift
            
            # Check for follow flag
            local follow=false
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --follow|-f)
                        follow=true
                        shift
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            local logs_dir="${HOME}/.vrooli/logs/scenarios/${scenario_name}"
            if [[ ! -d "$logs_dir" ]]; then
                log::warn "No logs found for scenario: $scenario_name"
                log::info "Available scenarios with logs:"
                ls -1 "${HOME}/.vrooli/logs/scenarios/" 2>/dev/null || echo "  (none found)"
                return 1
            fi
            
            # Check for log files
            local log_files=("$logs_dir"/*.log)
            if [[ ! -e "${log_files[0]}" ]]; then
                log::warn "No log files found in $logs_dir"
                return 1
            fi
            
            # Display logs
            if [[ "$follow" == "true" ]]; then
                log::info "Following logs for scenario: $scenario_name"
                log::info "Press Ctrl+C to stop viewing"
                echo ""
                tail -f "$logs_dir"/*.log
            else
                log::info "Showing recent logs for scenario: $scenario_name"
                echo ""
                # Show last 50 lines from each log file
                for log_file in "$logs_dir"/*.log; do
                    if [[ -f "$log_file" ]]; then
                        echo "==> $(basename "$log_file") <=="
                        tail -50 "$log_file"
                        echo ""
                    fi
                done
                log::info "Tip: Use --follow or -f to watch logs in real-time"
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
            if ! curl -s -f --connect-timeout 2 --max-time 5 "${orchestrator_api}/health" >/dev/null 2>&1; then
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
                response=$(curl -s "${orchestrator_api}/apps" 2>/dev/null)
                
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
                
                # Parse JSON response
                local total running
                total=$(echo "$response" | jq -r '.total // 0' 2>/dev/null)
                running=$(echo "$response" | jq -r '.running // 0' 2>/dev/null)
                
                echo "📊 SCENARIO STATUS SUMMARY"
                echo "═══════════════════════════════════════"
                echo "Total Scenarios: $total"
                echo "Running: $running"
                echo "Stopped: $((total - running))"
                echo ""
                
                if [[ "$total" -gt 0 ]]; then
                    echo "SCENARIO                          STATUS      PORT(S)"
                    echo "────────────────────────────────────────────────────────"
                    
                    # Parse individual apps - format ports in jq to avoid shell parsing issues
                    echo "$response" | jq -r '.apps[] | "\(.name)|\(.status)|" + (.allocated_ports | if . == {} or . == null then "" else (. | to_entries | map("\(.key):\(.value)") | join(", ")) end) + "|" + (.actual_health // "unknown")' 2>/dev/null | while IFS='|' read -r name status port_list health; do
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
                        
                        printf "%-33s %-11s %s\n" "$name" "$status_display" "$port_list"
                    done
                fi
            else
                # Show specific scenario status
                if [[ "$json_output" != "true" ]]; then
                    log::info "Fetching status for scenario: $scenario_name"
                    echo ""
                fi
                
                local response
                response=$(curl -s "${orchestrator_api}/apps/${scenario_name}/status" 2>/dev/null)
                
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
                
                # Parse and display detailed status
                local status pid started_at stopped_at restart_count port_info
                status=$(echo "$response" | jq -r '.status // "unknown"' 2>/dev/null)
                pid=$(echo "$response" | jq -r '.pid // "N/A"' 2>/dev/null)
                started_at=$(echo "$response" | jq -r '.started_at // "never"' 2>/dev/null)
                stopped_at=$(echo "$response" | jq -r '.stopped_at // "N/A"' 2>/dev/null)
                restart_count=$(echo "$response" | jq -r '.restart_count // 0' 2>/dev/null)
                
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
                [[ "$stopped_at" != "null" ]] && [[ "$stopped_at" != "N/A" ]] && echo "Stopped:       $stopped_at"
                [[ "$restart_count" != "0" ]] && echo "Restarts:      $restart_count"
                
                # Show allocated ports
                local ports
                ports=$(echo "$response" | jq -r '.allocated_ports // {}' 2>/dev/null)
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
                    echo "$port_status" | jq -r 'to_entries[] | "  \(.key): port \(.value.port) - \(if .value.listening then "✓ listening" else "✗ not listening" end)"' 2>/dev/null
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