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
    
    # Check for optional flags
    local clean_stale=false
    local allow_skip_missing_runtime=false
    local manage_runtime=false
    local had_prior_allow_var=false
    local prior_allow_value=""
    local had_prior_manage_var=false
    local prior_manage_value=""

    if [[ -n "${TEST_ALLOW_SKIP_MISSING_RUNTIME+x}" ]]; then
        had_prior_allow_var=true
        prior_allow_value="${TEST_ALLOW_SKIP_MISSING_RUNTIME}"
    fi

    if [[ -n "${TEST_MANAGE_RUNTIME+x}" ]]; then
        had_prior_manage_var=true
        prior_manage_value="${TEST_MANAGE_RUNTIME}"
    fi

    local -a remaining_args=()
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --clean-stale)
                clean_stale=true
                shift
                ;;
            --allow-skip-missing-runtime)
                allow_skip_missing_runtime=true
                shift
                ;;
            --manage-runtime)
                manage_runtime=true
                shift
                ;;
            *)
                remaining_args+=("$1")
                shift
                ;;
        esac
    done

    if [[ "$allow_skip_missing_runtime" == "true" && "$manage_runtime" == "true" ]]; then
        log::warning "⚠️  --manage-runtime overrides --allow-skip-missing-runtime"
        allow_skip_missing_runtime=false
    fi
    
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
    
    # Run stale lock cleanup if requested
    if [[ "$clean_stale" == "true" ]]; then
        log::info "🧹 Cleaning stale locks before starting scenario..."
        
        # Source clean commands for lock cleanup functionality
        if [[ -f "${var_ROOT_DIR}/cli/commands/clean-commands.sh" ]]; then
            # Source the clean functions
            source "${var_ROOT_DIR}/cli/commands/clean-commands.sh"
            
            # Run the lock cleanup (not dry-run)
            clean::stale_locks 2>&1 | while IFS= read -r line; do
                log::info "  $line"
            done
            
            log::success "✅ Stale lock cleanup completed"
        else
            log::warning "⚠️  Clean commands not found - skipping stale lock cleanup"
        fi
    fi
    
    # Set up logging for the scenario lifecycle execution
    local lifecycle_log="${HOME}/.vrooli/logs/${scenario_name}.log"
    mkdir -p "$(dirname "$lifecycle_log")" 2>/dev/null || true

    if ! ( : > "$lifecycle_log" ) 2>/dev/null; then
        lifecycle_log="${scenario_path}/logs/${scenario_name}.lifecycle.log"
        mkdir -p "$(dirname "$lifecycle_log")"
        : > "$lifecycle_log"
    fi
    
    # Call lifecycle.sh directly, capturing output to both console and log file
    log::info "Running scenario '$scenario_name' with direct lifecycle execution"
    
    # Optionally allow skipping runtime-dependent phases (tests only)
    if [[ "$allow_skip_missing_runtime" == "true" ]]; then
        export TEST_ALLOW_SKIP_MISSING_RUNTIME="true"
    fi

    if [[ "$manage_runtime" == "true" ]]; then
        export TEST_MANAGE_RUNTIME="true"
    fi

    # Use tee to show output on console AND write to log file
    # This preserves real-time output while capturing for later review
    "${SCRIPT_DIR}/../utils/lifecycle.sh" "$scenario_name" "$phase" "${remaining_args[@]}" 2>&1 | tee -a "$lifecycle_log"
    local run_exit="${PIPESTATUS[0]}"

    if [[ "$allow_skip_missing_runtime" == "true" ]]; then
        if [[ "$had_prior_allow_var" == "true" ]]; then
            export TEST_ALLOW_SKIP_MISSING_RUNTIME="${prior_allow_value}"
        else
            unset TEST_ALLOW_SKIP_MISSING_RUNTIME || true
        fi
    fi

    if [[ "$manage_runtime" == "true" ]]; then
        if [[ "$had_prior_manage_var" == "true" ]]; then
            export TEST_MANAGE_RUNTIME="${prior_manage_value}"
        else
            unset TEST_MANAGE_RUNTIME || true
        fi
    fi

    return "$run_exit"
}
scenario::list() {
    local json_output=false
    local include_ports=false
    local -a positional=()

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                json_output=true
                ;;
            --include-ports)
                include_ports=true
                ;;
            *)
                positional+=("$1")
                ;;
        esac
        shift
    done

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
            
            local ports_json="[]"
            if [[ "$include_ports" == "true" ]]; then
                local port_dir="$HOME/.vrooli/processes/scenarios/${name}"
                if compgen -G "${port_dir}"'/*.json' >/dev/null 2>&1; then
                    local ports_result
                    ports_result=$(scenario::ports::get_all "$name" true 2>/dev/null || true)
                    if [[ -n "$ports_result" ]]; then
                        ports_json=$(echo "$ports_result" | jq '.ports // []' 2>/dev/null || echo "[]")
                    fi
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
                    --arg include_ports "$include_ports" \
                    --argjson ports "$ports_json" \
                    '{
                        name: $name,
                        description: $description,
                        version: $version,
                        status: $status,
                        tags: (if $tags == "" then [] else ($tags | split(",")) end),
                        path: $path,
                        ports: (if $include_ports == "true" then $ports else [] end)
                    }')
                scenarios_json=$(echo "$scenarios_json" | jq ". += [$scenario_obj]")
            else
                # Build text output
                local line
                if [[ -n "$description" ]]; then
                    line="  • $name - $description"
                else
                    line="  • $name"
                fi

                if [[ "$include_ports" == "true" && "$ports_json" != "[]" ]]; then
                    local ports_text
                    ports_text=$(echo "$ports_json" | jq -r 'map("\(.key)=\(.port)") | join(", ")' 2>/dev/null || echo "")
                    if [[ -n "$ports_text" ]]; then
                        line+=" (ports: $ports_text)"
                    fi
                fi

                text_output+="$line"$'\n'
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
