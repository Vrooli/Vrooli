#!/usr/bin/env bash
set -euo pipefail

# Get script directory for sourcing utils
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
source "${SCRIPT_DIR}/../utils/var.sh"
source "${SCRIPT_DIR}/../utils/log.sh"
source "${SCRIPT_DIR}/dependencies.sh"

scenario::run() {
    local scenario_name="$1"
    shift

    # Check for optional flags (must happen before phase parsing)
    local custom_path=""
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

    # Get the phase (default to 'develop' if not specified)
    local phase="${1:-develop}"
    shift || true

    local selection=""
    local -a remaining_args=()
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --path)
                custom_path="$2"
                shift 2
                ;;
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
                if [[ "$phase" == "test" && -z "$selection" && "$1" != "-"* ]]; then
                    selection="$1"
                    shift
                else
                    remaining_args+=("$1")
                    shift
                fi
                ;;
        esac
    done

    # Resolve scenario path (support both default and custom paths)
    local scenario_path
    if [[ -n "$custom_path" ]]; then
        # Custom path provided - make it absolute
        if [[ "$custom_path" = /* ]]; then
            scenario_path="$custom_path"
        else
            scenario_path="$(cd "$(dirname "$custom_path")" 2>/dev/null && pwd)/$(basename "$custom_path")"
        fi
    else
        # Default: look in standard scenarios directory
        scenario_path="${var_ROOT_DIR}/scenarios/${scenario_name}"
    fi

    if [[ ! -d "$scenario_path" ]]; then
        log::error "Scenario not found: $scenario_name (path: $scenario_path)"
        return 1
    fi

    if [[ ${#SCENARIO_DEPENDENCY_STACK[@]} -eq 0 ]]; then
        scenario::dependencies::ready_reset
    fi

    if scenario::dependencies::phase_requires_bootstrap "$phase"; then
        scenario::dependencies::stack_push "$scenario_name"
        if ! scenario::dependencies::ensure_started "$scenario_name" "$phase"; then
            scenario::dependencies::stack_pop "$scenario_name"
            return 1
        fi
        scenario::dependencies::stack_pop "$scenario_name"
    fi

    if [[ "$allow_skip_missing_runtime" == "true" && "$manage_runtime" == "true" ]]; then
        log::warning "⚠️  --manage-runtime overrides --allow-skip-missing-runtime"
        allow_skip_missing_runtime=false
    fi

    if [[ "$phase" == "test" && -n "$selection" ]]; then
        local -a valid_selections=(structure dependencies unit integration business performance all e2e)
        local is_valid=false
        for sel in "${valid_selections[@]}"; do
            if [[ "$selection" == "$sel" ]]; then
                is_valid=true
                break
            fi
        done

        if [[ "$is_valid" == "false" ]]; then
            log::error "Invalid test selector: $selection"
            log::info "Valid selections: ${valid_selections[*]}"
            return 1
        fi

        if [[ "$selection" == "e2e" ]]; then
            log::info "Note: 'e2e' runs the integration phase (current end-to-end coverage)."
            selection="integration"
        fi

        if [[ "$selection" == "all" ]]; then
            remaining_args+=("all")
        else
            remaining_args+=("$selection")
        fi
    fi
    
    # For develop phase, check if already running and healthy
    if [[ "$phase" == "develop" ]]; then
        # Source lifecycle.sh to get the idempotency functions
        source "${SCRIPT_DIR}/../utils/lifecycle.sh"
        
        # Check if scenario is already running and healthy
        if lifecycle::is_scenario_running "$scenario_name"; then
            if lifecycle::is_scenario_healthy "$scenario_name"; then
                # Even if running and healthy, check if setup is needed (stale code)
                # This ensures code changes trigger rebuild even when scenario is running
                cd "$scenario_path" || return 1

                # Source setup utilities for staleness detection
                source "${SCRIPT_DIR}/../utils/setup.sh" 2>/dev/null || true

                local force_setup="${FORCE_SETUP:-false}"
                local force_setup_target="${FORCE_SETUP_SCENARIO:-}"
                local force_setup_applies=false
                if [[ "$force_setup" == "true" ]]; then
                    if [[ -z "$force_setup_target" || "$force_setup_target" == "$scenario_name" ]]; then
                        # Only honor forced rebuilds for the scenario the user
                        # explicitly restarted so dependencies stay untouched.
                        force_setup_applies=true
                    fi
                fi
                if command -v setup::is_needed >/dev/null 2>&1; then
                    if setup::is_needed "$scenario_path"; then
                        log::warning "⚠️  Scenario running but code is stale (${SETUP_REASONS[*]:-binaries/bundles outdated}), restarting..."
                        lifecycle::stop_scenario_processes "$scenario_name"
                        sleep 2
                        # Continue to normal execution below
                    elif [[ "$force_setup_applies" == "true" ]]; then
                        log::info "🔄 Forced restart requested, stopping and rebuilding..."
                        lifecycle::stop_scenario_processes "$scenario_name"
                        sleep 2
                        # Continue to normal execution below
                    else
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

                        return 0  # Already running, healthy, and code is current
                    fi
                else
                    # setup::is_needed not available, fall back to old behavior
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

                    return 0  # Already running and healthy
                fi
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

            # Run the lock cleanup directly - NO PIPES
            # The function writes formatted output directly to stdout
            clean::stale_locks || {
                log::warning "Lock cleanup encountered errors but continuing"
            }

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

    # Export custom path if provided, so lifecycle.sh can use it
    if [[ -n "$custom_path" ]]; then
        export SCENARIO_CUSTOM_PATH="$scenario_path"
    fi

    # Use tee to show output on console AND write to log file
    # This preserves real-time output while capturing for later review
    "${SCRIPT_DIR}/../utils/lifecycle.sh" "$scenario_name" "$phase" "${remaining_args[@]}" 2>&1 | tee -a "$lifecycle_log"
    local run_exit="${PIPESTATUS[0]}"

    # Clean up custom path export
    if [[ -n "$custom_path" ]]; then
        unset SCENARIO_CUSTOM_PATH
    fi

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
