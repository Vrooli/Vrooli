#!/usr/bin/env bash
# Scenario Status Display Module
# Handles status formatting, display, and API communication

set -euo pipefail

# Source test infrastructure validator
TEST_VALIDATOR="${SCENARIO_CMD_DIR}/validators/test-validator.sh"
if [[ -f "$TEST_VALIDATOR" ]]; then
    source "$TEST_VALIDATOR"
fi

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
    local timeout_seconds=30
    local curl_output=""
    local curl_status=0
    curl_output=$(timeout "${timeout_seconds}" curl -s -o /dev/null "${api_url}/health" 2>&1) || curl_status=$?

    if [[ ${curl_status} -ne 0 ]]; then
        if [[ ${curl_status} -eq 124 ]]; then
            log::error "Vrooli API health check timed out after ${timeout_seconds}s at ${api_url}/health"
            log::info "The API may still be running but responded too slowly. Inspect the API logs or try again."
        else
            log::error "Vrooli API is not accessible at ${api_url}"
            if [[ -n "${curl_output}" ]]; then
                log::info "curl error: ${curl_output}"
            fi
            log::info "The API may not be running. Start it with: vrooli develop"
        fi
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

    SCENARIO_STATUS_REQUIREMENTS_SUMMARY=""
    SCENARIO_STATUS_EXTRA_RECOMMENDATIONS=""
    SCENARIO_STATUS_EXTRA_DOC_LINKS=""

    # Extract status from response
    local status
    status=$(echo "$response" | jq -r '.data.status // "unknown"' 2>/dev/null)

    # Collect comprehensive diagnostic data
    local diagnostic_data
    diagnostic_data=$(scenario::health::collect_all_diagnostic_data "$scenario_name" "$response" "$status")

    local insights_json
    insights_json=$(scenario::insights::collect_data "$scenario_name" 2>/dev/null || echo '{"stack":{},"resources":{},"scenario_dependencies":{},"packages":{},"lifecycle":{}}')

    local scenario_path="${APP_ROOT}/scenarios/${scenario_name}"

    # Collect test infrastructure validation data
    local test_validation="{}"
    if [[ -d "$scenario_path" ]] && command -v scenario::test::validate_infrastructure >/dev/null 2>&1; then
        test_validation=$(scenario::test::validate_infrastructure "$scenario_name" "$scenario_path")
    fi

    local requirement_summary='{"status":"unavailable"}'
    if [[ -d "$scenario_path" ]] && command -v scenario::requirements::quick_check >/dev/null 2>&1; then
        requirement_summary=$(scenario::requirements::quick_check "$scenario_name")
    fi
    SCENARIO_STATUS_REQUIREMENTS_SUMMARY="$requirement_summary"

    # Create enhanced JSON response for individual scenario with diagnostics and test validation
    local enhanced_response=$(echo "$response" | jq --arg scenario_name "$scenario_name" --argjson diagnostics "$diagnostic_data" --argjson test_infrastructure "$test_validation" --argjson insights "$insights_json" --argjson requirements "$requirement_summary" '
    {
        "success": .success,
        "scenario_name": $scenario_name,
        "scenario_data": .data,
        "diagnostics": $diagnostics,
        "test_infrastructure": $test_infrastructure,
        "requirements": $requirements,
        "insights": $insights,
        "raw_response": .,
        "metadata": {
            "query_type": "individual_scenario",
            "timestamp": (now | strftime("%Y-%m-%d %H:%M:%S UTC")),
            "diagnostics_included": true,
            "test_validation_included": true
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

    SCENARIO_STATUS_REQUIREMENTS_SUMMARY=""
    SCENARIO_STATUS_EXTRA_RECOMMENDATIONS=""
    SCENARIO_STATUS_EXTRA_DOC_LINKS=""

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

    local insights_json
    insights_json=$(scenario::insights::collect_data "$scenario_name" 2>/dev/null || echo '{"stack":{},"resources":{},"scenario_dependencies":{},"packages":{},"lifecycle":{}}')
    
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

    scenario::insights::display_metadata "$insights_json" "$scenario_name"
    scenario::insights::display_documentation "$insights_json"

    scenario::insights::display_stack "$insights_json"

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
    scenario::insights::display_resources "$insights_json"
    scenario::insights::display_scenario_dependencies "$insights_json"
    scenario::insights::display_workspace_packages "$insights_json"
    scenario::insights::display_lifecycle "$insights_json"
    scenario::insights::display_health_config "$insights_json"

    # Check for production bundle requirement
    local production_bundle_check
    production_bundle_check=$(echo "$insights_json" | jq -r '.production_bundle' 2>/dev/null || echo '{}')
    local needs_conversion
    needs_conversion=$(echo "$production_bundle_check" | jq -r '.needs_conversion // false')

    if [[ "$needs_conversion" == "true" ]]; then
        if [[ -z "${SCENARIO_STATUS_EXTRA_RECOMMENDATIONS:-}" ]]; then
            SCENARIO_STATUS_EXTRA_RECOMMENDATIONS=""
        fi
        SCENARIO_STATUS_EXTRA_RECOMMENDATIONS="${SCENARIO_STATUS_EXTRA_RECOMMENDATIONS}Convert UI from dev server to production bundles for auto-rebuild support"$'\n'
    fi

    # Enhanced Health Checks and Diagnostics using unified data collection
    local diagnostic_data
    diagnostic_data=$(scenario::health::collect_all_diagnostic_data "$scenario_name" "$response" "$status")
    
    # Display the diagnostic information in text format
    scenario::status::display_diagnostic_data "$scenario_name" "$diagnostic_data" "$status"
    
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

# Display diagnostic data in text format (unified with JSON output)
scenario::status::display_diagnostic_data() {
    local scenario_name="$1"
    local diagnostic_data="$2"
    local status="$3"
    
    # For running scenarios, display health checks and enhanced diagnostics
    if [[ "$status" == "running" ]]; then
        scenario::status::display_health_checks "$diagnostic_data"
        scenario::status::display_running_diagnostics "$diagnostic_data" "$scenario_name"
    fi

    scenario::status::display_ui_smoke "$diagnostic_data" "$scenario_name"

    if command -v scenario::requirements::display_summary >/dev/null 2>&1; then
        if [[ -z "${SCENARIO_STATUS_REQUIREMENTS_SUMMARY:-}" ]] && command -v scenario::requirements::quick_check >/dev/null 2>&1; then
            SCENARIO_STATUS_REQUIREMENTS_SUMMARY=$(scenario::requirements::quick_check "$scenario_name")
        fi
        if [[ -n "${SCENARIO_STATUS_REQUIREMENTS_SUMMARY:-}" ]]; then
            echo ""
            scenario::requirements::display_summary "$SCENARIO_STATUS_REQUIREMENTS_SUMMARY" "$scenario_name"
        fi
    fi

    # Display production bundle warning if needed
    if [[ "$needs_conversion" == "true" ]]; then
        echo ""
        echo -e "\033[1;33m[WARNING]\033[0m UI Build: ⚠️  Using dev server instead of production bundles"
        echo "   • Production bundles enable cache-busting and consistent behavior"
        echo "   • Production bundles make behavior more predictable and closer to real-world behavior"
        echo "   • Production bundles make integration testing true to production behavior"

        # Set env var so test validator can add documentation link
        export SCENARIO_STATUS_NEEDS_PRODUCTION_BUNDLE="true"
    fi

    # Always display test infrastructure validation (for all statuses)
    scenario::status::display_test_infrastructure "$scenario_name"

    # Clean up env var
    unset SCENARIO_STATUS_NEEDS_PRODUCTION_BUNDLE

    if [[ -n "${SCENARIO_STATUS_EXTRA_RECOMMENDATIONS:-}" ]]; then
        echo ""
        echo "💡 Recommendations:"
        printf '%s' "$SCENARIO_STATUS_EXTRA_RECOMMENDATIONS" | while IFS= read -r line; do
            [[ -n "$line" ]] && echo "   • $line"
        done
        echo ""
        SCENARIO_STATUS_EXTRA_RECOMMENDATIONS=""
    fi

    # For stopped scenarios, display failure diagnostics after test insights
    if [[ "$status" == "stopped" ]]; then
        scenario::status::display_failure_diagnostics "$diagnostic_data" "$scenario_name"
    fi
}

# Display health check information for running scenarios
scenario::status::display_health_checks() {
    local diagnostic_data="$1"
    
    # Check if we have health check data
    local has_ui_health=$(echo "$diagnostic_data" | jq '.health_checks.ui != null')
    local has_api_health=$(echo "$diagnostic_data" | jq '.health_checks.api != null')
    
    if [[ "$has_ui_health" == "true" ]] || [[ "$has_api_health" == "true" ]]; then
        echo ""
        echo "Health Checks:"
        
        # Display UI health if available
        if [[ "$has_ui_health" == "true" ]]; then
            local ui_available=$(echo "$diagnostic_data" | jq -r '.health_checks.ui.available // false')
            local ui_port=$(echo "$diagnostic_data" | jq -r '.health_checks.ui.port // null')
            local ui_status=$(echo "$diagnostic_data" | jq -r '.health_checks.ui.status // "unknown"')
            
            if [[ "$ui_available" == "true" ]]; then
                echo "  UI Service (port $ui_port):"
                
                local schema_valid=$(echo "$diagnostic_data" | jq -r '.health_checks.ui.schema_valid // false')
                if [[ "$schema_valid" == "true" ]]; then
                    case "$ui_status" in
                        healthy)
                            echo "    Status:      ✅ $ui_status"
                            ;;
                        degraded)
                            echo "    Status:      ⚠️  $ui_status"
                            ;;
                        unhealthy)
                            echo "    Status:      ❌ $ui_status"
                            ;;
                        *)
                            echo "    Status:      ❓ $ui_status"
                            ;;
                    esac
                    
                    # Show API connectivity
                    local api_connected=$(echo "$diagnostic_data" | jq -r '.health_checks.ui.api_connectivity.connected // false')
                    local api_url=$(echo "$diagnostic_data" | jq -r '.health_checks.ui.api_connectivity.api_url // "unknown"')
                    
                    if [[ "$api_connected" == "true" ]]; then
                        echo "    API Link:    ✅ Connected to $api_url"
                        local latency=$(echo "$diagnostic_data" | jq -r '.health_checks.ui.api_connectivity.latency_ms // null')
                        [[ "$latency" != "null" ]] && echo "    Latency:     ${latency}ms"
                    elif [[ "$api_connected" == "false" ]]; then
                        echo "    API Link:    ❌ DISCONNECTED from $api_url"
                        local api_error=$(echo "$diagnostic_data" | jq -r '.health_checks.ui.api_connectivity.error // null')
                        [[ "$api_error" != "null" ]] && echo "    Error:       $api_error"
                    fi
                else
                    echo "    Status:      ⚠️  Invalid health response (not compliant with schema)"
                    echo "    💡 UI health endpoint must include 'api_connectivity' field"
                    echo "       See: ${SCENARIO_CMD_DIR}/schemas/health-ui.schema.json"
                fi
            else
                echo "  UI Service:    ⚠️  Health endpoint not responding"
            fi
        fi
        
        # Display API health if available
        if [[ "$has_api_health" == "true" ]]; then
            local api_available=$(echo "$diagnostic_data" | jq -r '.health_checks.api.available // false')
            local api_port=$(echo "$diagnostic_data" | jq -r '.health_checks.api.port // null')
            local api_status=$(echo "$diagnostic_data" | jq -r '.health_checks.api.status // "unknown"')
            
            if [[ "$api_available" == "true" ]]; then
                if [[ "$has_ui_health" != "true" ]]; then
                    echo "  API Service (port $api_port):"
                else
                    echo "  API Service (port $api_port):"
                fi
                
                local schema_valid=$(echo "$diagnostic_data" | jq -r '.health_checks.api.schema_valid // false')
                if [[ "$schema_valid" == "true" ]]; then
                    case "$api_status" in
                        healthy)
                            echo "    Status:      ✅ $api_status"
                            ;;
                        degraded)
                            echo "    Status:      ⚠️  $api_status"
                            ;;
                        unhealthy)
                            echo "    Status:      ❌ $api_status"
                            ;;
                        *)
                            echo "    Status:      ❓ $api_status"
                            ;;
                    esac
                    
                    # Show dependencies if available
                    local db_connected=$(echo "$diagnostic_data" | jq -r '.health_checks.api.dependencies.database.connected // null')
                    if [[ "$db_connected" != "null" ]]; then
                        if [[ "$db_connected" == "true" ]]; then
                            echo "    Database:    ✅ Connected"
                        else
                            echo "    Database:    ❌ Disconnected"
                        fi
                    fi
                else
                    echo "    Status:      ⚠️  Invalid health response"
                fi
            else
                if [[ "$has_ui_health" != "true" ]]; then
                    echo "  API Service:   ⚠️  Health endpoint not responding"
                else
                    echo "  API Service:   ⚠️  Health endpoint not responding"
                fi
            fi
        fi
    fi
}

