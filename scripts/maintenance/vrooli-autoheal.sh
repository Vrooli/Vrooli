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

log() {
    local level="$1"
    shift
    printf '%s :: [%s] %s\n' "$(date -Is)" "$level" "$*" | tee -a "$LOG_FILE" >/dev/null
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

check_scenario() {
    local name="$1"
    local info state api_ok ui_ok
    if ! info=$(vrooli scenario status "$name" --json 2>/dev/null); then
        log "WARN" "Failed to read status for scenario '$name'"
        return
    fi
    state=$(jq -r '.scenario_data.status // "unknown"' <<<"$info")
    api_ok=$(jq -r '.diagnostics.responsiveness.api.available // empty' <<<"$info")
    ui_ok=$(jq -r '.diagnostics.responsiveness.ui.available // empty' <<<"$info")

    local restart_required="false"
    if [[ "$state" != "running" ]]; then
        restart_required="true"
    fi
    if [[ -n "$api_ok" && "$api_ok" != "true" ]]; then
        restart_required="true"
    fi
    if [[ -n "$ui_ok" && "$ui_ok" != "true" ]]; then
        restart_required="true"
    fi

    if [[ "$restart_required" == "true" ]]; then
        log "INFO" "Restarting scenario '$name' (state=$state api=$api_ok ui=$ui_ok)"
        if ! vrooli scenario stop "$name" >/dev/null 2>&1; then
            log "DEBUG" "Stop command for '$name' returned non-zero, continuing"
        fi
        if ! vrooli scenario start "$name" --clean-stale >>"$LOG_FILE" 2>&1; then
            log "ERROR" "Failed to start scenario '$name'"
        else
            log "INFO" "Scenario '$name' restart requested"
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
