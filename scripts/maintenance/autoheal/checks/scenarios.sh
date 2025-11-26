#!/usr/bin/env bash
# =============================================================================
# Scenario Health Checks
# Application and microservice monitoring
# =============================================================================

find_scenario_dir() {
    local name="$1"
    local -a candidates=()

    if [[ -n "${VROOLI_ROOT:-}" ]]; then
        candidates+=("${VROOLI_ROOT}/scenarios/${name}")
    fi
    candidates+=(
        "${HOME}/Vrooli/scenarios/${name}"
        "${PWD}/scenarios/${name}"
    )

    local candidate
    for candidate in "${candidates[@]}"; do
        if [[ -d "$candidate" ]]; then
            printf '%s\n' "$candidate"
            return 0
        fi
    done
    return 1
}

collect_scenario_ports() {
    local name="$1"
    local proc_dir="${HOME}/.vrooli/processes/scenarios/${name}"
    declare -A seen_ports=()

    if [[ -d "$proc_dir" ]]; then
        local meta port
        shopt -s nullglob
        for meta in "$proc_dir"/*.json; do
            [[ -f "$meta" ]] || continue
            port=$(jq -r 'try .port catch ""' "$meta" 2>/dev/null || true)
            if [[ -n "$port" && "$port" != "null" && "$port" =~ ^[0-9]+$ ]]; then
                seen_ports["$port"]=1
            fi
        done
        shopt -u nullglob
    fi

    local scenario_dir service_file
    if scenario_dir=$(find_scenario_dir "$name" 2>/dev/null); then
        service_file="$scenario_dir/.vrooli/service.json"
        if [[ -f "$service_file" ]]; then
            while IFS= read -r port; do
                if [[ -n "$port" && "$port" =~ ^[0-9]+$ ]]; then
                    seen_ports["$port"]=1
                fi
            done < <(jq -r '.ports? | to_entries[] | select(.value.port != null) | .value.port' "$service_file" 2>/dev/null || true)
        fi
    fi

    local port
    for port in "${!seen_ports[@]}"; do
        printf '%s\n' "$port"
    done
}

append_scenario_logs() {
    set +e
    local scenario="$1"
    local base_log="${HOME}/.vrooli/logs/${scenario}.log"
    local lifecycle_dir="${HOME}/.vrooli/logs/scenarios/${scenario}"

    if [[ -f "$base_log" ]]; then
        log "DEBUG" "Lifecycle tail (last 50 lines) from $base_log"
        { tail -n 50 "$base_log" 2>/dev/null || true; } | sed 's/^/    /' | log_stream "TRACE" || true
    fi

    if [[ -d "$lifecycle_dir" ]]; then
        local log_file
        shopt -s nullglob
        local count=0
        for log_file in "$lifecycle_dir"/*.log; do
            [[ -f "$log_file" ]] || continue
            ((count++))
            if ((count > 5)); then
                log "DEBUG" "Additional background logs suppressed (showing first 5)"
                break
            fi
            log "DEBUG" "Background tail (last 200 lines) from $log_file"
            { tail -n 200 "$log_file" 2>/dev/null || true; } | sed 's/^/    /' | log_stream "TRACE" || true
        done
        shopt -u nullglob
    fi
    set -e
}

scenario_status_fields() {
    local info="$1"
    local __state_var="$2"
    local __api_var="$3"
    local __ui_var="$4"

    local parsed_state parsed_api parsed_ui
    parsed_state=$(jq -r '.scenario_data.status // "unknown"' <<<"$info")
    parsed_api=$(jq -r '.diagnostics.responsiveness.api.available // empty' <<<"$info")
    parsed_ui=$(jq -r '.diagnostics.responsiveness.ui.available // empty' <<<"$info")

    printf -v "$__state_var" '%s' "$parsed_state"
    printf -v "$__api_var" '%s' "$parsed_api"
    printf -v "$__ui_var" '%s' "$parsed_ui"
}

scenario_requires_restart() {
    local state="$1" api_ok="$2" ui_ok="$3"

    if [[ "$state" != "running" ]]; then
        return 0
    fi
    if [[ -n "$api_ok" && "$api_ok" != "true" ]]; then
        return 0
    fi
    if [[ -n "$ui_ok" && "$ui_ok" != "true" ]]; then
        return 0
    fi
    return 1
}

check_scenario() {
    local name="$1"
    local info state="" api_ok="" ui_ok=""

    if ! run_with_timeout_capture info vrooli scenario status "$name" --json; then
        log "WARN" "Scenario '$name' status check failed; attempting restart"
        restart_scenario "$name"
        return
    fi

    scenario_status_fields "$info" state api_ok ui_ok
    log "DEBUG" "Scenario '$name' status check: state=$state api=$api_ok ui=$ui_ok"

    if scenario_requires_restart "$state" "$api_ok" "$ui_ok"; then
        restart_scenario "$name" "$state" "$api_ok" "$ui_ok"
    fi
}

restart_scenario() {
    local name="$1"
    local state="${2:-unknown}" api_ok="${3:-unknown}" ui_ok="${4:-unknown}"

    log "INFO" "Restarting scenario '$name' (state=$state api=$api_ok ui=$ui_ok)"

    if ! run_with_timeout_stream vrooli scenario stop "$name" >>"$LOG_FILE" 2>&1; then
        log "DEBUG" "Stop command for '$name' returned non-zero, continuing"
    fi

    # Call cleanup function from recovery/cleanup.sh
    cleanup_scenario_ports "$name"

    if ! run_with_timeout_stream vrooli scenario start "$name" --clean-stale >>"$LOG_FILE" 2>&1; then
        log "ERROR" "Failed to start scenario '$name'"
        append_scenario_logs "$name"
        return
    fi

    log "INFO" "Scenario '$name' restart requested"

    if [[ "$VERIFY_DELAY" =~ ^[0-9]+$ && "$VERIFY_DELAY" -gt 0 ]]; then
        log "INFO" "Waiting ${VERIFY_DELAY}s before verification for '$name'"
        sleep "$VERIFY_DELAY"

        local verify_info verify_state="" verify_api="" verify_ui=""
        if run_with_timeout_capture verify_info vrooli scenario status "$name" --json; then
            scenario_status_fields "$verify_info" verify_state verify_api verify_ui
            if scenario_requires_restart "$verify_state" "$verify_api" "$verify_ui"; then
                log "ERROR" "Scenario '$name' still failing after restart (state=$verify_state api=$verify_api ui=$verify_ui)"
                append_scenario_logs "$name"
            else
                log "INFO" "Scenario '$name' reported healthy after restart"
            fi
        else
            log "WARN" "Unable to verify scenario '$name' after restart"
        fi
    fi
}
