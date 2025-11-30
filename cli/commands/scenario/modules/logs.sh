#!/usr/bin/env bash
# Scenario Log Management Module
# Handles log viewing, following, and discovery

set -euo pipefail

# View logs for a scenario
scenario::logs::view() {
    local scenario_name="${1:-}"
    [[ -z "$scenario_name" ]] && { 
        log::error "Scenario name required"
        log::info "Usage: vrooli scenario logs <name> [options]"
        log::info "Options:"
        log::info "  --follow, -f        Follow log output in real-time"
        log::info "  --step <name>       View specific background step log"
        log::info "  --runtime           View all background process logs"
        log::info "  --lifecycle         View lifecycle log (default behavior)"
        log::info "  --force-follow      Stream even in non-interactive environments"
        log::info ""
        log::info "Available scenarios with logs:"
        ls -1 "${HOME}/.vrooli/logs/scenarios/" 2>/dev/null || echo "  (none found)"
        return 1
    }
    shift
    
    # Parse flags
    local follow=false
    local force_follow=false
    local step_name=""
    local show_lifecycle=false
    local show_runtime=false
    local show_previous=false
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
            --previous)
                show_previous=true
                shift
                ;;
            --force-follow)
                follow=true
                force_follow=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Show runtime logs if requested
    if [[ "$show_runtime" == "true" ]]; then
        scenario::logs::show_runtime "$scenario_name" "$follow" "$force_follow"
        return $?
    fi

    # Show specific step log if requested
    if [[ -n "$step_name" ]]; then
        scenario::logs::show_step "$scenario_name" "$step_name" "$follow" "$force_follow" "$show_previous"
        return $?
    fi

    # Default behavior: show lifecycle log with discovery information
    scenario::logs::show_lifecycle "$scenario_name" "$follow" "$force_follow"
}

# Show runtime logs for a scenario
scenario::logs::show_runtime() {
    local scenario_name="$1"
    local follow="$2"
    local force_follow="${3:-false}"
    
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
        if scenario::logs::can_stream "$force_follow"; then
            log::info "Following runtime logs for scenario: $scenario_name"
            log::info "Press Ctrl+C to stop viewing"
            echo ""
            tail -f "$logs_dir"/*.log
            return $?
        fi
        scenario::logs::warn_snapshot_fallback
    fi

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
}

# Show specific step log for a scenario
scenario::logs::show_step() {
    local scenario_name="$1"
    local step_name="$2"
    local follow="$3"
    local force_follow="${4:-false}"
    local show_previous="${5:-false}"
    
    local logs_dir="${HOME}/.vrooli/logs/scenarios/${scenario_name}"
    
    # Determine the log file suffix
    local log_suffix=".log"
    if [[ "$show_previous" == "true" ]]; then
        log_suffix=".log.bak"
    fi

    # Find log files matching the step name
    local step_log=""
    shopt -s nullglob
    for log_file in "$logs_dir"/vrooli.*."${scenario_name}"."${step_name}"${log_suffix}; do
        if [[ -f "$log_file" ]]; then
            step_log="$log_file"
            break
        fi
    done
    shopt -u nullglob
    
    if [[ -z "$step_log" ]]; then
        if [[ "$show_previous" == "true" ]]; then
            log::error "No previous log found for step '$step_name'"
            log::info "A backup is only created when a scenario is started/restarted."
            return 1
        fi
        log::error "No log found for step '$step_name'"
        log::info "This could mean:"
        log::info "  • The step hasn't been reached yet (check earlier steps)"
        log::info "  • The step isn't a background process (check --lifecycle)"
        log::info "  • The step name is incorrect"
        echo ""
        log::info "Available background step logs:"
        scenario::logs::list_available_step_logs "$scenario_name"
        return 1
    fi
    
    if [[ "$follow" == "true" ]]; then
        if scenario::logs::can_stream "$force_follow"; then
            log::info "Following log for step '$step_name' in scenario: $scenario_name"
            log::info "Press Ctrl+C to stop viewing"
            echo ""
            tail -f "$step_log"
            return $?
        fi
        scenario::logs::warn_snapshot_fallback
    fi

    log::info "Showing recent log for step '$step_name' in scenario: $scenario_name"
    echo ""
    echo "==> $(basename "$step_log") <=="
    tail -100 "$step_log"
    echo ""
}

# Show lifecycle log with discovery information
scenario::logs::show_lifecycle() {
    local scenario_name="$1"
    local follow="$2"
    local force_follow="${3:-false}"
    
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
        if scenario::logs::can_stream "$force_follow"; then
            log::info "Following lifecycle log for scenario: $scenario_name"
            log::info "Press Ctrl+C to stop viewing"
            echo ""
            tail -f "$lifecycle_log"
            return $?
        fi
        scenario::logs::warn_snapshot_fallback
    fi

    log::info "Showing recent lifecycle execution for scenario: $scenario_name"
    echo ""
    echo "==> Lifecycle execution log <=="
    echo "────────────────────────────────────────────────────────"
    # Show more lines from lifecycle log to capture full execution flow
    tail -100 "$lifecycle_log"
    echo ""
    
    # Show discovery information
    scenario::logs::show_discovery "$scenario_name"
}

# Determine if streaming logs is allowed in the current environment
scenario::logs::can_stream() {
    local force_follow="${1:-false}"

    if [[ "$force_follow" == "true" ]]; then
        return 0
    fi

    [[ -t 1 ]]
}

# Emit a warning when falling back to a static snapshot instead of streaming
scenario::logs::warn_snapshot_fallback() {
    log::warn "Non-interactive environment detected; showing a static snapshot instead of streaming logs"
    log::info "Use --force-follow to stream anyway (may hang automation workflows)"
    echo ""
}

# Show log discovery information
scenario::logs::show_discovery() {
    local scenario_name="$1"
    local logs_dir="${HOME}/.vrooli/logs/scenarios/${scenario_name}"
    
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
        scenario::logs::check_expected_steps "$scenario_name" "$service_json" found_steps
    else
        # If we can't parse service.json, just note common missing steps
        scenario::logs::check_common_missing_steps "$scenario_name" "$logs_dir"
    fi
    
    echo "💡 Tips:"
    echo "  • Use --runtime to view all background process logs"
    echo "  • Use --step <name> to view a specific background process log"
    echo "  • Use --follow or -f to watch logs in real-time"
    echo "  • Missing logs usually mean the step wasn't reached"
    echo "  • The lifecycle log above shows the complete execution sequence"
}

# Check for expected background steps from service.json
scenario::logs::check_expected_steps() {
    local scenario_name="$1"
    local service_json="$2"
    local -n found_steps_ref=$3
    
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
            for fs in "${found_steps_ref[@]}"; do
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
}

# Check for common missing steps
scenario::logs::check_common_missing_steps() {
    local scenario_name="$1"
    local logs_dir="$2"
    
    if [[ ! -f "${logs_dir}/vrooli.develop.${scenario_name}.start-ui.log" ]] && \
       [[ -f "${logs_dir}/vrooli.develop.${scenario_name}.start-api.log" ]]; then
        echo "  ⚠️  start-ui (develop) - [NOT FOUND]"
        echo "     Expected but missing - step may not have been reached"
        echo "     Would view with: vrooli scenario logs ${scenario_name} --step start-ui"
        echo ""
    fi
}

# List available step logs for a scenario
scenario::logs::list_available_step_logs() {
    local scenario_name="$1"
    local logs_dir="${HOME}/.vrooli/logs/scenarios/${scenario_name}"
    
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
}