# Display running scenario diagnostics
scenario::status::display_running_diagnostics() {
    local diagnostic_data="$1"
    local scenario_name="$2"
    
    # Check if there are any issues to display
    local has_warnings=$(echo "$diagnostic_data" | jq '.log_analysis.recent_warnings | length > 0')
    local has_resource_issues=$(echo "$diagnostic_data" | jq '.log_analysis.resource_issues | length > 0')
    local has_performance_warnings=$(echo "$diagnostic_data" | jq '.log_analysis.performance_warnings | length > 0')
    
    # Check for slow response times
    local api_slow=$(echo "$diagnostic_data" | jq '.responsiveness.api.response_time > 3.0')
    local ui_slow=$(echo "$diagnostic_data" | jq '.responsiveness.ui.response_time > 3.0')
    local api_timeout=$(echo "$diagnostic_data" | jq -r '.responsiveness.api.timeout // false')
    local ui_timeout=$(echo "$diagnostic_data" | jq -r '.responsiveness.ui.timeout // false')
    
    local has_performance_issues=false
    if [[ "$api_slow" == "true" ]] || [[ "$ui_slow" == "true" ]] || [[ "$api_timeout" == "true" ]] || [[ "$ui_timeout" == "true" ]]; then
        has_performance_issues=true
    fi
    
    local analysis_printed=false

    # Display diagnostics if there are any issues
    if [[ "$has_warnings" == "true" ]] || [[ "$has_resource_issues" == "true" ]] || [[ "$has_performance_warnings" == "true" ]] || [[ "$has_performance_issues" == "true" ]]; then
        echo ""
        echo "🔍 Running Scenario Analysis:"
        analysis_printed=true
        
        # Show recent warnings
        if [[ "$has_warnings" == "true" ]]; then
            echo "  Recent Warnings:"
            echo "$diagnostic_data" | jq -r '.log_analysis.recent_warnings[] | "    ⚠️  " + .component + ": " + .message'
        fi
        
        # Show resource issues
        if [[ "$has_resource_issues" == "true" ]]; then
            echo "  Resource Issues:"
            echo "$diagnostic_data" | jq -r '.log_analysis.resource_issues[] | "    🔴 " + .component + ": " + .message'
        fi
        
        # Show performance warnings from logs
        if [[ "$has_performance_warnings" == "true" ]]; then
            echo "  Performance Warnings:"
            echo "$diagnostic_data" | jq -r '.log_analysis.performance_warnings[] | "    🟡 " + .component + ": " + .message'
        fi
        
        # Show responsiveness issues
        if [[ "$has_performance_issues" == "true" ]]; then
            echo "  Response Time Issues:"
            if [[ "$api_timeout" == "true" ]]; then
                echo "    🔴 API health endpoint timing out (>10s)"
            elif [[ "$api_slow" == "true" ]]; then
                local api_time=$(echo "$diagnostic_data" | jq -r '.responsiveness.api.response_time')
                echo "    🟡 API health endpoint slow (${api_time}s response)"
            fi
            
            if [[ "$ui_timeout" == "true" ]]; then
                echo "    🔴 UI health endpoint timing out (>10s)"
            elif [[ "$ui_slow" == "true" ]]; then
                local ui_time=$(echo "$diagnostic_data" | jq -r '.responsiveness.ui.response_time')
                echo "    🟡 UI health endpoint slow (${ui_time}s response)"
            fi
        fi
        
        echo "  💡 Check detailed logs: vrooli scenario logs $scenario_name"
        
        # Show enhanced diagnostics
        echo ""
        echo "💡 Diagnostics:"
        local api_connected=$(echo "$diagnostic_data" | jq -r '.health_checks.ui.api_connectivity.connected // null')
        if [[ "$api_connected" == "false" ]]; then
            echo "  • UI cannot reach API - check API logs and configuration"
            echo "  • Verify API_URL environment variable in UI matches API port"
        fi
        echo "  • Check logs: vrooli scenario logs $scenario_name"
        echo "  • Validate health endpoints comply with schemas in:"
        echo "    ${SCENARIO_CMD_DIR}/schemas/"
    fi

    local api_event ui_event
    api_event=$(echo "$diagnostic_data" | jq -c '.log_analysis.recent_events.api // empty' 2>/dev/null || echo "")
    ui_event=$(echo "$diagnostic_data" | jq -c '.log_analysis.recent_events.ui // empty' 2>/dev/null || echo "")

    if [[ -n "$api_event" ]] || [[ -n "$ui_event" ]]; then
        if [[ "$analysis_printed" != "true" ]]; then
            echo ""
        fi
        echo "Recent Signals:"

        if [[ -n "$api_event" ]]; then
            local api_type api_message api_step api_icon
            api_type=$(echo "$api_event" | jq -r '.type // "info"')
            api_message=$(echo "$api_event" | jq -r '.message // ""')
            api_step=$(echo "$api_event" | jq -r '.step // "start-api"')
            case "$api_type" in
                error)
                    api_icon="🔴"
                    ;;
                warning)
                    api_icon="⚠️"
                    ;;
                *)
                    api_icon="ℹ️"
                    ;;
            esac
            printf '  %s API logs (%s): %s\n' "$api_icon" "$api_step" "$api_message"
            if [[ "$api_type" != "info" ]]; then
                echo "     • Investigate: vrooli scenario logs $scenario_name --step $api_step"
            fi
        fi

        if [[ -n "$ui_event" ]]; then
            local ui_type ui_message ui_step ui_icon
            ui_type=$(echo "$ui_event" | jq -r '.type // "info"')
            ui_message=$(echo "$ui_event" | jq -r '.message // ""')
            ui_step=$(echo "$ui_event" | jq -r '.step // "start-ui"')
            case "$ui_type" in
                error)
                    ui_icon="🔴"
                    ;;
                warning)
                    ui_icon="⚠️"
                    ;;
                *)
                    ui_icon="ℹ️"
                    ;;
            esac
            printf '  %s UI logs (%s): %s\n' "$ui_icon" "$ui_step" "$ui_message"
            if [[ "$ui_type" != "info" ]]; then
                echo "     • Investigate: vrooli scenario logs $scenario_name --step $ui_step"
            fi
        fi

        echo ""
    fi
}

