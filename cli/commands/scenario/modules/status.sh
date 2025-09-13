#!/usr/bin/env bash
# Scenario Status Display Module
# Handles status formatting, display, and API communication

set -euo pipefail

# Show scenario status
scenario::status::show() {
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
    if ! timeout 10 curl -s "${api_url}/health" >/dev/null 2>&1; then
        log::error "Vrooli API is not accessible at ${api_url}"
        log::info "The API may not be running. Start it with: vrooli develop"
        return 1
    fi
    
    if [[ -z "$scenario_name" ]]; then
        scenario::status::show_all "$api_url" "$json_output"
    else
        scenario::status::show_individual "$api_url" "$scenario_name" "$json_output"
    fi
}

# Show status for all scenarios
scenario::status::show_all() {
    local api_url="$1"
    local json_output="$2"
    
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
        scenario::status::format_json_all "$response"
        return 0
    fi
    
    scenario::status::format_display_all "$response"
}

# Show status for individual scenario
scenario::status::show_individual() {
    local api_url="$1"
    local scenario_name="$2"
    local json_output="$3"
    
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
        scenario::status::format_json_individual "$response" "$scenario_name"
        return 0
    fi
    
    scenario::status::format_display_individual "$response" "$scenario_name"
}

# Format JSON output for all scenarios
scenario::status::format_json_all() {
    local response="$1"
    
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
}

# Format JSON output for individual scenario
scenario::status::format_json_individual() {
    local response="$1"
    local scenario_name="$2"
    
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
}

# Format display output for all scenarios
scenario::status::format_display_all() {
    local response="$1"
    
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
        
        scenario::status::format_scenario_list "$response"
    fi
}

# Format individual scenario display
scenario::status::format_display_individual() {
    local response="$1"
    local scenario_name="$2"
    
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
    
    scenario::status::format_status "$status" "$health_status" "$last_error"
    
    [[ "$pid" != "null" ]] && [[ "$pid" != "N/A" ]] && echo "Process ID:    $pid"
    [[ "$started_at" != "null" ]] && [[ "$started_at" != "never" ]] && echo "Started:       $started_at"
    [[ "$runtime_formatted" != "null" ]] && [[ "$runtime_formatted" != "N/A" ]] && echo "Runtime:       $runtime_formatted"
    [[ "$stopped_at" != "null" ]] && [[ "$stopped_at" != "N/A" ]] && echo "Stopped:       $stopped_at"
    [[ "$restart_count" != "0" ]] && echo "Restarts:      $restart_count"
    
    # Automatic failure diagnosis for stopped scenarios
    if [[ "$status" == "stopped" ]]; then
        scenario::health::diagnose_failure "$scenario_name" "false"
    fi
    
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
    
    # Enhanced Health Checks with Schema Validation
    scenario::health::check_scenario "$scenario_name" "$response" "$status"
    
    echo ""
    if [[ "$status" == "stopped" ]]; then
        log::info "Start with: vrooli scenario start $scenario_name"
    elif [[ "$status" == "running" ]]; then
        log::info "View logs with: vrooli scenario logs $scenario_name"
    fi
}

# Format scenario list for all scenarios display
scenario::status::format_scenario_list() {
    local response="$1"
    
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
}

# Format status with colors and detailed information
scenario::status::format_status() {
    local status="$1"
    local health_status="$2"
    local last_error="$3"
    
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
}