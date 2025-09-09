#!/usr/bin/env bash
################################################################################
# Vrooli CLI - System Status Command
# 
# Provides a comprehensive health check and status overview of the Vrooli system.
# Displays resources, scenarios, and system diagnostics in a concise format.
#
# Usage:
#   vrooli status [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
CLI_DIR="${APP_ROOT}/cli/commands"
VROOLI_ROOT="$APP_ROOT"

# Source utilities
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${APP_ROOT}/scripts/lib/utils/log.sh}"
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/cli/lib/arg-parser.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/cli/lib/output-formatter.sh"

# Configuration paths
RESOURCES_CONFIG="${var_ROOT_DIR}/.vrooli/service.json"
API_BASE="http://localhost:${VROOLI_API_PORT:-8092}"

# Show help for status command
show_status_help() {
    cat << EOF
🚀 Vrooli Status Command

USAGE:
    vrooli status [options]

OPTIONS:
    --json              Output in JSON format (alias for --format json)
    --format <type>     Output format: text, json
    --verbose, -v       Show detailed information
    --fast              Use fast mode for resource checks (default)
    --no-fast           Use detailed mode for resource checks (slower but more accurate)
    --resources         Show only resource status
    --scenarios        Show only scenario status
    --help, -h          Show this help message

EXAMPLES:
    vrooli status                  # Show full system status (fast mode)
    vrooli status --json           # Output as JSON
    vrooli status --resources      # Show only resource status
    vrooli status --scenarios      # Show only scenario status
    vrooli status --verbose        # Show detailed status
    vrooli status --no-fast        # Use detailed health checks (slower)

EOF
}

# Check if API is available
check_api() {
	if ! curl -s -f --connect-timeout 2 --max-time 5 "${API_BASE}/health" >/dev/null 2>&1; then
		return 1
	fi
	return 0
}


# Get resource status using its CLI
get_resource_status() {
    local resource_name="$1"
    local use_fast="${2:-false}"
    local cli_command="resource-${resource_name}"
    
    # Check if CLI exists
    if ! command -v "$cli_command" >/dev/null 2>&1; then
        echo "no_cli"
        return 1
    fi
    
    # Get status from resource CLI (pass --fast flag if requested)
    local status_output
    local fast_flag=""
    if [[ "$use_fast" == "true" ]]; then
        fast_flag="--fast"
    fi
    # Get status output regardless of exit code, as resources may return non-zero for stopped services
    # Resources may output JSON to stderr, so capture both stdout and stderr
    status_output=$("$cli_command" status --format json $fast_flag 2>&1)
    # Check if we have valid JSON output (regardless of exit code)
    if [[ -n "$status_output" ]] && echo "$status_output" | jq -e '.' >/dev/null 2>&1; then
        # Parse JSON to get running and healthy status
        # Some resources use 'healthy', others use 'health' 
        local running healthy health
        running=$(echo "$status_output" | jq -r '.running // false' 2>/dev/null)
        healthy=$(echo "$status_output" | jq -r '.healthy // false' 2>/dev/null)
        health=$(echo "$status_output" | jq -r '.health // ""' 2>/dev/null)
        
        # Check both 'healthy' field and 'health' field
        if [[ "$running" == "true" ]] && ([[ "$healthy" == "true" ]] || [[ "$health" == "healthy" ]]); then
            echo "healthy"
        elif [[ "$running" == "true" ]]; then
            echo "running"
        else
            echo "stopped"
        fi
        return 0
    else
        echo "error"
        return 1
    fi
}

# Helper function to check a single resource status with timing
check_resource_with_timing() {
    local name="$1"
    local result_file="$2"
    local use_fast="${3:-true}"  # Default to fast mode for backward compatibility
    
    local start_time end_time duration_ms
    start_time=$(date +%s%3N)  # milliseconds
    
    # Get resource status with configurable fast flag
    local resource_status
    resource_status=$(get_resource_status "${name}" "$use_fast")
    
    end_time=$(date +%s%3N)
    duration_ms=$((end_time - start_time))
    
    # Write result to file
    echo "${name}:${resource_status}:${duration_ms}" >> "$result_file"
}