# Display failure diagnostics for stopped scenarios
scenario::status::display_failure_diagnostics() {
    local diagnostic_data="$1"
    local scenario_name="$2"
    
    # Check if we have failure data
    local has_api_failures=$(echo "$diagnostic_data" | jq '.log_analysis.api_failures | length > 0')
    local has_ui_failures=$(echo "$diagnostic_data" | jq '.log_analysis.ui_failures | length > 0')
    local has_general_recs=$(echo "$diagnostic_data" | jq '.log_analysis.general_recommendations | length > 0')
    
    if [[ "$has_api_failures" == "true" ]] || [[ "$has_ui_failures" == "true" ]] || [[ "$has_general_recs" == "true" ]]; then
        echo ""
        
        # Display API failures
        if [[ "$has_api_failures" == "true" ]]; then
            echo "$diagnostic_data" | jq -r '.log_analysis.api_failures[] | 
                "🔍 API Startup Failure:\n   " + .message + "\n   💡 " + .recommendation + "\n"'
        fi
        
        # Display UI failures
        if [[ "$has_ui_failures" == "true" ]]; then
            echo "$diagnostic_data" | jq -r '.log_analysis.ui_failures[] | 
                "🔍 UI Startup Failure:\n   " + .message + "\n   💡 " + .recommendation + "\n"'
        fi
        
        # Display general recommendations if no specific failures found
        if [[ "$has_general_recs" == "true" ]] && [[ "$has_api_failures" == "false" ]] && [[ "$has_ui_failures" == "false" ]]; then
            echo "💡 Troubleshooting tips:"
            echo "$diagnostic_data" | jq -r '.log_analysis.general_recommendations[] | "   • " + .'
        fi
    fi
}

