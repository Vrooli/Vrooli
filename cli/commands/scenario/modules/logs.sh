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
        log::info "  --clean             Remove orphaned log files"
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
    local clean_logs=false
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
            --clean)
                clean_logs=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done

    if [[ "$clean_logs" == "true" ]]; then
        scenario::logs::clean_orphaned_logs "$scenario_name"
        return $?
    fi
    
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

# Clean orphaned logs
scenario::logs::clean_orphaned_logs() {
    local scenario_name="$1"
    log::info "Searching for orphaned logs for scenario: ${scenario_name}..."

    local logs_dir="${HOME}/.vrooli/logs/scenarios/${scenario_name}"
    if [[ ! -d "$logs_dir" ]]; then
        log::info "No log directory found for scenario: $scenario_name. Nothing to clean."
        return 0
    fi

    # Find service.json
    local service_json=""
    if [[ -f "${APP_ROOT}/scenarios/${scenario_name}/.vrooli/service.json" ]]; then
        service_json="${APP_ROOT}/scenarios/${scenario_name}/.vrooli/service.json"
    elif [[ -f "${APP_ROOT}/scenarios/${scenario_name}/service.json" ]]; then
        service_json="${APP_ROOT}/scenarios/${scenario_name}/service.json"
    fi

    local expected_steps_str=""
    if [[ -n "$service_json" ]] && [[ -f "$service_json" ]] && command -v jq >/dev/null 2>&1; then
        expected_steps_str=$(jq -r '
            .lifecycle |
            to_entries[] |
            select(.value | type == "object") |
            select(.value.steps) |
            .key as $phase |
            .value.steps[]? |
            select(.background == true) |
            "\(.name):\($phase)"
        ' "$service_json" 2>/dev/null || true)
    fi

    local orphaned_logs=()
    shopt -s nullglob
    for log_file in "$logs_dir"/vrooli.*.log; do
        local basename
        basename=$(basename "$log_file")
        if [[ "$basename" =~ vrooli\.([^.]+)\.${scenario_name}\.(.+)\.log ]]; then
            local phase="${BASH_REMATCH[1]}"
            local step="${BASH_REMATCH[2]}"
            local step_id="${step}:${phase}"
            
            if [[ -n "$expected_steps_str" ]] && grep -q -x -F "$step_id" <<< "$expected_steps_str"; then
                : # valid
            else
                orphaned_logs+=("$log_file")
            fi
        else
            # File doesn't match naming convention, consider it orphaned
            orphaned_logs+=("$log_file")
        fi
    done
    shopt -u nullglob

    if [[ ${#orphaned_logs[@]} -eq 0 ]]; then
        log::success "No orphaned logs found."
        return 0
    fi

    log::info "Found ${#orphaned_logs[@]} orphaned log(s). Removing..."
    for log_file in "${orphaned_logs[@]}"; do
        # Also remove .bak file if it exists
        if rm -f "$log_file" "${log_file}.bak"; then
            log::info "  - Removed $(basename "$log_file") (and backup)"
        else
            log::warn "  - Failed to remove $(basename "$log_file")"
        fi
    done

    log::success "Orphaned log cleanup complete."
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

    # Find service.json
    local service_json=""
    if [[ -f "${APP_ROOT}/scenarios/${scenario_name}/.vrooli/service.json" ]]; then
        service_json="${APP_ROOT}/scenarios/${scenario_name}/.vrooli/service.json"
    elif [[ -f "${APP_ROOT}/scenarios/${scenario_name}/service.json" ]]; then
        service_json="${APP_ROOT}/scenarios/${scenario_name}/service.json"
    fi

    local expected_steps_str=""
    declare -A expected_steps_map=()
    if [[ -n "$service_json" ]] && [[ -f "$service_json" ]] && command -v jq >/dev/null 2>&1; then
        expected_steps_str=$(jq -r '
            .lifecycle |
            to_entries[] |
            select(.value | type == "object") |
            select(.value.steps) |
            .key as $phase |
            .value.steps[]? |
            select(.background == true) |
            "\(.name):\($phase)"
        ' "$service_json" 2>/dev/null || true)
        
        while IFS= read -r line; do
            [[ -n "$line" ]] && expected_steps_map["$line"]=1
        done <<< "$expected_steps_str"
    fi

    declare -A found_steps_map=()
    local orphaned_logs=()
    local valid_logs=()
    local has_logs=false

    shopt -s nullglob
    for log_file in "$logs_dir"/vrooli.*.log; do
        has_logs=true
        local basename
        basename=$(basename "$log_file")
        if [[ "$basename" =~ vrooli\.([^.]+)\.${scenario_name}\.(.+)\.log ]]; then
            local phase="${BASH_REMATCH[1]}"
            local step="${BASH_REMATCH[2]}"
            local step_id="${step}:${phase}"
            
            found_steps_map["$step_id"]=1

            if [[ -n "${expected_steps_map[$step_id]:-}" ]]; then
                valid_logs+=("$log_file")
            else
                orphaned_logs+=("$log_file")
            fi
        else
            orphaned_logs+=("$log_file")
        fi
    done
    shopt -u nullglob

    if ! $has_logs; then
        echo "  (no background step logs found)"
    fi

    for log_file in "${valid_logs[@]}"; do
        local basename
        basename=$(basename "$log_file")
        if [[ "$basename" =~ vrooli\.([^.]+)\.${scenario_name}\.(.+)\.log ]]; then
            local phase="${BASH_REMATCH[1]}"
            local step="${BASH_REMATCH[2]}"
            echo "  ✅ ${step} (${phase})"
            echo "     View: vrooli scenario logs ${scenario_name} --step ${step}"
            echo ""
        fi
    done

    if [[ -n "$expected_steps_str" ]]; then
        while IFS= read -r expected; do
            if [[ -n "$expected" ]] && [[ -z "${found_steps_map[$expected]:-}" ]]; then
                IFS=':' read -r step phase <<< "$expected"
                echo "  ⚠️  ${step} (${phase}) - [NOT FOUND]"
                echo "     Expected but missing - step may not have been reached"
                echo "     Would view with: vrooli scenario logs ${scenario_name} --step ${step}"
                echo ""
            fi
        done <<< "$expected_steps_str"
    fi

    if [[ ${#orphaned_logs[@]} -gt 0 ]]; then
        echo "────────────────────────────────────────────────────────"
        echo "🗑️  ORPHANED BACKGROUND STEP LOGS:"
        echo ""
        for log_file in "${orphaned_logs[@]}"; do
            local basename
            basename=$(basename "$log_file")
            local extracted_step=""
            if [[ "$basename" =~ vrooli\.([^.]+)\.${scenario_name}\.(.+)\.log ]]; then
                local phase="${BASH_REMATCH[1]}"
                local step="${BASH_REMATCH[2]}"
                extracted_step="$step"
                echo "  🚫 ${step} (${phase}) - [ORPHANED]"
                echo "     This step is not defined as a background task in service.json"
            else
                extracted_step=$(echo "$basename" | sed -E "s/vrooli\.[^.]+\.${scenario_name}\.(.+)\.log/\1/")
                echo "  🚫 $(basename "$log_file") - [ORPHANED-NAMING]"
                echo "     Log file does not match expected naming convention"
            fi
            echo "     View: vrooli scenario logs ${scenario_name} --step ${extracted_step}"
            echo ""
        done
        echo "💡 Tip: Clean up orphaned logs with: vrooli scenario logs ${scenario_name} --clean"
        echo ""
    fi
    
    echo "💡 Tips:"
    echo "  • Use --runtime to view all background process logs"
    echo "  • Use --step <name> to view a specific background process log"
    echo "  • Use --follow or -f to watch logs in real-time"
    echo "  • Missing logs usually mean the step wasn't reached"
    echo "  • The lifecycle log above shows the complete execution sequence"
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