#!/usr/bin/env bash
set -euo pipefail

# Get script directory for sourcing utils
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
source "${SCRIPT_DIR}/../utils/var.sh"
source "${SCRIPT_DIR}/../utils/log.sh"

scenario::run() {
    local scenario_name="$1"
    shift
    
    local scenario_path="${var_ROOT_DIR}/scenarios/${scenario_name}"
    
    if [[ ! -d "$scenario_path" ]]; then
        log::error "Scenario not found: $scenario_name"
        return 1
    fi
    
    # Get the phase (default to 'develop' if not specified)
    local phase="${1:-develop}"
    shift || true
    
    # For develop phase, check if already running and healthy
    if [[ "$phase" == "develop" ]]; then
        # Source lifecycle.sh to get the idempotency functions
        source "${SCRIPT_DIR}/../utils/lifecycle.sh"
        
        # Check if scenario is already running and healthy
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
                # Continue to normal execution below
            fi
        fi
    fi
    
    # Set up logging for the scenario lifecycle execution
    local lifecycle_log="${HOME}/.vrooli/logs/${scenario_name}.log"
    mkdir -p "$(dirname "$lifecycle_log")"
    
    # Clear the log file for fresh execution (no confusion from old runs)
    > "$lifecycle_log"
    
    # Call lifecycle.sh directly, capturing output to both console and log file
    log::info "Running scenario '$scenario_name' with direct lifecycle execution"
    
    # Use tee to show output on console AND write to log file
    # This preserves real-time output while capturing for later review
    "${SCRIPT_DIR}/../utils/lifecycle.sh" "$scenario_name" "$phase" "$@" 2>&1 | tee -a "$lifecycle_log"
    
    # Preserve the exit code from the pipeline
    return "${PIPESTATUS[0]}"
}
scenario::list() {
    local json_output=false
    
    # Check for --json flag
    if [[ "${1:-}" == "--json" ]]; then
        json_output=true
    fi
    
    # Collect scenario data
    local scenarios_json="[]"
    local text_output=""
    
    for scenario in "${var_ROOT_DIR}"/scenarios/*/; do
        if [[ -d "$scenario" ]]; then
            local name="${scenario%/}"
            name="${name##*/}"
            local service_json="${scenario}/.vrooli/service.json"
            local description=""
            local version=""
            local status="available"
            local tags=""
            
            if [[ -f "$service_json" ]]; then
                description=$(jq -r '.service.description // ""' "$service_json" 2>/dev/null || echo "")
                version=$(jq -r '.service.version // ""' "$service_json" 2>/dev/null || echo "")
                tags=$(jq -r '.service.tags // [] | join(",")' "$service_json" 2>/dev/null || echo "")
            fi
            
            # Check if scenario is running (reuse lifecycle check if available)
            if command -v lifecycle::is_scenario_running &>/dev/null; then
                source "${SCRIPT_DIR}/../utils/lifecycle.sh" 2>/dev/null || true
                if lifecycle::is_scenario_running "$name" 2>/dev/null; then
                    status="running"
                fi
            fi
            
            if [[ "$json_output" == "true" ]]; then
                # Build JSON object for this scenario
                local scenario_obj=$(jq -n \
                    --arg name "$name" \
                    --arg description "$description" \
                    --arg version "$version" \
                    --arg status "$status" \
                    --arg tags "$tags" \
                    --arg path "${scenario}" \
                    '{
                        name: $name,
                        description: $description,
                        version: $version,
                        status: $status,
                        tags: (if $tags == "" then [] else ($tags | split(",")) end),
                        path: $path
                    }')
                scenarios_json=$(echo "$scenarios_json" | jq ". += [$scenario_obj]")
            else
                # Build text output
                if [[ -n "$description" ]]; then
                    text_output+="  • $name - $description"$'\n'
                else
                    text_output+="  • $name"$'\n'
                fi
            fi
        fi
    done
    
    # Output results
    if [[ "$json_output" == "true" ]]; then
        # Create final JSON response with metadata
        local total_count=$(echo "$scenarios_json" | jq 'length')
        local running_count=$(echo "$scenarios_json" | jq '[.[] | select(.status == "running")] | length')
        
        jq -n \
            --argjson scenarios "$scenarios_json" \
            --argjson total "$total_count" \
            --argjson running "$running_count" \
            '{
                success: true,
                summary: {
                    total_scenarios: $total,
                    running: $running,
                    available: ($total - $running)
                },
                scenarios: $scenarios
            }'
    else
        log::info "Available scenarios:"
        echo -n "$text_output"
    fi
}

scenario::test() {
    local scenario_name="$1"
    shift
    scenario::run "$scenario_name" test "$@"
}