# Display test infrastructure validation
scenario::status::display_test_infrastructure() {
    local scenario_name="$1"
    local scenario_path="${APP_ROOT}/scenarios/${scenario_name}"
    
    # Skip if scenario directory doesn't exist
    if [[ ! -d "$scenario_path" ]]; then
        return 0
    fi
    
    # Only validate if test validator is available
    if command -v scenario::test::validate_infrastructure >/dev/null 2>&1; then
        local validation_result
        validation_result=$(scenario::test::validate_infrastructure "$scenario_name" "$scenario_path")
        
        if [[ -n "$validation_result" ]]; then
            scenario::test::display_validation "$scenario_name" "$validation_result"
        fi
    fi
}

scenario::status::display_ui_smoke() {
    local diagnostic_data="$1"
    local scenario_name="$2"

    if ! command -v jq >/dev/null 2>&1; then
        return
    fi

    local smoke_json
    smoke_json=$(echo "$diagnostic_data" | jq -c '.ui_smoke // empty' 2>/dev/null || echo '')
    if [[ -z "$smoke_json" || "$smoke_json" == "null" ]]; then
        local scenario_dir="${APP_ROOT}/scenarios/${scenario_name}"
        if [[ -d "$scenario_dir/ui" ]]; then
            echo ""
            echo "UI Smoke: not yet run"
            echo "  ↳ Run: vrooli scenario ui-smoke ${scenario_name}"
        fi
        return
    fi

    local status message timestamp duration handshake handshake_duration handshake_error bundle_fresh bundle_reason screenshot_path handshake_present storage_summary storage_patched
    status=$(echo "$smoke_json" | jq -r '.status // "unknown"')
    message=$(echo "$smoke_json" | jq -r '.message // ""')
    timestamp=$(echo "$smoke_json" | jq -r '.timestamp // ""')
    duration=$(echo "$smoke_json" | jq -r '.duration_ms // 0')
    handshake=$(echo "$smoke_json" | jq -r '.raw.handshake.signaled // false')
    handshake_duration=$(echo "$smoke_json" | jq -r '.raw.handshake.durationMs // 0')
    handshake_error=$(echo "$smoke_json" | jq -r '.raw.handshake.error // ""')
    handshake_present=$(echo "$smoke_json" | jq -r 'if (.raw? and .raw.handshake?) then "true" else "false" end' 2>/dev/null || echo "false")
    bundle_fresh=$(echo "$smoke_json" | jq -r '.bundle.fresh // true')
    bundle_reason=$(echo "$smoke_json" | jq -r '.bundle.reason // ""')
    screenshot_path=$(echo "$smoke_json" | jq -r '.artifacts.screenshot // ""')
    local network_errors
    network_errors=$(echo "$smoke_json" | jq -r '(.raw.network // []) | length')
    local page_errors
    page_errors=$(echo "$smoke_json" | jq -r '(.raw.pageErrors // []) | length')
    storage_summary=$(echo "$smoke_json" | jq -r '.storage_shim // [] | @json' 2>/dev/null || echo '[]')
    storage_patched=$(echo "$smoke_json" | jq -r '(.storage_shim // []) | map(select(.patched == true)) | length' 2>/dev/null || echo '0')

    # Detect if smoke test results are stale (older than UI bundle)
    local smoke_is_stale=false
    local scenario_dir="${APP_ROOT}/scenarios/${scenario_name}"
    if [[ -d "$scenario_dir/ui" && -f "$scenario_dir/ui/dist/index.html" && -n "$timestamp" ]]; then
        local bundle_mtime
        bundle_mtime=$(stat -c %Y "$scenario_dir/ui/dist/index.html" 2>/dev/null || echo "0")
        if [[ "$bundle_mtime" != "0" ]]; then
            local smoke_timestamp_epoch
            smoke_timestamp_epoch=$(date -d "$timestamp" +%s 2>/dev/null || echo "0")
            if [[ "$smoke_timestamp_epoch" != "0" && "$smoke_timestamp_epoch" -lt "$bundle_mtime" ]]; then
                smoke_is_stale=true
            fi
        fi
    fi

    echo ""
    echo "UI Smoke: $status (${duration}ms${timestamp:+, $timestamp})"
    # Skip displaying cached message if we detect staleness (it's outdated info)
    if [[ -n "$message" && "$message" != "null" && "$smoke_is_stale" != "true" ]]; then
        # Format multiline messages with proper indentation
        local formatted_message=$(echo "$message" | sed '2,$s/^/  /')
        echo "  ↳ $formatted_message"
    fi
    if [[ "$handshake_present" = "true" ]]; then
        if [[ "$handshake" = "true" ]]; then
            echo "  ↳ Handshake: ✅ ${handshake_duration}ms"
        else
            local detail="${handshake_error:-Bridge never signaled ready}"
            echo "  ↳ Handshake: ❌ $detail"
        fi
    elif [[ "$status" = "skipped" ]]; then
        echo "  ↳ Handshake: (skipped)"
    fi
    if [[ "$bundle_fresh" != "true" ]]; then
        echo "  ↳ Bundle Status: ⚠️  ${bundle_reason:-UI bundle stale}"
        echo "  ↳ Fix: vrooli scenario restart ${scenario_name}"
    fi
    if [[ "$network_errors" -gt 0 ]]; then
        local network_path
        network_path=$(echo "$smoke_json" | jq -r '.artifacts.network // ""')
        echo "  ↳ Network errors: ${network_errors}${network_path:+ (see $network_path)}"
    fi
    if [[ "$page_errors" -gt 0 ]]; then
        echo "  ↳ UI exceptions: ${page_errors} recorded"
    fi
    if [[ -n "$screenshot_path" && "$screenshot_path" != "null" ]]; then
        echo "  ↳ Screenshot: $screenshot_path"
    fi
    if [[ "$storage_patched" != "0" ]]; then
        local storage_props
        storage_props=$(echo "$smoke_json" | jq -r '(.storage_shim // []) | map(select(.patched == true) | .prop) | join(", ")' 2>/dev/null || echo "localStorage")
        echo "  ↳ Storage shim active for: ${storage_props}"
    fi

    # Show staleness warning if test results are older than bundle
    if [[ "$smoke_is_stale" = true ]]; then
        echo "  ↳ ⚠️  Smoke test results outdated (bundle rebuilt since last test)"
        echo "  ↳ Rerun: vrooli scenario ui-smoke ${scenario_name}"
    fi
}
