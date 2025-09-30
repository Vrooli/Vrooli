#!/usr/bin/env bash
set -euo pipefail

# Ensure common paths exist for cron/systemd invocations
PATH="/usr/local/bin:/usr/bin:/bin:${PATH:-}"

# Configuration (override via environment variables)
GRACE_SECONDS="${VROOLI_AUTOHEAL_GRACE_SECONDS:-60}"
LOCK_FILE="${VROOLI_AUTOHEAL_LOCK_FILE:-/tmp/vrooli-autoheal.lock}"
LOG_FILE="${VROOLI_AUTOHEAL_LOG_FILE:-/var/log/vrooli-autoheal.log}"
RESOURCE_LIST="${VROOLI_AUTOHEAL_RESOURCES:-postgres,redis,qdrant}"
SCENARIO_LIST="${VROOLI_AUTOHEAL_SCENARIOS:-app-monitor,system-monitor,ecosystem-manager,maintenance-orchestrator,app-issue-tracker,vrooli-orchestrator}"
VERIFY_DELAY="${VROOLI_AUTOHEAL_VERIFY_DELAY:-30}"

log() {
    local level="$1"
    shift
    printf '%s :: [%s] %s\n' "$(date -Is)" "$level" "$*" | tee -a "$LOG_FILE" >/dev/null
}

log_stream() {
    local level="$1"
    while IFS= read -r line; do
        [[ -z "$line" ]] && continue
        log "$level" "$line"
    done
    return 0
}

fatal() {
    log "ERROR" "$*"
    exit 1
}

ensure_prereqs() {
    command -v vrooli >/dev/null 2>&1 || fatal "vrooli CLI not found in PATH"
    command -v jq >/dev/null 2>&1 || fatal "jq is required but not found in PATH"
    local target_log="$LOG_FILE"
    local fallback_log="${HOME}/.vrooli/logs/vrooli-autoheal.log"

    if ! mkdir -p "$(dirname "$target_log")" 2>/dev/null || ! touch "$target_log" 2>/dev/null; then
        LOG_FILE="$fallback_log"
        mkdir -p "$(dirname "$LOG_FILE")" 2>/dev/null || true
        if ! touch "$LOG_FILE" 2>/dev/null; then
            printf '%s\n' "ERROR: Unable to create log file at '$target_log' or fallback '$LOG_FILE'" >&2
            exit 1
        fi
        printf '%s :: [WARN] Falling back to log file %s (unable to write %s)\n' "$(date -Is)" "$LOG_FILE" "$target_log" >>"$LOG_FILE"
    else
        LOG_FILE="$target_log"
    fi
}

acquire_lock() {
    if command -v flock >/dev/null 2>&1; then
        exec 9>"$LOCK_FILE"
        if ! flock -n 9; then
            log "DEBUG" "Another autoheal run is already in progress"
            return 1
        fi
        LOCK_METHOD="flock"
    else
        if ( set -o noclobber; : >"$LOCK_FILE" ) 2>/dev/null; then
            LOCK_METHOD="noclobber"
            trap cleanup_lock EXIT
        else
            log "DEBUG" "Another autoheal run is already in progress"
            return 1
        fi
    fi
    return 0
}

cleanup_lock() {
    if [[ "${LOCK_METHOD:-}" == "noclobber" ]]; then
        rm -f "$LOCK_FILE"
    fi
}

trim_token() {
    printf '%s' "$1" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//'
}

parse_list() {
    local raw="$1"
    local IFS=',' token trimmed
    for token in $raw; do
        trimmed=$(trim_token "$token")
        [[ -n "$trimmed" ]] && printf '%s\n' "$trimmed"
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

check_resource() {
    local name="$1"
    local info running healthy
    if ! info=$(vrooli resource status "$name" --json 2>/dev/null); then
        log "WARN" "Failed to read status for resource '$name'"
        return
    fi
    running=$(jq -r '.running // false' <<<"$info")
    healthy=$(jq -r '.healthy // false' <<<"$info")

    local total_instances healthy_instances
    total_instances=$(jq -r '.total_instances // empty' <<<"$info")
    healthy_instances=$(jq -r '.healthy_instances // empty' <<<"$info")

    if [[ "$healthy" != "true" && -n "$total_instances" && -n "$healthy_instances" ]]; then
        if (( healthy_instances > 0 )); then
            log "WARN" "Resource '$name' degraded but $healthy_instances/$total_instances instances healthy; skipping restart"
            healthy="true"
        fi
    fi
    if [[ "$running" != "true" || "$healthy" != "true" ]]; then
        log "INFO" "Restarting resource '$name' (running=$running healthy=$healthy)"
        if ! vrooli resource stop "$name" >/dev/null 2>&1; then
            log "DEBUG" "Stop command for '$name' returned non-zero, continuing"
        fi
        if ! vrooli resource start "$name" >>"$LOG_FILE" 2>&1; then
            log "ERROR" "Failed to start resource '$name'"
        else
            log "INFO" "Resource '$name' restart requested"
        fi
    fi
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
    if ! info=$(vrooli scenario status "$name" --json 2>/dev/null); then
        log "WARN" "Failed to read status for scenario '$name'"
        return
    fi
    scenario_status_fields "$info" state api_ok ui_ok
    log "DEBUG" "Scenario '$name' status check: state=$state api=$api_ok ui=$ui_ok"

    if scenario_requires_restart "$state" "$api_ok" "$ui_ok"; then
        log "INFO" "Restarting scenario '$name' (state=$state api=$api_ok ui=$ui_ok)"
        if ! vrooli scenario stop "$name" >/dev/null 2>&1; then
            log "DEBUG" "Stop command for '$name' returned non-zero, continuing"
        fi
        if ! vrooli scenario start "$name" --clean-stale >>"$LOG_FILE" 2>&1; then
            log "ERROR" "Failed to start scenario '$name'"
            append_scenario_logs "$name"
        else
            log "INFO" "Scenario '$name' restart requested"
            if [[ "$VERIFY_DELAY" =~ ^[0-9]+$ && "$VERIFY_DELAY" -gt 0 ]]; then
                log "INFO" "Waiting ${VERIFY_DELAY}s before verification for '$name'"
                sleep "$VERIFY_DELAY"
                local verify_info verify_state="" verify_api="" verify_ui=""
                if verify_info=$(vrooli scenario status "$name" --json 2>/dev/null); then
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
        fi
    fi
}

main() {
    ensure_prereqs

    LOCK_METHOD=""
    if ! acquire_lock; then
        return 0
    fi

    log "INFO" "Starting autoheal run (grace=${GRACE_SECONDS}s)"
    sleep "$GRACE_SECONDS"

    local resources=() scenarios=()

    while IFS= read -r item; do
        resources+=("$item")
    done < <(parse_list "$RESOURCE_LIST")

    while IFS= read -r item; do
        scenarios+=("$item")
    done < <(parse_list "$SCENARIO_LIST")

    local item
    for item in "${resources[@]}"; do
        [[ -z "$item" ]] && continue
        check_resource "$item"
    done

    for item in "${scenarios[@]}"; do
        [[ -z "$item" ]] && continue
        check_scenario "$item"
    done

    log "INFO" "Autoheal run complete"
    cleanup_lock
}

main "$@"