# Collect resource data (format-agnostic) - PARALLEL VERSION
collect_resource_data() {
    local verbose="${1:-false}"
    local use_fast="${2:-true}"  # Default to fast mode for performance
    
    # Check if config file exists
    if [[ ! -f "$RESOURCES_CONFIG" ]]; then
        echo "error:No resource configuration found at $RESOURCES_CONFIG"
        return
    fi
    
    # Create temporary directory for parallel execution results
    local temp_dir
    temp_dir=$(mktemp -d)
    local result_file="${temp_dir}/results"
    
    # Cleanup function
    cleanup_temp() {
        if [[ -n "${temp_dir:-}" ]]; then
            rm -rf "$temp_dir" 2>/dev/null
        fi
    }
    trap cleanup_temp EXIT
    
    # Parse resources from config and launch parallel checks
    local jq_query='
        .resources | to_entries[] | 
        select(.value.enabled == true) | 
        "\(.key)"
    '
    
    local -a pids=()
    local enabled_count=0
    
    # Launch all status checks in parallel
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local name="$line"
            ((enabled_count++))
            
            # Launch background process
            check_resource_with_timing "$name" "$result_file" "$use_fast" &
            pids+=($!)
        fi
    done < <(jq -r "$jq_query" "$RESOURCES_CONFIG" 2>/dev/null | sort || true)
    
    # Wait for all background processes to complete
    for pid in "${pids[@]}"; do
        wait "$pid" 2>/dev/null || true
    done
    
    # Process results
    local running_count=0
    local healthy_count=0
    local -a timing_data=()
    
    if [[ -f "$result_file" ]]; then
        # Sort results by resource name for consistent ordering
        while IFS=: read -r name status duration_ms; do
            case "$status" in
                "healthy")
                    ((running_count++))
                    ((healthy_count++))
                    ;;
                "running")
                    ((running_count++))
                    ;;
                "stopped"|"error"|"no_cli")
                    # Not running
                    ;;
            esac
            
            # Store timing data for later analysis
            timing_data+=("${duration_ms}:${name}:${status}")
            
            if [[ "$verbose" == "true" ]]; then
                echo "resource:${name}:${status}"
            fi
        done < <(sort "$result_file")
    fi
    
    # Display timing information for verbose mode
    if [[ "$verbose" == "true" && ${#timing_data[@]} -gt 0 ]]; then
        # Sort by duration (descending) and show top 5
        local -a sorted_timings
        mapfile -t sorted_timings < <(printf '%s\n' "${timing_data[@]}" | sort -nr)
        
        echo "timing:header"
        local count=0
        for timing in "${sorted_timings[@]}"; do
            if [[ $count -ge 5 ]]; then
                break
            fi
            IFS=: read -r duration_ms name status <<< "$timing"
            echo "timing:${name}:${duration_ms}ms:${status}"
            ((count++))
        done
    fi
    
    # Always output summary
    echo "enabled:${enabled_count}"
    echo "running:${running_count}"
    echo "healthy:${healthy_count}"
}

# Get structured resource data (format-agnostic)
get_resource_data() {
    local verbose="${1:-false}"
    local use_fast="${2:-true}"  # Default to fast mode
    
    local raw_data
    raw_data=$(collect_resource_data "$verbose" "$use_fast")
    
    # Check for errors
    if echo "$raw_data" | grep -q "^error:"; then
        local error_msg
        error_msg=$(echo "$raw_data" | grep "^error:" | cut -d: -f2-)
        echo "type:error"
        echo "message:$error_msg"
        return
    fi
    
    # Parse collected data
    local enabled_count running_count healthy_count
    enabled_count=$(echo "$raw_data" | grep "^enabled:" | cut -d: -f2)
    running_count=$(echo "$raw_data" | grep "^running:" | cut -d: -f2)
    healthy_count=$(echo "$raw_data" | grep "^healthy:" | cut -d: -f2)
    
    echo "type:status"
    echo "enabled:${enabled_count}"
    echo "running:${running_count}"
    echo "healthy:${healthy_count}"
    echo "summary:${healthy_count}/${enabled_count} healthy (${running_count} running)"
    
    if [[ "$verbose" == "true" ]]; then
        echo "details:true"
        while IFS= read -r line; do
            if [[ "$line" =~ ^resource:(.+):(.+)$ ]]; then
                echo "item:${BASH_REMATCH[1]}:${BASH_REMATCH[2]}"
            elif [[ "$line" =~ ^timing:(.*)$ ]]; then
                # Pass through timing information
                echo "$line"
            fi
        done <<< "$raw_data"
    else
        echo "details:false"
    fi
}


# Collect app data (format-agnostic) - Using API with PID fallback
collect_scenario_data() {
    local verbose="${1:-false}"
    
    local total_scenarios=0
    local running_scenarios=0
    local -A scenario_statuses  # For detailed output
    
    # Try to use API for health-aware status (like scenario-commands.sh does)
    local api_port="${VROOLI_API_PORT:-8092}"
    local api_url="http://localhost:${api_port}"
    local api_available=false
    
    # Check if API is available (quick check)
    if curl -s --connect-timeout 2 --max-time 3 "${api_url}/health" >/dev/null 2>&1; then
        api_available=true
        
        # Get scenarios from API
        local response
        response=$(curl -s --connect-timeout 3 --max-time 5 "${api_url}/scenarios" 2>/dev/null)
        
        if [[ -n "$response" ]] && echo "$response" | jq -e '.success' >/dev/null 2>&1; then
            # Parse API response for health-aware status
            local data_check=$(echo "$response" | jq -r '.data' 2>/dev/null)
            
            if [[ "$data_check" != "null" ]] && [[ "$data_check" != "" ]]; then
                total_scenarios=$(echo "$response" | jq -r '.data | length' 2>/dev/null)
                
                # Parse each scenario with health status
                while IFS='|' read -r name status health_status; do
                    [[ -z "$name" || "$name" == "null" ]] && continue
                    
                    local final_status="stopped"
                    if [[ "$status" == "running" ]]; then
                        running_scenarios=$((running_scenarios + 1))
                        # Use health status if available, otherwise default to "running"
                        case "$health_status" in
                            "healthy")
                                final_status="healthy"
                                ;;
                            "degraded"|"unhealthy") 
                                final_status="$health_status"
                                ;;
                            *)
                                final_status="running"
                                ;;
                        esac
                    else
                        final_status="stopped"
                    fi
                    
                    scenario_statuses["$name"]="$final_status"
                done < <(echo "$response" | jq -r '.data[]? | "\(.name)|\(.status)|\(.health_status // "unknown")"' 2>/dev/null)
                
            else
                # API returned success but no data (no running scenarios)
                total_scenarios=0
                running_scenarios=0
            fi
        else
            # API call failed, fall back to directory-based detection
            api_available=false
        fi
    fi
    
    # Fallback: Use directory-based detection if API unavailable
    if [[ "$api_available" == "false" ]]; then
        local scenarios_dir="${VROOLI_ROOT:-$HOME/Vrooli}/scenarios"
        
        if [[ ! -d "$scenarios_dir" ]]; then
            echo "error:Scenarios directory not found and API unavailable"
            return
        fi
        
        # Count scenarios and check which are running using PID detection
        for scenario_path in "$scenarios_dir"/*; do
            [[ ! -d "$scenario_path" ]] && continue
            
            local name="$(basename "$scenario_path")"
            [[ "$name" == "*" ]] && continue  # No scenarios found
            
            total_scenarios=$((total_scenarios + 1))
            
            # Check if scenario is running using PID-based detection
            local is_running=false
            
            # Check for PID-tracked processes in the new system
            local scenario_dir="$HOME/.vrooli/processes/scenarios/$name"
            if [[ -d "$scenario_dir" ]]; then
                for step_file in "$scenario_dir"/*.json; do
                    [[ -f "$step_file" ]] || continue
                    
                    local pid=$(jq -r '.pid' "$step_file" 2>/dev/null)
                    
                    # Check if process is actually running
                    if [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 "$pid" 2>/dev/null; then
                        is_running=true
                        break  # Found at least one running process
                    else
                        # Cleanup dead process metadata
                        local step_name=$(jq -r '.step' "$step_file" 2>/dev/null)
                        rm -f "$step_file" "$scenario_dir/${step_name}.pid" 2>/dev/null
                    fi
                done
            fi
            
            # Fallback: Check for old pm2.pid files (backward compatibility)
            if [[ "$is_running" == "false" && -f "$HOME/.vrooli/processes/scenarios/$name/pm2.pid" ]]; then
                local pid=$(cat "$HOME/.vrooli/processes/scenarios/$name/pm2.pid" 2>/dev/null)
                if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                    is_running=true
                fi
            fi
            
            if [[ "$is_running" == "true" ]]; then
                running_scenarios=$((running_scenarios + 1))
                scenario_statuses["$name"]="running"  # Can't determine health without API
            else
                scenario_statuses["$name"]="stopped"
            fi
        done
    fi
    
    # Output summary
    echo "total:${total_scenarios}"
    echo "running:${running_scenarios}"
    
    # Output details if verbose (sorted alphabetically)
    if [[ "$verbose" == "true" && "$total_scenarios" -gt 0 ]]; then
        # Sort scenario names alphabetically
        for name in $(printf '%s\n' "${!scenario_statuses[@]}" | sort); do
            echo "scenario:${name}:${scenario_statuses[$name]}"
        done
    fi
}

# Legacy collect_scenario_data for fallback
collect_scenario_data_legacy() {
    local verbose="${1:-false}"
    
    # Get scenario list from orchestrator if available
    local response
    response=$(curl -s --connect-timeout 2 --max-time 5 "http://localhost:${ORCHESTRATOR_PORT:-9500}/apps" 2>/dev/null || echo '{"success": false}')
    
    if ! echo "$response" | jq -e '.apps' >/dev/null 2>&1; then
        echo "error:Failed to get scenario list from orchestrator"
        return
    fi
    
    # Check if orchestrator is available for app status
    local orchestrator_available=false
    if curl -s --connect-timeout 3 "http://localhost:${ORCHESTRATOR_PORT:-9500}/health" >/dev/null 2>&1; then
        orchestrator_available=true
    fi
    
    local total_scenarios=0
    local running_scenarios=0
    local -A scenario_statuses  # For detailed output
    
    # Process each app and check runtime status via process manager
    while IFS= read -r app_json; do
        # Skip empty lines
        [[ -z "$app_json" ]] && continue
        
        # Extract app data
        local name
        name=$(echo "$app_json" | jq -r '.name' 2>/dev/null)
        [[ -z "$name" || "$name" == "null" ]] && continue
        
        total_scenarios=$((total_scenarios + 1))
        
        # Check if app is running via orchestrator API
        local is_running=false
        
        if [[ "$orchestrator_available" == "true" ]]; then
            # Get app status from orchestrator
            local orchestrator_response
            orchestrator_response=$(curl -s --connect-timeout 2 "http://localhost:${ORCHESTRATOR_PORT:-9500}/apps" 2>/dev/null)
            
            if echo "$orchestrator_response" | jq -e '.apps' >/dev/null 2>&1; then
                # Check if this specific scenario is running in orchestrator
                local scenario_status
                scenario_status=$(echo "$orchestrator_response" | jq -r --arg name "$name" '.apps[] | select(.name == $name) | .status' 2>/dev/null)
                if [[ "$scenario_status" == "running" ]]; then
                    is_running=true
                fi
            fi
        fi
        
        # Update counts and store status
        if [[ "$is_running" == "true" ]]; then
            running_scenarios=$((running_scenarios + 1))
            scenario_statuses["$name"]="running"
        else
            scenario_statuses["$name"]="stopped"
        fi
        
    done < <(echo "$response" | jq -c '.data[]' 2>/dev/null)
    
    # Output summary
    echo "total:${total_scenarios}"
    echo "running:${running_scenarios}"
    
    # Output details if verbose (sorted alphabetically)
    if [[ "$verbose" == "true" && "$total_scenarios" -gt 0 ]]; then
        # Sort scenario names alphabetically
        for name in $(printf '%s\n' "${!scenario_statuses[@]}" | sort); do
            echo "scenario:${name}:${scenario_statuses[$name]}"
        done
    fi
}

# Get structured scenario data (format-agnostic)
get_scenario_data() {
    local verbose="${1:-false}"
    
    local raw_data
    raw_data=$(collect_scenario_data "$verbose")
    
    # Check for errors
    if echo "$raw_data" | grep -q "^error:"; then
        local error_msg
        error_msg=$(echo "$raw_data" | grep "^error:" | cut -d: -f2-)
        echo "type:error"
        echo "message:$error_msg"
        return
    fi
    
    # Parse collected data
    local total_scenarios running_scenarios
    total_scenarios=$(echo "$raw_data" | grep "^total:" | cut -d: -f2)
    running_scenarios=$(echo "$raw_data" | grep "^running:" | cut -d: -f2)
    
    echo "type:status"
    echo "total:${total_scenarios}"
    echo "running:${running_scenarios}"
    echo "summary:${running_scenarios}/${total_scenarios} running"
    
    if [[ "$verbose" == "true" && "$total_scenarios" -gt 0 ]]; then
        echo "details:true"
        while IFS= read -r line; do
            if [[ "$line" =~ ^scenario:(.+):(.+)$ ]]; then
                echo "item:${BASH_REMATCH[1]}:${BASH_REMATCH[2]}"
            fi
        done <<< "$raw_data"
    else
        echo "details:false"
    fi
}


# Collect system data (format-agnostic)
collect_system_data() {
    # Get memory usage
    local mem_info
    mem_info=$(free -m | awk 'NR==2{printf "%.1f", $3*100/$2}')
    
    # Get disk usage for Vrooli directory
    local disk_info
    disk_info=$(df -h "${var_ROOT_DIR}" | awk 'NR==2{print $5}' | sed 's/%//')
    
    # Get load average
    local load_avg
    load_avg=$(uptime | awk -F'load average:' '{print $2}' | xargs)
    
    # Check Docker status
    local docker_status="healthy"
    if ! docker info >/dev/null 2>&1; then
        docker_status="unavailable"
    fi
    
    # Count zombie processes (ALL defunct processes)
    local zombie_count=0
    zombie_count=$(ps aux | grep '<defunct>' | grep -v grep | wc -l 2>/dev/null || echo "0")
    
    # Count orphaned Vrooli processes 
    local orphan_count=0
    orphan_count=$(bash -c '
        processes_dir="$HOME/.vrooli/processes/scenarios"
        orphan_count=0
        
        # Get Vrooli-related processes into a temporary file to avoid subshell issues
        temp_file=$(mktemp)
        ps aux | grep -E "(vrooli|/scenarios/.*/(api|ui)|node_modules/.bin/vite|ecosystem-manager|picker-wheel)" | grep -v grep | grep -v "bash -c" > "$temp_file"
        
        while IFS= read -r line; do
            if [[ -z "$line" ]]; then continue; fi
            
            # Extract PID (second field)
            pid=$(echo "$line" | awk "{print \$2}")
            if [[ ! "$pid" =~ ^[0-9]+$ ]]; then continue; fi
            
            # Skip own processes
            if echo "$line" | grep -q "vrooli-api\|status-command"; then continue; fi
            
            # Check if PID is tracked
            is_tracked=false
            if [[ -d "$processes_dir" ]]; then
                for json_file in "$processes_dir"/*/*.json; do
                    if [[ -f "$json_file" ]]; then
                        tracked_pid=$(jq -r ".pid // empty" "$json_file" 2>/dev/null)
                        if [[ "$tracked_pid" == "$pid" ]]; then
                            is_tracked=true
                            break
                        fi
                    fi
                done
            fi
            
            # If not tracked, its an orphan
            if [[ "$is_tracked" == "false" ]]; then
                ((orphan_count++))
            fi
        done < "$temp_file"
        
        rm -f "$temp_file"
        echo $orphan_count
    ' 2>/dev/null || echo "0")
    
    # Output raw data
    echo "memory:${mem_info}"
    echo "disk:${disk_info}"
    echo "load:${load_avg}"
    echo "docker:${docker_status}"
    echo "zombies:${zombie_count}"
    echo "orphans:${orphan_count}"
}

get_system_data() {
    local raw_data
    raw_data=$(collect_system_data)
    
    # Parse collected data
    local mem_info disk_info load_avg docker_status zombie_count orphan_count
    mem_info=$(echo "$raw_data" | grep "^memory:" | cut -d: -f2)
    disk_info=$(echo "$raw_data" | grep "^disk:" | cut -d: -f2)
    load_avg=$(echo "$raw_data" | grep "^load:" | cut -d: -f2-)
    docker_status=$(echo "$raw_data" | grep "^docker:" | cut -d: -f2)
    zombie_count=$(echo "$raw_data" | grep "^zombies:" | cut -d: -f2)
    orphan_count=$(echo "$raw_data" | grep "^orphans:" | cut -d: -f2)
    
    echo "type:system"
    echo "memory:${mem_info}"
    echo "disk:${disk_info}"
    echo "load:${load_avg}"
    echo "docker:${docker_status}"
    echo "zombies:${zombie_count}"
    echo "orphans:${orphan_count}"
}

################################################################################
# Status-Specific Formatters (using generic format.sh functions)
################################################################################

# Format system data for display
format_system_data() {
    local format="$1"
    local raw_data="$2"
    
    local memory disk load docker zombies orphans
    memory=$(echo "$raw_data" | grep "^memory:" | cut -d: -f2)
    disk=$(echo "$raw_data" | grep "^disk:" | cut -d: -f2)
    load=$(echo "$raw_data" | grep "^load:" | cut -d: -f2-)
    docker=$(echo "$raw_data" | grep "^docker:" | cut -d: -f2)
    zombies=$(echo "$raw_data" | grep "^zombies:" | cut -d: -f2)
    orphans=$(echo "$raw_data" | grep "^orphans:" | cut -d: -f2)
    
    if [[ "$format" == "json" ]]; then
        format::key_value json \
            memory_usage "${memory}%" \
            disk_usage "${disk}%" \
            load_average "${load}" \
            docker "${docker}" \
            zombie_processes "${zombies}" \
            orphan_processes "${orphans}"
    else
        local docker_icon="✅"
        [[ "$docker" == "unavailable" ]] && docker_icon="❌"
        
        # Determine zombie status icon
        local zombie_icon="✅"
        local zombie_val="${zombies:-0}"
        if [[ $zombie_val -gt 20 ]]; then
            zombie_icon="🔴"
        elif [[ $zombie_val -gt 5 ]]; then
            zombie_icon="⚠️"
        fi
        
        # Determine orphan status icon
        local orphan_icon="✅"
        local orphan_val="${orphans:-0}"
        if [[ $orphan_val -gt 10 ]]; then
            orphan_icon="🔴"
        elif [[ $orphan_val -gt 3 ]]; then
            orphan_icon="⚠️"
        fi
        
        local rows=()
        rows+=("Memory:${memory}% used:$([ -n "${memory}" ] && [ ${memory%.*} -gt 80 ] && echo '⚠️' || echo '✅')")
        rows+=("Disk:${disk}% used:$([ -n "${disk}" ] && [ ${disk%.*} -gt 80 ] && echo '⚠️' || echo '✅')")
        rows+=("Load:${load}:📊")
        rows+=("Zombies:${zombies}:${zombie_icon}")
        rows+=("Orphans:${orphans}:${orphan_icon}")
        rows+=("Docker:${docker}:${docker_icon}")
        
        format::table "$format" "Metric" "Value" "Status" -- "${rows[@]}"
    fi
}

# Format resource/app data for display
format_component_data() {
    local format="$1"
    local component="$2"  # resources or apps
    local raw_data="$3"
    
    # Check for errors first
    if echo "$raw_data" | grep -q "^type:error"; then
        local error_msg
        error_msg=$(echo "$raw_data" | grep "^message:" | cut -d: -f2-)
        if [[ "$format" == "json" ]]; then
            format::key_value json error "$error_msg"
        else
            echo "${component^}: $error_msg"
        fi
        return
    fi
    
    local summary
    summary=$(echo "$raw_data" | grep "^summary:" | cut -d: -f2-)
    
    if [[ "$format" == "json" ]]; then
        # Build JSON based on component type
        if [[ "$component" == "resources" ]]; then
            local enabled running healthy
            enabled=$(echo "$raw_data" | grep "^enabled:" | cut -d: -f2)
            running=$(echo "$raw_data" | grep "^running:" | cut -d: -f2)
            healthy=$(echo "$raw_data" | grep "^healthy:" | cut -d: -f2)
            
            if echo "$raw_data" | grep -q "^details:true"; then
                local details_json="["
                local first=true
                while IFS= read -r line; do
                    if [[ "$line" =~ ^item:(.+):(.+)$ ]]; then
                        [[ "$first" == "true" ]] && first=false || details_json="${details_json},"
                        details_json="${details_json}{\"name\":\"${BASH_REMATCH[1]}\",\"status\":\"${BASH_REMATCH[2]}\"}"
                    fi
                done <<< "$raw_data"
                details_json="${details_json}]"
                
                # Add timing information if present
                local timing_json=""
                if echo "$raw_data" | grep -q "^timing:header"; then
                    timing_json="["
                    local timing_first=true
                    while IFS= read -r line; do
                        if [[ "$line" =~ ^timing:(.+):(.+):(.+)$ ]]; then
                            [[ "$timing_first" == "true" ]] && timing_first=false || timing_json="${timing_json},"
                            local name="${BASH_REMATCH[1]}"
                            local duration="${BASH_REMATCH[2]}"
                            local status="${BASH_REMATCH[3]}"
                            # Remove 'ms' suffix from duration for JSON (store as number)
                            local duration_num="${duration%ms}"
                            timing_json="${timing_json}{\"name\":\"$name\",\"duration_ms\":$duration_num,\"status\":\"$status\"}"
                        fi
                    done <<< "$raw_data"
                    timing_json="${timing_json}]"
                    format::json_object "enabled" "$enabled" "running" "$running" "healthy" "$healthy" "details" "$details_json" "timing" "$timing_json"
                else
                    format::json_object "enabled" "$enabled" "running" "$running" "healthy" "$healthy" "details" "$details_json"
                fi
            else
                format::key_value json enabled "$enabled" running "$running" healthy "$healthy"
            fi
        else  # scenarios
            local total running
            total=$(echo "$raw_data" | grep "^total:" | cut -d: -f2)
            running=$(echo "$raw_data" | grep "^running:" | cut -d: -f2)
            
            if echo "$raw_data" | grep -q "^details:true"; then
                local details_json="["
                local first=true
                while IFS= read -r line; do
                    if [[ "$line" =~ ^item:(.+):(.+)$ ]]; then
                        [[ "$first" == "true" ]] && first=false || details_json="${details_json},"
                        details_json="${details_json}{\"name\":\"${BASH_REMATCH[1]}\",\"status\":\"${BASH_REMATCH[2]}\"}"
                    fi
                done <<< "$raw_data"
                details_json="${details_json}]"
                format::json_object "total" "$total" "running" "$running" "details" "$details_json"
            else
                format::key_value json total "$total" running "$running"
            fi
        fi
    else
        # Text format
        echo "${component^}: $summary"
        
        if echo "$raw_data" | grep -q "^details:true"; then
            echo ""
            local table_rows=()
            while IFS= read -r line; do
                if [[ "$line" =~ ^item:(.+):(.+)$ ]]; then
                    local name="${BASH_REMATCH[1]}"
                    local status="${BASH_REMATCH[2]}"
                    local status_icon="🔴"
                    case "$status" in
                        "healthy") status_icon="🟢" ;;
                        "degraded") status_icon="🟡" ;;
                        "unhealthy") status_icon="🔴" ;;
                        "running") status_icon="🟡" ;;
                        "stopped") status_icon="🔴" ;;
                        "error") status_icon="❌" ;;
                        "no_cli") status_icon="⚠️" ;;
                    esac
                    table_rows+=("${name}:${status}:${status_icon}")
                fi
            done <<< "$raw_data"
            
            if [[ ${#table_rows[@]} -gt 0 ]]; then
                local header_name="Resource"
                [[ "$component" == "scenarios" ]] && header_name="Scenario"
                format::table "$format" "$header_name" "Status" "" -- "${table_rows[@]}"
            fi
            
            # Display timing information if present
            if echo "$raw_data" | grep -q "^timing:header"; then
                echo ""
                echo "⏱️  Slowest Response Times:"
                local timing_rows=()
                while IFS= read -r line; do
                    if [[ "$line" =~ ^timing:(.+):(.+):(.+)$ ]]; then
                        local name="${BASH_REMATCH[1]}"
                        local duration="${BASH_REMATCH[2]}"
                        local status="${BASH_REMATCH[3]}"
                        timing_rows+=("${name}:${duration}:${status}")
                    fi
                done <<< "$raw_data"
                
                if [[ ${#timing_rows[@]} -gt 0 ]]; then
                    format::table "$format" "Resource" "Duration" "Status" -- "${timing_rows[@]}"
                fi
            fi
        else
            # When not in verbose mode, show hint about --verbose flag
            echo "(Use --verbose to see individual ${component})"
        fi
    fi
}


# Main status display
show_status() {
    local show_resources="true"
    local show_scenarios="true"
    local show_system="true"
    
    # Parse common arguments
    local parsed_args
    parsed_args=$(parse_combined_args "$@")
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    local verbose
    verbose=$(extract_arg "$parsed_args" "verbose")
    local help_requested
    help_requested=$(extract_arg "$parsed_args" "help")
    local output_format
    output_format=$(extract_arg "$parsed_args" "format")
    local remaining_args
    remaining_args=$(extract_arg "$parsed_args" "remaining")
    
    # Handle help request
    if [[ "$help_requested" == "true" ]]; then
        show_status_help
        return 0
    fi
    
    # Parse remaining arguments for status-specific options
    local args_array
    mapfile -t args_array < <(args_to_array "$remaining_args")
    
    local use_fast="true"  # Default to fast mode for performance
    
    for arg in "${args_array[@]}"; do
        case "$arg" in
            --resources)
                show_scenarios="false"
                show_system="false"
                ;;
            --scenarios)
                show_resources="false"
                show_system="false"
                ;;
            --fast)
                use_fast="true"
                ;;
            --no-fast|--detailed)
                use_fast="false"
                ;;
            *)
                cli::format_error "$output_format" "Unknown option: $arg"
                show_status_help
                return 1
                ;;
        esac
    done
    
    # Collect data
    local system_data=""
    local resource_data=""
    local scenario_data=""
    
    if [[ "$show_system" == "true" ]]; then
        system_data=$(get_system_data)
    fi
    
    if [[ "$show_resources" == "true" ]]; then
        resource_data=$(get_resource_data "$verbose" "$use_fast")
    fi
    
    if [[ "$show_scenarios" == "true" ]]; then
        scenario_data=$(get_scenario_data "$verbose")
    fi
    
    # Format output using standardized format.sh
    if [[ "$output_format" == "json" ]]; then
        # Build key-value pairs for format.sh
        local kv_pairs=()
        
        if [[ "$show_system" == "true" ]]; then
            # Extract system data
            local memory disk load docker
            memory=$(echo "$system_data" | grep "^memory:" | cut -d: -f2)
            disk=$(echo "$system_data" | grep "^disk:" | cut -d: -f2)
            load=$(echo "$system_data" | grep "^load:" | cut -d: -f2-)
            docker=$(echo "$system_data" | grep "^docker:" | cut -d: -f2)
            
            kv_pairs+=("system_memory_usage" "${memory}%")
            kv_pairs+=("system_disk_usage" "${disk}%")
            kv_pairs+=("system_load_average" "$load")
            kv_pairs+=("system_docker" "$docker")
        fi
        
        if [[ "$show_resources" == "true" ]]; then
            # Extract resource data
            local enabled running healthy
            enabled=$(echo "$resource_data" | grep "^enabled:" | cut -d: -f2)
            running=$(echo "$resource_data" | grep "^running:" | cut -d: -f2)
            healthy=$(echo "$resource_data" | grep "^healthy:" | cut -d: -f2)
            
            kv_pairs+=("resources_enabled" "$enabled")
            kv_pairs+=("resources_running" "$running")
            kv_pairs+=("resources_healthy" "$healthy")
            
            # Include detailed resource list when verbose
            if [[ "$verbose" == "true" ]]; then
                local resource_list=()
                while IFS= read -r line; do
                    if [[ "$line" =~ ^item:([^:]+):(.+)$ ]]; then
                        local res_name="${BASH_REMATCH[1]}"
                        local res_status="${BASH_REMATCH[2]}"
                        resource_list+=("$res_name:$res_status")
                    fi
                done <<< "$resource_data"
                
                if [[ ${#resource_list[@]} -gt 0 ]]; then
                    kv_pairs+=("resources_list" "$(IFS=,; echo "${resource_list[*]}")")
                fi
            fi
        fi
        
        if [[ "$show_scenarios" == "true" ]]; then
            # Extract app data
            local total running
            total=$(echo "$scenario_data" | grep "^total:" | cut -d: -f2)
            running=$(echo "$scenario_data" | grep "^running:" | cut -d: -f2)
            
            kv_pairs+=("scenarios_total" "$total")
            kv_pairs+=("scenarios_running" "$running")
        fi
        
        # Add overall health status
        local health_status="healthy"
        if ! docker info >/dev/null 2>&1; then
            health_status="warning"
        fi
        if [[ "$show_scenarios" == "true" ]] && [[ -z "$scenario_data" ]]; then
            health_status="warning"
        fi
        
        kv_pairs+=("health_status" "$health_status")
        
        # Output using format.sh
        format::output json "kv" "${kv_pairs[@]}"
    else
        # Text output
        cli::format_header text "🚀 Vrooli System Status"
        
        if [[ "$show_system" == "true" ]]; then
            cli::format_header text "📊 System Metrics"
            format_system_data text "$system_data" || true
            echo ""
        fi
        
        if [[ "$show_resources" == "true" ]]; then
            cli::format_header text "🔧 Resources"
            format_component_data text resources "$resource_data" || true
            echo ""
        fi
        
        if [[ "$show_scenarios" == "true" ]]; then
            cli::format_header text "📦 Scenarios"
            format_component_data text scenarios "$scenario_data" || true
            echo ""
        fi
        
        # Overall health status
        local health_status="healthy"
        local health_message="System is healthy"
        
        if ! docker info >/dev/null 2>&1; then
            health_status="warning"
            health_message="Docker unavailable"
        fi
        
        if [[ "$show_scenarios" == "true" ]] && [[ -z "$scenario_data" ]]; then
            health_status="warning"
            health_message="API offline"
        fi
        
        cli::format_status text "$health_status" "$health_message"
    fi
}

# Main entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    show_status "$@"
fi